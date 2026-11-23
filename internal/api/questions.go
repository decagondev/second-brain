package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/toolbus"
	"example.com/second-brain/internal/errs"
)

// Questions serves questions to the stewards and notes (sb-ask(3); SB-1704, SB-1705).
type Questions struct {
	Service ask.Stewards
	// Reviews are the traces the review list counts from, and ReviewAfter a space's
	// review_after_days (SB-1709); a nil Reviews doesn't mount the list.
	Reviews     ask.ReviewSource
	ReviewAfter func(space string) int
	Release     string
	NewID       func(prefix string) string
}

// Register mounts the endpoints.
func (h Questions) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/questions", h.send)
	mux.HandleFunc("GET /v1/questions", h.mine)
	mux.HandleFunc("GET /v1/spaces/{name}/questions", h.queue)
	mux.HandleFunc("POST /v1/spaces/{name}/questions/{id}/reply", h.reply)
	mux.HandleFunc("POST /v1/spaces/{name}/questions/{id}/note", h.note)
	if h.Reviews != nil {
		mux.HandleFunc("GET /v1/spaces/{name}/review", h.review)
	}
	mux.HandleFunc("POST /v1/spaces/{name}/evals", h.addEval)
	mux.HandleFunc("GET /v1/spaces/{name}/evals", h.evals)
	mux.HandleFunc("DELETE /v1/spaces/{name}/evals/{id}", h.deleteEval)
}

func (h Questions) k() Knowledge { return Knowledge{Release: h.Release, NewID: h.NewID} }

func (h Questions) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ask.ErrNotSteward):
		h.k().out().fail(w, "", errs.New(errs.EPERM, "Only the space's stewards can see or answer its questions."))
	case errors.Is(err, kill.ErrKilled):
		h.k().out().fail(w, "", errs.New(errs.EKILLED, "Publishing notes is switched off for now."))
	case errors.Is(err, toolbus.ErrPerm):
		h.k().out().fail(w, "", errs.New(errs.EPERM, "The note can't be published by you."))
	default:
		h.k().out().fail(w, "", mapErr(err))
	}
}

func readJSON(r *http.Request, v any) bool {
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	return err == nil && len(b) <= MaxBody && json.Unmarshal(b, v) == nil
}

func (h Questions) send(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct{ Question, Trace string }
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with a question and its trace."))
		return
	}
	q, err := h.Service.Send(r.Context(), s, body.Question, body.Trace)
	if err != nil {
		h.fail(w, err)
		return
	}
	to := q.Space
	if len(to) > 4 && to[:4] == "spc_" {
		to = to[4:]
	}
	h.k().out().ok(w, "", map[string]string{"id": q.ID, "to": to})
}

func (h Questions) mine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	items, err := h.Service.Mine(r.Context(), s)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"items": items})
}

func (h Questions) queue(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	items, err := h.Service.Queue(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"items": items})
}

func (h Questions) reply(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Text  string   `json:"text"`
		Cites []string `json:"cites"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with text and cites."))
		return
	}
	q, err := h.Service.Reply(r.Context(), s, r.PathValue("name"), r.PathValue("id"), body.Text, body.Cites)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": q.ID, "state": "answered"})
}

func (h Questions) review(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	name := r.PathValue("name")
	after := 0
	if h.ReviewAfter != nil {
		after = h.ReviewAfter(name)
	}
	res, err := h.Service.Review(r.Context(), s, name, h.Reviews, after)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", res)
}

func (h Questions) note(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var req ask.NoteRequest
	if !readJSON(r, &req) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with a title, and confirm to publish."))
		return
	}
	res, err := h.Service.Note(r.Context(), s, r.PathValue("name"), r.PathValue("id"), req)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", res)
}
