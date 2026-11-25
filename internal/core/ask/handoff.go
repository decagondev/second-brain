package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// Hand off to a department (sb-ask(3), Ask each department and hand off; SB-4502): an
// answer whose question names a request category of another space the asker is a member
// of offers Hand off. The draft is the category, the question and the answer's cites the
// asker can open; nothing is stored or sent until the asker sends it as a request, with
// the cites they chose. The stewards see only the shared cites they can open (LAW-01).

// MaxShared is the most cites one request shares.
const MaxShared = 10

// Handoff is the space and category an answer offers to hand the question off to.
type Handoff struct {
	Space    string `json:"space"`
	Category string `json:"category"`
}

// SharedCite is a cite the sender shared with a request, and its title when they sent it.
type SharedCite struct {
	Cite  string `json:"cite"`
	Title string `json:"title,omitempty"`
}

// HandoffDraft is a drafted request: nothing is sent until the asker sends it.
type HandoffDraft struct {
	Space    string       `json:"space"`
	Category string       `json:"category"`
	Text     string       `json:"text"`
	Cites    []SharedCite `json:"cites"`
}

// handoff is the first space by name, other than the one asked in, that s's person is a
// member of and whose request category q names as whole words; nil for none.
func (a Agent) handoff(ctx context.Context, s tenancy.Scope, q string) *Handoff {
	if a.Requests == nil || a.Auth.Spaces == nil {
		return nil
	}
	all, _, err := a.Auth.Spaces(ctx)
	if err != nil {
		return nil // an offer only: no registry, no offer
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	for _, sp := range all {
		if sp.Tenant != a.Tenant || sp.ID == s.Space() || !spaces.Member(sp, s.Groups()) {
			continue
		}
		if cat := category(q, a.Requests(sp.Name)); cat != "" {
			return &Handoff{Space: sp.Name, Category: cat}
		}
	}
	return nil
}

// category is the first of cats that q names as whole words, in any case, or "".
func category(q string, cats []string) string {
	for _, c := range cats {
		if _, used := expand(q, []Term{{Term: strings.ToLower(c)}}); len(used) > 0 {
			return c
		}
	}
	return ""
}

// HandoffDraft drafts a request to space name from s's own answer trace: the category
// given (or the one q names, or the space's first), q as the text, and the answer's cited
// documents s can open now, each once. It stores and sends nothing.
func (x Stewards) HandoffDraft(ctx context.Context, s tenancy.Scope, name, trace, q, cat string) (HandoffDraft, error) {
	text, err := question(s, x.Agent.Tenant, q)
	if err != nil {
		return HandoffDraft{}, err
	}
	if err := x.member(ctx, s, name); err != nil {
		return HandoffDraft{}, err
	}
	var cats []string
	ok := false
	if x.Requests != nil {
		cats, _, ok = x.Requests(name)
	}
	if !ok || len(cats) == 0 {
		return HandoffDraft{}, fmt.Errorf("%w: space %q takes no requests", ErrInvalid, name)
	}
	pick := ""
	for _, c := range cats {
		if strings.EqualFold(c, strings.TrimSpace(cat)) {
			pick = c
		}
	}
	switch {
	case strings.TrimSpace(cat) != "" && pick == "":
		return HandoffDraft{}, fmt.Errorf("%w: category must be one of %s", ErrInvalid, strings.Join(cats, ", "))
	case pick == "":
		if pick = category(text, cats); pick == "" {
			pick = cats[0]
		}
	}
	t, err := x.Agent.Trace(ctx, s, trace) // the asker's own answer only
	if err != nil {
		return HandoffDraft{}, err
	}
	var cites []string
	titles, seen := map[string]string{}, map[string]bool{}
	for _, c := range t.Response.Answer.Claims {
		for _, ci := range c.Cites {
			doc, _, _ := strings.Cut(ci.Cite, "@")
			if !c.Uncited && !seen[doc] {
				seen[doc] = true
				cites = append(cites, ci.Cite)
				titles[ci.Cite] = ci.Title
			}
		}
	}
	open, err := x.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return HandoffDraft{}, err
	}
	d := HandoffDraft{Space: name, Category: pick, Text: text, Cites: []SharedCite{}}
	for _, c := range cites {
		if open[c] && len(d.Cites) < MaxShared {
			d.Cites = append(d.Cites, SharedCite{Cite: c, Title: titles[c]})
		}
	}
	return d, nil
}

// shared checks the cites a sender shares with a request: at most MaxShared, each one they
// can open now, with its title as indexed.
func (x Stewards) shared(ctx context.Context, s tenancy.Scope, cites []string) ([]SharedCite, error) {
	if len(cites) == 0 {
		return nil, nil
	}
	if len(cites) > MaxShared {
		return nil, fmt.Errorf("%w: share at most %d cites", ErrInvalid, MaxShared)
	}
	open, err := x.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return nil, err
	}
	titles, err := x.Agent.citeTitles(ctx, cites)
	if err != nil {
		return nil, err
	}
	var out []SharedCite
	seen := map[string]bool{}
	for _, c := range cites {
		if !open[c] {
			return nil, fmt.Errorf("%w: you can share only what you can open", ErrInvalid)
		}
		if !seen[c] {
			seen[c] = true
			out = append(out, SharedCite{Cite: c, Title: titles[c]})
		}
	}
	return out, nil
}

// sharedView is a request's shared cites as a steward may see them now: those they can
// open, with their titles, and how many they can't.
func (x Stewards) sharedView(ctx context.Context, s tenancy.Scope, sh []SharedCite) ([]SharedCite, int, error) {
	if len(sh) == 0 {
		return nil, 0, nil
	}
	var cites []string
	for _, c := range sh {
		cites = append(cites, c.Cite)
	}
	open, err := x.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return nil, 0, err
	}
	var out []SharedCite
	hidden := 0
	for _, c := range sh {
		if open[c.Cite] {
			out = append(out, c)
		} else {
			hidden++
		}
	}
	return out, hidden, nil
}

// citeTitles are the indexed titles of cites, by cite: the chunk at that version, or the
// document's newest.
func (a Agent) citeTitles(ctx context.Context, cites []string) (map[string]string, error) {
	out := map[string]string{}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return out, nil
	}
	var refs []ports.SourceRef
	for _, c := range cites {
		doc, _, _ := strings.Cut(c, "@")
		if kind, id, ok := strings.Cut(doc, ":"); ok {
			refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
		}
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	newest := firsts(latest(chunks))
	for _, c := range cites {
		doc, _, _ := strings.Cut(c, "@")
		for _, ch := range chunks {
			if ch.Cite == c && ch.Title != "" {
				out[c] = ch.Title
			}
		}
		if out[c] == "" {
			if ch, ok := newest[doc]; ok {
				out[c] = ch.Title
			}
		}
	}
	return out, nil
}
