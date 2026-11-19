package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
	"example.com/second-brain/internal/errs"
)

// Bus is the tool bus as the actions API uses it.
type Bus interface {
	ToolFor(workflow string) (toolbus.Tool, bool)
	Invoke(ctx context.Context, s tenancy.Scope, c toolbus.Call) (approvals.Action, error)
	Get(ctx context.Context, s tenancy.Scope, id string) (toolbus.View, error)
	Approve(ctx context.Context, s tenancy.Scope, id, hash, note string) (approvals.Action, error)
	Reject(ctx context.Context, s tenancy.Scope, id, note string) (approvals.Action, error)
}

// Actions serves sb-actions(3): propose, read, approve, reject.
type Actions struct {
	Bus     Bus
	Release string
	NewID   func(prefix string) string
}

// Register mounts the endpoints.
func (h Actions) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/actions", h.propose)
	mux.HandleFunc("GET /v1/actions/{id}", h.get)
	mux.HandleFunc("POST /v1/actions/{id}/approve", h.approve)
	mux.HandleFunc("POST /v1/actions/{id}/reject", h.reject)
}

// actionBody is the API shape of an action (sb-actions(3)).
type actionBody struct {
	ID            string          `json:"id"`
	State         string          `json:"state"`
	Workflow      string          `json:"workflow"`
	Target        string          `json:"target"`
	Payload       json.RawMessage `json:"payload"`
	PayloadSHA256 string          `json:"payload_sha256"`
	Requester     string          `json:"requester"`
	Approvers     string          `json:"approvers"`
	ExpiresAt     string          `json:"expires_at,omitempty"`
	PolicyReason  string          `json:"policy_reason,omitempty"`
	Approval      string          `json:"approval,omitempty"`
	ApprovedBy    string          `json:"approved_by,omitempty"`
	Receipt       string          `json:"receipt,omitempty"`
	Reason        string          `json:"reason,omitempty"`
}

func body(a approvals.Action) actionBody {
	b := actionBody{ID: a.ID, State: string(a.State), Workflow: a.Workflow, Target: a.Target, Payload: a.Payload,
		PayloadSHA256: a.PayloadSHA256, Requester: a.Requester, Approvers: "group:" + a.Approvers, PolicyReason: a.PolicyReason,
		Approval: a.Approval, ApprovedBy: a.ApprovedBy, Receipt: a.Receipt, Reason: a.Reason}
	if !a.ExpiresAt.IsZero() {
		b.ExpiresAt = a.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return b
}

func (h Actions) read(w http.ResponseWriter, r *http.Request, v any) (tenancy.Scope, bool) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.fail(w, errs.New(errs.EAUTH, "Sign in again."))
		return s, false
	}
	if v == nil {
		return s, true
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, v) != nil {
		h.fail(w, errs.New(errs.EINVAL, "The request body must be JSON as in sb-actions(3)."))
		return s, false
	}
	return s, true
}

func (h Actions) propose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Workflow string          `json:"workflow"`
		Target   string          `json:"target"`
		Payload  json.RawMessage `json:"payload"`
	}
	s, ok := h.read(w, r, &req)
	if !ok {
		return
	}
	tool, enabled := h.Bus.ToolFor(req.Workflow)
	if !enabled {
		h.fail(w, errs.New(errs.EPERM, "That workflow isn't enabled for this tenant."))
		return
	}
	if req.Target == "" || len(req.Payload) == 0 {
		h.fail(w, errs.New(errs.EINVAL, "target and payload are required."))
		return
	}
	a, err := h.Bus.Invoke(r.Context(), s, toolbus.Call{Agent: req.Workflow, Tool: tool.Name, Target: req.Target, Payload: req.Payload})
	if err != nil {
		h.fail(w, mapBusErr(err))
		return
	}
	h.ok(w, body(a))
}

func (h Actions) get(w http.ResponseWriter, r *http.Request) {
	s, ok := h.read(w, r, nil)
	if !ok {
		return
	}
	v, err := h.Bus.Get(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, mapBusErr(err))
		return
	}
	// diff is null when the connector can't read the target's current state.
	h.ok(w, struct {
		actionBody
		Diff []toolbus.Change `json:"diff"`
	}{body(v.Action), v.Diff})
}

func (h Actions) approve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PayloadSHA256 string `json:"payload_sha256"`
		Note          string `json:"note"`
	}
	s, ok := h.read(w, r, &req)
	if !ok {
		return
	}
	a, err := h.Bus.Approve(r.Context(), s, r.PathValue("id"), req.PayloadSHA256, req.Note)
	if err != nil {
		h.fail(w, mapBusErr(err))
		return
	}
	h.ok(w, body(a))
}

func (h Actions) reject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Note string `json:"note"`
	}
	s, ok := h.read(w, r, &req)
	if !ok {
		return
	}
	a, err := h.Bus.Reject(r.Context(), s, r.PathValue("id"), req.Note)
	if err != nil {
		h.fail(w, mapBusErr(err))
		return
	}
	h.ok(w, body(a))
}

func mapBusErr(err error) *errs.Error {
	switch {
	case errors.Is(err, toolbus.ErrNoApproval):
		return errs.New(errs.ENOAPPROVAL, "This action can't be approved or run as shown. Reload it and check the current payload.")
	case errors.Is(err, kill.ErrKilled):
		return errs.New(errs.EKILLED, "This is switched off for now.")
	case errors.Is(err, toolbus.ErrPerm):
		return errs.New(errs.EPERM, "You can't do that with this action.")
	case errors.Is(err, approvals.ErrPayload):
		return errs.New(errs.EINVAL, "The payload must be JSON with whole numbers only.")
	default:
		return errs.New(errs.EUNAVAIL, "Actions are unavailable right now.")
	}
}

func (h Actions) ok(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SB-Release", h.Release)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

func (h Actions) fail(w http.ResponseWriter, e *errs.Error) {
	trace := ""
	if h.NewID != nil {
		trace = h.NewID("tr_")
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SB-Release", h.Release)
	if trace != "" {
		w.Header().Set("X-SB-Trace", trace)
	}
	w.WriteHeader(errs.HTTP(e.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
		"code": string(e.Code), "message": e.Msg, "trace": trace, "remedy": errs.Remedy(e.Code)}})
}
