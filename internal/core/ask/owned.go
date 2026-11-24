package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// OwnedPastReview counts, by user id, the documents each owner owns that answers in traces
// cited at least ReviewCites times and that are stale now (SB-2806). scopeOf is the owner's
// scope now, by email, and false for an email that is not one active user, who is told
// nothing. Only the documents the owner can open are counted, checked as them (LAW-01).
func (a Agent) OwnedPastReview(ctx context.Context, traces []Trace, scopeOf func(ctx context.Context, email string) (tenancy.Scope, bool, error)) (map[string]int, error) {
	out := map[string]int{}
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
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	marks := &marksCache{store: a.Marks, by: map[string]map[string]ReviewMark{}}
	var stale []ports.Chunk
	for _, c := range firsts(latest(chunks)) {
		if c.Owner != "" && a.freshness(ctx, marks, c).stale {
			stale = append(stale, c)
		}
	}
	by, err := a.ownedBy(ctx, stale, scopeOf)
	if err != nil {
		return out, err
	}
	for u, docs := range by {
		out[u] = len(docs)
	}
	return out, nil
}

// ownedBy is, by user id, the documents of chunks each person looks after and can open
// now, checked as them: the owner at the source when they are one active user, else the
// document's caretaker when they are one (SB-3405). Nobody else is ever counted.
func (a Agent) ownedBy(ctx context.Context, chunks []ports.Chunk, scopeOf func(ctx context.Context, email string) (tenancy.Scope, bool, error)) (map[string]map[string]bool, error) {
	type who struct {
		s  tenancy.Scope
		ok bool
	}
	people := map[string]who{}
	look := func(email string) (who, error) {
		if w, ok := people[email]; ok {
			return w, nil
		}
		s, ok, err := scopeOf(ctx, email)
		if err != nil {
			return who{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		people[email] = who{s, ok}
		return people[email], nil
	}
	by := map[string][]ports.Chunk{}
	for _, c := range chunks {
		if c.Owner == "" {
			continue
		}
		email := c.Owner
		w, err := look(email)
		if err != nil {
			return nil, err
		}
		if !w.ok && a.Caretaker != nil {
			if e := a.Caretaker(ctx, c.Kind+":"+c.SourceID); e != "" {
				email = e
				if w, err = look(email); err != nil {
					return nil, err
				}
			}
		}
		if w.ok {
			by[email] = append(by[email], c)
		}
	}
	emails := make([]string, 0, len(by))
	for e := range by {
		emails = append(emails, e)
	}
	sort.Strings(emails)
	out := map[string]map[string]bool{}
	for _, e := range emails {
		s := people[e].s
		cand := by[e]
		sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
		res, err := retrieve.Allowed(ctx, s, ports.UserRef{ID: s.User(), Email: e}, a.Auth, cand, len(cand))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		for _, c := range res.Context {
			if out[s.User()] == nil {
				out[s.User()] = map[string]bool{}
			}
			out[s.User()][c.Kind+":"+c.SourceID] = true
		}
	}
	return out, nil
}

// HelpedOwners counts, by user id, the answers among traces rated up in fbs that cite a
// document the person looks after and can open now (SB-3406): counts only.
func (a Agent) HelpedOwners(ctx context.Context, traces []Trace, fbs []Feedback, scopeOf func(ctx context.Context, email string) (tenancy.Scope, bool, error)) (map[string]int, error) {
	out := map[string]int{}
	up := map[string]bool{}
	for _, f := range fbs {
		up[f.Trace] = f.Rating == "up" // the latest feedback on a trace replaces any earlier one
	}
	cites := map[string][]string{}
	var refs []ports.SourceRef
	seen := map[string]bool{}
	for _, t := range traces {
		if !up[t.ID] {
			continue
		}
		for _, cl := range t.Response.Answer.Claims {
			for _, ci := range cl.Cites {
				k := docKey(ci.Cite)
				cites[t.ID] = append(cites[t.ID], k)
				if !seen[k] {
					seen[k] = true
					kind, id, _ := strings.Cut(k, ":")
					refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
				}
			}
		}
	}
	if len(refs) == 0 {
		return out, nil
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var docs []ports.Chunk
	for _, c := range firsts(latest(chunks)) {
		docs = append(docs, c)
	}
	by, err := a.ownedBy(ctx, docs, scopeOf)
	if err != nil {
		return out, err
	}
	for u, mine := range by {
		for _, ks := range cites {
			for _, k := range ks {
				if mine[k] {
					out[u]++
					break
				}
			}
		}
	}
	return out, nil
}

// StaleCount is, of the answers that cite anything, how many carried the stale warning.
type StaleCount struct {
	Answers int `json:"answers"`
	Stale   int `json:"stale"`
}

// Freshness is a space's stale-answer counts this calendar month and last, UTC (SB-2807).
type Freshness struct {
	ThisMonth StaleCount `json:"this_month"`
	LastMonth StaleCount `json:"last_month"`
}

// StaleShare counts the answers among traces that cite anything, and those of them that
// carried notice_stale, in space (spc_NAME or company), or every space for "".
func StaleShare(traces []Trace, space string) StaleCount {
	var n StaleCount
	for _, t := range traces {
		if t.Response.Agent != "" && t.Response.Agent != "ask" || space != "" && t.Space != space {
			continue
		}
		cites := false
		for _, c := range t.Response.Answer.Claims {
			if len(c.Cites) > 0 {
				cites = true
				break
			}
		}
		if !cites {
			continue
		}
		n.Answers++
		if t.Response.NoticeStale != nil {
			n.Stale++
		}
	}
	return n
}

// MonthStart is the first instant of t's calendar month, UTC.
func MonthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// unusedTerms are the space's usable glossary terms no trace in the space used (SB-2805).
func unusedTerms(glossary []Term, traces []Trace, space string) []string {
	used := map[string]bool{}
	for _, t := range traces {
		if t.Space != space {
			continue
		}
		for _, x := range t.Response.Terms {
			used[normTitle(x.Term)] = true
		}
	}
	var out []string
	for _, g := range glossary {
		if !used[normTitle(g.Term)] {
			out = append(out, g.Term)
		}
	}
	sort.Strings(out)
	return out
}
