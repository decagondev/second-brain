package cellstore

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

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/core/tenancy"
)

// ---- Directory (ports.DirectoryStore, People, UserFinder) ----

// Directory is the SCIM-provisioned directory.
type Directory struct{ DB }

var (
	_ ports.DirectoryStore = Directory{}
	_ ports.People         = Directory{}
	_ ports.UserFinder     = Directory{}
)

func (d Directory) users(ctx context.Context, tenant string) ([]ports.User, error) {
	return list[ports.User](ctx, d.Pool, tenant, "user", "", "id")
}

// Active implements ports.Directory.
func (d Directory) Active(ctx context.Context, tenant, user string) (bool, bool, error) {
	u, found, err := d.User(ctx, tenant, user)
	return found && u.Active, found, err
}

// PutUser implements ports.DirectoryStore.
func (d Directory) PutUser(ctx context.Context, u ports.User) error {
	return put(ctx, d.Pool, u.Tenant, "user", u.ID, meta{at: d.now(), user: u.UserName}, u, false)
}

// User implements ports.DirectoryStore.
func (d Directory) User(ctx context.Context, tenant, id string) (ports.User, bool, error) {
	return get[ports.User](ctx, d.Pool, tenant, "user", id, false)
}

// UserByName implements ports.DirectoryStore.
func (d Directory) UserByName(ctx context.Context, tenant, name string) (ports.User, bool, error) {
	us, err := list[ports.User](ctx, d.Pool, tenant, "user", "usr = $3", "id", name)
	if err != nil || len(us) == 0 {
		return ports.User{}, false, err
	}
	return us[0], true, nil
}

// UsersByEmail implements ports.UserFinder, case-insensitively.
func (d Directory) UsersByEmail(ctx context.Context, tenant, email string) ([]ports.User, error) {
	all, err := d.users(ctx, tenant)
	if err != nil {
		return nil, err
	}
	var out []ports.User
	for _, u := range all {
		for _, e := range u.Emails {
			if strings.EqualFold(e, email) {
				out = append(out, u)
				break
			}
		}
	}
	return out, nil
}

// PutGroup implements ports.DirectoryStore.
func (d Directory) PutGroup(ctx context.Context, g ports.Group) error {
	return put(ctx, d.Pool, g.Tenant, "group", g.ID, meta{at: d.now(), user: g.Name}, g, false)
}

// Group implements ports.DirectoryStore.
func (d Directory) Group(ctx context.Context, tenant, id string) (ports.Group, bool, error) {
	return get[ports.Group](ctx, d.Pool, tenant, "group", id, false)
}

// DeleteGroup implements ports.DirectoryStore.
func (d Directory) DeleteGroup(ctx context.Context, tenant, id string) error {
	return del(ctx, d.Pool, tenant, "group", id)
}

// GroupsOf implements ports.DirectoryStore: the user's group names, sorted.
func (d Directory) GroupsOf(ctx context.Context, tenant, user string) ([]string, error) {
	gs, err := list[ports.Group](ctx, d.Pool, tenant, "group", "", "id")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, g := range gs {
		for _, m := range g.Members {
			if m == user {
				out = append(out, g.Name)
				break
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// Person implements ports.People: users of the scope's tenant only.
func (d Directory) Person(ctx context.Context, s tenancy.Scope, id string) (ports.User, bool, error) {
	if err := s.Check(); err != nil {
		return ports.User{}, false, err
	}
	return d.User(ctx, s.Tenant(), id)
}

// MyGroups implements ports.People.
func (d Directory) MyGroups(ctx context.Context, s tenancy.Scope) ([]string, error) {
	if err := s.Check(); err != nil {
		return nil, err
	}
	return d.GroupsOf(ctx, s.Tenant(), s.User())
}

// Members lists a group's user ids by its name, sorted; an unknown group is an error.
func (d Directory) Members(ctx context.Context, tenant, name string) ([]string, error) {
	gs, err := list[ports.Group](ctx, d.Pool, tenant, "group", "usr = $3", "id", name)
	if err != nil {
		return nil, err
	}
	if len(gs) == 0 {
		return nil, fmt.Errorf("directory: no group %q", name)
	}
	out := append([]string{}, gs[0].Members...)
	sort.Strings(out)
	return out, nil
}

// ActiveUsers counts the tenant's active users.
func (d Directory) ActiveUsers(ctx context.Context, tenant string) (int, error) {
	all, err := d.users(ctx, tenant)
	n := 0
	for _, u := range all {
		if u.Active {
			n++
		}
	}
	return n, err
}

// ---- Sessions (ports.Sessions) ----

// Sessions are the cell's sign-in sessions, keyed by a hash of the session id: the table
// never holds a cookie value.
type Sessions struct{ DB }

var _ ports.Sessions = Sessions{}

func sid(id string) string { s := sha256.Sum256([]byte(id)); return hex.EncodeToString(s[:]) }

// Put implements ports.Sessions.
func (s Sessions) Put(ctx context.Context, x ports.Session) error {
	return put(ctx, s.Pool, s.Tenant, "session", sid(x.ID), meta{at: x.SignedIn, user: x.User}, x, false)
}

// Get implements ports.Sessions.
func (s Sessions) Get(ctx context.Context, id string) (ports.Session, bool, error) {
	return get[ports.Session](ctx, s.Pool, s.Tenant, "session", sid(id), false)
}

// Delete implements ports.Sessions.
func (s Sessions) Delete(ctx context.Context, id string) error {
	return del(ctx, s.Pool, s.Tenant, "session", sid(id))
}

// RevokeUser implements ports.Sessions.
func (s Sessions) RevokeUser(ctx context.Context, tenant, user string) (int, error) {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM cell_docs WHERE tenant = $1 AND kind = 'session' AND usr = $2`, tenant, user)
	return int(tag.RowsAffected()), err
}

// ListUser implements ports.Sessions.
func (s Sessions) ListUser(ctx context.Context, tenant, user string) ([]ports.Session, error) {
	if tenant != s.Tenant {
		return nil, nil
	}
	return list[ports.Session](ctx, s.Pool, s.Tenant, "session", "usr = $3", "at", user)
}

// ListTokens is every read-only API token of tenant (SB-2606), for the expiry warning.
func (s Sessions) ListTokens(ctx context.Context, tenant string) ([]ports.Session, error) {
	if tenant != s.Tenant {
		return nil, nil
	}
	return list[ports.Session](ctx, s.Pool, s.Tenant, "session", "COALESCE(body->>'Token', '') <> ''", "at")
}

// ---- Traces, conversations and feedback (ask.Store, FeedbackStore, SpaceTagger) ----

// Traces keeps answers, conversations and feedback.
type Traces struct{ DB }

var (
	_ ask.Store          = Traces{}
	_ ask.FeedbackStore  = Traces{}
	_ ask.FeedbackReader = Traces{}
	_ ask.SpaceTagger    = Traces{}
)

type conv struct {
	Tenant, User string
	Questions    []string
	At           time.Time
	Space        string // the space it is asked in; "mixed" if more than one
}

// PutTrace implements ask.Store; ids are never reused.
func (t Traces) PutTrace(ctx context.Context, x ask.Trace) error {
	return put(ctx, t.Pool, x.Tenant, "trace", x.ID, meta{at: x.At, user: x.User, space: x.Space}, x, true)
}

// GetTrace implements ask.Store.
func (t Traces) GetTrace(ctx context.Context, tenant, id string) (ask.Trace, bool, error) {
	return get[ask.Trace](ctx, t.Pool, tenant, "trace", id, false)
}

// Questions implements ask.Store: only the asker's own conversation is found.
func (t Traces) Questions(ctx context.Context, tenant, user, id string) ([]string, bool, error) {
	c, found, err := get[conv](ctx, t.Pool, tenant, "conv", id, false)
	if err != nil || !found || c.User != user {
		return nil, false, err
	}
	return c.Questions, true, nil
}

func convMeta(c conv) meta { return meta{at: c.At, user: c.User, space: c.Space} }

// AppendQuestion implements ask.Store.
func (t Traces) AppendQuestion(ctx context.Context, tenant, user, id, q string) error {
	tx, err := t.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	c, found, err := get[conv](ctx, tx, tenant, "conv", id, true)
	if err != nil {
		return err
	}
	if found && c.User != user {
		return errors.New("traces: conversation belongs to someone else")
	}
	c.Tenant, c.User, c.At = tenant, user, t.now()
	c.Questions = append(c.Questions, q)
	if err := put(ctx, tx, tenant, "conv", id, convMeta(c), c, false); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var errNoConv = errors.New("traces: no such conversation")

// TagSpace implements ask.SpaceTagger.
func (t Traces) TagSpace(ctx context.Context, tenant, id, space string) error {
	_, err := update(ctx, t.Pool, tenant, "conv", id, errNoConv, convMeta, func(c *conv) error {
		switch {
		case c.Space == "":
			c.Space = space
		case c.Space != space:
			c.Space = "mixed"
		}
		return nil
	})
	return err
}

// PutFeedback implements ask.FeedbackStore; a later feedback replaces it.
func (t Traces) PutFeedback(ctx context.Context, f ask.Feedback) error {
	return put(ctx, t.Pool, f.Tenant, "feedback", f.Trace, meta{at: f.At, user: f.User}, f, false)
}

// GetFeedback implements ask.FeedbackReader.
func (t Traces) GetFeedback(ctx context.Context, tenant, trace string) (ask.Feedback, bool, error) {
	return get[ask.Feedback](ctx, t.Pool, tenant, "feedback", trace, false)
}

// FeedbackBetween implements ask.FeedbackStore, oldest first.
func (t Traces) FeedbackBetween(ctx context.Context, tenant string, from, to time.Time) ([]ask.Feedback, error) {
	return list[ask.Feedback](ctx, t.Pool, tenant, "feedback", "at >= $3 AND at < $4", "at, id", from, to)
}

// Between lists traces asked in [from, to), oldest first (the weekly report).
func (t Traces) Between(ctx context.Context, tenant string, from, to time.Time) ([]ask.Trace, error) {
	return list[ask.Trace](ctx, t.Pool, tenant, "trace", "at >= $3 AND at < $4", "at, id", from, to)
}

// Retention is the stores retention sweeps and erases: feedback first, since it takes its
// space from its trace, which must still be there.
func (t Traces) Retention() []retention.Store {
	return []retention.Store{view{t, "feedback"}, view{t, "conv"}, view{t, "trace"}, view{t, "saved"}, view{t, "share"}, view{t, "stq"}, view{t, "pref"}, view{t, "evalcase"}, view{t, "accessreq"}, view{t, "reviewmark"}, view{t, "replytemplate"}}
}

var _ notify.PrefStore = Traces{}

type prefRec struct {
	Tenant string `json:"tenant"`
	User   string `json:"user"`
	notify.Prefs
}

// NotifyPrefs implements notify.PrefStore (SB-2101): none is every kind on.
func (t Traces) NotifyPrefs(ctx context.Context, tenant, user string) (notify.Prefs, error) {
	r, _, err := get[prefRec](ctx, t.Pool, tenant, "pref", user, false)
	return r.Prefs, err
}

// PutNotifyPrefs implements notify.PrefStore.
func (t Traces) PutNotifyPrefs(ctx context.Context, tenant, user string, p notify.Prefs) error {
	return put(ctx, t.Pool, tenant, "pref", user, meta{at: time.Now().UTC(), user: user}, prefRec{tenant, user, p}, false)
}

var _ ask.AccessStore = Traces{}

type accessRec struct {
	Tenant string    `json:"tenant"`
	Key    string    `json:"key"`
	User   string    `json:"user"`
	At     time.Time `json:"at"`
}

// AccessRequested implements ask.AccessStore (SB-2205).
func (t Traces) AccessRequested(ctx context.Context, tenant, key string) (bool, error) {
	_, found, err := get[accessRec](ctx, t.Pool, tenant, "accessreq", key, false)
	return found, err
}

// PutAccessRequest implements ask.AccessStore.
func (t Traces) PutAccessRequest(ctx context.Context, tenant, key, user string, at time.Time) error {
	return put(ctx, t.Pool, tenant, "accessreq", key, meta{at: at, user: user}, accessRec{tenant, key, user, at}, false)
}

var _ ask.EvalStore = Traces{}

// PutEvalCase implements ask.EvalStore (SB-2204).
func (t Traces) PutEvalCase(ctx context.Context, c ask.EvalCase) error {
	return put(ctx, t.Pool, c.Tenant, "evalcase", c.ID, meta{at: c.At, user: c.By, space: c.Space}, c, false)
}

// EvalCases implements ask.EvalStore: oldest first.
func (t Traces) EvalCases(ctx context.Context, tenant string) ([]ask.EvalCase, error) {
	return list[ask.EvalCase](ctx, t.Pool, tenant, "evalcase", "", "at, id")
}

// DeleteEvalCase implements ask.EvalStore.
func (t Traces) DeleteEvalCase(ctx context.Context, tenant, id string) error {
	return del(ctx, t.Pool, tenant, "evalcase", id)
}

var (
	_ ask.MarkStore     = Traces{}
	_ ask.TemplateStore = Traces{}
)

// PutReviewMark implements ask.MarkStore (SB-2501): one per space and document.
func (t Traces) PutReviewMark(ctx context.Context, m ask.ReviewMark) error {
	return put(ctx, t.Pool, m.Tenant, "reviewmark", m.Space+"|"+m.Doc, meta{at: m.At, user: m.By, space: m.Space}, m, false)
}

// ReviewMarks implements ask.MarkStore.
func (t Traces) ReviewMarks(ctx context.Context, tenant, space string) ([]ask.ReviewMark, error) {
	return list[ask.ReviewMark](ctx, t.Pool, tenant, "reviewmark", "space = $3", "id", space)
}

// PutTemplate implements ask.TemplateStore (SB-2503).
func (t Traces) PutTemplate(ctx context.Context, rt ask.ReplyTemplate) error {
	return put(ctx, t.Pool, rt.Tenant, "replytemplate", rt.ID, meta{at: rt.At, user: rt.By, space: rt.Space}, rt, false)
}

// Templates implements ask.TemplateStore: by name.
func (t Traces) Templates(ctx context.Context, tenant, space string) ([]ask.ReplyTemplate, error) {
	return list[ask.ReplyTemplate](ctx, t.Pool, tenant, "replytemplate", "space = $3", "body->>'name', id", space)
}

// DeleteTemplate implements ask.TemplateStore.
func (t Traces) DeleteTemplate(ctx context.Context, tenant, space, id string) (bool, error) {
	tag, err := t.Pool.Exec(ctx, `DELETE FROM cell_docs WHERE tenant = $1 AND kind = 'replytemplate' AND id = $2 AND space = $3`, tenant, id, space)
	return err == nil && tag.RowsAffected() > 0, err
}

var _ ask.StewardStore = Traces{}

func stqMeta(q ask.StewardQuestion) meta { return meta{at: q.At, user: q.Asker, space: q.Space} }

// PutStewardQuestion implements ask.StewardStore (SB-1704).
func (t Traces) PutStewardQuestion(ctx context.Context, q ask.StewardQuestion) error {
	return put(ctx, t.Pool, q.Tenant, "stq", q.ID, stqMeta(q), q, true)
}

// GetStewardQuestion implements ask.StewardStore.
func (t Traces) GetStewardQuestion(ctx context.Context, tenant, id string) (ask.StewardQuestion, bool, error) {
	return get[ask.StewardQuestion](ctx, t.Pool, tenant, "stq", id, false)
}

// StewardQuestionsOf implements ask.StewardStore.
func (t Traces) StewardQuestionsOf(ctx context.Context, tenant, asker string) ([]ask.StewardQuestion, error) {
	return list[ask.StewardQuestion](ctx, t.Pool, tenant, "stq", "usr = $3", "at DESC, id DESC", asker)
}

// StewardQueue implements ask.StewardStore.
func (t Traces) StewardQueue(ctx context.Context, tenant, space string) ([]ask.StewardQuestion, error) {
	return list[ask.StewardQuestion](ctx, t.Pool, tenant, "stq", "space = $3", "at DESC, id DESC", space)
}

// StewardQuestionsBetween implements ask.StewardStore.
func (t Traces) StewardQuestionsBetween(ctx context.Context, tenant string, from, to time.Time) ([]ask.StewardQuestion, error) {
	return list[ask.StewardQuestion](ctx, t.Pool, tenant, "stq", "at >= $3 AND at < $4", "at, id", from, to)
}

// UpdateStewardQuestion implements ask.StewardStore, with the row locked.
func (t Traces) UpdateStewardQuestion(ctx context.Context, tenant, id string, f func(*ask.StewardQuestion) error) (ask.StewardQuestion, error) {
	return update(ctx, t.Pool, tenant, "stq", id, fmt.Errorf("questions: no question %s", id), stqMeta, f)
}

var _ ask.QuestionStore = Traces{}

// PutSaved implements ask.QuestionStore (SB-1703).
func (t Traces) PutSaved(ctx context.Context, q ask.SavedQuestion) error {
	return put(ctx, t.Pool, q.Tenant, "saved", q.ID, meta{at: q.At, user: q.User, space: q.Space}, savedRec{q, q.Tenant, q.User, q.Sources}, true)
}

// SavedOf implements ask.QuestionStore: newest first.
func (t Traces) SavedOf(ctx context.Context, tenant, user string) ([]ask.SavedQuestion, error) {
	rs, err := list[savedRec](ctx, t.Pool, tenant, "saved", "usr = $3", "at DESC, id DESC", user)
	out := make([]ask.SavedQuestion, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.full())
	}
	return out, err
}

// DeleteSaved implements ask.QuestionStore.
func (t Traces) DeleteSaved(ctx context.Context, tenant, user, id string) (bool, error) {
	tag, err := t.Pool.Exec(ctx, `DELETE FROM cell_docs WHERE tenant = $1 AND kind = 'saved' AND id = $2 AND usr = $3`, tenant, id, user)
	return tag.RowsAffected() == 1, err
}

// PutShare implements ask.QuestionStore.
func (t Traces) PutShare(ctx context.Context, sh ask.Share) error {
	return put(ctx, t.Pool, sh.Tenant, "share", sh.ID, meta{at: sh.At, user: sh.User, space: sh.Space}, shareRec{sh, sh.Tenant, sh.User, sh.At}, true)
}

// GetShare implements ask.QuestionStore.
func (t Traces) GetShare(ctx context.Context, tenant, id string) (ask.Share, bool, error) {
	r, found, err := get[shareRec](ctx, t.Pool, tenant, "share", id, false)
	return r.full(), found, err
}

// savedRec and shareRec keep the fields the API never shows (tenant, user, time).
type savedRec struct {
	ask.SavedQuestion
	T string `json:"tenant"`
	U string `json:"user"`
	H string `json:"sources,omitempty"` // the watch's hash (SB-2104)
}

func (r savedRec) full() ask.SavedQuestion {
	q := r.SavedQuestion
	q.Tenant, q.User, q.Sources = r.T, r.U, r.H
	return q
}

var _ ask.WatchStore = Traces{}

// UpdateSaved implements ask.WatchStore, with the row locked.
func (t Traces) UpdateSaved(ctx context.Context, tenant, user, id string, f func(*ask.SavedQuestion)) (bool, error) {
	errNone := errors.New("saved: none")
	_, err := update(ctx, t.Pool, tenant, "saved", id, errNone, func(r savedRec) meta {
		return meta{at: r.At, user: r.U, space: r.Space}
	}, func(r *savedRec) error {
		if r.U != user {
			return errNone
		}
		q := r.full()
		f(&q)
		*r = savedRec{q, tenant, user, q.Sources}
		return nil
	})
	if errors.Is(err, errNone) {
		return false, nil
	}
	return err == nil, err
}

// Watched implements ask.WatchStore: oldest first.
func (t Traces) Watched(ctx context.Context, tenant string) ([]ask.SavedQuestion, error) {
	rs, err := list[savedRec](ctx, t.Pool, tenant, "saved", "", "at, id")
	var out []ask.SavedQuestion
	for _, r := range rs {
		if r.Watch {
			out = append(out, r.full())
		}
	}
	return out, err
}

type shareRec struct {
	ask.Share
	T  string    `json:"tenant"`
	U  string    `json:"user"`
	At time.Time `json:"made"`
}

func (r shareRec) full() ask.Share {
	sh := r.Share
	sh.Tenant, sh.User, sh.At = r.T, r.U, r.At
	return sh
}

type view struct {
	t    Traces
	kind string
}

func (v view) Name() string {
	return map[string]string{"feedback": "feedback", "conv": "conversations", "trace": "traces", "saved": "saved questions", "share": "shares", "stq": "steward questions", "pref": "notification settings", "evalcase": "eval cases", "accessreq": "access requests", "reviewmark": "review marks", "replytemplate": "reply templates"}[v.kind]
}

// spaceOf is a record's space for holds: "" (unknown, so any hold may cover it) for one
// asked company-wide or in more than one space.
func spaceOf(s string) string {
	if s == "company" || s == "mixed" {
		return ""
	}
	return s
}

type rec struct {
	id, user, space string
}

func (v view) records(ctx context.Context, tenant string, before *time.Time, user string) ([]rec, error) {
	// Feedback takes its space from its trace (same id); a feedback whose trace is gone has
	// an unknown space, which any hold may cover.
	sql := `SELECT d.id, d.usr, CASE WHEN d.kind = 'feedback' THEN COALESCE(t.space, '') ELSE d.space END FROM cell_docs d
		LEFT JOIN cell_docs t ON d.kind = 'feedback' AND t.tenant = d.tenant AND t.kind = 'trace' AND t.id = d.id
		WHERE d.tenant = $1 AND d.kind = $2`
	args := []any{tenant, v.kind}
	if before != nil {
		sql += ` AND d.at < $3`
		args = append(args, *before)
	} else {
		sql += ` AND d.usr = $3`
		args = append(args, user)
	}
	rows, err := v.t.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.user, &r.space); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Expire implements retention.Store: records older than before, except held ones.
func (v view) Expire(ctx context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
	if v.kind == "pref" { // settings are kept until changed or erased
		return 0, 0, nil
	}
	rs, err := v.records(ctx, tenant, &before, "")
	if err != nil {
		return 0, 0, err
	}
	n, kept := 0, 0
	for _, r := range rs {
		if held.Is(spaceOf(r.space), r.user) {
			kept++
			continue
		}
		if err := del(ctx, v.t.Pool, tenant, v.kind, r.id); err != nil {
			return n, kept, err
		}
		n++
	}
	return n, kept, nil
}

// Plan implements retention.Store: Expire's selection, deleting nothing.
func (v view) Plan(ctx context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
	if v.kind == "pref" {
		return 0, 0, nil
	}
	rs, err := v.records(ctx, tenant, &before, "")
	if err != nil {
		return 0, 0, err
	}
	n, kept := 0, 0
	for _, r := range rs {
		if held.Is(spaceOf(r.space), r.user) {
			kept++
		} else {
			n++
		}
	}
	return n, kept, nil
}

// Export implements retention.Store: the bodies of the rows Erase deletes.
func (v view) Export(ctx context.Context, tenant, user string) ([]json.RawMessage, error) {
	rows, err := v.t.Pool.Query(ctx, `SELECT body FROM cell_docs WHERE tenant = $1 AND kind = $2 AND usr = $3 ORDER BY at, id`, tenant, v.kind, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []json.RawMessage
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, json.RawMessage(b))
	}
	return out, rows.Err()
}

// Erase implements retention.Store.
func (v view) Erase(ctx context.Context, tenant, user string) (int, error) {
	tag, err := v.t.Pool.Exec(ctx, `DELETE FROM cell_docs WHERE tenant = $1 AND kind = $2 AND usr = $3`, tenant, v.kind, user)
	return int(tag.RowsAffected()), err
}

// ---- Actions and receipts (approvals.Store, toolbus.Receipts) ----

// Actions are the tool bus's proposed actions.
type Actions struct{ DB }

var _ approvals.Store = Actions{}

func actionMeta(a approvals.Action) meta {
	at := a.FinishedAt
	if at.IsZero() {
		at = a.ExpiresAt
	}
	return meta{at: at, user: a.Requester, space: a.Space}
}

// Put implements approvals.Store; ids are never reused.
func (s Actions) Put(ctx context.Context, a approvals.Action) error {
	return put(ctx, s.Pool, a.Tenant, "action", a.ID, actionMeta(a), a, true)
}

// Get implements approvals.Store.
func (s Actions) Get(ctx context.Context, tenant, id string) (approvals.Action, bool, error) {
	return get[approvals.Action](ctx, s.Pool, tenant, "action", id, false)
}

// Move implements approvals.Store: the change applies only from state from, and only along
// a transition of sb-approvals(7).
func (s Actions) Move(ctx context.Context, tenant, id string, from approvals.State, change func(*approvals.Action) error) (approvals.Action, error) {
	return update(ctx, s.Pool, tenant, "action", id, fmt.Errorf("actions: no action %s", id), actionMeta, func(a *approvals.Action) error {
		if a.State != from {
			return approvals.ErrMoved
		}
		old := a.State
		if err := change(a); err != nil {
			return err
		}
		if a.State != old && !approvals.CanMove(old, a.State) {
			return fmt.Errorf("actions: %s -> %s is not a transition in sb-approvals(7)", old, a.State)
		}
		return nil
	})
}

// Unconfirmed implements approvals.Store.
func (s Actions) Unconfirmed(ctx context.Context, tenant string) ([]approvals.Action, error) {
	all, err := s.All(ctx, tenant)
	var out []approvals.Action
	for _, a := range all {
		if a.State == approvals.DidNotRun && a.Unconfirmed {
			out = append(out, a)
		}
	}
	return out, err
}

// All lists the tenant's actions by id (the backup snapshot).
func (s Actions) All(ctx context.Context, tenant string) ([]approvals.Action, error) {
	return list[approvals.Action](ctx, s.Pool, tenant, "action", "", "id")
}

// Receipts is the idempotency store: a key keeps its first receipt.
type Receipts struct{ DB }

// Get implements toolbus.Receipts.
func (r Receipts) Get(ctx context.Context, tenant, key string) (string, bool, error) {
	return get[string](ctx, r.Pool, tenant, "receipt", key, false)
}

// Put implements toolbus.Receipts.
func (r Receipts) Put(ctx context.Context, tenant, key, receipt string) error {
	b, _ := json.Marshal(receipt)
	_, err := r.Pool.Exec(ctx, `INSERT INTO cell_docs (tenant, kind, id, at, body) VALUES ($1, 'receipt', $2, $3, $4)
		ON CONFLICT (tenant, kind, id) DO NOTHING`, tenant, key, r.now(), b)
	return err
}

// ---- Proposals, reviews and holds ----

// Proposals are steward proposals (proposals.Store).
type Proposals struct{ DB }

var _ proposals.Store = Proposals{}

func (p Proposals) meta(x proposals.Proposal) meta { return meta{at: p.now()} }

// Put implements proposals.Store.
func (p Proposals) Put(ctx context.Context, x proposals.Proposal) error {
	return put(ctx, p.Pool, p.Tenant, "proposal", x.ID, p.meta(x), x, false)
}

// Get implements proposals.Store.
func (p Proposals) Get(ctx context.Context, id string) (proposals.Proposal, bool, error) {
	return get[proposals.Proposal](ctx, p.Pool, p.Tenant, "proposal", id, false)
}

// List implements proposals.Store, by id.
func (p Proposals) List(ctx context.Context) ([]proposals.Proposal, error) {
	return list[proposals.Proposal](ctx, p.Pool, p.Tenant, "proposal", "", "id")
}

// Update implements proposals.Store.
func (p Proposals) Update(ctx context.Context, id string, f func(x *proposals.Proposal) error) (proposals.Proposal, error) {
	return update(ctx, p.Pool, p.Tenant, "proposal", id, proposals.ErrNotFound, p.meta, f)
}

// Reviews are access reviews (reviews.Store).
type Reviews struct{ DB }

var _ reviews.Store = Reviews{}

func (r Reviews) meta(reviews.Review) meta { return meta{at: r.now()} }

// Put implements reviews.Store.
func (r Reviews) Put(ctx context.Context, x reviews.Review) error {
	return put(ctx, r.Pool, r.Tenant, "review", x.ID, r.meta(x), x, false)
}

// Get implements reviews.Store.
func (r Reviews) Get(ctx context.Context, id string) (reviews.Review, bool, error) {
	return get[reviews.Review](ctx, r.Pool, r.Tenant, "review", id, false)
}

// List implements reviews.Store, by id.
func (r Reviews) List(ctx context.Context) ([]reviews.Review, error) {
	return list[reviews.Review](ctx, r.Pool, r.Tenant, "review", "", "id")
}

// Update implements reviews.Store.
func (r Reviews) Update(ctx context.Context, id string, f func(x *reviews.Review) error) (reviews.Review, error) {
	return update(ctx, r.Pool, r.Tenant, "review", id, reviews.ErrNotFound, r.meta, f)
}

// Holds are legal holds (holds.Store).
type Holds struct{ DB }

var _ holds.Store = Holds{}

// Put implements holds.Store.
func (h Holds) Put(ctx context.Context, x holds.Hold) error {
	return put(ctx, h.Pool, h.Tenant, "hold", x.ID, meta{at: x.PlacedAt, space: x.Space}, x, false)
}

// Get implements holds.Store.
func (h Holds) Get(ctx context.Context, id string) (holds.Hold, bool, error) {
	return get[holds.Hold](ctx, h.Pool, h.Tenant, "hold", id, false)
}

// List implements holds.Store, by id.
func (h Holds) List(ctx context.Context) ([]holds.Hold, error) {
	return list[holds.Hold](ctx, h.Pool, h.Tenant, "hold", "", "id")
}

// ---- Queue (the chat events inbox) ----

// Queue is an append-only, gap-free list of JSON values; a position counts from 0.
type Queue struct {
	DB
	Kind string
}

// Append adds v at the end.
func (q Queue) Append(ctx context.Context, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tx, err := q.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1 || '/' || $2))`, q.Tenant, q.Kind); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO cell_queue (tenant, kind, pos, body)
		SELECT $1, $2, COALESCE(MAX(pos) + 1, 0), $3 FROM cell_queue WHERE tenant = $1 AND kind = $2`, q.Tenant, q.Kind, b); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Len is how many values there are.
func (q Queue) Len(ctx context.Context) (int, error) {
	var n int
	err := q.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM cell_queue WHERE tenant = $1 AND kind = $2`, q.Tenant, q.Kind).Scan(&n)
	return n, err
}

// ErrCursor is a position outside the queue.
var ErrCursor = errors.New("cellstore: position outside the queue")

// Read returns the values from position from, and the position after the last.
func (q Queue) Read(ctx context.Context, from int) ([]json.RawMessage, int, error) {
	n, err := q.Len(ctx)
	if err != nil {
		return nil, from, err
	}
	if from < 0 || from > n {
		return nil, from, ErrCursor
	}
	rows, err := q.Pool.Query(ctx, `SELECT body FROM cell_queue WHERE tenant = $1 AND kind = $2 AND pos >= $3 AND pos < $4 ORDER BY pos`,
		q.Tenant, q.Kind, from, n)
	if err != nil {
		return nil, from, err
	}
	defer rows.Close()
	var out []json.RawMessage
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, from, err
		}
		out = append(out, b)
	}
	return out, n, rows.Err()
}

// ActiveUserIDs lists the tenant's active users' ids, sorted.
func (d Directory) ActiveUserIDs(ctx context.Context, tenant string) ([]string, error) {
	all, err := d.users(ctx, tenant)
	var out []string
	for _, u := range all {
		if u.Active {
			out = append(out, u.ID)
		}
	}
	return out, err
}
