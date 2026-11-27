package ask

import (
	"context"
	"fmt"
	"strings"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Why not this document? (sb-ask(3), Why not this document; SB-5506) and the retrieval view
// for stewards (SB-5507): an answer's retrieval replayed, as the person asking why, never as
// anyone else. A document the person can't open answers like one that doesn't exist.

// The reasons Why not gives.
const (
	WhyNoSuch      = "no_such_document"
	WhyNotIndexed  = "not_indexed"
	WhyOutside     = "outside_scope"
	WhyRankedBelow = "ranked_below"
	WhySuspect     = "suspect"
	WhyHeld        = "held"
	WhyDropped     = "dropped"
	WhyExcluded    = "excluded"
	WhyRetired     = "retired"
	WhyUsed        = "used"
)

// NoSuchDocument is the one reply for a document the person can't open, or that isn't there.
const NoSuchDocument = "No document you can open by that name."

// WhyNot is the 200 body of POST /v1/traces/{id}/why-not.
type WhyNot struct {
	Reason string `json:"reason"`
	Text   string `json:"text"`
	Doc    string `json:"document,omitempty"`
	Title  string `json:"title,omitempty"`
	// Rank is the document's best place among the candidates (0: not among them), Cut how
	// many passages the answer read, and Closest the cite of its best passage.
	Rank    int    `json:"rank,omitempty"`
	Cut     int    `json:"cut,omitempty"`
	Closest string `json:"closest,omitempty"`
	// LastSync is when its connector last synced, for one not indexed yet.
	LastSync string `json:"last_sync,omitempty"`
}

func noSuch() WhyNot { return WhyNot{Reason: WhyNoSuch, Text: NoSuchDocument} }

// questionOf is trace t's question, from its asker's conversation; "" when it can't be read.
func (a Agent) questionOf(ctx context.Context, t Trace) string {
	qs, found, err := a.Store.Questions(ctx, a.Tenant, t.User, t.Response.Conversation)
	if err != nil || !found || t.Turn < 0 || t.Turn >= len(qs) {
		return ""
	}
	return qs[t.Turn]
}

// resolveDoc finds the document name points at: KIND:ID, a link to its source, or its
// title, among the documents of space (every space company-wide). "" for none.
func (a Agent) resolveDoc(ctx context.Context, s tenancy.Scope, name string) (string, []ports.Chunk, error) {
	name = strings.TrimSpace(name)
	lk, ok := a.Index.(ports.DocLookup)
	if !ok || name == "" {
		return "", nil, nil
	}
	if kind, id, ok := strings.Cut(name, ":"); ok && Kinds[kind] && id != "" && !strings.HasPrefix(id, "//") {
		d, _, _ := strings.Cut(name, "@")
		kind, id, _ = strings.Cut(d, ":")
		cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
		return d, latest(cs), err
	}
	var spaces []string
	if s.Space() != "company" && s.Space() != "" {
		spaces = []string{s.Space()}
	} else if a.Auth.Spaces != nil {
		all, _, err := a.Auth.Spaces(ctx)
		if err != nil {
			return "", nil, err
		}
		for _, sp := range all {
			spaces = append(spaces, sp.ID)
		}
	}
	for _, sp := range spaces {
		cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Space: sp})
		if err != nil {
			return "", nil, err
		}
		for _, c := range latest(cs) {
			link := ""
			if a.Link != nil {
				link = a.Link(c)
			}
			if strings.EqualFold(c.Title, name) || (link != "" && link == name) {
				k := c.Kind + ":" + c.SourceID
				var mine []ports.Chunk
				for _, x := range latest(cs) {
					if x.Kind+":"+x.SourceID == k {
						mine = append(mine, x)
					}
				}
				return k, mine, nil
			}
		}
	}
	return "", nil, nil
}

// WhyNot says why trace id's answer didn't use the document name points at, for whoever
// may read the trace, checked as them now.
func (a Agent) WhyNot(ctx context.Context, s tenancy.Scope, id, name string) (WhyNot, error) {
	t, err := a.Trace(ctx, s, id)
	if err != nil {
		return WhyNot{}, err
	}
	doc, chunks, err := a.resolveDoc(ctx, s, name)
	if err != nil {
		return WhyNot{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	if doc == "" {
		return noSuch(), nil
	}
	kind, sid, _ := strings.Cut(doc, ":")
	probe := ports.Chunk{Kind: kind, SourceID: sid}
	if len(chunks) > 0 {
		probe = firsts(chunks)[doc]
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return WhyNot{}, err
	}
	if v := a.Auth.Check(ctx, who, []ports.Chunk{probe}); v[doc] != ports.VerdictAllow {
		return noSuch(), nil // can't open it at its source: the same as no such document
	}
	out := WhyNot{Doc: doc, Title: probe.Title}
	if len(chunks) == 0 {
		out.Reason, out.Text = WhyNotIndexed, "It isn't indexed yet."
		if a.SyncLag != nil {
			if ns, err := a.SyncLag(ctx, []string{kind}); err == nil && len(ns) > 0 && ns[0].LastSuccess != nil {
				out.LastSync = ns[0].LastSuccess.UTC().Format("2006-01-02T15:04:05Z")
				out.Text = "It isn't indexed yet; its source last synced at " + out.LastSync + "."
			}
		}
		return out, nil
	}
	for _, c := range t.Response.Answer.Claims {
		for _, ci := range c.Cites {
			if docKey(ci.Cite) == doc {
				out.Reason, out.Text = WhyUsed, "The answer cites it."
				return out, nil
			}
		}
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, []ports.Chunk{probe}, 1)
	if err != nil {
		return WhyNot{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) == 0 {
		out.Reason, out.Text = WhyOutside, "It isn't in the space this was asked in, or its label keeps it out of answers here."
		return out, nil
	}
	if contains(t.Dropped, doc) {
		out.Reason, out.Text = WhyDropped, "You dropped it from this conversation."
		return out, nil
	}
	if ex, err := a.excluded(ctx, s); err != nil {
		return WhyNot{}, err
	} else if ex[doc] {
		out.Reason, out.Text = WhyExcluded, "You set it to not be used for you; undo that on My data."
		return out, nil
	}
	if len(t.Documents) > 0 && !contains(t.Documents, doc) {
		out.Reason, out.Text = WhyOutside, "The question asked about other documents only."
		return out, nil
	}
	if kept, err := a.dropRetired(ctx, s, res.Context); err != nil {
		return WhyNot{}, err
	} else if len(kept) == 0 {
		out.Reason, out.Text = WhyRetired, "A steward retired it from this space's answers."
		return out, nil
	}
	if kept, err := a.dropHeld(ctx, s, res.Context); err != nil {
		return WhyNot{}, err
	} else if len(kept) == 0 {
		out.Reason, out.Text = WhyHeld, "It is held back because its source shares it too widely; its owner can confirm or narrow that."
		return out, nil
	}
	suspect := true
	for _, c := range chunks {
		suspect = suspect && ingest.Suspect(c.Text, c.Flags)
	}
	if suspect && a.suspectMode(t.Space) == SuspectSkip {
		out.Reason, out.Text = WhySuspect, "Its passages read like instructions to the AI, so they were skipped."
		return out, nil
	}
	out.Reason, out.Cut = WhyRankedBelow, retrieve.Budget
	if q := a.questionOf(ctx, t); q != "" {
		cands, err := a.ranked(ctx, s, q)
		if err != nil {
			return WhyNot{}, err
		}
		for i, c := range cands {
			if docKey(c.Chunk.Cite) == doc {
				out.Rank, out.Closest = i+1, c.Chunk.Cite
				break
			}
		}
	}
	if out.Rank > 0 {
		out.Text = fmt.Sprintf("Other passages ranked higher: its closest passage came %d, and the answer read the top %d it could use.", out.Rank, out.Cut)
	} else {
		out.Text = fmt.Sprintf("None of its passages was among the %d closest to the question.", retrieve.Candidates)
	}
	return out, nil
}

// ranked is the fused candidates for q, as s would search, before any check.
func (a Agent) ranked(ctx context.Context, s tenancy.Scope, q string) ([]ports.Candidate, error) {
	var emails []string
	if a.Emails != nil {
		e, err := a.Emails(ctx, s.User())
		if err != nil {
			return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		emails = e
	}
	query := ports.Query{Text: q, Limit: retrieve.Candidates}
	if a.Embedder != nil {
		vs, err := a.Embedder.Embed(ctx, []string{q})
		if err != nil || len(vs) != 1 {
			return nil, fmt.Errorf("%w: embedder", ErrUnavailable)
		}
		query.Vector = vs[0]
	}
	cands, err := a.Index.Search(ctx, s, retrieve.Tokens(s, emails), query)
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	return retrieve.Fuse(cands), nil
}

// RetrievalRow is one ranked passage of a replayed ask, as a steward sees it (SB-5507).
type RetrievalRow struct {
	Rank        int     `json:"rank"`
	Cite        string  `json:"cite"`
	Title       string  `json:"title,omitempty"`
	KeywordRank int     `json:"keyword_rank,omitempty"`
	VectorRank  int     `json:"vector_rank,omitempty"`
	Score       float64 `json:"score"`
	// Why is kept, or why it was cut: budget, per_document, refused, retired, held, suspect.
	Why string `json:"why"`
}

// RetrievalView is the replay, rows the steward can open and a count of the rest.
type RetrievalView struct {
	Trace  string         `json:"trace"`
	Rows   []RetrievalRow `json:"rows"`
	Hidden int            `json:"hidden"`
}

// Retrieval replays the retrieval of a reported or unhelpful answer in space name, as the
// steward s (SB-5507): never as the asker, so the asker's memory and exclusions play no
// part and are never shown.
func (x Stewards) Retrieval(ctx context.Context, s tenancy.Scope, name, trace string) (RetrievalView, error) {
	out := RetrievalView{Trace: trace, Rows: []RetrievalRow{}}
	if err := x.reviewer(ctx, s, name); err != nil {
		return out, err
	}
	a := x.Agent
	t, found, err := a.Store.GetTrace(ctx, a.Tenant, trace)
	if err != nil {
		return out, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	if !found || t.Space != SpaceKey(name) {
		return out, fmt.Errorf("%w: no such answer in this space", ErrInvalid)
	}
	if ok, err := x.flagged(ctx, name, trace); err != nil {
		return out, err
	} else if !ok {
		return out, fmt.Errorf("%w: only a reported or unhelpful answer can be replayed", ErrInvalid)
	}
	q := a.questionOf(ctx, t)
	if q == "" {
		return out, fmt.Errorf("%w: the question is no longer kept", ErrInvalid)
	}
	cands, err := a.ranked(ctx, s, q)
	if err != nil {
		return out, err
	}
	var chunks []ports.Chunk
	for _, c := range cands {
		chunks = append(chunks, c.Chunk)
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, chunks, len(chunks))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	allowed := map[string]bool{}
	for _, c := range res.Context {
		allowed[c.ID] = true
	}
	notRetired, err := a.dropRetired(ctx, s, res.Context)
	if err != nil {
		return out, err
	}
	notHeld, err := a.dropHeld(ctx, s, notRetired)
	if err != nil {
		return out, err
	}
	in := func(cs []ports.Chunk) map[string]bool {
		m := map[string]bool{}
		for _, c := range cs {
			m[c.ID] = true
		}
		return m
	}
	retiredOK, heldOK := in(notRetired), in(notHeld)
	perDoc := map[string]int{}
	used := 0
	for i, c := range cands {
		if !allowed[c.Chunk.ID] {
			out.Hidden++ // the steward can't open it: counted, never named
			continue
		}
		row := RetrievalRow{Rank: i + 1, Cite: c.Chunk.Cite, Title: c.Chunk.Title, KeywordRank: c.KeywordRank, VectorRank: c.VectorRank, Score: retrieve.Score(c)}
		k := docKey(c.Chunk.Cite)
		switch {
		case !retiredOK[c.Chunk.ID]:
			row.Why = "retired"
		case !heldOK[c.Chunk.ID]:
			row.Why = "held"
		case ingest.Suspect(c.Chunk.Text, c.Chunk.Flags) && a.suspectMode(t.Space) == SuspectSkip:
			row.Why = "suspect"
		case perDoc[k] >= MaxPerDocument:
			row.Why = "per_document"
		case used >= retrieve.Budget:
			row.Why = "budget"
		default:
			row.Why = "kept"
			perDoc[k]++
			used++
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// flagged says trace was reported to the space's stewards (SB-3606) or rated unhelpful
// (SB-3604).
func (x Stewards) flagged(ctx context.Context, name, trace string) (bool, error) {
	if x.Feedback != nil {
		f, ok, err := x.Feedback.GetFeedback(ctx, x.Agent.Tenant, trace)
		if err != nil {
			return false, fmt.Errorf("%w: feedback: %v", ErrUnavailable, err)
		}
		if ok && f.Rating == "down" {
			return true, nil
		}
	}
	qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return false, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	for _, q := range qs {
		if q.Trace == trace && q.Report != nil {
			return true, nil
		}
	}
	return false, nil
}
