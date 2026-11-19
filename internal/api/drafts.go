package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Drafter is the draft-reply agent as the API uses it.
type Drafter interface {
	Draft(ctx context.Context, s tenancy.Scope, req ask.DraftRequest) (ask.DraftResponse, error)
	Outcome(ctx context.Context, s tenancy.Scope, trace, outcome string) error
}

// Drafts serves POST /v1/drafts and POST /v1/drafts/{trace}/outcome (sb-actions(3)).
// It shares Ask's response and error handling.
type Drafts struct {
	Agent Drafter
	Ask   Ask // for headers and errors
}

// Register mounts the endpoints.
func (h Drafts) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/drafts", h.draft)
	mux.HandleFunc("POST /v1/drafts/{trace}/outcome", h.outcome)
}

func (h Drafts) draft(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.Ask.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var req ask.DraftRequest
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil {
		h.Ask.fail(w, "", errs.New(errs.EINVAL, "The request body must be JSON with a context and an instruction."))
		return
	}
	resp, err := h.Agent.Draft(r.Context(), s, req)
	if err != nil {
		h.Ask.fail(w, "", mapErr(err))
		return
	}
	h.Ask.ok(w, resp.Trace, resp)
}

func (h Drafts) outcome(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.Ask.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var req struct {
		Outcome string `json:"outcome"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil {
		h.Ask.fail(w, "", errs.New(errs.EINVAL, "The request body must be JSON with an outcome."))
		return
	}
	trace := r.PathValue("trace")
	if err := h.Agent.Outcome(r.Context(), s, trace, req.Outcome); err != nil {
		h.Ask.fail(w, trace, mapErr(err))
		return
	}
	h.Ask.ok(w, trace, map[string]any{"trace": trace, "outcome": req.Outcome})
}
