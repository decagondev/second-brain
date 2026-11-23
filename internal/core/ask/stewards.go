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
}

// SpaceKey is a space name as a record's space: spc_NAME, or company.
func SpaceKey(name string) string {
	if name == "company" {
		return "company"
	}
	return "spc_" + name
}

func hashOf(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func auditSpace(space string) string {
	if space == "company" {
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
	if err != nil || t.User != s.User() || t.Space != s.Space() || t.Response.Agent == AgentSearch ||
		t.Response.Notice == nil || !Askable[*t.Response.Notice] {
		return StewardQuestion{}, fmt.Errorf("%w: send a question from your own answer that nothing answered, in the same space", ErrInvalid)
	}
	if _, _, ok := x.Facts(s.Space()); !ok {
		return StewardQuestion{}, fmt.Errorf("%w: nobody answers questions sent from here", ErrInvalid)
	}
	sq := StewardQuestion{ID: x.NewID("qs_"), Tenant: x.Agent.Tenant, Asker: s.User(), Question: q, Space: s.Space(), Trace: trace, At: x.Now().UTC()}
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
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	qs, err := x.Store.StewardQueue(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	out := []StewardItem{}
	for _, q := range qs {
		it := StewardItem{ID: q.ID, Question: q.Question, At: q.At, State: "open", Reply: q.Reply}
		if q.Reply != nil {
			it.State = "answered"
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

func (x Stewards) question(ctx context.Context, name, id string) (StewardQuestion, error) {
	q, found, err := x.Store.GetStewardQuestion(ctx, x.Agent.Tenant, id)
	if err != nil {
		return q, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	if !found || q.Space != SpaceKey(name) {
		return q, ErrNotSteward // another space's question looks like none
	}
	return q, nil
}

// Reply records a steward's answer. Every cite must be one the steward can open now.
func (x Stewards) Reply(ctx context.Context, s tenancy.Scope, name, id, text string, cites []string) (StewardQuestion, error) {
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
	// Confirm is the count of narrower cites the steward saw and accepts; nil previews.
	Confirm *int `json:"confirm,omitempty"`
}

// NotePreview is what the steward confirms before a note is published.
type NotePreview struct {
	Path     string `json:"path"`
	Readers  int    `json:"readers"`
	Narrower int    `json:"narrower"` // cites some of the note's readers can't open
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
	payload, _ := json.Marshal(NotePayload{Markdown: noteText(title, q.Reply), Readers: readers, Owner: owner, Label: label})
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
			q.Reply.Note, q.Reply.NoteTitle = a.Receipt, title
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

// noteText is the note's Markdown: the steward's title and reply, and the cites they gave.
func noteText(title string, r *StewardReply) string {
	var b strings.Builder
	b.WriteString("# " + title + "\n\n" + r.Text + "\n")
	if len(r.Cites) > 0 {
		b.WriteString("\nSources:\n\n")
		for _, c := range r.Cites {
			b.WriteString("- " + c + "\n")
		}
	}
	return b.String()
}
