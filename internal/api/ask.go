// Package api is the cell's HTTP API (sb-api(3)). Handlers run behind auth.Middleware,
// which puts the request's tenancy.Scope in the context; they never read a tenant or a
// user from the request itself.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// MaxBody bounds a request body.
const MaxBody = 64 << 10

// Asker is the ask agent as the handlers use it.
type Asker interface {
	Ask(ctx context.Context, s tenancy.Scope, req ask.Request) (ask.Response, error)
	Trace(ctx context.Context, s tenancy.Scope, id string) (ask.Trace, error)
}

// FeedbackGiver stores feedback on a trace (SB-310).
type FeedbackGiver interface {
	Give(ctx context.Context, s tenancy.Scope, req ask.FeedbackRequest) error
}

// Ask serves POST /v1/ask and GET /v1/traces/{id} (sb-ask(3)).
type Ask struct {
	Agent    Asker
	Feedback FeedbackGiver // nil: POST /v1/feedback is not mounted
	Release  string
	NewID    func(prefix string) string // trace ids for errors before an answer exists
}

// Register mounts the endpoints on mux.
func (h Ask) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/ask", h.ask)
	mux.HandleFunc("GET /v1/traces/{id}", h.trace)
	if h.Feedback != nil {
		mux.HandleFunc("POST /v1/feedback", h.feedback)
	}
}

func (h Ask) feedback(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var req ask.FeedbackRequest
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(body) > MaxBody || json.Unmarshal(body, &req) != nil {
		h.fail(w, "", errs.New(errs.EINVAL, "The request body must be JSON with a trace and a rating."))
		return
	}
	if err := h.Feedback.Give(r.Context(), s, req); err != nil {
		h.fail(w, req.Trace, mapErr(err))
		return
	}
	h.ok(w, req.Trace, map[string]any{"trace": req.Trace, "stored": true})
}

type traceBody struct {
	ask.Response
	Decisions any `json:"decisions"`
}

func (h Ask) ask(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var req ask.Request
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(body) > MaxBody || json.Unmarshal(body, &req) != nil {
		h.fail(w, "", errs.New(errs.EINVAL, "The request body must be JSON with a question."))
		return
	}
	resp, err := h.Agent.Ask(r.Context(), s, req)
	if err != nil {
		h.fail(w, "", mapErr(err))
		return
	}
	h.ok(w, resp.Trace, resp)
}

func (h Ask) trace(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	t, err := h.Agent.Trace(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, "", mapErr(err))
		return
	}
	h.ok(w, t.ID, traceBody{Response: t.Response, Decisions: t.Decisions})
}

// mapErr turns agent errors into sb-errors(7) codes with user-safe messages.
func mapErr(err error) *errs.Error {
	switch {
	case errors.Is(err, ask.ErrInvalid):
		return errs.New(errs.EINVAL, "%s", safe(err))
	case errors.Is(err, ask.ErrKilled):
		return errs.New(errs.EKILLED, "Answers are switched off for now.")
	case errors.Is(err, ask.ErrNotFound):
		return errs.New(errs.EPERM, "You can't see that trace.")
	case errors.Is(err, ask.ErrBudget):
		return errs.New(errs.EBUDGET, "The model budget or rate limit was reached.")
	default:
		return errs.New(errs.EUNAVAIL, "Answers are unavailable right now.")
	}
}

// safe keeps only the agent's own validation text (it never holds content).
func safe(err error) string {
	msg := err.Error()
	const p = "ask: EINVAL: "
	if len(msg) > len(p) && msg[:len(p)] == p {
		return msg[len(p):]
	}
	return "Malformed request."
}

func (h Ask) headers(w http.ResponseWriter, trace string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SB-Release", h.Release)
	if trace != "" {
		w.Header().Set("X-SB-Trace", trace)
	}
}

func (h Ask) ok(w http.ResponseWriter, trace string, v any) {
	h.headers(w, trace)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

func (h Ask) fail(w http.ResponseWriter, trace string, e *errs.Error) {
	if trace == "" && h.NewID != nil {
		trace = h.NewID("tr_")
	}
	h.headers(w, trace)
	w.WriteHeader(errs.HTTP(e.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
		"code": string(e.Code), "message": e.Msg, "trace": trace, "remedy": errs.Remedy(e.Code)}})
}
