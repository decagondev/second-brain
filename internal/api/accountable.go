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
	// Shadow is the open shadow windows of the caller's agents (SB-4704).
	Shadow func(ctx context.Context, s tenancy.Scope) (ask.ShadowReport, error)
}

// Register mounts the endpoints.
func (h AgentControl) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/agents/value", h.value)
	if h.Shadow != nil {
		mux.HandleFunc("GET /v1/agents/shadow", h.shadow)
	}
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

func (h AgentControl) shadow(w http.ResponseWriter, r *http.Request) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return
	}
	v, err := h.Shadow(r.Context(), s)
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

// aiUsage is GET /v1/admin/ai/usage[?format=csv]: the AI command centre (SB-5001).
func (h Admin) aiUsage(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.AIUsage == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "AI use can't be read in this cell."))
		return
	}
	u, err := h.AIUsage(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "AI use can't be read right now."))
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ai-usage-`+u.From+`.csv"`)
		_, _ = w.Write(u.CSV())
		return
	}
	h.ok(w, u)
}

// agentRecord is GET /v1/admin/agents/NAME/record[?format=md]: the agent's signed record, or
// its Markdown (SB-4707).
func (h Admin) agentRecord(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.AgentRecord == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Agent records can't be made in this cell."))
		return
	}
	sr, err := h.AgentRecord(r.Context(), r.PathValue("name"))
	switch {
	case errors.Is(err, ask.ErrInvalid):
		h.fail(w, errs.New(errs.EINVAL, "No agent %q in the bundle.", r.PathValue("name")))
		return
	case err != nil:
		h.fail(w, errs.New(errs.EUNAVAIL, "The agent's record can't be made right now."))
		return
	}
	if r.URL.Query().Get("format") == "md" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+sr.Record.Agent+`-record.md"`)
		_, _ = io.WriteString(w, ask.RecordMarkdown(sr))
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+sr.Record.Agent+`-record.json"`)
	h.ok(w, sr)
}

// checks is GET /v1/admin/checks: every kept check, newest first (SB-5005).
func (h Admin) checks(w http.ResponseWriter, _ *http.Request, _ tenancy.Scope) {
	if h.Checks == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Checks can't be read in this cell."))
		return
	}
	cs, err := h.Checks("")
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Checks can't be read right now."))
		return
	}
	h.ok(w, map[string]any{"checks": cs})
}

// check is GET /v1/admin/checks/ID: one check.
func (h Admin) check(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Checks == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Checks can't be read in this cell."))
		return
	}
	cs, err := h.Checks("")
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "Checks can't be read right now."))
		return
	}
	for _, c := range cs {
		if c.ID == r.PathValue("id") {
			h.ok(w, c)
			return
		}
	}
	h.fail(w, errs.New(errs.EINVAL, "No check %q.", r.PathValue("id")))
}

// inventory is GET /v1/admin/inventory[?format=csv]: the signed AI inventory, or its CSV
// (SB-5007).
func (h Admin) inventory(w http.ResponseWriter, r *http.Request, _ tenancy.Scope) {
	if h.Inventory == nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The inventory can't be made in this cell."))
		return
	}
	inv, err := h.Inventory(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EUNAVAIL, "The inventory can't be made right now."))
		return
	}
	day := inv.Inventory.Made
	if len(day) >= 10 {
		day = day[:10]
	}
	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ai-inventory-`+day+`.csv"`)
		_, _ = w.Write(inv.Inventory.CSV())
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="ai-inventory-`+day+`.json"`)
	h.ok(w, inv)
}
