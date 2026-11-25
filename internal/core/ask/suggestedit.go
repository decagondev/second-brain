package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Suggested edits to the source (sb-ask(3), Suggested edits; SB-5201): from the review list,
// a steward sends a document's owner a drafted change to a stale, contradicted, replaced or
// reported passage, composed as the owner from sources they can open. Nothing is written to
// the source: the owner makes the change there, or dismisses it, and a dismissed passage
// isn't suggested again.

// DraftSuggestion's kind of draft, and why one is made.
const (
	DraftEdit = "suggested-edit"

	WhyStale        = "stale"
	WhyContradicted = "contradicted"
	WhyReplaced     = "replaced"
	WhyReported     = "reported"
)

// Limits of a suggested edit.
const (
	MaxEditEvidence = 4
	MaxEditPassage  = 1500
	MaxDismissWhy   = 300
)

// EditRequest is a steward's request for a suggested edit: the passage's cite, why, and the
// documents that support the change (KIND:ID or a cite).
type EditRequest struct {
	Cite     string   `json:"cite"`
	Why      string   `json:"why"`
	Evidence []string `json:"evidence"`
}

// SuggestEdit drafts the change and puts it in the owner's own queue.
func (x Stewards) SuggestEdit(ctx context.Context, s tenancy.Scope, name string, req EditRequest) (StewardQuestion, error) {
	switch req.Why {
	case WhyStale, WhyContradicted, WhyReplaced, WhyReported:
	default:
		return StewardQuestion{}, fmt.Errorf("%w: why is stale, contradicted, replaced or reported", ErrInvalid)
	}
	if len(req.Evidence) > MaxEditEvidence {
		return StewardQuestion{}, fmt.Errorf("%w: at most %d documents of evidence", ErrInvalid, MaxEditEvidence)
	}
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return StewardQuestion{}, err
	}
	doc := docOf(req.Cite)
	if doc == "" {
		return StewardQuestion{}, fmt.Errorf("%w: cite is KIND:ID or KIND:ID@VERSION", ErrInvalid)
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok || x.Owner == nil || x.ScopeOf == nil {
		return StewardQuestion{}, fmt.Errorf("%w: this cell can't suggest edits", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Space: SpaceKey(name)})
	if err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var passage *ports.Chunk
	for _, c := range latest(cs) {
		if c.Kind+":"+c.SourceID == doc && (passage == nil || c.Cite == req.Cite) {
			c := c
			passage = &c
		}
	}
	if passage == nil {
		return StewardQuestion{}, fmt.Errorf("%w: the space holds no document %s", ErrInvalid, doc)
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return StewardQuestion{}, err
	}
	if res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, []ports.Chunk{*passage}, 1); err != nil || len(res.Context) == 0 {
		return StewardQuestion{}, fmt.Errorf("%w: you can't open %s", ErrInvalid, doc)
	}
	if passage.Owner == "" {
		return StewardQuestion{}, fmt.Errorf("%w: %s has no owner at its source to tell", ErrInvalid, doc)
	}
	owner, ok, err := x.Owner(ctx, passage.Owner)
	if err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	if !ok {
		return StewardQuestion{}, fmt.Errorf("%w: the owner isn't one active user", ErrInvalid) // LAW-12
	}
	queue, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, PersonKey(owner))
	if err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	for _, q := range queue {
		if q.Thread != nil && q.Thread.Kind == DraftEdit && q.Thread.Doc == passage.Cite {
			if q.Thread.Discarded {
				return StewardQuestion{}, fmt.Errorf("%w: the owner dismissed a suggestion for this passage", ErrInvalid)
			}
			if q.Reply == nil {
				return StewardQuestion{}, fmt.Errorf("%w: the owner already has a suggestion for this passage", ErrInvalid)
			}
		}
	}
	os, found, err := x.ScopeOf(ctx, passage.Owner)
	if err != nil || !found {
		return StewardQuestion{}, fmt.Errorf("%w: the owner can't be asked as now", ErrInvalid)
	}
	text := passage.Text
	if r := []rune(text); len(r) > MaxEditPassage {
		text = string(r[:MaxEditPassage])
	}
	docs := []string{doc}
	for _, e := range req.Evidence {
		if d := docOf(e); d != "" && !contains(docs, d) {
			docs = append(docs, d)
		}
	}
	ask := fmt.Sprintf("Suggest how to change this passage of %q, which is %s, so that it agrees with the other documents. Give the new wording and what supports it.\n\nPassage: %s",
		passage.Title, req.Why, strings.TrimSpace(text))
	resp, err := x.Agent.ask(ctx, os, Request{Question: ask, Documents: docs}, true) // as the owner, keeping no conversation
	if err != nil {
		return StewardQuestion{}, err
	}
	draft := strings.TrimRight(DraftText(resp.Answer), "\n")
	if draft == "" {
		return StewardQuestion{}, fmt.Errorf("%w: nothing the owner can open supports a change", ErrInvalid)
	}
	title := passage.Title
	if title == "" {
		title = doc
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: owner, Question: "Suggested edit to " + title + " (" + req.Why + ")",
		Space: PersonKey(owner), Trace: resp.Trace, At: x.Now().UTC(),
		Thread: &SavedThread{Agent: "ask", Kind: DraftEdit, Doc: passage.Cite, Text: draft, Sources: draftSources(resp), Asked: ask, Release: resp.Release}}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "suggested",
		PayloadSHA256: hashOf(draft), Reason: "suggested edit to " + passage.Cite + ": " + req.Why, Space: auditSpace(SpaceKey(name))}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// DismissEdit is the owner dismissing a suggested edit in their own queue, with a reason:
// the passage isn't suggested again.
func (x Stewards) DismissEdit(ctx context.Context, s tenancy.Scope, id, why string) error {
	why = strings.Join(strings.Fields(why), " ")
	if why == "" || utf8.RuneCountInString(why) > MaxDismissWhy {
		return fmt.Errorf("%w: a reason is 1 to %d characters", ErrInvalid, MaxDismissWhy)
	}
	q, found, err := x.Store.GetStewardQuestion(ctx, x.Agent.Tenant, id)
	if err != nil {
		return fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	if !found || q.Space != PersonKey(s.User()) || q.Thread == nil || q.Thread.Kind != DraftEdit || q.Thread.Discarded {
		return fmt.Errorf("%w: no open suggested edit %q for you", ErrInvalid, id)
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "dismissed",
		PayloadSHA256: hashOf(why), Reason: "suggested edit to " + q.Thread.Doc + " dismissed"}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	_, err = x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(cur *StewardQuestion) error {
		cur.Thread = &SavedThread{Discarded: true, Agent: cur.Thread.Agent, Kind: DraftEdit, Doc: cur.Thread.Doc}
		cur.Question = "(dismissed)"
		return nil
	})
	return err
}
