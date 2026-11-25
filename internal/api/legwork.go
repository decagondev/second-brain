package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/ask"
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
	mux.HandleFunc("POST /v1/prepare", h.prepare)
	mux.HandleFunc("GET /v1/me/events", h.events)
	mux.HandleFunc("GET /v1/me/actions", h.mine)
	mux.HandleFunc("POST /v1/me/actions/{id}/done", h.done)
	mux.HandleFunc("POST /v1/spaces/{name}/actions", h.confirm)
	mux.HandleFunc("GET /v1/spaces/{name}/actions", h.spaceActions)
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
	}
	if !decode(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	v, err := h.Service.Plan(r.Context(), s, body.Question, body.Documents)
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
