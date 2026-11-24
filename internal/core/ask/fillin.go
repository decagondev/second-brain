package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Fill it in once (sb-ask(3); SB-3705 to SB-3707): a draft from a found-nothing term,
// questions to a document's owner, and the documents a note would repeat.

// NoteDraft marks a steward question as a draft a steward started from a term (SB-3705).
type NoteDraft struct {
	Title   string `json:"title"`
	Comment string `json:"comment"`
}

// MaxDraftCount bounds a draft's count, and MaxDraftTerm its term.
const (
	MaxDraftCount = 10000
	MaxDraftTerm  = 120
)

// DraftNote puts a draft for term in the space's own queue, from the steward.
func (x Stewards) DraftNote(ctx context.Context, s tenancy.Scope, name, term string, count int) (StewardQuestion, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return StewardQuestion{}, err
	}
	if IsPerson(SpaceKey(name)) {
		return StewardQuestion{}, fmt.Errorf("%w: drafts are for a space", ErrInvalid)
	}
	term = strings.Join(strings.Fields(term), " ")
	if term == "" || utf8.RuneCountInString(term) > MaxDraftTerm || count < 1 || count > MaxDraftCount {
		return StewardQuestion{}, fmt.Errorf("%w: a term of 1 to %d characters and a count of 1 to %d", ErrInvalid, MaxDraftTerm, MaxDraftCount)
	}
	d := &NoteDraft{Title: term, Comment: fmt.Sprintf("Asked %d times this week, found nothing", count)}
	if count == 1 {
		d.Comment = "Asked once this week, found nothing"
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: term, Space: SpaceKey(name), At: x.Now().UTC(), Draft: d}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "drafted",
		PayloadSHA256: hashOf(term), Reason: "note drafted from a term", Space: auditSpace(sq.Space)}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// PersonKey is the queue of one person, by user id (SB-3706).
func PersonKey(user string) string { return "person:" + user }

// IsPerson reports whether a queue's space is one person's.
func IsPerson(space string) bool { return strings.HasPrefix(space, "person:") }

// AskOwner sends the asker's question to one person the answer trace names in Who to ask,
// who is one active user now.
func (x Stewards) AskOwner(ctx context.Context, s tenancy.Scope, text, trace, email string) (StewardQuestion, error) {
	q, err := question(s, x.Agent.Tenant, text)
	if err != nil {
		return StewardQuestion{}, err
	}
	t, err := x.Agent.Trace(ctx, s, trace)
	if err != nil || t.User != s.User() || t.Space != s.Space() || t.Response.Agent == AgentSearch {
		return StewardQuestion{}, fmt.Errorf("%w: ask from your own answer, in the same space", ErrInvalid)
	}
	named := false
	for _, p := range t.Response.People {
		named = named || strings.EqualFold(p.Email, strings.TrimSpace(email))
	}
	if !named || x.Owner == nil {
		return StewardQuestion{}, fmt.Errorf("%w: ask someone the answer names in Who to ask", ErrInvalid)
	}
	id, ok, err := x.Owner(ctx, strings.TrimSpace(email))
	if err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if !ok {
		return StewardQuestion{}, fmt.Errorf("%w: that person isn't one active user", ErrInvalid) // LAW-12
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: q, Space: PersonKey(id), Trace: trace, At: x.Now().UTC()}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "asked",
		PayloadSHA256: hashOf(q), Reason: "question to owner"}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// MaxSimilar is the most documents Already written? names (SB-3707).
const MaxSimilar = 3

// similar is up to MaxSimilar documents of the space whose chunks read like text, as the
// steward finds them in a search: checked live as them, so one they can't open is never
// named. Anything that can't be read names none: it is a warning only.
func (x Stewards) similar(ctx context.Context, s tenancy.Scope, name, text string) []Result {
	out := []Result{}
	a := x.Agent
	if a.Embedder == nil || strings.TrimSpace(text) == "" {
		return out
	}
	res, _, err := a.retrieve(ctx, s, text, nil, retrieve.Candidates)
	if err != nil {
		return out
	}
	var cs []ports.Chunk
	for _, c := range res.Context {
		if (SpaceKey(name) == "company" || inSpaces(c.Spaces, SpaceKey(name))) && !isNote(c) { // a note isn't a document it would repeat
			cs = append(cs, c)
		}
	}
	if len(cs) == 0 {
		return out
	}
	texts := []string{text}
	for _, c := range cs {
		texts = append(texts, c.Text)
	}
	vs, err := a.Embedder.Embed(ctx, texts)
	if err != nil || len(vs) != len(texts) {
		return out
	}
	type hit struct {
		c   ports.Chunk
		cos float64
	}
	var hits []hit
	for i, c := range cs {
		if cos := cosine(vs[0], vs[i+1]); cos >= LikeAt {
			hits = append(hits, hit{c, cos})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].cos > hits[j].cos })
	own := a.owners()
	seen := map[string]bool{}
	for _, h := range hits {
		k := h.c.Kind + ":" + h.c.SourceID
		if seen[k] || len(out) == MaxSimilar {
			continue
		}
		seen[k] = true
		r := a.result(ctx, own, h.c)
		r.Snippet = ""
		out = append(out, r)
	}
	return out
}

func inSpaces(spaces []string, name string) bool {
	for _, sp := range spaces {
		if sp == name {
			return true
		}
	}
	return false
}
