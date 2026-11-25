package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Why a start_here item is on the review list (SB-2907).
const (
	StartGone       = "gone"
	StartStale      = "stale"
	StartUnreadable = "unreadable"
)

// MaxStartMembers is the most members whose access a start_here check counts.
const MaxStartMembers = 200

// StartItem is a start_here item gone stale, for the space's stewards.
type StartItem struct {
	Item           string   `json:"item"`
	Result         *Result  `json:"result,omitempty"`
	Why            []string `json:"why"`
	MembersWithout int      `json:"members_without"`
	Members        int      `json:"members"`
}

// EvalCheck is an eval case of the space to check (SB-2904).
type EvalCheck struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Check    string `json:"check"`
}

// evalsToCheck are the space's cases whose expected document changed or is gone.
func (x Stewards) evalsToCheck(ctx context.Context, name string) ([]EvalCheck, error) {
	if x.Evals == nil {
		return nil, nil
	}
	all, err := x.Evals.EvalCases(ctx, x.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: eval cases: %v", ErrUnavailable, err)
	}
	var mine []EvalCase
	for _, c := range ActiveCases(all) {
		if c.Space == SpaceKey(name) {
			mine = append(mine, c)
		}
	}
	checks, err := x.Agent.CaseChecks(ctx, mine)
	if err != nil {
		return nil, err
	}
	var out []EvalCheck
	for _, c := range mine {
		if ch := checks[c.ID]; ch != "" {
			out = append(out, EvalCheck{ID: c.ID, Question: c.Question, Check: ch})
		}
	}
	return out, nil
}

// startStale judges the space's start_here items: gone from the index, stale, or more than
// a quarter of its active members can't open now, by the live ACL (Unknown is can't). The
// steward sees an item as they can open it, and only counts one they can't.
func (x Stewards) startStale(ctx context.Context, s tenancy.Scope, lk ports.DocLookup, name string) ([]StartItem, int, error) {
	if x.StartItems == nil {
		return nil, 0, nil
	}
	items := x.StartItems(name)
	if len(items) == 0 {
		return nil, 0, nil
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
	chunks, err := lk.Docs(ctx, x.Agent.Tenant, q)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	first := firsts(latest(chunks))
	docsOf := func(it string) []ports.Chunk {
		kind, id, _ := strings.Cut(it, ":")
		var out []ports.Chunk
		if !strings.HasPrefix(id, "/") {
			if c, ok := first[kind+":"+id]; ok {
				out = append(out, c)
			}
			return out
		}
		for _, c := range first {
			if c.Kind == kind && strings.HasPrefix(c.Location+"/", id) {
				out = append(out, c)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Location < out[j].Location })
		return out
	}
	opens, members, err := x.membersOpen(ctx, name, first)
	if err != nil {
		return nil, 0, err
	}
	marks := &marksCache{store: x.Agent.Marks, by: map[string]map[string]ReviewMark{}}
	var out []StartItem
	var shown []ports.Chunk
	for _, it := range items {
		docs := docsOf(it)
		si := StartItem{Item: it, Why: []string{}, Members: members}
		if len(docs) == 0 {
			si.Why = append(si.Why, StartGone)
		} else {
			stale := true
			for _, d := range docs {
				stale = stale && x.Agent.freshness(ctx, marks, d).stale
			}
			if stale {
				si.Why = append(si.Why, StartStale)
			}
			for _, can := range opens {
				any := false
				for _, d := range docs {
					any = any || can[d.Kind+":"+d.SourceID]
				}
				if !any {
					si.MembersWithout++
				}
			}
			if members > 0 && 4*si.MembersWithout > members {
				si.Why = append(si.Why, StartUnreadable)
			}
			shown = append(shown, docs[0])
		}
		if len(si.Why) > 0 {
			out = append(out, si)
		}
	}
	return x.asSteward(ctx, s, out, shown, docsOf)
}

// asSteward gives each listed item the document the steward can open, and counts, without
// listing, the items whose documents they can't open.
func (x Stewards) asSteward(ctx context.Context, s tenancy.Scope, items []StartItem, shown []ports.Chunk, docsOf func(string) []ports.Chunk) ([]StartItem, int, error) {
	if len(items) == 0 {
		return nil, 0, nil
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return nil, 0, err
	}
	sort.Slice(shown, func(i, j int) bool { return shown[i].ID < shown[j].ID })
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, shown, len(shown))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	allowed := map[string]ports.Chunk{}
	for _, c := range res.Context {
		allowed[c.Kind+":"+c.SourceID] = c
	}
	own := x.Agent.owners()
	var out []StartItem
	hidden := 0
	for _, si := range items {
		docs := docsOf(si.Item)
		if len(docs) == 0 {
			out = append(out, si) // gone: the item names only what the space's own list says
			continue
		}
		c, ok := allowed[docs[0].Kind+":"+docs[0].SourceID]
		if !ok {
			hidden++
			continue
		}
		r := x.Agent.result(ctx, own, c)
		r.Snippet = ""
		si.Result = &r
		out = append(out, si)
	}
	return out, hidden, nil
}

// membersOpen is, for each of the space's active members (at most MaxStartMembers), the
// documents of first they can open at the source now, one ACL call per member and kind.
func (x Stewards) membersOpen(ctx context.Context, name string, first map[string]ports.Chunk) ([]map[string]bool, int, error) {
	if x.Members == nil || len(first) == 0 {
		return nil, 0, nil
	}
	emails, err := x.Members(ctx, SpaceKey(name))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	sort.Strings(emails)
	if len(emails) > MaxStartMembers {
		emails = emails[:MaxStartMembers]
	}
	byKind := map[string][]ports.SourceRef{}
	keys := make([]string, 0, len(first))
	for k := range first {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c := first[k]
		byKind[c.Kind] = append(byKind[c.Kind], ports.SourceRef{Kind: c.Kind, ID: c.SourceID})
	}
	var out []map[string]bool
	for _, e := range emails {
		can := map[string]bool{}
		for kind, refs := range byKind {
			conn, ok := x.Agent.Auth.Connectors[kind]
			if !ok {
				continue // no connector: nobody can open it
			}
			vs, err := conn.ACL(ctx, ports.UserRef{Email: e}, refs)
			if err != nil || len(vs) != len(refs) {
				continue // unknown is can't (LAW-01)
			}
			for i, v := range vs {
				if v == ports.VerdictAllow {
					can[kind+":"+refs[i].ID] = true
				}
			}
		}
		out = append(out, can)
	}
	return out, len(emails), nil
}
