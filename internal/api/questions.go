package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/tenancy"
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
	mux.HandleFunc("DELETE /v1/spaces/{name}/questions/{id}", h.discard)
	mux.HandleFunc("POST /v1/spaces/{name}/drafts", h.draft)
	mux.HandleFunc("POST /v1/spaces/{name}/questions/{id}/note", h.note)
	if h.Reviews != nil {
		mux.HandleFunc("GET /v1/spaces/{name}/review", h.review)
	}
	mux.HandleFunc("POST /v1/spaces/{name}/review/marks", h.mark)
	mux.HandleFunc("POST /v1/documents/reviewed", h.markOwn)
	mux.HandleFunc("POST /v1/spaces/{name}/retired", h.retire)
	mux.HandleFunc("GET /v1/spaces/{name}/retired", h.retired)
	mux.HandleFunc("POST /v1/spaces/{name}/retired/restore", h.restore)
	mux.HandleFunc("POST /v1/spaces/{name}/questions/{id}/claim", h.claim)
	mux.HandleFunc("DELETE /v1/spaces/{name}/questions/{id}/claim", h.claim)
	mux.HandleFunc("GET /v1/spaces/{name}/templates", h.templates)
	mux.HandleFunc("POST /v1/spaces/{name}/templates", h.saveTemplate)
	mux.HandleFunc("PUT /v1/spaces/{name}/templates/{id}", h.saveTemplate)
	mux.HandleFunc("DELETE /v1/spaces/{name}/templates/{id}", h.deleteTemplate)
	mux.HandleFunc("GET /v1/spaces/{name}/suggested", h.suggested)
	if h.Reviews != nil {
		mux.HandleFunc("GET /v1/spaces/{name}/people", h.people)
	}
	mux.HandleFunc("GET /v1/spaces/{name}/goals", h.goals)
	mux.HandleFunc("GET /v1/spaces/{name}/requests", h.requestForm)
	mux.HandleFunc("POST /v1/spaces/{name}/requests", h.request)
	mux.HandleFunc("POST /v1/spaces/{name}/handoff", h.handoff)
	mux.HandleFunc("PUT /v1/spaces/{name}/waits", h.setWait)
	mux.HandleFunc("DELETE /v1/spaces/{name}/waits", h.clearWait)
	mux.HandleFunc("GET /v1/spaces/{name}/waits", h.waits)
	mux.HandleFunc("GET /v1/spaces/{name}/decisions", h.trail)
	mux.HandleFunc("GET /v1/spaces/{name}/service", h.service)
	h.registerShape(mux) // SB-4601 to SB-4603
	mux.HandleFunc("GET /v1/spaces/{name}/updates", h.updates)
	h.registerPolicies(mux)
	mux.HandleFunc("PUT /v1/spaces/{name}/suggested", h.setSuggested)
	mux.HandleFunc("GET /v1/spaces/{name}/suggested/candidates", h.starterCandidates) // SB-4906
	mux.HandleFunc("GET /v1/spaces/{name}/pins", h.pins)
	mux.HandleFunc("POST /v1/spaces/{name}/pins", h.pin)
	mux.HandleFunc("DELETE /v1/spaces/{name}/pins/{id}", h.unpin)
	mux.HandleFunc("PUT /v1/spaces/{name}/caretakers", h.caretaker)
	mux.HandleFunc("DELETE /v1/spaces/{name}/caretakers", h.caretaker)
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
	case errors.Is(err, ask.ErrNotMember):
		return errs.New(errs.EPERM, "Only the space's members can see this.")
	case errors.Is(err, ask.ErrNotOwner):
		return errs.New(errs.EPERM, "Only the document's current owner can mark it reviewed here.")
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
	var body struct {
		Question, Trace, Note, To string
		Report                    bool
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with a question and its trace."))
		return
	}
	send := h.Service.Send
	if body.To != "" { // SB-3706
		send = func(ctx context.Context, s tenancy.Scope, text, trace string) (ask.StewardQuestion, error) {
			return h.Service.AskOwner(ctx, s, text, trace, body.To)
		}
	}
	if body.Report { // SB-3606
		send = func(ctx context.Context, s tenancy.Scope, text, trace string) (ask.StewardQuestion, error) {
			return h.Service.Report(ctx, s, text, trace, body.Note)
		}
	}
	q, err := send(r.Context(), s, body.Question, body.Trace)
	if err != nil {
		h.fail(w, err)
		return
	}
	to := q.Space
	if len(to) > 4 && to[:4] == "spc_" {
		to = to[4:]
	}
	if ask.IsPerson(to) {
		to = body.To // never the person's user id
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
	items, err := h.Service.Queue(r.Context(), s, queueName(r, s))
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
		Both  bool     `json:"both"` // also to the question it looks like (SB-3402)
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with text and cites."))
		return
	}
	q, err := h.Service.ReplyBoth(r.Context(), s, queueName(r, s), r.PathValue("id"), body.Text, body.Cites, body.Both)
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
	if r.URL.Query().Get("section") == "due" { // SB-3105
		res, err := h.Service.ComingDue(r.Context(), s, name, h.Reviews, after)
		if err != nil {
			h.fail(w, err)
			return
		}
		h.k().out().ok(w, "", res)
		return
	}
	if r.URL.Query().Get("section") == "suspect" { // SB-5101
		res, err := h.Service.Suspect(r.Context(), s, name)
		if err != nil {
			h.fail(w, err)
			return
		}
		h.k().out().ok(w, "", res)
		return
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
	res, err := h.Service.Note(r.Context(), s, queueName(r, s), r.PathValue("id"), req)
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
		Days  int      `json:"days"` // SB-3603
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the document's cite."))
		return
	}
	if body.Cites != nil {
		h.bulk(w, body.Cites, func(c string) error {
			_, err := h.Service.MarkReviewed(r.Context(), s, r.PathValue("name"), c, body.Days)
			return err
		})
		return
	}
	m, err := h.Service.MarkReviewed(r.Context(), s, r.PathValue("name"), body.Cite, body.Days)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"document": m.Doc, "version": m.Version})
}

// markOwn is an owner's Reviewed, still right on their own document (SB-3106).
func (h Questions) markOwn(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Cite string `json:"cite"`
		Days int    `json:"days"` // SB-3603
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the document's cite."))
		return
	}
	m, err := h.Service.MarkOwn(r.Context(), s, body.Cite, body.Days)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", m)
}

// claim is I'll answer this, and DELETE lets it go (SB-2502).
func (h Questions) claim(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	c, err := h.Service.Claim(r.Context(), s, queueName(r, s), r.PathValue("id"), r.Method == http.MethodDelete)
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

// people is who's who in a space, for its members (SB-3906).
func (h Questions) people(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	p, err := h.Service.People(r.Context(), s, r.PathValue("name"), h.Reviews)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", p)
}

// goals are a space's branch's goals, for its members (SB-4101).
func (h Questions) goals(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	g, err := h.Service.Goals(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", g)
}

// updates are a department's teams' latest published updates, for its members (SB-4102).
func (h Questions) updates(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	u, err := h.Service.TeamUpdates(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", u)
}

// suggested is a space's suggested questions, for its members (SB-3307).
func (h Questions) suggested(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	qs, err := h.Service.SuggestedFor(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"questions": qs})
}

func (h Questions) setSuggested(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Questions []string `json:"questions"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<14)).Decode(&body); err != nil {
		h.fail(w, ask.ErrInvalid)
		return
	}
	qs, err := h.Service.SetSuggested(r.Context(), s, r.PathValue("name"), body.Questions)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"questions": qs})
}

// starterCandidates are the questions the cell suggests the space's stewards publish as
// starter questions (SB-4906): never an asker's name.
func (h Questions) starterCandidates(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	cs, err := h.Service.StarterCandidates(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"items": cs})
}

// pins lists the space's verified answers for its stewards (SB-3601).
func (h Questions) pins(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	ps, err := h.Service.PinsOf(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"items": ps})
}

// pin pins a cited reply as the space's verified answer to its question.
func (h Questions) pin(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		QuestionID string `json:"question_id"`
		Question   string `json:"question"`
	}
	if !readJSON(r, &body) || body.QuestionID == "" {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the question_id of an answered question."))
		return
	}
	p, err := h.Service.PinReply(r.Context(), s, r.PathValue("name"), body.QuestionID, body.Question)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", p)
}

func (h Questions) unpin(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	if err := h.Service.Unpin(r.Context(), s, r.PathValue("name"), r.PathValue("id")); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": r.PathValue("id"), "removed": true})
}

// queueName is the path's space, with me for the caller's own queue (SB-3706).
func queueName(r *http.Request, s tenancy.Scope) string {
	if n := r.PathValue("name"); n != "me" {
		return n
	}
	return ask.PersonKey(s.User())
}

// draft starts a note from a found-nothing term (SB-3705).
func (h Questions) draft(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Term  string `json:"term"`
		Count int    `json:"count"`
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with a term and its count."))
		return
	}
	q, err := h.Service.DraftNote(r.Context(), s, r.PathValue("name"), body.Term, body.Count)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": q.ID, "draft": q.Draft})
}

// discard drops an open saved chat thread (SB-3703).
func (h Questions) discard(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	if err := h.Service.Discard(r.Context(), s, queueName(r, s), r.PathValue("id")); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": r.PathValue("id"), "discarded": true})
}

// caretaker names or removes a document's caretaker (SB-3405).
func (h Questions) caretaker(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodDelete {
		if err := h.Service.RemoveCaretaker(r.Context(), s, r.PathValue("name"), r.URL.Query().Get("doc")); err != nil {
			h.fail(w, err)
			return
		}
		h.k().out().ok(w, "", map[string]any{"removed": true})
		return
	}
	var body struct {
		Doc   string `json:"doc"`
		Email string `json:"email"`
	}
	if !readJSON(r, &body) {
		h.fail(w, ask.ErrInvalid)
		return
	}
	c, err := h.Service.NameCaretaker(r.Context(), s, r.PathValue("name"), body.Doc, body.Email)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"doc": c.Doc, "email": c.Email})
}

// requestForm is a space's request categories for its members (SB-4104).
func (h Questions) requestForm(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	f, err := h.Service.RequestFormOf(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", f)
}

// request sends a member's request to a space's stewards (SB-4104).
func (h Questions) request(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct {
		Category, Text string
		Cites          []string // shared with the stewards (SB-4502)
	}
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with a category and text."))
		return
	}
	q, err := h.Service.RequestWith(r.Context(), s, r.PathValue("name"), body.Category, body.Text, body.Cites)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"id": q.ID, "to": r.PathValue("name"), "category": q.Request.Category, "due": q.Request.Due})
}

// handoff drafts a request to a space from the caller's own answer (SB-4502): nothing is
// stored or sent.
func (h Questions) handoff(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body struct{ Trace, Question, Category string }
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, "The body must be JSON with the answer's trace and question."))
		return
	}
	d, err := h.Service.HandoffDraft(r.Context(), s, r.PathValue("name"), body.Trace, body.Question, body.Category)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", d)
}

// setWait marks a request or action item of the space as waiting on another (SB-4504).
func (h Questions) setWait(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	var body ask.WaitMark
	if !readJSON(r, &body) {
		h.k().out().fail(w, "", errs.New(errs.EINVAL, `The body must be JSON such as {"kind": "request", "id": "qs_…", "on": "legal", "reason": "…", "until": "2026-12-01"}.`))
		return
	}
	wt, err := h.Service.SetWait(r.Context(), s, r.PathValue("name"), body)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", wt)
}

// clearWait ends one wait, from either side.
func (h Questions) clearWait(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	if err := h.Service.ClearWait(r.Context(), s, r.PathValue("name"), q.Get("kind"), q.Get("id")); err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", map[string]any{"cleared": true})
}

// waits are what the space waits on and what waits on it, for a member.
func (h Questions) waits(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	ws, err := h.Service.Waits(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", ws)
}

// trail is a topic's decisions in the space, for a member (SB-4505).
func (h Questions) trail(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	t, err := h.Service.Trail(r.Context(), s, r.PathValue("name"), r.URL.Query().Get("topic"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", t)
}

// service is the space's service levels, for its stewards (SB-4507).
func (h Questions) service(w http.ResponseWriter, r *http.Request) {
	s, ok := h.k().scope(w, r)
	if !ok {
		return
	}
	sv, err := h.Service.Service(r.Context(), s, r.PathValue("name"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.k().out().ok(w, "", sv)
}
