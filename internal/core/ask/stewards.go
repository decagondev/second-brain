package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// Questions to the stewards (sb-ask(3); SB-1704) and replies published as notes (SB-1705).
const (
	MaxReply       = 4000
	MaxReplyCites  = 10
	MaxNoteReaders = 500
	MaxNoteTitle   = 200
	// NoteTool is the tool a steward's publish calls on the folder connector.
	NoteTool = "note-publish"
)

// Askable are the notices after which a question may go to the stewards.
var Askable = map[string]bool{NoticeNoSources: true, NoticeNoAccess: true, NoticeMostUncited: true}

// StewardReply is a steward's answer: their own text and the cites they added.
type StewardReply struct {
	By    string    `json:"by"`
	Text  string    `json:"text"`
	Cites []string  `json:"cites"`
	At    time.Time `json:"at"`
	Note  string    `json:"note,omitempty"` // the note's receipt, once published (SB-1705)
	// NoteTitle is the note's title, so it can be republished to fewer readers (SB-2004).
	NoteTitle string `json:"note_title,omitempty"`
	// NoteAt is when it was first published as a note, so the asker is told once (SB-2102).
	NoteAt time.Time `json:"note_at,omitzero"`
	// NoteQuestion is whether the steward kept the question's text in the note (SB-3401).
	NoteQuestion bool `json:"note_question,omitempty"`
}

// StewardQuestion is one question sent to the people who know.
type StewardQuestion struct {
	ID       string        `json:"id"`
	Tenant   string        `json:"tenant"`
	Asker    string        `json:"asker"`
	Question string        `json:"question"`
	Space    string        `json:"space"` // spc_NAME or company
	Trace    string        `json:"trace"`
	At       time.Time     `json:"at"`
	Reply    *StewardReply `json:"reply,omitempty"`
	// Claim is a steward's "I'll answer this" (SB-2502); a reply ends it.
	Claim *StewardClaim `json:"claim,omitempty"`
	// FromSpace is the space it was sent from when that space had no active steward, so
	// it went to the company queue (SB-3102), by name.
	FromSpace string `json:"from_space,omitempty"`
	// Like is the question of the same queue it looks like when it was sent (SB-3402).
	Like string `json:"like,omitempty"`
	// Also is company when the company-wide answerers see it too, because it waited
	// Waited working days in its space (SB-3404).
	Also   string `json:"also,omitempty"`
	Waited int    `json:"waited,omitempty"`
	// Up are the queues above its own it has reached, in order, by space key: a team's
	// department's, then company (SB-3904). It stays in each one it reached.
	Up []string `json:"up,omitempty"`
	// Report marks a report of the answer Trace by its asker (SB-3606).
	Report *AnswerReport `json:"report,omitempty"`
	// Thread marks a chat thread saved to the stewards (SB-3703).
	Thread *SavedThread `json:"thread,omitempty"`
	// Draft marks a draft a steward started from a found-nothing term (SB-3705).
	Draft *NoteDraft `json:"draft,omitempty"`
}

// StewardStore keeps questions to stewards, under retention, by asker and by space.
type StewardStore interface {
	PutStewardQuestion(ctx context.Context, q StewardQuestion) error
	GetStewardQuestion(ctx context.Context, tenant, id string) (StewardQuestion, bool, error)
	StewardQuestionsOf(ctx context.Context, tenant, asker string) ([]StewardQuestion, error)
	StewardQueue(ctx context.Context, tenant, space string) ([]StewardQuestion, error)
	UpdateStewardQuestion(ctx context.Context, tenant, id string, f func(*StewardQuestion) error) (StewardQuestion, error)
	StewardQuestionsBetween(ctx context.Context, tenant string, from, to time.Time) ([]StewardQuestion, error)
}

// NoteBus is the tool bus as notes use it.
type NoteBus interface {
	Invoke(ctx context.Context, s tenancy.Scope, c toolbus.Call) (approvals.Action, error)
}

// SpaceFacts are who answers a space's questions (its stewards, or questions_to for
// company) and its label; ok is false when nobody does.
type SpaceFacts func(space string) (stewards []string, label string, ok bool)

// Stewards is the questions-to-stewards service for one cell.
type Stewards struct {
	Agent  Agent
	Store  StewardStore
	Audit  Auditor
	Policy policy.Kernel
	Facts  SpaceFacts
	NewID  func(prefix string) string
	Now    func() time.Time
	// Notes (SB-1705): the bus with note-publish, and the emails of a space's members now
	// (every active user for company). A nil Bus is a cell without notes.
	Bus     NoteBus
	Readers func(ctx context.Context, space string) ([]string, error)
	// Gone says whether an owner's email is no active user in the directory now (SB-2201).
	// nil is a cell that can't tell: its review list names no document for its owner.
	Gone func(ctx context.Context, email string) (bool, error)
	// Evals keeps the tenant's eval cases, and Feedback finds the report one is made from
	// (SB-2204); nil keeps none.
	Evals    EvalStore
	Feedback FeedbackReader
	// Marks keeps review marks, Claimant says whether a claim's holder is a steward now, and
	// Replies keeps reply templates (SB-2501 to SB-2503); nil keeps or shows none.
	Marks    MarkStore
	Claimant Claimant
	Replies  TemplateStore
	// Suggested keeps each space's suggested questions (SB-3307); nil keeps none.
	Suggested SuggestedStore
	// StartItems is a space's start_here list by name, and Members the emails of its active
	// members now (SB-2907); nil checks no start_here items.
	StartItems func(name string) []string
	Members    func(ctx context.Context, space string) ([]string, error)
	// Unstewarded says whether a space (spc_NAME) has no active steward now (none), and
	// whether the directory could tell (known) (SB-3102); nil never reroutes.
	Unstewarded func(ctx context.Context, space string) (none, known bool, err error)
	// Caretakers keeps documents' caretakers (SB-3405); nil keeps none.
	Caretakers CaretakerStore
	// Zone is the working week's time zone (SB-3404); nil is UTC.
	Zone *time.Location
	// Children are a department's team spaces, by name (SB-3905); nil is a flat cell.
	Children func(name string) []string
	// ReviewAfter is a space's review_after_days, for its row in a roll-up; nil uses the
	// department's.
	ReviewAfter func(name string) int
	// Parent is a space's department by name, "" for none, and StewardPeople the active
	// people in a space's steward groups now (SB-3906); nil names none.
	Parent        func(name string) string
	StewardPeople func(ctx context.Context, space string) ([]answer.Person, error)
	// Owner is the one active user with an email, ok false for none or several (SB-3706);
	// nil asks no owner.
	Owner func(ctx context.Context, email string) (id string, ok bool, err error)
}

// ErrNotSteward is a caller who isn't a steward of the space (EPERM).
var ErrNotSteward = errors.New("ask: EPERM not a steward of this space")

// ReplyView is a reply as the asker sees it: cites checked live for them, the rest counted.
type ReplyView struct {
	Text    string    `json:"text"`
	At      time.Time `json:"at"`
	Cites   []Result  `json:"cites"`
	Dropped int       `json:"dropped"`
	Note    bool      `json:"note"`
}

// AskerItem is one of the asker's own questions.
type AskerItem struct {
	ID       string     `json:"id"`
	Question string     `json:"question"`
	Space    string     `json:"space"`
	At       time.Time  `json:"at"`
	State    string     `json:"state"` // open or answered
	Reply    *ReplyView `json:"reply,omitempty"`
}

// StewardItem is a question as stewards see it: never who asked.
type StewardItem struct {
	ID       string        `json:"id"`
	Question string        `json:"question"`
	At       time.Time     `json:"at"`
	State    string        `json:"state"`
	Reply    *StewardReply `json:"reply,omitempty"`
	Claim    *ClaimView    `json:"claim,omitempty"`
	// FromSpace is the space with no active steward it came from (SB-3102), or the space
	// that left it open Waited working days (SB-3404).
	FromSpace string `json:"from_space,omitempty"`
	Waited    int    `json:"waited,omitempty"`
	// Like is the question of the queue it looks like (SB-3402), and Overdue an open
	// question OverdueAfter working days old (SB-3404).
	Like    string `json:"like,omitempty"`
	Overdue bool   `json:"overdue,omitempty"`
	// Report is a reported answer as the steward may see it (SB-3606): it names its asker.
	Report *ReportView `json:"report,omitempty"`
	// Thread is a saved chat thread (SB-3703): it names who saved it.
	Thread *ThreadView `json:"thread,omitempty"`
	Draft  *NoteDraft  `json:"draft,omitempty"`
}

// SpaceKey is a space name as a record's space: spc_NAME, or company.
func SpaceKey(name string) string {
	if name == "company" || IsPerson(name) {
		return name
	}
	return "spc_" + name
}

func hashOf(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func auditSpace(space string) string {
	if space == "company" || IsPerson(space) {
		return ""
	}
	return space
}

// Send sends the asker's question to the people who answer the space it was asked in.
// trace must be the asker's own answer there, one nothing answered.
func (x Stewards) Send(ctx context.Context, s tenancy.Scope, text, trace string) (StewardQuestion, error) {
	q, err := question(s, x.Agent.Tenant, text)
	if err != nil {
		return StewardQuestion{}, err
	}
	t, err := x.Agent.Trace(ctx, s, trace)
	if err != nil || t.User != s.User() || t.Space != s.Space() || t.Response.Agent == AgentSearch || t.Response.Agent == AgentLimited ||
		t.Response.Notice == nil || !Askable[*t.Response.Notice] {
		return StewardQuestion{}, fmt.Errorf("%w: send a question from your own answer that nothing answered, in the same space", ErrInvalid)
	}
	if _, _, ok := x.Facts(s.Space()); !ok {
		return StewardQuestion{}, fmt.Errorf("%w: nobody answers questions sent from here", ErrInvalid)
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: q, Space: s.Space(), Trace: trace, At: x.Now().UTC()}
	if sq.Space != "company" && x.Unstewarded != nil {
		// Only a definite "nobody" reroutes: an unknown directory keeps it in the space,
		// which never widens who reads it on a guess (LAW-03).
		if none, known, err := x.Unstewarded(ctx, sq.Space); err == nil && known && none {
			if _, _, ok := x.Facts("company"); ok {
				sq.FromSpace, sq.Space = strings.TrimPrefix(sq.Space, "spc_"), "company"
			}
		}
	}
	sq.Like = x.repeat(ctx, sq)
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: sq.ID, State: "asked",
		PayloadSHA256: hashOf(q), Reason: "question to stewards", Space: auditSpace(sq.Space)}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Store.PutStewardQuestion(ctx, sq); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq, nil
}

// Mine lists the asker's own questions, with each reply's cites checked live for them.
func (x Stewards) Mine(ctx context.Context, s tenancy.Scope) ([]AskerItem, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return nil, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	qs, err := x.Store.StewardQuestionsOf(ctx, x.Agent.Tenant, s.User())
	if err != nil {
		return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	out := []AskerItem{}
	for _, q := range qs {
		it := AskerItem{ID: q.ID, Question: q.Question, Space: q.Space, At: q.At, State: "open"}
		if q.Reply != nil {
			it.State = "answered"
			cites, dropped, err := x.Agent.CheckCites(ctx, s, q.Reply.Cites)
			if err != nil {
				return nil, err
			}
			it.Reply = &ReplyView{Text: q.Reply.Text, At: q.Reply.At, Cites: cites, Dropped: dropped, Note: q.Reply.Note != ""}
		}
		out = append(out, it)
	}
	return out, nil
}

// steward decides, now, whether s answers questions for space name (or company).
func (x Stewards) steward(ctx context.Context, s tenancy.Scope, name string) ([]string, string, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return nil, "", ErrNotSteward
	}
	key := SpaceKey(name)
	if IsPerson(key) { // one person's own queue (SB-3706): theirs alone
		if key != PersonKey(s.User()) {
			return nil, "", ErrNotSteward
		}
		return []string{"user:" + s.User()}, "", nil
	}
	stewards, label, ok := x.Facts(key)
	if !ok {
		return nil, "", ErrNotSteward
	}
	var groups []string
	for _, g := range s.Groups() {
		groups = append(groups, "group:"+g)
	}
	dec := x.Policy.Authorise(ctx, s, "answer_questions", policy.Entity{Type: "SpaceConfig", ID: name, Tenant: x.Agent.Tenant,
		Attrs: map[string]any{"stewards": stewards}}, map[string]any{"groups": groups})
	if dec.Decision != policy.Allow {
		return nil, "", ErrNotSteward
	}
	return stewards, label, nil
}

// Queue is the space's questions for its stewards: open ones first, then newest.
func (x Stewards) Queue(ctx context.Context, s tenancy.Scope, name string) ([]StewardItem, error) {
	stewards, _, err := x.steward(ctx, s, name)
	if err != nil {
		return nil, err
	}
	qs, err := x.queue(ctx, name)
	if err != nil {
		return nil, err
	}
	out := []StewardItem{}
	now := x.Now().UTC()
	for _, q := range qs {
		if q.Thread != nil && q.Thread.Discarded {
			continue
		}
		it := StewardItem{ID: q.ID, Question: q.Question, At: q.At, State: "open", Reply: q.Reply, FromSpace: q.FromSpace, Like: partner(q, qs)}
		if q.Thread != nil {
			it.Thread = &ThreadView{By: q.Asker, Link: q.Thread.Link, Text: q.Thread.Text, Agent: q.Thread.Agent, Kind: q.Thread.Kind}
			if q.Thread.Agent != "" && q.Asker != s.User() {
				it.Thread.Text, it.Thread.Hidden = "", true
			}
		}
		it.Draft = q.Draft
		if waitedFor(q, SpaceKey(name)) {
			it.FromSpace, it.Waited = strings.TrimPrefix(q.Space, "spc_"), q.Waited
		}
		if it.Report, err = x.reportView(ctx, s, q); err != nil {
			return nil, err
		}
		if q.Reply != nil {
			it.State = "answered"
		} else if it.Claim, err = x.claimView(ctx, s, q.Claim, stewards, now); err != nil {
			return nil, err
		} else {
			it.Overdue = WorkingDays(q.At, now, x.Zone) >= OverdueAfter
		}
		out = append(out, it)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].State == "open") != (out[j].State == "open") {
			return out[i].State == "open"
		}
		return out[i].At.After(out[j].At)
	})
	return out, nil
}

// queue is the space's questions, newest first: on the company-wide queue, with those
// that waited in their space (SB-3404).
func (x Stewards) queue(ctx context.Context, name string) ([]StewardQuestion, error) {
	qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	if IsPerson(SpaceKey(name)) {
		return qs, nil
	}
	all, err := x.Store.StewardQuestionsBetween(ctx, x.Agent.Tenant, time.Time{}, x.Now().UTC().Add(time.Minute))
	if err != nil {
		return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	for _, q := range all {
		if q.Space != SpaceKey(name) && waitedFor(q, SpaceKey(name)) {
			qs = append(qs, q)
		}
	}
	sort.SliceStable(qs, func(i, j int) bool { return qs[i].At.After(qs[j].At) })
	return qs, nil
}

func (x Stewards) question(ctx context.Context, name, id string) (StewardQuestion, error) {
	q, found, err := x.Store.GetStewardQuestion(ctx, x.Agent.Tenant, id)
	if err != nil {
		return q, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	if !found || (q.Space != SpaceKey(name) && !waitedFor(q, SpaceKey(name))) {
		return q, ErrNotSteward // another space's question looks like none
	}
	return q, nil
}

// Reply records a steward's answer. Every cite must be one the steward can open now.
func (x Stewards) Reply(ctx context.Context, s tenancy.Scope, name, id, text string, cites []string) (StewardQuestion, error) {
	return x.ReplyBoth(ctx, s, name, id, text, cites, false)
}

// ReplyBoth is Reply; with both, the same reply also goes to the open question id looks
// like (SB-3402), each its own reply and audit row.
func (x Stewards) ReplyBoth(ctx context.Context, s tenancy.Scope, name, id, text string, cites []string, both bool) (StewardQuestion, error) {
	q, err := x.reply(ctx, s, name, id, text, cites)
	if err != nil || !both {
		return q, err
	}
	qs, err := x.queue(ctx, name)
	if err != nil {
		return q, err
	}
	if other := partner(q, qs); other != "" {
		if _, err := x.reply(ctx, s, name, other, text, cites); err != nil && !errors.Is(err, ErrInvalid) {
			return q, err // already answered is fine: Answer both answers what's still open
		}
	}
	return q, nil
}

func (x Stewards) reply(ctx context.Context, s tenancy.Scope, name, id, text string, cites []string) (StewardQuestion, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return StewardQuestion{}, err
	}
	if _, err := x.question(ctx, name, id); err != nil {
		return StewardQuestion{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > MaxReply {
		return StewardQuestion{}, fmt.Errorf("%w: a reply is 1 to %d characters", ErrInvalid, MaxReply)
	}
	uniq := map[string]bool{}
	for _, c := range cites {
		uniq[c] = true
	}
	if len(cites) > MaxReplyCites || len(uniq) != len(cites) {
		return StewardQuestion{}, fmt.Errorf("%w: at most %d cites, each once", ErrInvalid, MaxReplyCites)
	}
	if len(cites) > 0 {
		ok, dropped, err := x.Agent.CheckCites(ctx, s, cites)
		if err != nil {
			return StewardQuestion{}, err
		}
		if dropped > 0 || len(ok) != len(cites) {
			return StewardQuestion{}, fmt.Errorf("%w: cite only documents you can open now", ErrInvalid)
		}
	}
	if cites == nil {
		cites = []string{}
	}
	r := &StewardReply{By: s.User(), Text: text, Cites: cites, At: x.Now().UTC()}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: id, State: "answered",
		PayloadSHA256: hashOf(text), Reason: "steward reply", Space: auditSpace(SpaceKey(name))}); err != nil {
		return StewardQuestion{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(q *StewardQuestion) error {
		if q.Reply != nil {
			return fmt.Errorf("%w: this question already has a reply", ErrInvalid)
		}
		q.Reply = r
		q.Claim = nil
		return nil
	})
}

// CheckCites shows a person the cites they can open now, as a search would, and counts the rest.
func (a Agent) CheckCites(ctx context.Context, s tenancy.Scope, cites []string) ([]Result, int, error) {
	if len(cites) == 0 {
		return []Result{}, 0, nil
	}
	lk, ok := a.Index.(ports.CiteLookup)
	if !ok {
		return nil, 0, fmt.Errorf("%w: the index can't look up cites", ErrUnavailable)
	}
	chunks, err := lk.ChunksOf(ctx, a.Tenant, cites)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return nil, 0, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, chunks, len(chunks))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	byCite := map[string]ports.Chunk{}
	for _, c := range res.Context {
		if _, seen := byCite[c.Cite]; !seen {
			byCite[c.Cite] = c
		}
	}
	own := a.owners()
	out := []Result{}
	seen := map[string]bool{}
	for _, c := range cites {
		ch, ok := byCite[c]
		if !ok || seen[c] {
			continue
		}
		seen[c] = true
		r := a.result(ctx, own, ch)
		r.Snippet = ""
		out = append(out, r)
	}
	return out, len(seen0(cites)) - len(out), nil
}

func seen0(cites []string) map[string]bool {
	m := map[string]bool{}
	for _, c := range cites {
		m[c] = true
	}
	return m
}

// who is the scope's user as connectors know them: by the directory's primary email.
func (a Agent) who(ctx context.Context, s tenancy.Scope) (ports.UserRef, error) {
	who := ports.UserRef{ID: s.User()}
	if a.Emails != nil {
		e, err := a.Emails(ctx, s.User())
		if err != nil {
			return who, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		if len(e) > 0 {
			who.Email = e[0]
		}
	}
	return who, nil
}

// NoteRequest is a steward's note: a title they write, and readers, if fewer than the default.
type NoteRequest struct {
	Title   string   `json:"title"`
	Readers []string `json:"readers,omitempty"`
	// KeepQuestion puts the question's text in the note (SB-3401); a republish keeps what
	// the first publish chose.
	KeepQuestion bool `json:"keep_question,omitempty"`
	// Confirm is the count of narrower cites the steward saw and accepts; nil previews.
	Confirm *int `json:"confirm,omitempty"`
}

// NotePreview is what the steward confirms before a note is published.
type NotePreview struct {
	Path     string `json:"path"`
	Readers  int    `json:"readers"`
	Narrower int    `json:"narrower"` // cites some of the note's readers can't open
	// Similar are documents of the space that read the same, checked as the steward (SB-3707).
	Similar []Result `json:"similar,omitempty"`
}

// NoteResult is a published note.
type NoteResult struct {
	NotePreview
	Action  string `json:"action"`
	State   string `json:"state"`
	Receipt string `json:"receipt,omitempty"`
}

// NoteSchema is note-publish's payload schema, pinned by the release bundle (sb-release(7)).
const NoteSchema = `{"type":"object","additionalProperties":false,"required":["markdown","readers","owner"],"properties":{"markdown":{"type":"string","minLength":1,"maxLength":65536},"readers":{"type":"array","minItems":1,"maxItems":500,"items":{"type":"string"}},"owner":{"type":"string"},"label":{"type":"string"}}}`

// NotePayload is note-publish's payload on the folder connector (sb-connector(3), Notes).
type NotePayload struct {
	Markdown string   `json:"markdown"`
	Readers  []string `json:"readers"`
	Owner    string   `json:"owner"`
	Label    string   `json:"label,omitempty"`
}

// Note previews, or publishes, a reply as a note (SB-1705). It needs a reply, readers no
// wider than the space's members, and the steward's confirmation of the narrower count.
func (x Stewards) Note(ctx context.Context, s tenancy.Scope, name, id string, req NoteRequest) (NoteResult, error) {
	if IsPerson(SpaceKey(name)) {
		return NoteResult{}, fmt.Errorf("%w: a reply to you can't be published as a note", ErrInvalid)
	}
	if x.Bus == nil || x.Readers == nil {
		return NoteResult{}, fmt.Errorf("%w: this cell doesn't publish notes (sb-tenant(5), notes)", ErrInvalid)
	}
	stewards, label, err := x.steward(ctx, s, name)
	if err != nil {
		return NoteResult{}, err
	}
	q, err := x.question(ctx, name, id)
	if err != nil {
		return NoteResult{}, err
	}
	if q.Reply == nil {
		return NoteResult{}, fmt.Errorf("%w: reply first; a note is a reply", ErrInvalid)
	}
	if waitedFor(q, SpaceKey(name)) {
		return NoteResult{}, fmt.Errorf("%w: publish this note from its own space, %s", ErrInvalid, strings.TrimPrefix(q.Space, "spc_"))
	}
	// A reply that is a note already is republished: the same text to fewer readers, the
	// note's readers now that are still members, or fewer (SB-2004). Never wider.
	republish := q.Reply.Note != ""
	title := strings.TrimSpace(req.Title)
	if republish && title == "" {
		title = q.Reply.NoteTitle
	}
	if title == "" || utf8.RuneCountInString(title) > MaxNoteTitle || strings.ContainsAny(title, "\n\r") {
		return NoteResult{}, fmt.Errorf("%w: a note needs a one-line title of 1 to %d characters", ErrInvalid, MaxNoteTitle)
	}
	all, err := x.Readers(ctx, q.Space)
	if err != nil {
		return NoteResult{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	path := "notes/" + name + "/" + id + ".md"
	var was ports.Chunk
	if republish {
		if was, err = x.noteNow(ctx, path); err != nil {
			return NoteResult{}, err
		}
		all = still(all, was)
	}
	readers, err := narrowed(all, req.Readers)
	if err != nil {
		return NoteResult{}, err
	}
	p := NotePreview{Path: path, Readers: len(readers)}
	if p.Narrower, err = x.narrower(ctx, q.Reply.Cites, readers); err != nil {
		return NoteResult{}, err
	}
	if req.Confirm == nil {
		p.Similar = x.similar(ctx, s, name, q.Reply.Text) // SB-3707: a warning only
		return NoteResult{NotePreview: p}, nil
	}
	if *req.Confirm != p.Narrower {
		return NoteResult{}, fmt.Errorf("%w: %d of the cites are narrower than the note now; preview again and confirm that", ErrInvalid, p.Narrower)
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil || who.Email == "" {
		return NoteResult{}, fmt.Errorf("%w: the steward has no email to own the note", ErrUnavailable)
	}
	owner := strings.ToLower(who.Email)
	if republish && was.Owner != "" { // the same note: its owner stays, whoever republishes it
		owner = strings.ToLower(was.Owner)
	}
	keep := req.KeepQuestion
	if republish {
		keep = q.Reply.NoteQuestion
	}
	asked := ""
	if keep {
		asked = q.Question
	}
	payload, _ := json.Marshal(NotePayload{Markdown: noteText(title, q.Reply, asked, x.Now().UTC().Add(NoteReviewIn)), Readers: readers, Owner: owner, Label: label})
	var groups []string
	for _, g := range s.Groups() {
		groups = append(groups, "group:"+g)
	}
	a, err := x.Bus.Invoke(ctx, s, toolbus.Call{Agent: "steward", Tool: NoteTool, Target: p.Path, Payload: payload,
		Context: map[string]any{"stewards": stewards, "groups": groups}})
	if err != nil {
		return NoteResult{}, err
	}
	out := NoteResult{NotePreview: p, Action: a.ID, State: string(a.State), Receipt: a.Receipt}
	if a.State == approvals.Ran && a.Receipt != "" {
		if _, err := x.Store.UpdateStewardQuestion(ctx, x.Agent.Tenant, id, func(q *StewardQuestion) error {
			if q.Reply.Note == "" {
				q.Reply.NoteAt = x.Now().UTC()
			}
			q.Reply.Note, q.Reply.NoteTitle, q.Reply.NoteQuestion = a.Receipt, title, keep
			return nil
		}); err != nil {
			return out, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
	}
	return out, nil
}

// noteKind is the connector kind notes are written to and indexed from.
const noteKind = "docs"

// noteNow is a published note as the index holds it now: its readers and owner.
func (x Stewards) noteNow(ctx context.Context, path string) (ports.Chunk, error) {
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return ports.Chunk{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: noteKind, ID: path}}})
	if err != nil {
		return ports.Chunk{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	for _, c := range firsts(latest(cs)) {
		return c, nil
	}
	return ports.Chunk{}, fmt.Errorf("%w: the note isn't indexed yet; republish it after the next sync", ErrUnavailable)
}

// still is the members who can read the note now: never anyone it wasn't published to.
func still(members []string, note ports.Chunk) []string {
	had := map[string]bool{}
	for _, r := range note.ACLPrefilter {
		had[strings.ToLower(r)] = true
	}
	var out []string
	for _, m := range members {
		if had[strings.ToLower(m)] {
			out = append(out, m)
		}
	}
	return out
}

// wider is how many of a note's readers aren't members of its space now (SB-2004): its
// owner aside, who is the steward that published it.
func wider(note ports.Chunk, members map[string]bool) int {
	n := 0
	for _, r := range note.ACLPrefilter {
		if r = strings.ToLower(r); r != strings.ToLower(note.Owner) && !members[r] {
			n++
		}
	}
	return n
}

// narrowed is the note's readers: the default, or a subset of it. Never wider.
func narrowed(all, asked []string) ([]string, error) {
	have := map[string]bool{}
	for _, e := range all {
		have[strings.ToLower(e)] = true
	}
	var out []string
	if len(asked) == 0 {
		for e := range have {
			out = append(out, e)
		}
	} else {
		seen := map[string]bool{}
		for _, e := range asked {
			e = strings.ToLower(strings.TrimSpace(e))
			if !have[e] {
				return nil, fmt.Errorf("%w: a note's readers can only be fewer than the space's members", ErrInvalid)
			}
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	sort.Strings(out)
	if len(out) == 0 || len(out) > MaxNoteReaders {
		return nil, fmt.Errorf("%w: a note has 1 to %d readers", ErrInvalid, MaxNoteReaders)
	}
	return out, nil
}

// narrower counts the reply's cites that some of the note's readers can't open at the
// source now: a note is new text, and the source's ACL can't follow it.
func (x Stewards) narrower(ctx context.Context, cites, readers []string) (int, error) {
	if len(cites) == 0 {
		return 0, nil
	}
	lk, ok := x.Agent.Index.(ports.CiteLookup)
	if !ok {
		return 0, fmt.Errorf("%w: the index can't look up cites", ErrUnavailable)
	}
	chunks, err := lk.ChunksOf(ctx, x.Agent.Tenant, cites)
	if err != nil {
		return 0, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	narrow := map[string]bool{}
	for _, c := range cites {
		found := false
		for _, ch := range chunks {
			found = found || ch.Cite == c
		}
		if !found {
			narrow[c] = true // gone from the index: nobody can be shown it
		}
	}
	for _, r := range readers {
		vs := x.Agent.Auth.Check(ctx, ports.UserRef{ID: r, Email: r}, chunks)
		for _, ch := range chunks {
			if vs[ch.Kind+":"+ch.SourceID] != ports.VerdictAllow {
				narrow[ch.Cite] = true
			}
		}
	}
	return len(narrow), nil
}

// NoteReviewIn is how far ahead a note's review_by is when it is published (SB-3403).
const NoteReviewIn = 180 * 24 * time.Hour

// noteText is the note's Markdown: front matter with its review_by date, the steward's
// title, the question when they kept it, the reply, and the cites they gave. Never the asker.
func noteText(title string, r *StewardReply, question string, due time.Time) string {
	var b strings.Builder
	b.WriteString("---\nreview_by: " + due.UTC().Format("2006-01-02") + "\n---\n")
	b.WriteString("# " + title + "\n\n")
	if question != "" {
		b.WriteString("Question: " + question + "\n\n")
	}
	b.WriteString(r.Text + "\n")
	if len(r.Cites) > 0 {
		b.WriteString("\nSources:\n\n")
		for _, c := range r.Cites {
			b.WriteString("- " + c + "\n")
		}
	}
	return b.String()
}
