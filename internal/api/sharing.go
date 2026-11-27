package api

import (
	"errors"
	"net/http"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Know who can see (sb-api(3); SB-5401 to SB-5404): the owner's Shared on purpose, and who
// can open a document or every cite of a shared answer.
func (h Questions) registerSharing(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/documents/sharing/confirm", h.confirmWide) // SB-5402
	mux.HandleFunc("GET /v1/documents/{id}/audience", h.audience)       // SB-5404
	mux.HandleFunc("GET /v1/audience", h.audience)                      // SB-5404: doc=KIND:ID, repeated
}

// confirmWide is Shared on purpose, for the document's owner.
func (h Questions) confirmWide(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Cite   string `json:"cite"`
		Reason string `json:"reason"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the document's cite and a reason."))
		return
	}
	c, err := h.Service.ConfirmWide(r.Context(), s, body.Cite, body.Reason)
	if errors.Is(err, ask.ErrNotOwner) {
		h.k().out().fail(w, "", errs.New(errs.EPERM, "Only the document's current owner can confirm its sharing."))
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", c)
}

// audience is who can open a document, or every document named, now.
func (h Questions) audience(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	docs := r.URL.Query()["doc"]
	if id := r.PathValue("id"); id != "" {
		docs = []string{id}
	}
	a, err := h.Service.Agent.Audience(r.Context(), s, docs)
	if errors.Is(err, ask.ErrNotFound) {
		h.k().out().fail(w, "", errs.New(errs.EPERM, "No document you can open by that id."))
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", a)
}

// agentReach is what an agent can read now, for operators (SB-5406).
func (h Admin) agentReach(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.AgentReach == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Agent reach can't be worked out in this cell."))
		return
	}
	rc, err := h.AgentReach(r.Context(), r.PathValue("name"))
	switch {
	case errors.Is(err, ask.ErrInvalid):
		h.fail(w, errs.New(errs.EINVAL, "No agent %q in the bundle.", r.PathValue("name")))
		return
	case err != nil:
		h.fail(w, errs.New(errs.EUNAVAIL, "The agent's reach can't be worked out right now."))
		return
	}
	h.ok(w, rc)
}

// sensitive is every space's sensitive counts, for operators (SB-5407).
func (h Admin) sensitive(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Sensitive == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The sensitive map can't be counted in this cell."))
		return
	}
	sm, err := h.Sensitive(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The sensitive map can't be counted right now."))
		return
	}
	h.ok(w, map[string]any{"spaces": sm})
}
