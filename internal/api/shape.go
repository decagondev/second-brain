package api

import (
	"net/http"

	"example.com/second-brain/internal/errs"
)

// Shape the work (sb-api(3); SB-4601 to SB-4603): a space's skills, Ask for changes on an
// agent draft, and routines shared with a space.
func (h Questions) registerShape(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/spaces/{name}/skills", h.skills)
	mux.HandleFunc("POST /v1/spaces/{name}/questions/{id}/changes", h.changes)
	mux.HandleFunc("POST /v1/spaces/{name}/questions/{id}/comments", h.comment) // SB-4902
	mux.HandleFunc("GET /v1/spaces/{name}/routines", h.sharedRoutines)
	mux.HandleFunc("PUT /v1/spaces/{name}/routines/{id}", h.shareRoutine)
	mux.HandleFunc("DELETE /v1/spaces/{name}/routines/{id}", h.shareRoutine)
	mux.HandleFunc("POST /v1/spaces/{name}/routines/{id}/join", h.joinRoutine)
	mux.HandleFunc("DELETE /v1/spaces/{name}/routines/{id}/join", h.joinRoutine)
}

// skills are a member's skills in the space.
func (h Questions) skills(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	sk, err := h.Service.SkillsIn(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"skills": sk})
}

// changes redrafts an agent draft with a note, for the steward it ran as (SB-4602).
func (h Questions) changes(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"note": "Lead with the decisions."}.`))
		return
	}
	out, err := h.Service.AskForChanges(r.Context(), s, r.PathValue("name"), r.PathValue("id"), body.Note)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", out)
}

// sharedRoutines are the routines shared with the space, for a member (SB-4603).
func (h Questions) sharedRoutines(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	rs, err := h.Service.SharedRoutines(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"routines": rs})
}

// shareRoutine shares a steward's own routine with the space (PUT) or stops (DELETE).
func (h Questions) shareRoutine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	q, err := h.Service.ShareRoutine(r.Context(), s, r.PathValue("name"), r.PathValue("id"), r.Method == http.MethodPut)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", q)
}

// joinRoutine opts a member in to a shared routine (POST) or out (DELETE).
func (h Questions) joinRoutine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	on := r.Method == http.MethodPost
	q, err := h.Service.JoinRoutine(r.Context(), s, r.PathValue("name"), r.PathValue("id"), on)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": q.ID, "from": r.PathValue("id"), "joined": on, "routine": q.Routine})
}

// comment adds the reviewer's comment on one section of an agent draft (SB-4902).
func (h Questions) comment(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Section int    `json:"section"`
		Text    string `json:"text"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"section": 2, "text": "Say when it starts."}.`))
		return
	}
	cs, err := h.Service.CommentSection(r.Context(), s, r.PathValue("name"), r.PathValue("id"), body.Section, body.Text)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"comments": cs})
}
