// Package api is the cell's HTTP API (sb-api(3)). Handlers run behind auth.Middleware,
// which puts the request's tenancy.Scope in the context; they never read a tenant or a
// user from the request itself.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// MaxBody bounds a request body, and MaxAskBody an ask's, which may carry a file of up to
// 2 MB in base64 (SB-3802).
const (
	MaxBody    = 64 << 10
	MaxAskBody = 3 << 20
)

// Asker is the ask agent as the handlers use it.
type Asker interface {
	Ask(ctx context.Context, s tenancy.Scope, req ask.Request) (ask.Response, error)
	Trace(ctx context.Context, s tenancy.Scope, id string) (ask.Trace, error)
}

// Hower says how an answer was made, checked as the caller now (SB-4606).
type Hower interface {
	How(ctx context.Context, s tenancy.Scope, id string) (ask.HowMade, error)
}

// Passager returns a cite's passage, checked live (SB-3001).
type Passager interface {
	Passage(ctx context.Context, s tenancy.Scope, trace, cite string, start int) (ask.Passage, error)
}

// Brancher starts a conversation from an earlier question of one (SB-5206).
type Brancher interface {
	Branch(ctx context.Context, s tenancy.Scope, conv string, at int) (string, error)
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
	if _, ok := h.Agent.(Hower); ok {
		mux.HandleFunc("GET /v1/traces/{id}/how", h.how) // SB-4606
	}
	if _, ok := h.Agent.(Passager); ok {
		mux.HandleFunc("GET /v1/traces/{id}/passage", h.passage)
	}
	if _, ok := h.Agent.(Contexter); ok {
		mux.HandleFunc("GET /v1/traces/{id}/context", h.contextPanel) // SB-5501
	}
	if _, ok := h.Agent.(WhyNoter); ok {
		mux.HandleFunc("POST /v1/traces/{id}/why-not", h.whyNot) // SB-5506
	}
	if h.Feedback != nil {
		mux.HandleFunc("POST /v1/feedback", h.feedback)
	}
	if _, ok := h.Agent.(Brancher); ok {
		mux.HandleFunc("POST /v1/conversations/{id}/branch", h.branch) // SB-5206
	}
}

// branch is POST /v1/conversations/ID/branch {"at": N}: a new conversation with the first N
// questions of ID (SB-5206).
func (h Ask) branch(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	var body struct {
		At int `json:"at"`
	}
	if b, err := io.ReadAll(io.LimitReader(r.Body, 1<<10)); err != nil || json.Unmarshal(b, &body) != nil {
		h.fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"at": 2}.`))
		return
	}
	next, err := h.Agent.(Brancher).Branch(r.Context(), s, r.PathValue("id"), body.At)
	if err != nil {
		h.fail(w, "", mapErr(err))
		return
	}
	h.ok(w, "", map[string]any{"conversation": next, "from": r.PathValue("id"), "at": body.At})
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
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxAskBody+1))
	if err != nil || len(body) > MaxAskBody {
		h.fail(w, "", errs.New(errs.EINVAL, "The request is too large: a file is at most 2 MB."))
		return
	}
	if json.Unmarshal(body, &req) != nil {
		h.fail(w, "", errs.New(errs.EINVAL, "The request body must be JSON with a question."))
		return
	}
	resp, err := h.Agent.Ask(r.Context(), s, req)
	var over *ask.RateError
	if errors.As(err, &over) {
		h.limited(w, over)
		return
	}
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

// how is how one answer or draft was made, checked as the caller now (SB-4606).
func (h Ask) how(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	v, err := h.Agent.(Hower).How(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, "", mapErr(err))
		return
	}
	h.ok(w, v.Trace, v)
}

func (h Ask) passage(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	cite := r.URL.Query().Get("cite")
	start := -1
	if v := r.URL.Query().Get("start"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			h.fail(w, "", errs.New(errs.EINVAL, "start must be a whole number."))
			return
		}
		start = n
	}
	if cite == "" {
		h.fail(w, "", errs.New(errs.EINVAL, "Name the cite."))
		return
	}
	id := r.PathValue("id")
	p, err := h.Agent.(Passager).Passage(r.Context(), s, id, cite, start)
	if errors.Is(err, ask.ErrSourceDenied) {
		h.fail(w, id, errs.New(errs.EPERM, "You can't open this source now."))
		return
	}
	if err != nil {
		h.fail(w, id, mapErr(err))
		return
	}
	h.ok(w, id, p)
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

// limited answers an ask over asks_per_minute (SB-2305): 429 ERATE with the whole seconds
// to wait, at least 1, in Retry-After and retry_after.
func (h Ask) limited(w http.ResponseWriter, e *ask.RateError) {
	secs := int(math.Ceil(e.Wait.Seconds()))
	if secs < 1 {
		secs = 1
	}
	h.headers(w, e.Trace)
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	w.WriteHeader(errs.HTTP(errs.ERATE))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
		"code": string(errs.ERATE), "message": "You've asked a lot in the last minute.", "trace": e.Trace,
		"remedy": fmt.Sprintf("Wait %d seconds, then ask again.", secs), "retry_after": secs}})
}
