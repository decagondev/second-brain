package api

import (
	"net/http"

	"example.com/second-brain/internal/core/ask"
)

// Collections serves the caller's collections (sb-ask(3), Context for the work; SB-4304):
// ids only, each reader seeing only the documents they can open now.
type Collections struct {
	Service ask.CollectionService
	Release string
	NewID   func(prefix string) string
}

// Register mounts the endpoints.
func (h Collections) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/collections", h.list)
	mux.HandleFunc("POST /v1/collections", h.create)
	mux.HandleFunc("PUT /v1/collections/{id}", h.update)
	mux.HandleFunc("DELETE /v1/collections/{id}", h.remove)
}

func (h Collections) lw() Legwork { return Legwork{Release: h.Release, NewID: h.NewID} }

func (h Collections) list(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	cs, err := h.Service.List(r.Context(), s)
	if err != nil {
		h.lw().fail(w, err)
		return
	}
	h.lw().k().out().ok(w, "", map[string]any{"items": cs})
}

func (h Collections) create(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in ask.CollectionInput
	if !decode(r, &in) {
		h.lw().fail(w, ask.ErrInvalid)
		return
	}
	v, err := h.Service.Create(r.Context(), s, in)
	if err != nil {
		h.lw().fail(w, err)
		return
	}
	h.lw().k().out().ok(w, "", v)
}

func (h Collections) update(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in ask.CollectionInput
	if !decode(r, &in) {
		h.lw().fail(w, ask.ErrInvalid)
		return
	}
	v, err := h.Service.Update(r.Context(), s, r.PathValue("id"), in)
	if err != nil {
		h.lw().fail(w, err)
		return
	}
	h.lw().k().out().ok(w, "", v)
}

func (h Collections) remove(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	if err := h.Service.Delete(r.Context(), s, r.PathValue("id")); err != nil {
		h.lw().fail(w, err)
		return
	}
	h.lw().k().out().ok(w, "", map[string]bool{"deleted": true})
}
