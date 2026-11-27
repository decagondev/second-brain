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

// Legwork mounts the legwork agents (sb-ask(3), Research, meeting prep and action items;
// SB-4201 to SB-4203): every endpoint is for the signed-in person only, and a stored result
// is checked again as them when read (LAW-26).
type Legwork struct {
	Service ask.Legwork
	Release string
	NewID   func(prefix string) string
}

// Register mounts the endpoints.
func (h Legwork) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/research", h.plan)
	mux.HandleFunc("GET /v1/research", h.list)
	mux.HandleFunc("GET /v1/research/{id}", h.open)
	mux.HandleFunc("POST /v1/research/{id}/run", h.run)
	mux.HandleFunc("POST /v1/research/{id}/steer", h.steer) // SB-4903
	mux.HandleFunc("POST /v1/research/{id}/stop", h.stopLong)
	mux.HandleFunc("POST /v1/research/{id}/go-on", h.goOn)
	mux.HandleFunc("POST /v1/prepare", h.prepare)
	mux.HandleFunc("GET /v1/research/{id}/how", h.how)       // SB-4606
	mux.HandleFunc("POST /v1/account-brief", h.accountBrief) // SB-4605
	mux.HandleFunc("GET /v1/me/events", h.events)
	mux.HandleFunc("GET /v1/me/actions", h.mine)
	mux.HandleFunc("POST /v1/me/actions/{id}/done", h.done)
	mux.HandleFunc("POST /v1/spaces/{name}/actions", h.confirm)
	mux.HandleFunc("GET /v1/spaces/{name}/actions", h.spaceActions)
	mux.HandleFunc("POST /v1/spaces/{name}/review/fix", h.fix)
	mux.HandleFunc("GET /v1/me/inbox", h.inbox)
	mux.HandleFunc("GET /v1/me/left-off", h.leftOff) // SB-5805
	mux.HandleFunc("POST /v1/catch-up", h.catchUp)
	mux.HandleFunc("GET /v1/me/agent-runs", h.agentRuns)
	mux.HandleFunc("GET /v1/agents/cases", h.cases)
	mux.HandleFunc("POST /v1/agents/cases/{id}", h.decideCase)
}

// cases are the caller's candidate cases from feedback (sb-evals(7), Cases from feedback;
// SB-4401).
func (h Legwork) cases(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	cs, err := h.Service.Stewards.CaseCandidates(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"items": cs})
}

// decideCase accepts a candidate with an expect, or dismisses it.
func (h Legwork) decideCase(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body ask.CaseDecision
	if !decode(r, &body) || (body.Dismiss == (body.Expect != "")) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	c, err := h.Service.Stewards.DecideCase(r.Context(), s, r.PathValue("id"), body)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", c)
}

// agentRuns are the runs agents made as the caller (sb-ask(3), What agents did as me; SB-4305).
func (h Legwork) agentRuns(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	runs, err := h.Service.AgentRuns(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"items": runs})
}

// inbox is what waits on the caller across every space (sb-ask(3), One inbox; SB-4301).
func (h Legwork) inbox(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	in, err := h.Service.Inbox(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", in)
}

// leftOff is where the caller left off: up to 3 of their own open things (SB-5805).
func (h Legwork) leftOff(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	lo, err := h.Service.LeftOff(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", lo)
}

func (h Legwork) fix(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Doc string `json:"doc"`
	}
	if !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	out, err := h.Service.DraftFix(r.Context(), s, r.PathValue("name"), body.Doc)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", out)
}

func (h Legwork) k() Knowledge { return Knowledge{Release: h.Release, NewID: h.NewID} }

func (h Legwork) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, ask.ErrNotFound) {
		h.k().out().fail(w, "", errs.New(errs.EPERM, "You can't see that."))
		return
	}
	h.k().out().fail(w, "", questionErr(err))
}

func decode(r *http.Request, v any) bool {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(v) == nil
}

func (h Legwork) plan(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Question  string   `json:"question"`
		Documents []string `json:"documents"`
		Long      bool     `json:"long"`
	}
	if !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	plan := h.Service.Plan
	if body.Long { // SB-4903
		plan = h.Service.PlanLong
	}
	v, err := plan(r.Context(), s, body.Question, body.Documents)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

func (h Legwork) list(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	items, err := h.Service.List(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"items": items})
}

func (h Legwork) open(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	v, err := h.Service.Open(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

func (h Legwork) run(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	v, err := h.Service.Run(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

func (h Legwork) prepare(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body ask.PrepRequest
	if !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	v, err := h.Service.Prepare(r.Context(), s, body)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

func (h Legwork) how(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	v, err := h.Service.How(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

func (h Legwork) accountBrief(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body ask.AccountRequest
	if !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	v, err := h.Service.AccountBrief(r.Context(), s, body)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

func (h Legwork) events(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	evs, err := h.Service.Events(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"events": evs})
}

func (h Legwork) mine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	as, err := h.Service.Mine(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"actions": as})
}

func (h Legwork) done(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	a, err := h.Service.Done(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", a)
}

func (h Legwork) confirm(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Draft string            `json:"draft"`
		Items []ask.DraftAction `json:"items"`
	}
	if !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	as, err := h.Service.Confirm(r.Context(), s, r.PathValue("name"), body.Draft, body.Items)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"actions": as})
}

func (h Legwork) spaceActions(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	as, hidden, err := h.Service.SpaceActions(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"actions": as, "hidden": hidden})
}

// catchUp is Catch me up in the scope's space, branch or a collection (SB-4506).
func (h Legwork) catchUp(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body ask.CatchUpRequest
	if r.ContentLength != 0 && !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	out, err := h.Service.CatchUp(r.Context(), s, body)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", out)
}

// steer changes a long report's sub-questions not yet asked, at a checkpoint (SB-4903).
func (h Legwork) steer(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body ask.Steer
	if !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	v, err := h.Service.SteerLong(r.Context(), s, r.PathValue("id"), body)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}

// stopLong stops a long report at a checkpoint with what it has.
func (h Legwork) stopLong(w http.ResponseWriter, r *http.Request) {
	h.checkpoint(w, r, h.Service.StopLong)
}

// goOn lets a long report at a checkpoint go on now.
func (h Legwork) goOn(w http.ResponseWriter, r *http.Request) {
	h.checkpoint(w, r, h.Service.GoOn)
}

func (h Legwork) checkpoint(w http.ResponseWriter, r *http.Request, f func(context.Context, tenancy.Scope, string) (ask.ResearchView, error)) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	v, err := f(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", v)
}
