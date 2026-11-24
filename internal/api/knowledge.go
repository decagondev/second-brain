package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Searcher is search without the model (sb-ask(3), Search; SB-1701).
type Searcher interface {
	Search(ctx context.Context, s tenancy.Scope, req ask.SearchRequest) (ask.SearchResponse, error)
}

// Knowledge serves search and saved and shared questions (sb-ask(3); SB-1701, SB-1703).
type Knowledge struct {
	Search    Searcher
	Questions ask.Questions
	// Start is the reading list of the request's space (SB-1706); nil serves no list.
	Start func(ctx context.Context, s tenancy.Scope) (ask.StartResponse, error)
	// Handover is the handover brief (SB-1707); nil doesn't mount it.
	Handover func(ctx context.Context, s tenancy.Scope, person string) (ask.HandoverResponse, error)
	// People finds whom the caller may brief (SB-2106); nil doesn't mount it.
	People func(ctx context.Context, s tenancy.Scope, q string) ([]ask.Briefable, error)
	// Changes is what changed (SB-1708); nil doesn't mount it.
	Changes func(ctx context.Context, s tenancy.Scope, days int) (ask.ChangesResponse, error)
	// Access asks the owners of sources an answer couldn't show for access (SB-2205); nil
	// doesn't mount it.
	Access  func(ctx context.Context, s tenancy.Scope, trace string) (ask.AccessResponse, error)
	Release string
	NewID   func(prefix string) string
}

// Register mounts the endpoints on mux.
func (h Knowledge) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/search", h.search)
	if h.Start != nil {
		mux.HandleFunc("GET /v1/start", h.start)
	}
	if h.Handover != nil {
		mux.HandleFunc("POST /v1/handover", h.handover)
	}
	if h.Changes != nil {
		mux.HandleFunc("GET /v1/changes", h.changes)
	}
	if h.People != nil {
		mux.HandleFunc("GET /v1/handover/people", h.people)
	}
	if h.Access != nil {
		mux.HandleFunc("POST /v1/access-requests", h.access)
	}
	if h.Questions.Store != nil {
		mux.HandleFunc("POST /v1/saved", h.save)
		mux.HandleFunc("GET /v1/saved", h.saved)
		mux.HandleFunc("DELETE /v1/saved/{id}", h.unsave)
		mux.HandleFunc("PUT /v1/saved/{id}/watch", h.watch)
		mux.HandleFunc("POST /v1/shares", h.share)
		mux.HandleFunc("GET /v1/shares/{id}", h.openShare)
	}
}

func (h Knowledge) out() Ask { return Ask{Release: h.Release, NewID: h.NewID} }

func (h Knowledge) scope(w http.ResponseWriter, r *http.Request) (tenancy.Scope, bool) {
	s, err := tenancy.FromContext(r.Context())
	if err != nil {
		h.out().fail(w, "", errs.New(errs.EAUTH, "Sign in again."))
		return s, false
	}
	return s, true
}

func (h Knowledge) search(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	req := ask.SearchRequest{Query: r.URL.Query().Get("q"), Owner: r.URL.Query().Get("owner")}
	if src := r.URL.Query().Get("sources"); src != "" {
		req.Sources = strings.Split(src, ",")
	}
	if ty := r.URL.Query().Get("type"); ty != "" {
		req.Types = strings.Split(ty, ",")
	}
	if v := r.URL.Query().Get("since"); v != "" {
		n, err := strconv.Atoi(strings.TrimSuffix(v, "d"))
		if err != nil || !strings.HasSuffix(v, "d") || n < 1 || n > ask.MaxSinceDays {
			h.out().fail(w, "", errs.New(errs.EINVAL, "since is a number of days from 1 to 365, such as 30d."))
			return
		}
		req.SinceDays = n
	}
	resp, err := h.Search.Search(r.Context(), s, req)
	if err != nil {
		h.out().fail(w, "", mapErr(err))
		return
	}
	h.out().ok(w, resp.Trace, resp)
}

func (h Knowledge) start(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	resp, err := h.Start(r.Context(), s)
	if err != nil {
		h.out().fail(w, "", mapErr(err))
		return
	}
	h.out().ok(w, "", resp)
}

func (h Knowledge) changes(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	days := 7
	if v := r.URL.Query().Get("since"); v != "" {
		n, err := strconv.Atoi(strings.TrimSuffix(v, "d"))
		if err != nil || !strings.HasSuffix(v, "d") {
			h.out().fail(w, "", errs.New(errs.EINVAL, "since is a number of days, such as 7d."))
			return
		}
		days = n
	}
	resp, err := h.Changes(r.Context(), s, days)
	if err != nil {
		h.out().fail(w, "", mapErr(err))
		return
	}
	h.out().ok(w, "", resp)
}

func (h Knowledge) handover(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Person string `json:"person"`
	}
	if !readJSON(r, &body) || body.Person == "" {
		h.out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with a person."))
		return
	}
	resp, err := h.Handover(r.Context(), s, body.Person)
	if errors.Is(err, ask.ErrNoPerson) {
		h.out().fail(w, "", errs.New(errs.EPERM, "Only the person, their manager or a steward of a space they are in can have their handover brief, and only of someone in the directory."))
		return
	}
	if err != nil {
		h.out().fail(w, "", mapErr(err))
		return
	}
	h.out().ok(w, resp.Trace, resp)
}

func (h Knowledge) people(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	items, err := h.People(r.Context(), s, r.URL.Query().Get("q"))
	if err != nil {
		h.out().fail(w, "", mapErr(err))
		return
	}
	h.out().ok(w, "", map[string]any{"items": items})
}

func questionBody(r *http.Request) (string, bool) {
	var body struct {
		Question string `json:"question"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &body) != nil {
		return "", false
	}
	return body.Question, true
}

func (h Knowledge) qfail(w http.ResponseWriter, err error) {
	if errors.Is(err, ask.ErrNoShare) {
		h.out().fail(w, "", errs.New(errs.EPERM, "That question isn't yours, has expired or doesn't exist."))
		return
	}
	h.out().fail(w, "", mapErr(err))
}

func (h Knowledge) save(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	q, ok := questionBody(r)
	if !ok {
		h.out().fail(w, "", errs.New(errs.EINVAL, "The request body must be JSON with a question."))
		return
	}
	sq, err := h.Questions.Save(r.Context(), s, q)
	if err != nil {
		h.qfail(w, err)
		return
	}
	h.out().ok(w, "", map[string]string{"id": sq.ID})
}

func (h Knowledge) saved(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	items, err := h.Questions.Saved(r.Context(), s)
	if err != nil {
		h.qfail(w, err)
		return
	}
	h.out().ok(w, "", map[string]any{"items": items})
}

func (h Knowledge) unsave(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	if err := h.Questions.Unsave(r.Context(), s, r.PathValue("id")); err != nil {
		h.qfail(w, err)
		return
	}
	h.out().ok(w, "", map[string]bool{"removed": true})
}

func (h Knowledge) watch(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Watch *bool `json:"watch"`
	}
	if !readJSON(r, &body) || body.Watch == nil {
		h.out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"watch": true}.`))
		return
	}
	q, err := h.Questions.Watch(r.Context(), s, r.PathValue("id"), *body.Watch)
	if err != nil {
		h.qfail(w, err)
		return
	}
	h.out().ok(w, "", map[string]any{"id": q.ID, "watch": q.Watch})
}

func (h Knowledge) share(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	q, ok := questionBody(r)
	if !ok {
		h.out().fail(w, "", errs.New(errs.EINVAL, "The request body must be JSON with a question."))
		return
	}
	sh, err := h.Questions.Share(r.Context(), s, q)
	if err != nil {
		h.qfail(w, err)
		return
	}
	h.out().ok(w, "", map[string]any{"id": sh.ID, "url": "/q/" + sh.ID, "expires": sh.Expires})
}

func (h Knowledge) openShare(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	sh, err := h.Questions.Open(r.Context(), s, r.PathValue("id"))
	if err != nil {
		h.qfail(w, err)
		return
	}
	h.out().ok(w, "", sh)
}

func (h Knowledge) access(w http.ResponseWriter, r *http.Request) {
	s, ok := h.scope(w, r)
	if !ok {
		return
	}
	var req struct {
		Trace string `json:"trace"`
	}
	if !readJSON(r, &req) || req.Trace == "" {
		h.out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with trace, the answer's trace."))
		return
	}
	res, err := h.Access(r.Context(), s, req.Trace)
	if err != nil {
		h.out().fail(w, "", mapErr(err))
		return
	}
	h.out().ok(w, "", res)
}
