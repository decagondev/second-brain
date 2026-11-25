package api

import (
	"net/http"

	"example.com/second-brain/internal/core/ask"
)

// Rhythms serves a space's rhythms and projects (sb-departments(7), Rhythms; sb-spaces(7),
// Projects; SB-4801 to SB-4804). Every step is a draft in its approver's queue; nothing
// here publishes or sends.
type Rhythms struct {
	Service ask.Rhythms
	Release string
	NewID   func(prefix string) string
}

// Register mounts the endpoints.
func (h Rhythms) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/spaces/{name}/rhythms", h.list)
	mux.HandleFunc("POST /v1/spaces/{name}/rhythms", h.start)
	mux.HandleFunc("POST /v1/rhythms/{id}/approve", h.approve)
	mux.HandleFunc("POST /v1/rhythms/{id}/stop", h.stop)
	mux.HandleFunc("POST /v1/rhythms/{id}/consent", h.consent)
	mux.HandleFunc("POST /v1/rhythms/{id}/line", h.line)
	mux.HandleFunc("GET /v1/spaces/{name}/planning", h.planning)
	mux.HandleFunc("GET /v1/spaces/{name}/projects", h.projects)
	mux.HandleFunc("POST /v1/spaces/{name}/projects", h.startProject)
	mux.HandleFunc("GET /v1/projects/{id}", h.project)
	mux.HandleFunc("POST /v1/projects/{id}/decisions", h.decide)
	mux.HandleFunc("POST /v1/projects/{id}/members", h.member)
	mux.HandleFunc("POST /v1/projects/{id}/close", h.close)
}

func (h Rhythms) lw() Legwork { return Legwork{Release: h.Release, NewID: h.NewID} }

// do runs one call as the caller and answers with its result.
func do[T any](h Rhythms, w http.ResponseWriter, r *http.Request, in any, call func() (T, error)) {
	if in != nil && !decode(r, in) {
		h.lw().fail(w, ask.ErrInvalid)
		return
	}
	v, err := call()
	if err != nil {
		h.lw().fail(w, err)
		return
	}
	h.lw().k().out().ok(w, "", v)
}

func (h Rhythms) list(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	do(h, w, r, nil, func() (map[string]any, error) {
		rs, err := h.Service.For(r.Context(), s, r.PathValue("name"))
		return map[string]any{"items": rs}, err
	})
}

func (h Rhythms) start(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in ask.RhythmStart
	do(h, w, r, &in, func() (ask.RhythmView, error) { return h.Service.Start(r.Context(), s, r.PathValue("name"), in) })
}

func (h Rhythms) approve(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	do(h, w, r, nil, func() (ask.RhythmView, error) { return h.Service.Approve(r.Context(), s, r.PathValue("id")) })
}

func (h Rhythms) stop(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	do(h, w, r, nil, func() (ask.RhythmView, error) { return h.Service.Stop(r.Context(), s, r.PathValue("id")) })
}

func (h Rhythms) consent(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in struct {
		Share bool `json:"share"`
	}
	do(h, w, r, &in, func() (ask.RhythmView, error) { return h.Service.Consent(r.Context(), s, r.PathValue("id"), in.Share) })
}

func (h Rhythms) projects(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	do(h, w, r, nil, func() (map[string]any, error) {
		ps, err := h.Service.ProjectsIn(r.Context(), s, r.PathValue("name"))
		return map[string]any{"items": ps}, err
	})
}

func (h Rhythms) startProject(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in ask.ProjectInput
	do(h, w, r, &in, func() (ask.ProjectView, error) {
		return h.Service.StartProject(r.Context(), s, r.PathValue("name"), in)
	})
}

func (h Rhythms) project(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	do(h, w, r, nil, func() (ask.ProjectView, error) { return h.Service.Project(r.Context(), s, r.PathValue("id")) })
}

func (h Rhythms) decide(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in ask.DecisionInput
	do(h, w, r, &in, func() (ask.ProjectView, error) { return h.Service.Decide(r.Context(), s, r.PathValue("id"), in) })
}

func (h Rhythms) member(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in struct {
		Email string `json:"email"`
	}
	do(h, w, r, &in, func() (map[string]any, error) {
		return map[string]any{"added": in.Email}, h.Service.AddMember(r.Context(), s, r.PathValue("id"), in.Email)
	})
}

func (h Rhythms) close(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in ask.CloseInput
	do(h, w, r, &in, func() (ask.ProjectView, error) { return h.Service.CloseProject(r.Context(), s, r.PathValue("id"), in) })
}

func (h Rhythms) line(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	do(h, w, r, &in, func() (ask.RhythmView, error) { return h.Service.AddLine(r.Context(), s, r.PathValue("id"), in.Text) })
}

func (h Rhythms) planning(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lw().k().scope(w, r)
	if !ok {
		return
	}
	do(h, w, r, nil, func() (ask.PlanningResponse, error) { return h.Service.Planning(r.Context(), s, r.PathValue("name")) })
}
