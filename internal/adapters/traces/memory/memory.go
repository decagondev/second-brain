// Package memory is an in-memory ask.Store for development and tests: traces and
// conversations for one cell. A Postgres store replaces it with the cell runtime.
package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/retention"
)

// Store is safe for concurrent use.
type Store struct {
	mu       sync.Mutex
	traces   map[string]ask.Trace
	convs    map[string]conv
	feedback map[string]ask.Feedback // tenant/trace
	saved    map[string]ask.SavedQuestion
	shares   map[string]ask.Share
	stq      map[string]ask.StewardQuestion
	prefs    map[string]prefRec // tenant/user
	evals    map[string]ask.EvalCase
	access   map[string]accessRec         // tenant/key
	marks    map[string]ask.ReviewMark    // tenant/space/doc
	retired  map[string]ask.Retirement    // tenant/space/doc
	tmpls    map[string]ask.ReplyTemplate // tenant/space/id
	notices  map[string]ask.CellNotice    // tenant
	suggest  map[string]ask.Suggested     // tenant/space
	pins     map[string]ask.Pins          // tenant/space
	cares    map[string]ask.Caretaker     // tenant/doc
	policies map[string]ask.SpacePolicy   // tenant/space/doc (SB-4103)
	reads    map[string]ask.PolicyRead    // tenant/space/doc/email/version
	research map[string]ask.Research      // tenant/id (SB-4201)
	actions  map[string]ask.Action        // tenant/id (SB-4203)
	memory   map[string]ask.Memory        // tenant/user (SB-4303)
	colls    map[string]ask.Collection    // tenant/id (SB-4304)
	shadows  map[string]ask.ShadowRun     // tenant/id (SB-4704)
	windows  map[string]ask.ShadowWindow  // tenant/agent|release
	rhythms  map[string]ask.RhythmRun     // tenant/id (SB-4801)
	rosters  map[string]ask.RhythmRoster  // tenant/space (SB-4802)
	projects map[string]ask.Project       // tenant/id (SB-4803)
	wide     map[string]ask.WideConfirm   // tenant/doc (SB-5402)
	sharing  []ask.SharingEvent           // SB-5403
	// Now stamps conversations for retention; nil is time.Now.
	Now func() time.Time
}

type conv struct {
	tenant, user string
	questions    []string
	at           time.Time // the last question
	space        string    // the space it is asked in; "mixed" if more than one
}

// New returns an empty store.
func New() *Store {
	return &Store{traces: map[string]ask.Trace{}, convs: map[string]conv{}, feedback: map[string]ask.Feedback{},
		saved: map[string]ask.SavedQuestion{}, shares: map[string]ask.Share{}, stq: map[string]ask.StewardQuestion{}, prefs: map[string]prefRec{},
		evals: map[string]ask.EvalCase{}, access: map[string]accessRec{}, marks: map[string]ask.ReviewMark{}, retired: map[string]ask.Retirement{}, tmpls: map[string]ask.ReplyTemplate{}, notices: map[string]ask.CellNotice{}, suggest: map[string]ask.Suggested{}, pins: map[string]ask.Pins{}, cares: map[string]ask.Caretaker{}, policies: map[string]ask.SpacePolicy{}, reads: map[string]ask.PolicyRead{}, research: map[string]ask.Research{}, actions: map[string]ask.Action{}, memory: map[string]ask.Memory{}, colls: map[string]ask.Collection{}, shadows: map[string]ask.ShadowRun{}, windows: map[string]ask.ShadowWindow{},
		rhythms: map[string]ask.RhythmRun{}, rosters: map[string]ask.RhythmRoster{}, projects: map[string]ask.Project{}}
}

// PutTrace implements ask.Store.
func (s *Store) PutTrace(_ context.Context, t ask.Trace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.traces[t.ID]; dup {
		return errors.New("traces: duplicate trace id")
	}
	s.traces[t.ID] = t
	return nil
}

// GetTrace implements ask.Store.
func (s *Store) GetTrace(_ context.Context, tenant, id string) (ask.Trace, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.traces[id]
	if !ok || t.Tenant != tenant {
		return ask.Trace{}, false, nil
	}
	return t, true, nil
}

// Questions implements ask.Store.
func (s *Store) Questions(_ context.Context, tenant, user, id string) ([]string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.convs[id]
	if !ok || c.tenant != tenant || c.user != user {
		return nil, false, nil
	}
	return append([]string(nil), c.questions...), true, nil
}

// TagSpace implements ask.SpaceTagger.
func (s *Store) TagSpace(_ context.Context, tenant, id, space string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.convs[id]
	if !ok || c.tenant != tenant {
		return errors.New("traces: no such conversation")
	}
	switch {
	case c.space == "":
		c.space = space
	case c.space != space:
		c.space = "mixed" // asked in two spaces: a hold on either keeps it
	}
	s.convs[id] = c
	return nil
}

// AppendQuestion implements ask.Store.
func (s *Store) AppendQuestion(_ context.Context, tenant, user, id, q string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.convs[id]
	if ok && (c.tenant != tenant || c.user != user) {
		return errors.New("traces: conversation belongs to someone else")
	}
	c.tenant, c.user = tenant, user
	c.questions = append(c.questions, q)
	c.at = time.Now()
	if s.Now != nil {
		c.at = s.Now()
	}
	s.convs[id] = c
	return nil
}

// PutFeedback implements ask.FeedbackStore; a later feedback on a trace replaces it.
func (s *Store) PutFeedback(_ context.Context, f ask.Feedback) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.feedback[f.Tenant+"/"+f.Trace] = f
	return nil
}

// GetFeedback implements ask.FeedbackReader.
func (s *Store) GetFeedback(_ context.Context, tenant, trace string) (ask.Feedback, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.feedback[tenant+"/"+trace]
	return f, ok, nil
}

// FeedbackBetween implements ask.FeedbackStore: the tenant's feedback in [from, to), oldest first.
func (s *Store) FeedbackBetween(_ context.Context, tenant string, from, to time.Time) ([]ask.Feedback, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return window(s.feedback, func(f ask.Feedback) (string, time.Time, string) { return f.Tenant, f.At, f.Trace }, tenant, from, to), nil
}

// window is a map's values of tenant in [from, to), by time and then key.
func window[T any](m map[string]T, of func(T) (tenant string, at time.Time, key string), tenant string, from, to time.Time) []T {
	var out []T
	for _, v := range m {
		if t, at, _ := of(v); t == tenant && !at.Before(from) && at.Before(to) {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		_, a, ka := of(out[i])
		_, b, kb := of(out[j])
		return a.Before(b) || (a.Equal(b) && ka < kb)
	})
	return out
}

// Retention returns the store's three retention stores: traces (answers and drafts),
// conversations and feedback (sb-tenant(5), Retention and deletion).
func (s *Store) Retention() []retention.Store {
	// Feedback first: it takes its space from its trace, which must still be there (SB-1401).
	return []retention.Store{view{s, "feedback"}, view{s, "conversations"}, view{s, "traces"}, view{s, "saved questions"}, view{s, "shares"}, view{s, "steward questions"}, view{s, PrefsName}, view{s, "eval cases"}, view{s, "access requests"}, view{s, "review marks"}, view{s, "reply templates"}, view{s, MemoryName}, view{s, CollectionsName}}
}

// MemoryName is the memory's store name (SB-4303): erased with a person and by their own
// Forget, never expired.
const MemoryName = "memory"

// CollectionsName is the collections' store name (SB-4304): erased with their owner, never
// expired.
const CollectionsName = "collections"

// PrefsName is the notification settings' store name: erased with a person, never expired.
const PrefsName = "notification settings"

type view struct {
	s    *Store
	name string
}

func (v view) Name() string { return v.name }

// Expire implements retention.Store: a trace carries its space; feedback takes its trace's;
// a conversation carries the space it was asked in, and one asked in several, or not yet
// tagged, is kept by any hold that could cover it.
func (v view) Expire(_ context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
	if v.name == PrefsName || v.name == MemoryName || v.name == CollectionsName {
		return 0, 0, nil
	}
	n, k := v.drop(aged(tenant, before), held, false, nil)
	return n, k, nil
}

// Plan implements retention.Store: Expire's selection, deleting nothing.
func (v view) Plan(_ context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
	if v.name == PrefsName || v.name == MemoryName || v.name == CollectionsName {
		return 0, 0, nil
	}
	n, k := v.drop(aged(tenant, before), held, true, nil)
	return n, k, nil
}

// Erase implements retention.Store.
func (v view) Erase(_ context.Context, tenant, user string) (int, error) {
	n, _ := v.drop(about(tenant, user), nil, false, nil)
	return n, nil
}

// Export implements retention.Store: Erase's selection, deleting nothing.
func (v view) Export(_ context.Context, tenant, user string) ([]json.RawMessage, error) {
	var out []json.RawMessage
	v.drop(about(tenant, user), nil, true, &out)
	return out, nil
}

type matcher func(tenant, user, space string, at time.Time) bool

func aged(tenant string, before time.Time) matcher {
	return func(t, _, _ string, at time.Time) bool { return t == tenant && at.Before(before) }
}

func about(tenant, user string) matcher {
	return func(t, u, _ string, _ time.Time) bool { return t == tenant && u == user }
}

// convJSON is a conversation as the Postgres store keeps it, for an export.
type convJSON struct {
	Tenant, User string
	Questions    []string
	At           time.Time
	Space        string
}

// drop selects with match, keeps what held holds, and deletes the rest unless dry; with
// got, every selected record is added to it as JSON. One selection serves all four of
// Expire, Plan, Erase and Export, so the dry runs count what the deletes delete.
func (v view) drop(match matcher, held retention.Held, dry bool, got *[]json.RawMessage) (int, int) {
	s := v.s
	s.mu.Lock()
	defer s.mu.Unlock()
	n, kept := 0, 0
	try := func(tenant, user, space string, at time.Time, rec any) bool {
		if !match(tenant, user, space, at) {
			return false
		}
		if held.Is(space, user) {
			kept++
			return false
		}
		n++
		if got != nil {
			b, _ := json.Marshal(rec)
			*got = append(*got, b)
		}
		return !dry
	}
	switch v.name {
	case "traces":
		for k, t := range s.traces {
			if try(t.Tenant, t.User, spaceOf(t.Space), t.At, t) {
				delete(s.traces, k)
			}
		}
	case "conversations":
		for k, c := range s.convs {
			if try(c.tenant, c.user, spaceOf(c.space), c.at, convJSON{c.tenant, c.user, c.questions, c.at, c.space}) {
				delete(s.convs, k)
			}
		}
	case "saved questions":
		for k, q := range s.saved {
			if try(q.Tenant, q.User, spaceOf(q.Space), q.At, q) {
				delete(s.saved, k)
			}
		}
	case "steward questions":
		for k, q := range s.stq {
			if try(q.Tenant, q.Asker, spaceOf(q.Space), q.At, q) {
				delete(s.stq, k)
			}
		}
	case "shares":
		for k, sh := range s.shares {
			if try(sh.Tenant, sh.User, spaceOf(sh.Space), sh.At, sh) {
				delete(s.shares, k)
			}
		}
	case "access requests":
		for k, a := range s.access {
			if try(a.Tenant, a.User, "", a.At, a) {
				delete(s.access, k)
			}
		}
	case "eval cases":
		for k, c := range s.evals {
			if try(c.Tenant, c.By, spaceOf(c.Space), c.At, c) {
				delete(s.evals, k)
			}
		}
	case "review marks":
		for k, m := range s.marks {
			if try(m.Tenant, m.By, spaceOf(m.Space), m.At, m) {
				delete(s.marks, k)
			}
		}
	case "reply templates":
		for k, t := range s.tmpls {
			if try(t.Tenant, t.By, spaceOf(t.Space), t.At, t) {
				delete(s.tmpls, k)
			}
		}
	case MemoryName, CollectionsName:
		dropOwn(s, v.name, try)
	case PrefsName:
		for k, p := range s.prefs {
			if try(p.Tenant, p.User, "", time.Time{}, p) {
				delete(s.prefs, k)
			}
		}
	case "feedback":
		for k, f := range s.feedback {
			space := ""
			if t, ok := s.traces[f.Trace]; ok && t.Tenant == f.Tenant {
				space = spaceOf(t.Space)
			}
			if try(f.Tenant, f.User, space, f.At, f) {
				delete(s.feedback, k)
			}
		}
	}
	return n, kept
}

// spaceOf is a record's space for holds: "" for one asked company-wide, which any hold on
// a space might still cover through its sources, so it is treated as unknown.
func spaceOf(s string) string {
	if s == "company" || s == "mixed" {
		return ""
	}
	return s
}

var _ ask.QuestionStore = (*Store)(nil)

// PutSaved implements ask.QuestionStore.
func (s *Store) PutSaved(_ context.Context, q ask.SavedQuestion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.saved[q.Tenant+"/"+q.ID]; dup {
		return errors.New("traces: duplicate saved question id")
	}
	s.saved[q.Tenant+"/"+q.ID] = q
	return nil
}

// SavedOf implements ask.QuestionStore: newest first.
func (s *Store) SavedOf(_ context.Context, tenant, user string) ([]ask.SavedQuestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ask.SavedQuestion
	for _, q := range s.saved {
		if q.Tenant == tenant && q.User == user {
			out = append(out, q)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.After(out[j].At) || (out[i].At.Equal(out[j].At) && out[i].ID > out[j].ID)
	})
	return out, nil
}

// DeleteSaved implements ask.QuestionStore.
func (s *Store) DeleteSaved(_ context.Context, tenant, user, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.saved[tenant+"/"+id]
	if !ok || q.User != user {
		return false, nil
	}
	delete(s.saved, tenant+"/"+id)
	return true, nil
}

// PutShare implements ask.QuestionStore.
func (s *Store) PutShare(_ context.Context, sh ask.Share) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.shares[sh.Tenant+"/"+sh.ID]; dup {
		return errors.New("traces: duplicate share id")
	}
	s.shares[sh.Tenant+"/"+sh.ID] = sh
	return nil
}

// GetShare implements ask.QuestionStore.
func (s *Store) GetShare(_ context.Context, tenant, id string) (ask.Share, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[tenant+"/"+id]
	return sh, ok, nil
}

var _ ask.StewardStore = (*Store)(nil)

// PutStewardQuestion implements ask.StewardStore.
func (s *Store) PutStewardQuestion(_ context.Context, q ask.StewardQuestion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.stq[q.Tenant+"/"+q.ID]; dup {
		return errors.New("traces: duplicate question id")
	}
	s.stq[q.Tenant+"/"+q.ID] = q
	return nil
}

// GetStewardQuestion implements ask.StewardStore.
func (s *Store) GetStewardQuestion(_ context.Context, tenant, id string) (ask.StewardQuestion, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.stq[tenant+"/"+id]
	return q, ok, nil
}

func (s *Store) stqWhere(tenant string, keep func(ask.StewardQuestion) bool) []ask.StewardQuestion {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ask.StewardQuestion
	for _, q := range s.stq {
		if q.Tenant == tenant && keep(q) {
			out = append(out, q)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.After(out[j].At) || (out[i].At.Equal(out[j].At) && out[i].ID > out[j].ID)
	})
	return out
}

// StewardQuestionsOf implements ask.StewardStore: the asker's, newest first.
func (s *Store) StewardQuestionsOf(_ context.Context, tenant, asker string) ([]ask.StewardQuestion, error) {
	return s.stqWhere(tenant, func(q ask.StewardQuestion) bool { return q.Asker == asker }), nil
}

// StewardQueue implements ask.StewardStore: one space's, newest first.
func (s *Store) StewardQueue(_ context.Context, tenant, space string) ([]ask.StewardQuestion, error) {
	return s.stqWhere(tenant, func(q ask.StewardQuestion) bool { return q.Space == space }), nil
}

// StewardQuestionsBetween implements ask.StewardStore: asked in [from, to).
func (s *Store) StewardQuestionsBetween(_ context.Context, tenant string, from, to time.Time) ([]ask.StewardQuestion, error) {
	return s.stqWhere(tenant, func(q ask.StewardQuestion) bool { return !q.At.Before(from) && q.At.Before(to) }), nil
}

// UpdateStewardQuestion implements ask.StewardStore: f's change is kept only if it returns nil.
func (s *Store) UpdateStewardQuestion(_ context.Context, tenant, id string, f func(*ask.StewardQuestion) error) (ask.StewardQuestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.stq[tenant+"/"+id]
	if !ok {
		return q, errors.New("traces: no such question")
	}
	if q.Reply != nil {
		r := *q.Reply
		q.Reply = &r
	}
	if err := f(&q); err != nil {
		return q, err
	}
	s.stq[tenant+"/"+id] = q
	return q, nil
}

// Between lists a tenant's traces asked in [from, to), oldest first (the weekly report).
func (s *Store) Between(_ context.Context, tenant string, from, to time.Time) ([]ask.Trace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return window(s.traces, func(t ask.Trace) (string, time.Time, string) { return t.Tenant, t.At, t.ID }, tenant, from, to), nil
}

// prefRec is a person's notification settings as an export shows them.
type prefRec struct {
	Tenant string `json:"tenant"`
	User   string `json:"user"`
	notify.Prefs
}

var _ notify.PrefStore = (*Store)(nil)

// NotifyPrefs implements notify.PrefStore.
func (s *Store) NotifyPrefs(_ context.Context, tenant, user string) (notify.Prefs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prefs[tenant+"/"+user].Prefs, nil
}

// PutNotifyPrefs implements notify.PrefStore.
func (s *Store) PutNotifyPrefs(_ context.Context, tenant, user string, p notify.Prefs) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefs[tenant+"/"+user] = prefRec{tenant, user, p}
	return nil
}

// accessRec is one access request sent (SB-2205): a key, who asked, and when.
type accessRec struct {
	Tenant string    `json:"tenant"`
	Key    string    `json:"key"`
	User   string    `json:"user"`
	At     time.Time `json:"at"`
}

var _ ask.AccessStore = (*Store)(nil)

// AccessRequested implements ask.AccessStore.
func (s *Store) AccessRequested(_ context.Context, tenant, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.access[tenant+"/"+key]
	return ok, nil
}

// PutAccessRequest implements ask.AccessStore.
func (s *Store) PutAccessRequest(_ context.Context, tenant, key, user string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.access[tenant+"/"+key] = accessRec{tenant, key, user, at}
	return nil
}

var _ ask.EvalStore = (*Store)(nil)

// PutEvalCase implements ask.EvalStore (SB-2204).
func (s *Store) PutEvalCase(_ context.Context, c ask.EvalCase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evals[c.Tenant+"/"+c.ID] = c
	return nil
}

// EvalCases implements ask.EvalStore: oldest first.
func (s *Store) EvalCases(_ context.Context, tenant string) ([]ask.EvalCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.EvalCase{}
	for _, c := range s.evals {
		if c.Tenant == tenant {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// DeleteEvalCase implements ask.EvalStore.
func (s *Store) DeleteEvalCase(_ context.Context, tenant, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.evals, tenant+"/"+id)
	return nil
}

var _ ask.WatchStore = (*Store)(nil)

// UpdateSaved implements ask.WatchStore.
func (s *Store) UpdateSaved(_ context.Context, tenant, user, id string, f func(*ask.SavedQuestion)) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.saved[tenant+"/"+id]
	if !ok || q.User != user {
		return false, nil
	}
	f(&q)
	q.Tenant, q.User, q.ID = tenant, user, id
	s.saved[tenant+"/"+id] = q
	return true, nil
}

// Routines implements ask.RoutineStore: oldest first.
func (s *Store) Routines(_ context.Context, tenant string) ([]ask.SavedQuestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ask.SavedQuestion
	for _, q := range s.saved {
		if q.Tenant == tenant && q.Routine != nil {
			out = append(out, q)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || out[i].At.Equal(out[j].At) && out[i].ID < out[j].ID
	})
	return out, nil
}

// MemoryOf implements ask.MemoryStore.
func (s *Store) MemoryOf(_ context.Context, tenant, user string) (ask.Memory, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.memory[tenant+"/"+user]
	m.Lines = append([]ask.MemoryLine(nil), m.Lines...)
	return m, ok, nil
}

// PutMemory implements ask.MemoryStore; a memory with no lines is removed.
func (s *Store) PutMemory(_ context.Context, m ask.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(m.Lines) == 0 && len(m.Excluded) == 0 { // SB-5503: exclusions live here too
		delete(s.memory, m.Tenant+"/"+m.User)
		return nil
	}
	m.Lines = append([]ask.MemoryLine(nil), m.Lines...)
	m.Excluded = append([]ask.Exclusion(nil), m.Excluded...)
	s.memory[m.Tenant+"/"+m.User] = m
	return nil
}

// Watched implements ask.WatchStore: oldest first.
func (s *Store) Watched(_ context.Context, tenant string) ([]ask.SavedQuestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ask.SavedQuestion
	for _, q := range s.saved {
		if q.Tenant == tenant && q.Watch {
			out = append(out, q)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || out[i].At.Equal(out[j].At) && out[i].ID < out[j].ID
	})
	return out, nil
}

var (
	_ ask.MarkStore     = (*Store)(nil)
	_ ask.TemplateStore = (*Store)(nil)
)

// PutReviewMark implements ask.MarkStore (SB-2501): one per space and document.
func (s *Store) PutReviewMark(_ context.Context, m ask.ReviewMark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.marks[m.Tenant+"/"+m.Space+"/"+m.Doc] = m
	return nil
}

// ReviewMarks implements ask.MarkStore.
func (s *Store) ReviewMarks(_ context.Context, tenant, space string) ([]ask.ReviewMark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.ReviewMark{}
	for _, m := range s.marks {
		if m.Tenant == tenant && m.Space == space {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Doc < out[j].Doc })
	return out, nil
}

// PutTemplate implements ask.TemplateStore (SB-2503).
func (s *Store) PutTemplate(_ context.Context, t ask.ReplyTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tmpls[t.Tenant+"/"+t.Space+"/"+t.ID] = t
	return nil
}

// Templates implements ask.TemplateStore: by name.
func (s *Store) Templates(_ context.Context, tenant, space string) ([]ask.ReplyTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.ReplyTemplate{}
	for _, t := range s.tmpls {
		if t.Tenant == tenant && t.Space == space {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name+out[i].ID < out[j].Name+out[j].ID })
	return out, nil
}

// DeleteTemplate implements ask.TemplateStore.
func (s *Store) DeleteTemplate(_ context.Context, tenant, space, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := tenant + "/" + space + "/" + id
	_, ok := s.tmpls[k]
	delete(s.tmpls, k)
	return ok, nil
}

var _ ask.RetireStore = (*Store)(nil)

// PutRetirement implements ask.RetireStore (SB-2801): one per space and document.
func (s *Store) PutRetirement(_ context.Context, r ask.Retirement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retired[r.Tenant+"/"+r.Space+"/"+r.Doc] = r
	return nil
}

// Retirements implements ask.RetireStore.
func (s *Store) Retirements(_ context.Context, tenant, space string) ([]ask.Retirement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.Retirement{}
	for _, r := range s.retired {
		if r.Tenant == tenant && r.Space == space {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Doc < out[j].Doc })
	return out, nil
}

// DeleteRetirement implements ask.RetireStore.
func (s *Store) DeleteRetirement(_ context.Context, tenant, space, doc string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := tenant + "/" + space + "/" + doc
	_, ok := s.retired[k]
	delete(s.retired, k)
	return ok, nil
}

var _ ask.NoticeStore = (*Store)(nil)

// PutNotice implements ask.NoticeStore (SB-3207): one per tenant.
func (s *Store) PutNotice(_ context.Context, n ask.CellNotice) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notices[n.Tenant] = n
	return nil
}

// Notice implements ask.NoticeStore.
func (s *Store) Notice(_ context.Context, tenant string) (ask.CellNotice, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notices[tenant]
	return n, ok, nil
}

// DeleteNotice implements ask.NoticeStore.
func (s *Store) DeleteNotice(_ context.Context, tenant string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.notices, tenant)
	return nil
}

var _ ask.SuggestedStore = (*Store)(nil)

// PutSuggested implements ask.SuggestedStore (SB-3307): one list per space.
func (s *Store) PutSuggested(_ context.Context, x ask.Suggested) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suggest[x.Tenant+"/"+x.Space] = x
	return nil
}

// Suggested implements ask.SuggestedStore.
func (s *Store) Suggested(_ context.Context, tenant, space string) (ask.Suggested, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.suggest[tenant+"/"+space]
	return x, ok, nil
}

var _ ask.PinStore = (*Store)(nil)

// PutPins implements ask.PinStore (SB-3601): one record per space.
func (s *Store) PutPins(_ context.Context, p ask.Pins) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.Items = append([]ask.Pin(nil), p.Items...)
	s.pins[p.Tenant+"/"+p.Space] = p
	return nil
}

// Pins implements ask.PinStore.
func (s *Store) Pins(_ context.Context, tenant, space string) (ask.Pins, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pins[tenant+"/"+space]
	p.Items = append([]ask.Pin(nil), p.Items...)
	return p, ok, nil
}

var _ ask.CaretakerStore = (*Store)(nil)

// PutCaretaker implements ask.CaretakerStore (SB-3405): one per document.
func (s *Store) PutCaretaker(_ context.Context, c ask.Caretaker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cares[c.Tenant+"/"+c.Doc] = c
	return nil
}

// CaretakerOf implements ask.CaretakerStore.
func (s *Store) CaretakerOf(_ context.Context, tenant, doc string) (ask.Caretaker, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cares[tenant+"/"+doc]
	return c, ok, nil
}

// DeleteCaretaker implements ask.CaretakerStore.
func (s *Store) DeleteCaretaker(_ context.Context, tenant, doc string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cares, tenant+"/"+doc)
	return nil
}

var _ ask.PolicyStore = (*Store)(nil)

// PutPolicy implements ask.PolicyStore (SB-4103): one per space and document.
func (s *Store) PutPolicy(_ context.Context, p ask.SpacePolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[p.Tenant+"/"+p.Space+"/"+p.Doc] = p
	return nil
}

// Policies implements ask.PolicyStore, by document.
func (s *Store) Policies(_ context.Context, tenant, space string) ([]ask.SpacePolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.SpacePolicy{}
	for _, p := range s.policies {
		if p.Tenant == tenant && p.Space == space {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Doc < out[j].Doc })
	return out, nil
}

// PutPolicyRead implements ask.PolicyStore: one per person and version.
func (s *Store) PutPolicyRead(_ context.Context, r ask.PolicyRead) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads[fmt.Sprintf("%s/%s/%s/%s/%d", r.Tenant, r.Space, r.Doc, r.Email, r.Version)] = r
	return nil
}

// PolicyReads implements ask.PolicyStore.
func (s *Store) PolicyReads(_ context.Context, tenant, space, doc string) ([]ask.PolicyRead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.PolicyRead{}
	for _, r := range s.reads {
		if r.Tenant == tenant && r.Space == space && r.Doc == doc {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Email < out[j].Email || out[i].Email == out[j].Email && out[i].Version < out[j].Version
	})
	return out, nil
}

var _ ask.ResearchStore = (*Store)(nil)

// PutResearch implements ask.ResearchStore (SB-4201).
func (s *Store) PutResearch(_ context.Context, r ask.Research) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.research[r.Tenant+"/"+r.ID] = r
	return nil
}

// GetResearch implements ask.ResearchStore.
func (s *Store) GetResearch(_ context.Context, tenant, id string) (ask.Research, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.research[tenant+"/"+id]
	return r, ok, nil
}

// ResearchOf implements ask.ResearchStore.
func (s *Store) ResearchOf(_ context.Context, tenant, user string, since time.Time) ([]ask.Research, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.Research{}
	for _, r := range s.research {
		if r.Tenant == tenant && r.User == user && !r.At.Before(since) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || out[i].At.Equal(out[j].At) && out[i].ID < out[j].ID
	})
	return out, nil
}

// QueuedResearch implements ask.ResearchStore.
func (s *Store) QueuedResearch(_ context.Context, tenant string) ([]ask.Research, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.Research{}
	for _, r := range s.research {
		if r.Tenant == tenant && r.State == ask.StateQueued {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || out[i].At.Equal(out[j].At) && out[i].ID < out[j].ID
	})
	return out, nil
}

var _ ask.ActionStore = (*Store)(nil)

// PutAction implements ask.ActionStore (SB-4203).
func (s *Store) PutAction(_ context.Context, a ask.Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[a.Tenant+"/"+a.ID] = a
	return nil
}

// GetAction implements ask.ActionStore.
func (s *Store) GetAction(_ context.Context, tenant, id string) (ask.Action, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.actions[tenant+"/"+id]
	return a, ok, nil
}

func (s *Store) actionsWhere(tenant string, keep func(ask.Action) bool) []ask.Action {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.Action{}
	for _, a := range s.actions {
		if a.Tenant == tenant && keep(a) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || out[i].At.Equal(out[j].At) && out[i].ID < out[j].ID
	})
	return out
}

// ActionsOf implements ask.ActionStore.
func (s *Store) ActionsOf(_ context.Context, tenant, user string) ([]ask.Action, error) {
	return s.actionsWhere(tenant, func(a ask.Action) bool { return a.User == user }), nil
}

// ActionsIn implements ask.ActionStore.
func (s *Store) ActionsIn(_ context.Context, tenant, space string) ([]ask.Action, error) {
	return s.actionsWhere(tenant, func(a ask.Action) bool { return a.Space == space }), nil
}

// ActionsSince implements ask.ActionStore.
func (s *Store) ActionsSince(_ context.Context, tenant string, since time.Time) ([]ask.Action, error) {
	return s.actionsWhere(tenant, func(a ask.Action) bool { return !a.At.Before(since) }), nil
}

// dropOwn drops a person's own records that retention never expires: memory, collections.
func dropOwn(s *Store, name string, try func(tenant, user, space string, at time.Time, rec any) bool) {
	if name == CollectionsName {
		for k, c := range s.colls {
			if try(c.Tenant, c.Owner, "", c.At, c) {
				delete(s.colls, k)
			}
		}
		return
	}
	for k, m := range s.memory {
		if try(m.Tenant, m.User, "", m.At, m) {
			delete(s.memory, k)
		}
	}
}

var _ ask.CollectionStore = (*Store)(nil)

func copyCollection(c ask.Collection) ask.Collection {
	c.Items = append([]string(nil), c.Items...)
	c.Shared = append([]string(nil), c.Shared...)
	return c
}

// PutCollection implements ask.CollectionStore.
func (s *Store) PutCollection(_ context.Context, c ask.Collection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.colls[c.Tenant+"/"+c.ID] = copyCollection(c)
	return nil
}

// GetCollection implements ask.CollectionStore.
func (s *Store) GetCollection(_ context.Context, tenant, id string) (ask.Collection, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.colls[tenant+"/"+id]
	return copyCollection(c), ok, nil
}

// Collections implements ask.CollectionStore: the tenant's, oldest first.
func (s *Store) Collections(_ context.Context, tenant string) ([]ask.Collection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.Collection{}
	for _, c := range s.colls {
		if c.Tenant == tenant {
			out = append(out, copyCollection(c))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || (out[i].At.Equal(out[j].At) && out[i].ID < out[j].ID)
	})
	return out, nil
}

// DeleteCollection implements ask.CollectionStore.
func (s *Store) DeleteCollection(_ context.Context, tenant, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.colls, tenant+"/"+id)
	return nil
}

var _ ask.ShadowStore = (*Store)(nil)

// OpenShadow implements ask.ShadowStore (SB-4704).
func (s *Store) OpenShadow(_ context.Context, w ask.ShadowWindow) (ask.ShadowWindow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := w.Tenant + "/" + w.Agent + "|" + w.Release
	if got, ok := s.windows[k]; ok {
		return got, nil
	}
	s.windows[k] = w
	return w, nil
}

// PutShadow implements ask.ShadowStore.
func (s *Store) PutShadow(_ context.Context, r ask.ShadowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shadows[r.Tenant+"/"+r.ID] = r
	return nil
}

// ShadowRuns implements ask.ShadowStore.
func (s *Store) ShadowRuns(_ context.Context, tenant string) ([]ask.ShadowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.ShadowRun{}
	for _, r := range s.shadows {
		if r.Tenant == tenant {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ShadowWindows implements ask.ShadowStore.
func (s *Store) ShadowWindows(_ context.Context, tenant string) ([]ask.ShadowWindow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ask.ShadowWindow{}
	for _, w := range s.windows {
		if w.Tenant == tenant {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Agent+out[i].Release < out[j].Agent+out[j].Release })
	return out, nil
}

// DeleteShadows implements ask.ShadowStore.
func (s *Store) DeleteShadows(_ context.Context, tenant string, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for k, r := range s.shadows {
		if r.Tenant == tenant && !now.Before(r.Until) {
			delete(s.shadows, k)
			n++
		}
	}
	return n, nil
}

var _ ask.RhythmStore = (*Store)(nil)

func copyRun(r ask.RhythmRun) ask.RhythmRun {
	r.Steps = append([]ask.RhythmRunStep(nil), r.Steps...)
	return r
}

// PutRhythmRun implements ask.RhythmStore (SB-4801).
func (s *Store) PutRhythmRun(_ context.Context, r ask.RhythmRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rhythms[r.Tenant+"/"+r.ID] = copyRun(r)
	return nil
}

// GetRhythmRun implements ask.RhythmStore.
func (s *Store) GetRhythmRun(_ context.Context, tenant, id string) (ask.RhythmRun, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rhythms[tenant+"/"+id]
	return copyRun(r), ok, nil
}

// RhythmRuns implements ask.RhythmStore: the tenant's, oldest first.
func (s *Store) RhythmRuns(_ context.Context, tenant string) ([]ask.RhythmRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return oldestFirst(s.rhythms, func(r ask.RhythmRun) (string, time.Time, string) { return r.Tenant, r.At, r.ID }, tenant, copyRun), nil
}

// oldestFirst are a tenant's records, by time then id, each copied.
func oldestFirst[T any](m map[string]T, key func(T) (string, time.Time, string), tenant string, cp func(T) T) []T {
	out := []T{}
	for _, v := range m {
		if t, _, _ := key(v); t == tenant {
			out = append(out, cp(v))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		_, ai, aid := key(out[i])
		_, bi, bid := key(out[j])
		if !ai.Equal(bi) {
			return ai.Before(bi)
		}
		return aid < bid
	})
	return out
}

// PutRoster implements ask.RhythmStore (SB-4802).
func (s *Store) PutRoster(_ context.Context, r ask.RhythmRoster) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.Members = append([]string(nil), r.Members...)
	s.rosters[r.Tenant+"/"+r.Space] = r
	return nil
}

// GetRoster implements ask.RhythmStore.
func (s *Store) GetRoster(_ context.Context, tenant, space string) (ask.RhythmRoster, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rosters[tenant+"/"+space]
	return r, ok, nil
}

var _ ask.ProjectStore = (*Store)(nil)

func copyProject(p ask.Project) ask.Project {
	p.Pending = append([]ask.PendingMember(nil), p.Pending...)
	p.Decisions = append([]ask.ProjectDecision(nil), p.Decisions...)
	p.Lessons = append([]ask.Lesson(nil), p.Lessons...)
	return p
}

// PutProject implements ask.ProjectStore (SB-4803).
func (s *Store) PutProject(_ context.Context, p ask.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.Tenant+"/"+p.ID] = copyProject(p)
	return nil
}

// GetProject implements ask.ProjectStore.
func (s *Store) GetProject(_ context.Context, tenant, id string) (ask.Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[tenant+"/"+id]
	return copyProject(p), ok, nil
}

// Projects implements ask.ProjectStore: the tenant's, oldest first.
func (s *Store) Projects(_ context.Context, tenant string) ([]ask.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return oldestFirst(s.projects, func(p ask.Project) (string, time.Time, string) { return p.Tenant, p.At, p.ID }, tenant, copyProject), nil
}
