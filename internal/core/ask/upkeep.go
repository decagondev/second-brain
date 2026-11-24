package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// DueWithin is how far ahead the coming-due section looks, and MaxDue the most it lists
// (SB-3105).
const (
	DueWithin = 14 * 24 * time.Hour
	MaxDue    = 100
)

// ErrNotOwner is a mark by someone who isn't the document's current owner (SB-3106).
var ErrNotOwner = errors.New("ask: EPERM not the document's owner")

// DueItem is a document on the coming-due section, with the whole days until it is stale.
type DueItem struct {
	Result
	DueInDays int `json:"due_in_days"`
}

// DueList is the 200 body of GET /v1/spaces/NAME/review?section=due.
type DueList struct {
	Items  []DueItem    `json:"items"`
	Hidden []HiddenItem `json:"hidden"`
}

// staleAt is when a document turns stale: after days from its version's first sight or a
// mark at that version, whichever is later, or its own review-by date if sooner. A
// document whose age the index doesn't know has no date.
func staleAt(c ports.Chunk, m ReviewMark, marked bool, after int) (time.Time, bool) {
	if c.Seen.IsZero() && c.ReviewBy.IsZero() {
		return time.Time{}, false
	}
	var at time.Time
	if !c.Seen.IsZero() {
		base, days := c.Seen, after
		if marked && m.Version == c.Version && m.At.After(base) {
			base, days = m.At, markAfter(m, after) // SB-3603
		}
		at = base.Add(time.Duration(days) * 24 * time.Hour)
	}
	if !c.ReviewBy.IsZero() && (at.IsZero() || c.ReviewBy.Before(at)) {
		at = c.ReviewBy
	}
	return at, true
}

// ComingDue lists the documents the space holds that answers lean on and that turn stale
// within DueWithin, soonest first, checked as the steward (SB-3105).
func (x Stewards) ComingDue(ctx context.Context, s tenancy.Scope, name string, src ReviewSource, after int) (DueList, error) {
	out := DueList{Items: []DueItem{}, Hidden: []HiddenItem{}}
	if err := x.reviewer(ctx, s, name); err != nil {
		return out, err
	}
	if after <= 0 {
		after = DefaultReviewDays
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	now := x.Now().UTC()
	traces, err := src.Between(ctx, x.Agent.Tenant, now.Add(-ReviewWindow), now)
	if err != nil {
		return out, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	cited, _, _ := reviewCounts(traces, nil)
	var refs []ports.SourceRef
	for k, n := range cited {
		if n >= ReviewCites {
			kind, id, _ := strings.Cut(k, ":")
			refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
		}
	}
	if len(refs) == 0 {
		return out, nil
	}
	chunks, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	marks, err := x.marked(ctx, name)
	if err != nil {
		return out, err
	}
	when := map[string]time.Time{}
	var cand []ports.Chunk
	for k, c := range firsts(latest(chunks)) {
		if !contains(c.Spaces, SpaceKey(name)) {
			continue
		}
		m, marked := marks[k]
		at, known := staleAt(c, m, marked, after)
		if known && now.Before(at) && !at.After(now.Add(DueWithin)) {
			when[k] = at
			cand = append(cand, c)
		}
	}
	sort.Slice(cand, func(i, j int) bool {
		a, b := when[cand[i].Kind+":"+cand[i].SourceID], when[cand[j].Kind+":"+cand[j].SourceID]
		if !a.Equal(b) {
			return a.Before(b)
		}
		return cand[i].ID < cand[j].ID
	})
	if len(cand) > MaxDue {
		cand = cand[:MaxDue]
	}
	if len(cand) == 0 {
		return out, nil
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	gone, err := x.ownersGone(ctx, cand)
	if err != nil {
		return out, err
	}
	allowed := map[string]bool{}
	own := x.Agent.owners()
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		allowed[k] = true
		r := x.Agent.result(ctx, own, c)
		r.Snippet = ""
		days := int((when[k].Sub(now) + 24*time.Hour - 1) / (24 * time.Hour))
		out.Items = append(out.Items, DueItem{Result: r, DueInDays: days})
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].DueInDays < out.Items[j].DueInDays })
	out.Hidden = hiddenOf(cand, allowed, gone)
	return out, nil
}

// addresses are the caller's own addresses in the directory now, lowercased; none when
// the directory can't say, so nothing is theirs on a guess.
func (a Agent) addresses(ctx context.Context, s tenancy.Scope) map[string]bool {
	out := map[string]bool{}
	if a.Emails == nil {
		return out
	}
	es, err := a.Emails(ctx, s.User())
	if err != nil {
		return out
	}
	for _, e := range es {
		out[strings.ToLower(e)] = true
	}
	return out
}

// OwnMark is the 200 body of POST /v1/documents/reviewed.
type OwnMark struct {
	Doc     string   `json:"document"`
	Version int      `json:"version"`
	Spaces  []string `json:"spaces"`
}

// MarkOwn marks the caller's own document Reviewed, still right at its current version,
// in every space that holds it, as the stewards' record (SB-3106). Only its current owner
// at the source, who can open it now, may.
func (x Stewards) MarkOwn(ctx context.Context, s tenancy.Scope, cite string, days int) (OwnMark, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return OwnMark{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if days != 0 && !MarkDays[days] {
		return OwnMark{}, fmt.Errorf("%w: review again in 30, 90, 180 or 365 days", ErrInvalid)
	}
	if x.Marks == nil {
		return OwnMark{}, fmt.Errorf("%w: this cell keeps no review marks", ErrUnavailable)
	}
	doc := docKey(cite)
	ver, err := strconv.Atoi(strings.TrimPrefix(cite, doc+"@"))
	kind, id, _ := strings.Cut(doc, ":")
	if err != nil || id == "" {
		return OwnMark{}, fmt.Errorf("%w: a cite is KIND:ID@VERSION", ErrInvalid)
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return OwnMark{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return OwnMark{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	cur := firsts(latest(cs))[doc]
	mine := x.Agent.addresses(ctx, s)
	if cur.ID == "" || cur.Owner == "" || !mine[strings.ToLower(cur.Owner)] {
		return OwnMark{}, ErrNotOwner // someone else's, or none, looks the same
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return OwnMark{}, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, []ports.Chunk{cur}, 1)
	if err != nil {
		return OwnMark{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) == 0 {
		return OwnMark{}, ErrNotOwner
	}
	if res.Context[0].Version != ver {
		return OwnMark{}, fmt.Errorf("%w: mark the document's current version, %d", ErrInvalid, res.Context[0].Version)
	}
	if len(cur.Spaces) == 0 {
		return OwnMark{}, fmt.Errorf("%w: no space holds this document, so nobody reviews it", ErrInvalid)
	}
	out := OwnMark{Doc: doc, Version: ver, Spaces: []string{}}
	at := x.Now().UTC()
	for _, sp := range cur.Spaces {
		m := ReviewMark{Tenant: x.Agent.Tenant, Space: sp, Doc: doc, Version: ver, By: s.User(), At: at, Days: days}
		if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: cite, State: "reviewed",
			Reason: "reviewed by its owner", Space: auditSpace(sp)}); err != nil {
			return out, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
		}
		if err := x.Marks.PutReviewMark(ctx, m); err != nil {
			return out, fmt.Errorf("%w: review marks: %v", ErrUnavailable, err)
		}
		out.Spaces = append(out.Spaces, strings.TrimPrefix(sp, "spc_"))
	}
	return out, nil
}
