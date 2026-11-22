package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// StartResponse is a space's reading list for one member (sb-spaces(7), Start here; SB-1706).
type StartResponse struct {
	Items  []Result `json:"items"`
	Hidden int      `json:"hidden"`
}

// latest keeps each document's newest indexed version only.
func latest(chunks []ports.Chunk) []ports.Chunk {
	top := map[string]int{}
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		if c.Version > top[k] {
			top[k] = c.Version
		}
	}
	var out []ports.Chunk
	for _, c := range chunks {
		if c.Version == top[c.Kind+":"+c.SourceID] {
			out = append(out, c)
		}
	}
	return out
}

// firsts is each document's first chunk (its title chunk), in the order given.
func firsts(chunks []ports.Chunk) map[string]ports.Chunk {
	out := map[string]ports.Chunk{}
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		if have, ok := out[k]; !ok || c.Start < have.Start {
			out[k] = c
		}
	}
	return out
}

// StartHere resolves items (KIND:ID or KIND:/Folder/) for the scope's user: each document
// checked as a search checks it, now; the ones they can't open are counted, not named.
func (a Agent) StartHere(ctx context.Context, s tenancy.Scope, items []string) (StartResponse, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return StartResponse{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if a.Killed == nil || a.Killed(ctx) || spaceKilled(a.SpaceKilled, s) {
		return StartResponse{}, ErrKilled
	}
	out := StartResponse{Items: []Result{}}
	if len(items) == 0 {
		return out, nil
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var q ports.DocQuery
	for _, it := range items {
		kind, id, _ := strings.Cut(it, ":")
		if strings.HasPrefix(id, "/") {
			q.Under = append(q.Under, ports.SourceRef{Kind: kind, ID: id})
		} else {
			q.Refs = append(q.Refs, ports.SourceRef{Kind: kind, ID: id})
		}
	}
	chunks, err := lk.Docs(ctx, a.Tenant, q)
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	first := firsts(latest(chunks))
	var cand []ports.Chunk
	for _, c := range first {
		cand = append(cand, c)
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := a.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	allowed := map[string]ports.Chunk{}
	for _, c := range res.Context {
		allowed[c.Kind+":"+c.SourceID] = c
	}
	own := a.owners()
	shown := map[string]bool{}
	add := func(c ports.Chunk) {
		k := c.Kind + ":" + c.SourceID
		if shown[k] || len(out.Items) == MaxStart {
			return
		}
		shown[k] = true
		r := a.result(ctx, own, c)
		r.Snippet = ""
		out.Items = append(out.Items, r)
	}
	for _, it := range items {
		kind, id, _ := strings.Cut(it, ":")
		if !strings.HasPrefix(id, "/") {
			if c, ok := allowed[kind+":"+id]; ok {
				add(c)
			} else {
				out.Hidden++ // refused, Unknown, late or not indexed: counted, never named
			}
			continue
		}
		var under []ports.Chunk
		hidden := 0
		for k, c := range first {
			if c.Kind != kind || !strings.HasPrefix(c.Location+"/", id) {
				continue
			}
			if ac, ok := allowed[k]; ok {
				under = append(under, ac)
			} else {
				hidden++
			}
		}
		sort.Slice(under, func(i, j int) bool { return under[i].Location < under[j].Location })
		for _, c := range under {
			add(c)
		}
		out.Hidden += hidden
	}
	return out, nil
}

// MaxStart is the most documents a start list shows.
const MaxStart = 10

// Handover (sb-agents(7), Handover; SB-1707).
const (
	AgentHandover = "handover"
	// HandoverDocs is the most of the person's documents briefed, newest first.
	HandoverDocs = retrieve.Budget
)

// HandoverResponse is the brief: every claim cited, and counts of what was left out.
type HandoverResponse struct {
	Trace   string `json:"trace"`
	Release string `json:"release"`
	Person  string `json:"person"`
	Answer  Answer `json:"answer"`
	Dropped int    `json:"dropped"`
	Hidden  int    `json:"hidden"`
}

// Relations says how the asker stands to the person: self, manager, steward (sb-policy(5)).
type Relations func(ctx context.Context, s tenancy.Scope, person string) ([]string, error)

// ErrNoPerson is a person the asker may not brief, or who doesn't exist (EPERM).
var ErrNoPerson = errors.New("ask: EPERM no handover of this person for you")

// Handover briefs the documents person owns that the asker can open now.
func (a Agent) Handover(ctx context.Context, s tenancy.Scope, person string, emails func(ctx context.Context, user string) ([]string, error),
	rel Relations, k policy.Kernel) (HandoverResponse, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return HandoverResponse{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if a.Killed == nil || a.Killed(ctx) || (a.AgentKilled != nil && a.AgentKilled(AgentHandover)) || spaceKilled(a.SpaceKilled, s) {
		return HandoverResponse{}, ErrKilled
	}
	rels, err := rel(ctx, s, person)
	if err != nil {
		return HandoverResponse{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if k.Authorise(ctx, s, "brief_handover", policy.Entity{Type: "User", ID: person, Tenant: a.Tenant}, map[string]any{"relations": rels}).Decision != policy.Allow {
		return HandoverResponse{}, ErrNoPerson
	}
	mails, err := emails(ctx, person)
	if err != nil {
		return HandoverResponse{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if len(mails) == 0 {
		return HandoverResponse{}, ErrNoPerson
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return HandoverResponse{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var owners []string
	for _, e := range mails {
		owners = append(owners, strings.ToLower(strings.TrimSpace(e)))
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Owners: owners})
	if err != nil {
		return HandoverResponse{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	first := firsts(latest(chunks))
	var cand []ports.Chunk
	for _, c := range first {
		cand = append(cand, c)
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := a.who(ctx, s)
	if err != nil {
		return HandoverResponse{}, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, len(cand))
	if err != nil {
		return HandoverResponse{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	ok2 := res.Context
	out := HandoverResponse{Release: a.Release, Person: person, Hidden: len(first) - len(ok2)}
	// Newest first, then by location, so the brief groups by where the documents live.
	sort.SliceStable(ok2, func(i, j int) bool { return ok2[i].Seen.After(ok2[j].Seen) })
	if len(ok2) > HandoverDocs {
		ok2 = ok2[:HandoverDocs]
	}
	sort.SliceStable(ok2, func(i, j int) bool { return ok2[i].Location < ok2[j].Location })
	trace := a.NewID("tr_")
	out.Trace = trace
	if len(ok2) == 0 {
		out.Answer.Claims = []answer.Claim{{Text: NothingToCite, Cites: []answer.Cite{}, Uncited: true}}
	} else {
		claims, err := a.composeWith(withUser(ctx, s.User()), AgentHandover, "Write the handover brief of the documents above.", trace, "", nil, ok2)
		if err != nil {
			return HandoverResponse{}, err
		}
		claims, out.Dropped = citedOnly(claims)
		out.Answer.Claims = claims
	}
	t := Trace{ID: trace, Tenant: a.Tenant, User: s.User(), Space: s.Space(), At: a.Now(),
		Response:  Response{Trace: trace, Release: a.Release, Agent: AgentHandover, Refused: out.Hidden, Answer: out.Answer, Dropped: out.Dropped, People: []PersonRef{}},
		Decisions: res.Decisions}
	if err := a.Store.PutTrace(ctx, t); err != nil {
		return HandoverResponse{}, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	return out, nil
}
