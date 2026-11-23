package ask

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Keep it current (sb-ask(3); SB-1708, SB-1709).
const (
	MaxChanges        = 50
	MaxChangeDays     = 90
	DefaultReviewDays = 365
	ReviewCites       = 5
	ReviewWindow      = 30 * 24 * time.Hour
)

// ChangesResponse is what changed in a window, newest first.
type ChangesResponse struct {
	Since time.Time `json:"since"`
	Items []Result  `json:"items"`
}

// memberSpaces are the ids of the spaces the scope's user is a member of now, and whether
// the cell has spaces at all.
func (a Agent) memberSpaces(ctx context.Context, s tenancy.Scope) (map[string]bool, bool, error) {
	if a.Auth.Spaces == nil {
		return nil, false, nil
	}
	all, _, err := a.Auth.Spaces(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	if len(all) == 0 {
		return nil, false, nil
	}
	mine := map[string]bool{}
	for _, g := range s.Groups() {
		mine["group:"+g] = true
		mine[g] = true
	}
	out := map[string]bool{}
	for _, sp := range all {
		for _, m := range sp.Members {
			if mine[m] {
				out[sp.ID] = true
			}
		}
	}
	return out, true, nil
}

// Changes lists documents with a version first seen in the last days days, in the scope's
// space or, company-wide, in the person's own spaces, each checked live as them.
func (a Agent) Changes(ctx context.Context, s tenancy.Scope, days int) (ChangesResponse, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return ChangesResponse{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if days < 1 || days > MaxChangeDays {
		return ChangesResponse{}, fmt.Errorf("%w: since is 1d to %dd", ErrInvalid, MaxChangeDays)
	}
	if a.Killed == nil || a.Killed(ctx) || spaceKilled(a.SpaceKilled, s) {
		return ChangesResponse{}, ErrKilled
	}
	since := a.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	out := ChangesResponse{Since: since, Items: []Result{}}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{SeenSince: since})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	mine, spaced, err := a.memberSpaces(ctx, s)
	if err != nil {
		return out, err
	}
	var cand []ports.Chunk
	for _, c := range firsts(latest(chunks)) {
		if spaced && s.Space() == "company" && !anyOf(c.Spaces, mine) {
			continue // company-wide: only what the person's own spaces hold
		}
		cand = append(cand, c)
	}
	sort.Slice(cand, func(i, j int) bool {
		if !cand[i].Seen.Equal(cand[j].Seen) {
			return cand[i].Seen.After(cand[j].Seen)
		}
		return cand[i].ID < cand[j].ID
	})
	who, err := a.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, MaxChanges)
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	own := a.owners()
	for _, c := range res.Context {
		r := a.result(ctx, own, c)
		r.Snippet = ""
		out.Items = append(out.Items, r)
	}
	return out, nil
}

func anyOf(xs []string, set map[string]bool) bool {
	for _, x := range xs {
		if set[x] {
			return true
		}
	}
	return false
}

// ReviewSource is where answers' cites and wrong-citation marks come from: the traces.
type ReviewSource interface {
	Between(ctx context.Context, tenant string, from, to time.Time) ([]Trace, error)
	FeedbackBetween(ctx context.Context, tenant string, from, to time.Time) ([]Feedback, error)
}

// ReviewItem is a document to review, shown because the steward can open it now.
type ReviewItem struct {
	Result
	Cited   int `json:"cited"`
	Flagged int `json:"flagged"`
	// Wider is, for a note, how many of its readers are no longer members of the space
	// (SB-2004): a count, never who. Question is the note's question, to republish it.
	Wider    int    `json:"wider,omitempty"`
	Question string `json:"question,omitempty"`
}

// HiddenItem is a document to review the steward can't open: only its owner, if any.
type HiddenItem struct {
	Owner string `json:"owner,omitempty"`
}

// ReviewResponse is the stewards' review list.
type ReviewResponse struct {
	Items  []ReviewItem `json:"items"`
	Hidden []HiddenItem `json:"hidden"`
	// Personal is how many documents the space holds with personal data, an IBAN or a
	// card number (SB-1904): a count, never which.
	Personal int `json:"personal_data"`
}

// Review lists the documents a space holds that answers lean on but that are older than
// its review_after_days, and any marked as a wrong citation, for its stewards (SB-1709).
func (x Stewards) Review(ctx context.Context, s tenancy.Scope, name string, src ReviewSource, after int) (ReviewResponse, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant || name == "company" {
		return ReviewResponse{}, ErrNotSteward
	}
	stewards, _, ok := x.Facts(SpaceKey(name))
	if !ok {
		return ReviewResponse{}, ErrNotSteward
	}
	var groups []string
	for _, g := range s.Groups() {
		groups = append(groups, "group:"+g)
	}
	if x.Policy.Authorise(ctx, s, "review_space", policy.Entity{Type: "SpaceConfig", ID: name, Tenant: x.Agent.Tenant,
		Attrs: map[string]any{"stewards": stewards}}, map[string]any{"groups": groups}).Decision != policy.Allow {
		return ReviewResponse{}, ErrNotSteward
	}
	if after <= 0 {
		after = DefaultReviewDays
	}
	now := x.Now().UTC()
	traces, err := src.Between(ctx, x.Agent.Tenant, now.Add(-ReviewWindow), now)
	if err != nil {
		return ReviewResponse{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	fbs, err := src.FeedbackBetween(ctx, x.Agent.Tenant, now.Add(-ReviewWindow), now)
	if err != nil {
		return ReviewResponse{}, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
	}
	cited, flagged, refs := reviewCounts(traces, fbs)
	out := ReviewResponse{Items: []ReviewItem{}, Hidden: []HiddenItem{}}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	if out.Personal, err = x.personalIn(ctx, lk, name); err != nil {
		return out, err
	}
	widerBy, notes, err := x.widerNotes(ctx, lk, name)
	if err != nil {
		return out, err
	}
	if len(refs) == 0 && len(notes) == 0 {
		return out, nil
	}
	var chunks []ports.Chunk
	if len(refs) > 0 {
		if chunks, err = lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: refs}); err != nil {
			return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
	}
	id := SpaceKey(name)
	var cand []ports.Chunk
	in := map[string]bool{}
	for k, c := range firsts(latest(chunks)) {
		age, known := ageDays(c.Seen, now)
		stale := known && age > after && cited[k] >= ReviewCites
		if contains(c.Spaces, id) && (stale || flagged[k] > 0) {
			cand = append(cand, c)
			in[k] = true
		}
	}
	for _, c := range notes {
		if !in[c.Kind+":"+c.SourceID] {
			cand = append(cand, c)
		}
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	allowed := map[string]bool{}
	own := x.Agent.owners()
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		allowed[k] = true
		r := x.Agent.result(ctx, own, c)
		r.Snippet = ""
		it := ReviewItem{Result: r, Cited: cited[k], Flagged: flagged[k], Wider: widerBy[k]}
		if it.Wider > 0 {
			it.Question = strings.TrimSuffix(path.Base(c.SourceID), ".md")
		}
		out.Items = append(out.Items, it)
	}
	for _, c := range cand {
		if !allowed[c.Kind+":"+c.SourceID] {
			out.Hidden = append(out.Hidden, HiddenItem{Owner: c.Owner}) // never its title or cite
		}
	}
	sort.SliceStable(out.Items, func(i, j int) bool {
		a, b := out.Items[i], out.Items[j]
		if (a.Wider > 0) != (b.Wider > 0) {
			return a.Wider > 0 // a note read wider than its space comes first
		}
		return a.Flagged+a.Cited > b.Flagged+b.Cited
	})
	return out, nil
}

// reviewCounts counts, by document, the answers that cited it and the wrong-citation
// marks on it, and lists every document either names.
func reviewCounts(traces []Trace, fbs []Feedback) (cited, flagged map[string]int, refs []ports.SourceRef) {
	cited, flagged = map[string]int{}, map[string]int{}
	for _, t := range traces {
		seen := map[string]bool{}
		for _, c := range t.Response.Answer.Claims {
			for _, ci := range c.Cites {
				if k := docKey(ci.Cite); !seen[k] {
					seen[k] = true
					cited[k]++
				}
			}
		}
	}
	for _, f := range fbs {
		if f.WrongCitation != "" {
			flagged[docKey(f.WrongCitation)]++
		}
	}
	for k := range cited {
		kind, id, _ := strings.Cut(k, ":")
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	for k := range flagged {
		if cited[k] == 0 {
			kind, id, _ := strings.Cut(k, ":")
			refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
		}
	}
	return cited, flagged, refs
}

// personalIn counts the documents the space holds with personal data (SB-1904).
func (x Stewards) personalIn(ctx context.Context, lk ports.DocLookup, name string) (int, error) {
	personal, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Flag: ingest.FlagPersonal})
	if err != nil {
		return 0, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var withPII []ports.Chunk
	for _, c := range personal {
		if contains(c.Flags, ingest.FlagPersonal) {
			withPII = append(withPII, c)
		}
	}
	n := 0
	for _, c := range firsts(withPII) {
		if contains(c.Spaces, SpaceKey(name)) {
			n++
		}
	}
	return n, nil
}

// widerNotes are the space's notes whose readers are wider than the space is now
// (SB-2004), with how many readers each has to lose.
func (x Stewards) widerNotes(ctx context.Context, lk ports.DocLookup, name string) (map[string]int, []ports.Chunk, error) {
	by := map[string]int{}
	if x.Readers == nil {
		return by, nil, nil
	}
	ncs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Under: []ports.SourceRef{{Kind: noteKind, ID: "/notes/" + name + "/"}}})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	if len(ncs) == 0 {
		return by, nil, nil
	}
	ms, err := x.Readers(ctx, SpaceKey(name))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	members := map[string]bool{}
	for _, m := range ms {
		members[strings.ToLower(m)] = true
	}
	var notes []ports.Chunk
	for k, c := range firsts(latest(ncs)) {
		if n := wider(c, members); n > 0 {
			by[k] = n
			notes = append(notes, c)
		}
	}
	return by, notes, nil
}

func docKey(cite string) string {
	if i := strings.LastIndex(cite, "@"); i >= 0 {
		return cite[:i]
	}
	return cite
}

func (a Agent) reviewAfter(spaces []string) int {
	if a.ReviewAfter != nil && len(spaces) > 0 {
		if d := a.ReviewAfter(spaces); d > 0 {
			return d
		}
	}
	return DefaultReviewDays
}
