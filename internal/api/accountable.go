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

// Every agent accountable (sb-agents(7), Value, Pause here; SB-4701, SB-4702): an owner's
// value of their agents, and Pause here for a space's stewards and an agent's owner.

// AgentControl mounts GET /v1/agents/value and POST /v1/spaces/NAME/agents/AGENT/pause|resume.
type AgentControl struct {
	Value func(ctx context.Context, s tenancy.Scope) (ask.ValueReport, error)
	Pause func(ctx context.Context, s tenancy.Scope, space, agent, reason string, on bool) error
}

// Register mounts the endpoints.
func (h AgentControl) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/agents/value", h.value)
	mux.HandleFunc("POST /v1/spaces/{name}/agents/{agent}/pause", h.pause(true))
	mux.HandleFunc("POST /v1/spaces/{name}/agents/{agent}/resume", h.pause(false))
}

func (h AgentControl) value(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	v, err := h.Value(r.Context(), s)
	if err != nil {
		fail(w, mapErr(err))
		return
	}
	writeJSON(w, v)
}

func (h AgentControl) pause(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := tenancy.FromContext(r.Context())
		if err != nil {
			fail(w, errs.New(errs.EAUTH, "Sign in again."))
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		if b, err := io.ReadAll(io.LimitReader(r.Body, 1<<12)); err != nil || (len(b) > 0 && json.Unmarshal(b, &body) != nil) {
			fail(w, errs.New(errs.EINVAL, "The body must be {\"reason\": \"…\"}."))
			return
		}
		err = h.Pause(r.Context(), s, r.PathValue("name"), r.PathValue("agent"), body.Reason, on)
		switch {
		case errors.Is(err, ask.ErrNotSteward):
			fail(w, errs.New(errs.EPERM, "Only the space's stewards or the agent's owner can pause it here."))
		case err != nil:
			fail(w, mapErr(err))
		default:
			writeJSON(w, map[string]any{"space": r.PathValue("name"), "agent": r.PathValue("agent"), "paused": on})
		}
	}
}

// agentValue is GET /v1/admin/agents/value: every agent's value (SB-4701).
func (h Admin) agentValue(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.AgentValue == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Agent value can't be read in this cell."))
		return
	}
	v, err := h.AgentValue(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Agent value can't be read right now."))
		return
	}
	h.ok(w, v)
}
