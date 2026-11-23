package ask

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// A tidy corpus (sb-ask(3), Documents to review; SB-2802 to SB-2804): documents with the
// same name and different text, documents no answer cites, and notes whose sources moved.

// Bounds of the never-cited section (SB-2803).
const (
	UncitedWindow = 180 * 24 * time.Hour
	UncitedAge    = 90 * 24 * time.Hour
	MaxUncited    = 100
)

// SameName is a group of documents with the same title and different text (SB-2802).
type SameName struct {
	Title  string   `json:"title"`
	Items  []Result `json:"items"`
	Hidden int      `json:"hidden"`
}

// UncitedList is the never-cited section: documents the steward can open, and the owners
// of the others.
type UncitedList struct {
	Items  []Result     `json:"items"`
	Hidden []HiddenItem `json:"hidden"`
	// UnusedTerms are the space's glossary terms no question used in the window (SB-2805).
	UnusedTerms []string `json:"unused_terms,omitempty"`
}

// normTitle is a title with case and spacing ignored.
func normTitle(t string) string { return strings.Join(strings.Fields(strings.ToLower(t)), " ") }

// sameName groups held documents by title where the text differs, each text once (the
// newest copy), and checks each group as the steward.
func (x Stewards) sameName(ctx context.Context, s tenancy.Scope, held []ports.Chunk) ([]SameName, error) {
	byTitle := map[string]map[string]ports.Chunk{} // title -> text hash -> newest copy
	for _, c := range held {
		t := normTitle(c.Title)
		if t == "" {
			continue
		}
		if byTitle[t] == nil {
			byTitle[t] = map[string]ports.Chunk{}
		}
		if have, ok := byTitle[t][c.TextSHA256]; !ok || c.Seen.After(have.Seen) {
			byTitle[t][c.TextSHA256] = c
		}
	}
	var titles []string
	var cand []ports.Chunk
	for t, m := range byTitle {
		if len(m) < 2 {
			continue
		}
		titles = append(titles, t)
		for _, c := range m {
			cand = append(cand, c)
		}
	}
	if len(cand) == 0 {
		return nil, nil
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	allowed := map[string]ports.Chunk{}
	for _, c := range res.Context {
		allowed[c.ID] = c
	}
	own := x.Agent.owners()
	var out []SameName
	sort.Strings(titles)
	for _, t := range titles {
		var group []ports.Chunk
		for _, c := range byTitle[t] {
			group = append(group, c)
		}
		sort.Slice(group, func(i, j int) bool { return group[i].Seen.After(group[j].Seen) })
		g := SameName{Items: []Result{}}
		for _, g0 := range group {
			c, ok := allowed[g0.ID]
			if !ok {
				g.Hidden++
				continue
			}
			r := x.Agent.result(ctx, own, c)
			r.Snippet = ""
			g.Items = append(g.Items, r)
			if g.Title == "" {
				g.Title = c.Title
			}
		}
		out = append(out, g)
	}
	return out, nil
}

// Uncited lists the documents the space holds that no answer cited in UncitedWindow and
// that the index has held for UncitedAge, oldest first (SB-2803).
func (x Stewards) Uncited(ctx context.Context, s tenancy.Scope, name string, src ReviewSource) (UncitedList, error) {
	out := UncitedList{Items: []Result{}, Hidden: []HiddenItem{}}
	if err := x.reviewer(ctx, s, name); err != nil {
		return out, err
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	now := x.Now().UTC()
	traces, err := src.Between(ctx, x.Agent.Tenant, now.Add(-UncitedWindow), now)
	if err != nil {
		return out, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	cited, _, _ := reviewCounts(traces, nil)
	if x.Agent.Glossary != nil {
		ok, _ := usable(x.Agent.Glossary(name))
		out.UnusedTerms = unusedTerms(ok, traces, SpaceKey(name))
	}
	held, err := x.heldBy(ctx, lk, name)
	if err != nil {
		return out, err
	}
	var cand []ports.Chunk
	for _, c := range held {
		if cited[c.Kind+":"+c.SourceID] == 0 && !c.Seen.IsZero() && now.Sub(c.Seen) > UncitedAge {
			cand = append(cand, c)
		}
	}
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].Seen.Before(cand[j].Seen) })
	if len(cand) > MaxUncited {
		cand = cand[:MaxUncited]
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
		allowed[c.Kind+":"+c.SourceID] = true
		r := x.Agent.result(ctx, own, c)
		r.Snippet = ""
		out.Items = append(out.Items, r)
	}
	out.Hidden = hiddenOf(cand, allowed, gone)
	return out, nil
}

// movedNotes are the space's notes with a cited source that has a newer version or is gone,
// by note, with how many, unless a mark at the note's version came after the change (SB-2804).
func (x Stewards) movedNotes(ctx context.Context, lk ports.DocLookup, name string, marks map[string]ReviewMark) (map[string]int, []ports.Chunk, error) {
	by := map[string]int{}
	ncs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Under: []ports.SourceRef{{Kind: noteKind, ID: "/notes/" + name + "/"}}})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var notes []ports.Chunk
	for k, c := range firsts(latest(ncs)) {
		id := strings.TrimSuffix(path.Base(c.SourceID), ".md")
		q, found, err := x.Store.GetStewardQuestion(ctx, x.Agent.Tenant, id)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		if !found || q.Reply == nil || len(q.Reply.Cites) == 0 {
			continue
		}
		n, at, err := x.moved(ctx, lk, q.Reply.Cites)
		if err != nil {
			return nil, nil, err
		}
		if m, ok := marks[k]; n == 0 || (ok && m.Version == c.Version && m.At.After(at)) {
			continue
		}
		by[k] = n
		notes = append(notes, c)
	}
	return by, notes, nil
}

// moved counts the cites with a newer version in the index, or gone from it, and the
// newest time such a version was seen (zero when only gone ones moved).
func (x Stewards) moved(ctx context.Context, lk ports.DocLookup, cites []string) (int, time.Time, error) {
	var refs []ports.SourceRef
	want := map[string]int{}
	for _, ci := range cites {
		k := docKey(ci)
		v, err := strconv.Atoi(strings.TrimPrefix(ci, k+"@"))
		kind, id, ok := strings.Cut(k, ":")
		if err != nil || !ok {
			continue
		}
		want[k] = v
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	if len(refs) == 0 {
		return 0, time.Time{}, nil
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	cur := firsts(latest(cs))
	n := 0
	var at time.Time
	for k, v := range want {
		c, ok := cur[k]
		switch {
		case !ok:
			n++
		case c.Version > v:
			n++
			if c.Seen.After(at) {
				at = c.Seen
			}
		}
	}
	return n, at, nil
}
