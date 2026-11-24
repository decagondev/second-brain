package api

import (
	"encoding/json"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/ask"
)

// Policies to read (sb-spaces(7), Policies to read; SB-4103).

func (h Questions) registerPolicies(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/spaces/{name}/policies", h.markPolicy)
	mux.HandleFunc("GET /v1/spaces/{name}/policies", h.policies)
	mux.HandleFunc("POST /v1/spaces/{name}/policies/read", h.readPolicy)
	mux.HandleFunc("GET /v1/spaces/{name}/policies/status", h.policyStatus)
}

func (h Questions) markPolicy(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Doc string `json:"doc"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<12)).Decode(&body); err != nil {
		h.fail(w, ask.ErrInvalid)
		return
	}
	p, err := h.Service.MarkPolicy(r.Context(), s, r.PathValue("name"), body.Doc)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", p)
}

func (h Questions) policies(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	ps, err := h.Service.PoliciesFor(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"policies": ps})
}

func (h Questions) readPolicy(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Doc     string `json:"doc"`
		Version int    `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<12)).Decode(&body); err != nil {
		h.fail(w, ask.ErrInvalid)
		return
	}
	if err := h.Service.ReadPolicy(r.Context(), s, r.PathValue("name"), body.Doc, body.Version); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"doc": body.Doc, "version": body.Version, "read": true})
}

func (h Questions) policyStatus(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	st, err := h.Service.PolicyStatuses(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"policies": st})
}
