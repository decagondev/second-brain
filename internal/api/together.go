package api

import (
	"net/http"
	"strconv"

	"example.com/second-brain/internal/core/ask"
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
