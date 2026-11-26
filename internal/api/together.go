package api

import (
	"net/http"
	"strconv"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Edit a draft together (sb-api(3), SB-5202): the reviewer edits a section in place, and
// accepts or rejects each suggestion Ask for changes made.

// editSection is PUT /v1/spaces/NAME/questions/ID/sections/N {"text": "…"}.
func (h Questions) editSection(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"text": "The dock opens at 7."}, at a section number.`))
		return
	}
	t, err := h.Service.EditSection(r.Context(), s, r.PathValue("name"), r.PathValue("id"), n, body.Text)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"text": t.Text, "editors": t.Editors, "suggestions": t.Suggestions})
}

// decide is POST /v1/spaces/NAME/questions/ID/suggestions/N/accept|reject.
func (h Questions) decideSuggestion(accept bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := h.k().scope(w, r)
		if !ok {
			return
		}
		n, err := strconv.Atoi(r.PathValue("n"))
		if err != nil {
			h.k().out().fail(w, "", errs.New(errs.EINVAL, "No suggestion %q.", r.PathValue("n")))
			return
		}
		t, err := h.Service.Decide(r.Context(), s, r.PathValue("name"), r.PathValue("id"), n, accept)
		if err != nil {
			h.fail(w, err)
			return
		}
		h.k().out().ok(w, "", map[string]any{"text": t.Text, "suggestions": t.Suggestions})
	}
}

// suggestEdit is POST /v1/spaces/NAME/review/suggest: a steward's suggested edit, sent to
// the document's owner (SB-5201).
func (h Questions) suggestEdit(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var req ask.EditRequest
	if !readJSON(r, &req) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"cite": "docs:…@3", "why": "stale", "evidence": ["docs:…"]}.`))
		return
	}
	q, err := h.Service.SuggestEdit(r.Context(), s, r.PathValue("name"), req)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": q.ID, "to": "owner", "doc": q.Thread.Doc}) // never the owner's address or the draft
}

// dismissEdit is POST /v1/me/suggestions/ID/dismiss {"reason": "…"}, for the owner.
func (h Questions) dismissEdit(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"reason": "It is right as it is."}.`))
		return
	}
	if err := h.Service.DismissEdit(r.Context(), s, r.PathValue("id"), body.Reason); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": r.PathValue("id"), "dismissed": true})
}

// tryTrigger is POST /v1/spaces/NAME/triggers/ON/try: what the trigger would have drafted in
// the last 14 days, as the steward, sending nothing (SB-5302).
func (h Questions) tryTrigger(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	if h.TryTrigger == nil {
		h.k().out().fail(w, "", errs.New(errs.EUNAVAIL, "Triggers can't be tried in this cell."))
		return
	}
	res, err := h.TryTrigger(r.Context(), s, r.PathValue("name"), r.PathValue("on"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", res)
}

// requestLoan is POST /v1/spaces/NAME/loans {"lender", "space", "purpose"} (SB-5303).
func (h Questions) requestLoan(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var req ask.LoanRequest
	if !readJSON(r, &req) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"lender": "…", "space": "finance", "purpose": "…"}.`))
		return
	}
	q, err := h.Service.RequestLoan(r.Context(), s, r.PathValue("name"), req)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": q.ID, "to": "lender"})
}

// lend is POST /v1/me/loans/ID/approve|decline, for the lender (SB-5303).
func (h Questions) lend(yes bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := h.k().scope(w, r)
		if !ok {
			return
		}
		if h.LendAs == nil {
			h.k().out().fail(w, "", errs.New(errs.EUNAVAIL, "Access can't be lent in this cell."))
			return
		}
		_, err := h.Service.Lend(r.Context(), s, r.PathValue("id"), yes, func(space string) (tenancy.Scope, error) {
			return h.LendAs(r.Context(), s.User(), space)
		})
		if err != nil {
			h.fail(w, err)
			return
		}
		h.k().out().ok(w, "", map[string]any{"id": r.PathValue("id"), "lent": yes})
	}
}

// draftAction is POST /v1/actions/draft {"trace", "kind"}: the next action drafted from the
// caller's own answer, for them to edit (SB-5304). Nothing is sent.
func (h Questions) draftAction(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct{ Trace, Kind string }
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"trace": "tr_…", "kind": "ticket"}.`))
		return
	}
	d, err := h.Service.Agent.DraftAction(r.Context(), s, body.Trace, body.Kind)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", d)
}

// sendMessage is POST /v1/spaces/NAME/messages {"channel", "text"}: the person's edited
// message, posted by the bot in one of the space's channels, marked as sent by them (SB-5304).
func (h Questions) sendMessage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct{ Channel, Text string }
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"channel": "#ops", "text": "…"}.`))
		return
	}
	m, err := h.Service.SendMessage(r.Context(), s, r.PathValue("name"), body.Channel, body.Text, h.Poster, h.Channels)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", m)
}
