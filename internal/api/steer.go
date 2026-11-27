package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// See and steer the context (sb-api(3); SB-5501 to SB-5504).

// Contexter returns an answer's context panel (SB-5501).
type Contexter interface {
	Context(ctx context.Context, s tenancy.Scope, id string) (ask.ContextPanel, error)
}

func (h Ask) contextPanel(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	v, err := h.Agent.(Contexter).Context(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, "", mapErr(err))
		return
	}
	h.ok(w, v.Trace, v)
}

// registerExclusions mounts the caller's own Don't use this for me (SB-5503).
func (h Me) registerExclusions(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/me/exclusions", h.exclusions)
	mux.HandleFunc("POST /v1/me/exclusions", h.exclude(true))
	mux.HandleFunc("DELETE /v1/me/exclusions", h.exclude(false)) // ?document=KIND:ID
}

func (h Me) exclusionErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ask.ErrInvalid) {
		fail(w, errs.New(errs.EINVAL, "%s", strings.TrimPrefix(err.Error(), ask.ErrInvalid.Error()+": ")))
		return
	}
	fail(w, errs.New(errs.EUNAVAIL, "Your list can't be read or changed right now; nothing was changed."))
}

func (h Me) exclusions(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	v, err := h.Agent.ExcludedFor(r.Context(), s)
	if err != nil {
		h.exclusionErr(w, err)
		return
	}
	writeJSON(w, v)
}

func (h Me) exclude(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := tenancy.FromContext(r.Context())
		if err != nil {
			fail(w, errs.New(errs.EAUTH, "Sign in again."))
			return
		}
		doc := r.URL.Query().Get("document")
		if on {
			var body struct {
				Document string `json:"document"`
			}
			if !readJSON(r, &body) {
				fail(w, errs.New(errs.EINVAL, `The body must be JSON such as {"document": "docs:ID"}.`))
				return
			}
			doc = body.Document
		}
		v, err := h.Agent.Exclude(r.Context(), s, doc, on)
		if err != nil {
			h.exclusionErr(w, err)
			return
		}
		writeJSON(w, v)
	}
}
