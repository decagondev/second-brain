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
	mux.HandleFunc("POST /v1/spaces/{name}/review/marks", h.mark)
	mux.HandleFunc("POST /v1/spaces/{name}/retired", h.retire)
	mux.HandleFunc("GET /v1/spaces/{name}/retired", h.retired)
	mux.HandleFunc("POST /v1/spaces/{name}/retired/restore", h.restore)
	mux.HandleFunc("POST /v1/spaces/{name}/questions/{id}/claim", h.claim)
	mux.HandleFunc("DELETE /v1/spaces/{name}/questions/{id}/claim", h.claim)
	mux.HandleFunc("GET /v1/spaces/{name}/templates", h.templates)
	mux.HandleFunc("POST /v1/spaces/{name}/templates", h.saveTemplate)
	mux.HandleFunc("PUT /v1/spaces/{name}/templates/{id}", h.saveTemplate)
	mux.HandleFunc("DELETE /v1/spaces/{name}/templates/{id}", h.deleteTemplate)
	mux.HandleFunc("POST /v1/spaces/{name}/evals", h.addEval)
	mux.HandleFunc("GET /v1/spaces/{name}/evals", h.evals)
	mux.HandleFunc("DELETE /v1/spaces/{name}/evals/{id}", h.deleteEval)
}

func (h Questions) k() Knowledge { return Knowledge{Release: h.Release, NewID: h.NewID} }

func (h Questions) fail(w http.ResponseWriter, err error) {
	h.k().out().fail(w, "", questionErr(err))
}

func questionErr(err error) *errs.Error {
	switch {
	case errors.Is(err, ask.ErrNotSteward):
		return errs.New(errs.EPERM, "Only the space's stewards can see or answer its questions.")
	case errors.Is(err, kill.ErrKilled):
		return errs.New(errs.EKILLED, "Publishing notes is switched off for now.")
	case errors.Is(err, toolbus.ErrPerm):
		return errs.New(errs.EPERM, "The note can't be published by you.")
	default:
		return mapErr(err)
	}
}

// MaxBulk is the most documents one bulk mark or retirement takes (SB-3103).
const MaxBulk = 50

// bulkItem is one document's outcome in a bulk action.
type bulkItem struct {
	Cite  string            `json:"cite"`
	OK    bool              `json:"ok,omitempty"`
	Error map[string]string `json:"error,omitempty"`
}

// bulk runs one action per cite, each on its own: a failure stops nothing (SB-3103).
func (h Questions) bulk(w http.ResponseWriter, cites []string, one func(cite string) error) {
	if len(cites) > MaxBulk {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "At most %d documents at once.", MaxBulk))
		return
	}
	items := make([]bulkItem, 0, len(cites))
	for _, c := range cites {
		if err := one(c); err != nil {
			e := questionErr(err)
			items = append(items, bulkItem{Cite: c, Error: map[string]string{"code": string(e.Code), "message": e.Msg}})
			continue
		}
		items = append(items, bulkItem{Cite: c, OK: true})
	}
	h.k().out().ok(w, "", map[string]any{"items": items})
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
	if r.URL.Query().Get("section") == "uncited" { // SB-2803
		res, err := h.Service.Uncited(r.Context(), s, name, h.Reviews)
		if err != nil {
			h.fail(w, err)
			return
		}
		h.k().out().ok(w, "", res)
		return
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

// mark is Reviewed, still right (SB-2501).
func (h Questions) mark(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Cite  string   `json:"cite"`
		Cites []string `json:"cites"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the document's cite."))
		return
	}
	if body.Cites != nil {
		h.bulk(w, body.Cites, func(c string) error {
			_, err := h.Service.MarkReviewed(r.Context(), s, r.PathValue("name"), c)
			return err
		})
		return
	}
	m, err := h.Service.MarkReviewed(r.Context(), s, r.PathValue("name"), body.Cite)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"document": m.Doc, "version": m.Version})
}

// claim is I'll answer this, and DELETE lets it go (SB-2502).
func (h Questions) claim(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	c, err := h.Service.Claim(r.Context(), s, r.PathValue("name"), r.PathValue("id"), r.Method == http.MethodDelete)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"claim": c})
}

type templateView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

// templates are the space's reply templates (SB-2503).
func (h Questions) templates(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	ts, err := h.Service.Templates(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	out := []templateView{}
	for _, t := range ts {
		out = append(out, templateView{t.ID, t.Name, t.Text})
	}
	h.k().out().ok(w, "", map[string]any{"items": out})
}

func (h Questions) saveTemplate(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
		Text string `json:"text"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with a name and a text."))
		return
	}
	t, err := h.Service.SaveTemplate(r.Context(), s, r.PathValue("name"), r.PathValue("id"), body.Name, body.Text)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", templateView{t.ID, t.Name, t.Text})
}

func (h Questions) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteTemplate(r.Context(), s, r.PathValue("name"), r.PathValue("id")); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": r.PathValue("id"), "deleted": true})
}

// retire takes a document out of the space's answers (SB-2801).
func (h Questions) retire(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Cite       string   `json:"cite"`
		Cites      []string `json:"cites"`
		Reason     string   `json:"reason"`
		ReplacedBy string   `json:"replaced_by"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the document's cite and a reason."))
		return
	}
	if body.Cites != nil {
		if body.ReplacedBy != "" {
			h.k().out().fail(w, "", errs.New(errs.EINVAL, "A replacement is named for one document at a time."))
			return
		}
		h.bulk(w, body.Cites, func(c string) error {
			_, err := h.Service.Retire(r.Context(), s, r.PathValue("name"), c, body.Reason, "")
			return err
		})
		return
	}
	ret, err := h.Service.Retire(r.Context(), s, r.PathValue("name"), body.Cite, body.Reason, body.ReplacedBy)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"document": ret.Doc, "reason": ret.Reason, "replaced_by": ret.ReplacedBy, "at": ret.At})
}

// retired lists the space's retirements (SB-2801).
func (h Questions) retired(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	res, err := h.Service.RetiredOf(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", res)
}

// restore brings a retired document back (SB-2801).
func (h Questions) restore(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Document string `json:"document"`
	}
	if !readJSON(r, &body) || body.Document == "" {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the document."))
		return
	}
	if err := h.Service.Restore(r.Context(), s, r.PathValue("name"), body.Document); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"document": body.Document, "restored": true})
}
