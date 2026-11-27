package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// See and steer the context (sb-ask(3), The context of an answer; SB-5501 to SB-5504): what
// an answer was built from, shown from its trace and checked as the viewer now; Keep and
// Drop for the rest of a conversation, sent by the browser with each ask; and a person's own
// Don't use this for me, kept with their memory and applied to everything made as them.

// MaxSteer is the most documents one ask keeps or drops, and MaxExcluded the most a person
// excludes.
const (
	MaxSteer    = 20
	MaxExcluded = 200
)

// Exclusion is one document a person keeps out of everything made as them (SB-5503).
type Exclusion struct {
	Doc string    `json:"document"` // KIND:ID
	At  time.Time `json:"at"`
}

// steerKeys checks a list of KIND:ID documents.
func steerKeys(ds []string, what string) ([]string, error) {
	if len(ds) > MaxSteer {
		return nil, fmt.Errorf("%w: at most %d documents to %s", ErrInvalid, MaxSteer, what)
	}
	var out []string
	for _, d := range ds {
		d, _, _ = strings.Cut(strings.TrimSpace(d), "@")
		kind, id, ok := strings.Cut(d, ":")
		if !ok || !Kinds[kind] || id == "" {
			return nil, fmt.Errorf("%w: documents to %s must be KIND:ID", ErrInvalid, what)
		}
		if !contains(out, d) {
			out = append(out, d)
		}
	}
	return out, nil
}

// excluded are s's person's excluded documents, by KIND:ID. An unreadable list excludes
// nothing it can't read, so it fails the ask instead of answering from a document the
// person turned off.
func (a Agent) excluded(ctx context.Context, s tenancy.Scope) (map[string]bool, error) {
	out := map[string]bool{}
	if a.Exclusions == nil {
		return out, nil
	}
	m, _, err := a.Exclusions.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return nil, fmt.Errorf("%w: your exclusions: %v", ErrUnavailable, err)
	}
	if m.User != "" && m.User != s.User() {
		return out, nil
	}
	for _, x := range m.Excluded {
		out[x.Doc] = true
	}
	return out, nil
}

// dropDocs leaves out of chunks the documents in drop.
func dropDocs(chunks []ports.Chunk, drop map[string]bool) []ports.Chunk {
	if len(drop) == 0 {
		return chunks
	}
	out := chunks[:0:0]
	for _, c := range chunks {
		if !drop[c.Kind+":"+c.SourceID] {
			out = append(out, c)
		}
	}
	return out
}

// dropExcluded takes the person's own exclusions out of chunks (SB-5503): only narrows.
func (a Agent) dropExcluded(ctx context.Context, s tenancy.Scope, chunks []ports.Chunk) ([]ports.Chunk, error) {
	ex, err := a.excluded(ctx, s)
	if err != nil {
		return nil, err
	}
	return dropDocs(chunks, ex), nil
}

type steerKey struct{}

// steer is one ask's Keep and Drop (SB-5502).
type steer struct{ keep, drop []string }

func withSteer(ctx context.Context, keep, drop []string) context.Context {
	if len(keep) == 0 && len(drop) == 0 {
		return ctx
	}
	return context.WithValue(ctx, steerKey{}, steer{keep: keep, drop: drop})
}

func steerOf(ctx context.Context) steer {
	st, _ := ctx.Value(steerKey{}).(steer)
	return st
}

// steered drops the ask's dropped documents from res and puts its kept ones first, each
// kept document's first chunks checked as s now like any chunk, and still left out when
// held, excluded or retired.
func (a Agent) steered(ctx context.Context, s tenancy.Scope, chunks []ports.Chunk) ([]ports.Chunk, error) {
	st := steerOf(ctx)
	drop := map[string]bool{}
	for _, d := range st.drop {
		drop[d] = true
	}
	chunks = dropDocs(chunks, drop)
	if len(st.keep) == 0 {
		return chunks, nil
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return chunks, nil
	}
	var q ports.DocQuery
	for _, d := range st.keep {
		if drop[d] {
			continue
		}
		kind, id, _ := strings.Cut(d, ":")
		q.Refs = append(q.Refs, ports.SourceRef{Kind: kind, ID: id})
	}
	if len(q.Refs) == 0 {
		return chunks, nil
	}
	all, err := lk.Docs(ctx, a.Tenant, q)
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	all = latest(all)
	sort.SliceStable(all, func(i, j int) bool { return all[i].Start < all[j].Start })
	all = retrieve.PerDocument(all, MaxPerDocument)
	who, err := a.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, all, len(all))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	kept := res.Context
	for _, f := range []func(context.Context, tenancy.Scope, []ports.Chunk) ([]ports.Chunk, error){a.dropRetired, a.dropHeld, a.dropExcluded} {
		if kept, err = f(ctx, s, kept); err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{}
	var out []ports.Chunk
	for _, c := range append(kept, chunks...) {
		if !seen[c.ID] {
			seen[c.ID] = true
			out = append(out, c)
		}
	}
	return out, nil
}

// ExcludedView is a person's exclusions as they see them on My data: each document's title
// when they can open it now, the rest counted.
type ExcludedView struct {
	Items  []ExcludedItem `json:"items"`
	Hidden int            `json:"hidden"`
}

// ExcludedItem is one exclusion the person can open.
type ExcludedItem struct {
	Doc   string    `json:"document"`
	Title string    `json:"title,omitempty"`
	At    time.Time `json:"at"`
}

func (a Agent) exclusionScope(s tenancy.Scope) error {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if a.Exclusions == nil {
		return fmt.Errorf("%w: this cell keeps no exclusions", ErrUnavailable)
	}
	return nil
}

// ExcludedFor lists the person's own exclusions.
func (a Agent) ExcludedFor(ctx context.Context, s tenancy.Scope) (ExcludedView, error) {
	out := ExcludedView{Items: []ExcludedItem{}}
	if err := a.exclusionScope(s); err != nil {
		return out, err
	}
	m, _, err := a.Exclusions.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return out, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	var docs []string
	for _, x := range m.Excluded {
		docs = append(docs, x.Doc)
	}
	titles, hidden, err := a.docTitles(ctx, s, docs)
	if err != nil {
		return out, err
	}
	out.Hidden = hidden
	for _, x := range m.Excluded {
		if t, ok := titles[x.Doc]; ok {
			out.Items = append(out.Items, ExcludedItem{Doc: x.Doc, Title: t, At: x.At})
		}
	}
	return out, nil
}

// Exclude is Don't use this for me (SB-5503), or its undo: kept with the person's memory,
// so it is exported with it and erased with it and by their Forget (LAW-25). Audited by
// the document's id only.
func (a Agent) Exclude(ctx context.Context, s tenancy.Scope, doc string, on bool) (ExcludedView, error) {
	if err := a.exclusionScope(s); err != nil {
		return ExcludedView{}, err
	}
	keys, err := steerKeys([]string{doc}, "exclude")
	if err != nil || len(keys) != 1 {
		return ExcludedView{}, fmt.Errorf("%w: a document is KIND:ID", ErrInvalid)
	}
	d := keys[0]
	m, _, err := a.Exclusions.MemoryOf(ctx, a.Tenant, s.User())
	if err != nil {
		return ExcludedView{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	var keep []Exclusion
	had := false
	for _, x := range m.Excluded {
		if x.Doc == d {
			had = true
			continue
		}
		keep = append(keep, x)
	}
	if on {
		if !had && len(keep) >= MaxExcluded {
			return ExcludedView{}, fmt.Errorf("%w: at most %d documents; undo one first", ErrInvalid, MaxExcluded)
		}
		keep = append(keep, Exclusion{Doc: d, At: a.now()})
	} else if !had {
		return ExcludedView{}, fmt.Errorf("%w: that document isn't excluded", ErrInvalid)
	}
	state := map[bool]string{true: "excluded", false: "included"}[on]
	if a.Audit != nil {
		if _, err := a.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: "exclusion:" + d, State: state, Reason: "don't use this for me"}); err != nil {
			return ExcludedView{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
		}
	}
	m.Tenant, m.User, m.At, m.Excluded = a.Tenant, s.User(), a.now(), keep
	if err := a.Exclusions.PutMemory(ctx, m); err != nil {
		return ExcludedView{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
	}
	return a.ExcludedFor(ctx, s)
}

// ContextPanel is what one answer was built from (SB-5501), from its trace, checked as the
// viewer now: what they can't open now is counted, never named.
type ContextPanel struct {
	HowMade
	Memory         []MemoryLine `json:"memory"`
	MemoryGone     int          `json:"memory_gone,omitempty"`
	Collection     string       `json:"collection,omitempty"`
	Documents      []HowSource  `json:"documents"`
	Kept           []HowSource  `json:"kept"`
	Dropped        []HowSource  `json:"dropped"`
	DocsHidden     int          `json:"documents_hidden"`
	EarlierKept    int          `json:"earlier_kept"`
	EarlierDropped int          `json:"earlier_dropped"`
	Skipped        int          `json:"skipped"`
	Held           int          `json:"held"`
	// Translated is the question as it was also searched in the cell's other languages
	// (SB-6103), shown only to the asker, whose question it is.
	Translated []string `json:"translated,omitempty"`
}

// Context is the panel of trace id's answer, for whoever may read the trace; the memory
// lines are shown only to the asker, who wrote them (LAW-25).
func (a Agent) Context(ctx context.Context, s tenancy.Scope, id string) (ContextPanel, error) {
	t, err := a.Trace(ctx, s, id)
	if err != nil {
		return ContextPanel{}, err
	}
	out := ContextPanel{HowMade: HowMade{Trace: t.ID, Agent: t.Response.Agent, Release: t.Response.Release, Skill: t.Response.Skill, Tier: t.Tier},
		Memory: []MemoryLine{}, Documents: []HowSource{}, Kept: []HowSource{}, Dropped: []HowSource{},
		Collection: t.Collection, EarlierDropped: t.Response.EarlierDropped, Skipped: t.Response.Skipped, Held: t.Response.Held}
	if t.User == s.User() {
		out.Translated = t.Translated
	}
	if err := a.howOf(ctx, s, []Trace{t}, &out.HowMade); err != nil {
		return ContextPanel{}, err
	}
	out.EarlierKept = t.Turn - t.Response.EarlierDropped
	if out.EarlierKept < 0 {
		out.EarlierKept = 0
	}
	lists := []struct {
		docs []string
		to   *[]HowSource
	}{{t.Documents, &out.Documents}, {t.Kept, &out.Kept}, {t.Dropped, &out.Dropped}}
	for _, l := range lists {
		titles, hidden, err := a.docTitles(ctx, s, l.docs)
		if err != nil {
			return ContextPanel{}, err
		}
		out.DocsHidden += hidden
		for _, d := range l.docs {
			if ti, ok := titles[d]; ok {
				*l.to = append(*l.to, HowSource{Cite: d, Title: ti})
			}
		}
	}
	if len(t.Response.Memory) > 0 && t.User == s.User() && a.Memory != nil {
		m, _, err := a.Memory.MemoryOf(ctx, a.Tenant, s.User())
		if err != nil {
			return ContextPanel{}, fmt.Errorf("%w: memory: %v", ErrUnavailable, err)
		}
		have := map[string]MemoryLine{}
		for _, l := range m.Lines {
			have[l.ID] = l
		}
		for _, id := range t.Response.Memory {
			if l, ok := have[id]; ok {
				out.Memory = append(out.Memory, l)
			} else {
				out.MemoryGone++ // forgotten since
			}
		}
	}
	return out, nil
}

func memoryTexts(lines []MemoryLine) []string {
	var out []string
	for _, l := range lines {
		out = append(out, l.Text)
	}
	return out
}

// steerOfRequest checks an ask's Keep and Drop.
func steerOfRequest(req Request) (keep, drop []string, err error) {
	if keep, err = steerKeys(req.Keep, "keep"); err != nil {
		return nil, nil, err
	}
	if drop, err = steerKeys(req.Drop, "drop"); err != nil {
		return nil, nil, err
	}
	return keep, drop, nil
}

// steerAndRemember gives ctx the asker's memory lines (SB-4303: their own, ask agent only,
// LAW-25) and the ask's Keep and Drop (SB-5502).
func (a Agent) steerAndRemember(ctx context.Context, s tenancy.Scope, req Request, probe bool) (context.Context, []MemoryLine, []string, []string, error) {
	used := a.memoryUsed(ctx, s, req.Agent, probe)
	keep, drop, err := steerOfRequest(req)
	if err != nil {
		return ctx, nil, nil, nil, err
	}
	return withSteer(withMemory(ctx, memoryTexts(used)), keep, drop), used, keep, drop, nil
}
