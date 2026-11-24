// Package notify tells people when something waits for them (sb-ask(3), Notifications;
// SB-2101, SB-2102). A message is a kind, a count, a space's name and a link to the cell's
// own page: never a question, a reply, a cite, a document's title or who asked (LAW-07).
// Who is told is decided when the message is sent, from the directory as it is then
// (LAW-21, LAW-12).
package notify

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/reviews"
)

// Kinds a person can turn off.
const (
	KindQuestions = "questions" // new questions, to stewards (SB-2101)
	KindReplies   = "replies"   // replies and notes, to askers (SB-2102)
	KindDigest    = "digest"    // the weekly digest (SB-2103)
	KindWatch     = "watch"     // a watched saved question's sources changed (SB-2104)
	KindAccess    = "access"    // someone asked for access to a document (SB-2205)
	KindTokens    = "tokens"    // a read-only API token expires in TokenWarn (SB-2606)
	KindOwned     = "owned"     // an owner's often-cited documents are past review, monthly (SB-2806)
)

// TokenWarn is how long before a token expires its owner is told.
const TokenWarn = 7 * 24 * time.Hour

// Expiring is a read-only API token: its owner, its name and its end. Never the token.
type Expiring struct {
	Owner   string
	Name    string
	Expires time.Time
}

// KindTest is a test a person or an operator asks for (SB-2602): always sent, never off.
const KindTest = "test"

// TestText is the test's whole text, fixed, so a test says nothing else.
const TestText = "This is a test from Second Brain. Its notifications reach you here."

// Kinds are every kind, for validating a person's settings.
var Kinds = map[string]bool{KindQuestions: true, KindReplies: true, KindDigest: true, KindWatch: true, KindAccess: true, KindTokens: true, KindOwned: true}

// Person is one recipient: a directory user and the email the chat finds them by.
type Person struct {
	ID    string
	Email string
}

// Message is what a person is told. Text holds counts and names of spaces only.
type Message struct {
	Kind string
	Text string
	Link string // an https URL on the cell's own web UI
}

// Sender delivers a message to one person: a chat bot's direct message, or email.
type Sender interface {
	Send(ctx context.Context, to Person, m Message) error
}

// ErrNoSuchPerson is a person the chat doesn't know by their email: skipped, not failed.
var ErrNoSuchPerson = errors.New("notify: the chat knows no one by this email")

// Prefs are a person's settings: the kinds they turned off, and until when everything is
// paused (SB-2506).
type Prefs struct {
	Off         []string   `json:"off"`
	PausedUntil *time.Time `json:"paused_until,omitempty"`
}

// MaxPauseDays is the longest pause.
const MaxPauseDays = 30

// ErrPause is a pause outside 0 to MaxPauseDays days.
var ErrPause = errors.New("notify: EINVAL a pause is 0 to 30 days")

// Pause sets the pause to days from now; 0 ends it.
func (p Prefs) Pause(days int, now time.Time) (Prefs, error) {
	if days < 0 || days > MaxPauseDays {
		return p, ErrPause
	}
	p.PausedUntil = nil
	if days > 0 {
		u := now.UTC().Add(time.Duration(days) * 24 * time.Hour)
		p.PausedUntil = &u
	}
	return p, nil
}

// Paused reports whether everything is paused at now.
func (p Prefs) Paused(now time.Time) bool {
	return p.PausedUntil != nil && now.Before(*p.PausedUntil)
}

// Wants reports whether the person wants messages of kind.
func (p Prefs) Wants(kind string) bool {
	for _, k := range p.Off {
		if k == kind {
			return false
		}
	}
	return true
}

// ErrKind is a setting that names no kind.
var ErrKind = errors.New("notify: EINVAL not a notification kind")

// Check validates settings and returns them sorted, each kind once.
func (p Prefs) Check() (Prefs, error) {
	seen := map[string]bool{}
	out := Prefs{Off: []string{}, PausedUntil: p.PausedUntil}
	for _, k := range p.Off {
		if !Kinds[k] {
			return Prefs{}, fmt.Errorf("%w: %q (one of questions, replies, digest, watch, access, tokens, owned)", ErrKind, k)
		}
		if !seen[k] {
			seen[k] = true
			out.Off = append(out.Off, k)
		}
	}
	sort.Strings(out.Off)
	return out, nil
}

// PrefStore keeps each person's settings, until they change them or are erased.
type PrefStore interface {
	NotifyPrefs(ctx context.Context, tenant, user string) (Prefs, error) // none is every kind on
	PutNotifyPrefs(ctx context.Context, tenant, user string, p Prefs) error
}

// Clock keeps when the job last did each part: "last" (questions and replies) and
// "weekly" (the digest and watched questions).
type Clock interface {
	Last(ctx context.Context, part string) (t time.Time, found bool, err error)
	Mark(ctx context.Context, part string, t time.Time) error
}

// Directory is what the job reads about people, at send time.
type Directory interface {
	Members(ctx context.Context, tenant, group string) ([]string, error)
	User(ctx context.Context, tenant, id string) (ports.User, bool, error)
}

// Job is one cell's notify job.
type Job struct {
	Tenant    string
	Base      string // the cell's base_url
	Questions ask.StewardStore
	Facts     ask.SpaceFacts
	Dir       Directory
	// FoundNothing counts, per space, the asks in a window that found nothing, for the
	// stewards' line (SB-3101); nil adds nothing.
	FoundNothing func(ctx context.Context, from, to time.Time) (map[string]int, error)
	Prefs        PrefStore
	Send         Sender // nil sends nothing
	Clock        Clock
	Now          func() time.Time
	// Lookback is how far back a question can be and still have its reply told; the
	// tenant's retention, since older questions are gone.
	Lookback time.Duration
	// Failed counts a send that failed, by kind; nil counts nothing.
	Failed func(kind string)

	// The weekly part (SB-2103, SB-2104) runs once a week, the first run after Monday
	// 08:00 in Zone (nil is UTC). Digest false sends no digest.
	Zone   *time.Location
	Digest bool
	// Askers are the people who asked anything in [from, to).
	Askers func(ctx context.Context, from, to time.Time) ([]string, error)
	// Changes are the titles of the documents that changed this week in user's spaces,
	// as user, now (GET /v1/changes), at most MaxDigest, and how many more.
	Changes func(ctx context.Context, user string) (titles []string, more int, err error)
	// Watches and Sources check watched saved questions: Sources is the hash of the
	// question's search results as user, in space, now.
	Watches ask.WatchStore
	Sources func(ctx context.Context, user, space, question string) (string, error)
	// Tokens are the read-only API tokens that expire in [from, to), revoked ones gone
	// (SB-2606); nil tells no one.
	Tokens func(ctx context.Context, from, to time.Time) ([]Expiring, error)
	// Owned is, by user id, how many documents each active owner owns, can open now, and
	// answers often cite while past review (SB-2806); nil tells no one.
	Owned func(ctx context.Context, now time.Time) (map[string]int, error)
	// Helped is, by user id, how many answers rated helpful last calendar month cited the
	// documents each person looks after (SB-3406); nil adds no line.
	Helped func(ctx context.Context, now time.Time) (map[string]int, error)
	// Audit writes a row for each question shown to the company-wide answerers because it
	// waited (SB-3404); nil shows none there.
	Audit ask.Auditor
	// Reviews are the access reviews (SB-3504): the stewards hear once when one opens and
	// once when it becomes overdue; nil tells no one.
	Reviews func(ctx context.Context) ([]reviews.Review, error)
}

// The digest's limits.
const (
	MaxDigest    = 10
	DigestAskers = 90 * 24 * time.Hour // who asked in this long gets one
)

// FirstLookback is how far back the job reads on its first run.
const FirstLookback = 10 * time.Minute

// Result is what one run did.
type Result struct {
	Sent, Skipped, Failed int
}

// RunRecord is the last run as the cell keeps it for health and the weekly report
// (SB-2607): when, the channel, and counts only, never who.
type RunRecord struct {
	At      time.Time `json:"at"`
	Via     string    `json:"via"`
	Sent    int       `json:"sent"`
	Skipped int       `json:"skipped"`
	Failed  int       `json:"failed"`
	Error   bool      `json:"error"` // the run stopped or some sends failed
}

// ErrSend is a run in which some sends failed; the events are not sent again.
var ErrSend = errors.New("notify: some messages couldn't be sent")

// Run tells what happened in [last run, now).
func (j Job) Run(ctx context.Context) (Result, error) {
	now := j.Now().UTC()
	from, found, err := j.Clock.Last(ctx, "last")
	if err != nil {
		return Result{}, fmt.Errorf("notify: last run: %w", err)
	}
	if !found || from.After(now) {
		from = now.Add(-FirstLookback)
	}
	moved, err := j.reroute(ctx, now)
	if err != nil {
		return Result{}, err
	}
	var out []outgoing
	if j.Send != nil {
		if out, err = j.plan(ctx, from, now); err != nil {
			return Result{}, err
		}
		more, err := j.waited(ctx, moved)
		if err != nil {
			return Result{}, err
		}
		out = append(out, more...)
	}
	revs, revMarks, err := j.reviews(ctx, now)
	if err != nil {
		return Result{}, err
	}
	if j.Send != nil {
		out = append(out, revs...)
	}
	week, due, err := j.weeklyDue(ctx, now)
	if err != nil {
		return Result{}, err
	}
	marks := revMarks
	if due {
		more, mark, err := j.weekly(ctx, now)
		if err != nil {
			return Result{}, err
		}
		if j.Send != nil {
			out = append(out, more...)
		}
		marks = append(marks, mark...)
	}
	var res Result
	for _, o := range out {
		if !j.wants(ctx, o.to.ID, o.msg.Kind, now) {
			res.Skipped++
			continue
		}
		if err := j.Send.Send(ctx, o.to, o.msg); errors.Is(err, ErrNoSuchPerson) {
			res.Skipped++
			continue
		} else if err != nil {
			res.Failed++
			if j.Failed != nil {
				j.Failed(o.msg.Kind)
			}
			continue
		}
		res.Sent++
	}
	for _, m := range marks {
		if err := m(); err != nil {
			return res, fmt.Errorf("notify: watched questions: %w", err)
		}
	}
	if err := j.Clock.Mark(ctx, "last", now); err != nil {
		return res, fmt.Errorf("notify: mark: %w", err)
	}
	if due {
		if err := j.Clock.Mark(ctx, "weekly", week); err != nil {
			return res, fmt.Errorf("notify: mark: %w", err)
		}
	}
	if res.Failed > 0 {
		return res, fmt.Errorf("%w (%d)", ErrSend, res.Failed)
	}
	return res, nil
}

// wants reads the person's settings; settings that can't be read send nothing, and a
// paused person gets nothing until the pause ends.
func (j Job) wants(ctx context.Context, user, kind string, now time.Time) bool {
	if j.Prefs == nil {
		return true
	}
	p, err := j.Prefs.NotifyPrefs(ctx, j.Tenant, user)
	return err == nil && p.Wants(kind) && !p.Paused(now)
}

type outgoing struct {
	to  Person
	msg Message
}

// plan reads the window and decides every message before any is sent.
func (j Job) plan(ctx context.Context, from, now time.Time) ([]outgoing, error) {
	asked, err := j.Questions.StewardQuestionsBetween(ctx, j.Tenant, from, now)
	if err != nil {
		return nil, fmt.Errorf("notify: questions: %w", err)
	}
	back := j.Lookback
	if back <= 0 {
		back = 400 * 24 * time.Hour
	}
	all, err := j.Questions.StewardQuestionsBetween(ctx, j.Tenant, now.Add(-back), now)
	if err != nil {
		return nil, fmt.Errorf("notify: questions: %w", err)
	}
	out, err := j.stewards(ctx, asked)
	if err != nil {
		return nil, err
	}
	more, err := j.askers(ctx, all, from, now)
	if err != nil {
		return nil, err
	}
	out = append(out, more...)
	toks, err := j.tokens(ctx, from, now)
	return append(out, toks...), err
}

// reroute shows the company-wide answerers each question open RerouteAfter working days in
// its space, with no live claim (SB-3404). It stays in its space's queue too. Each one is
// audited, counts and ids only.
func (j Job) reroute(ctx context.Context, now time.Time) (int, error) {
	if j.Audit == nil || j.Facts == nil {
		return 0, nil
	}
	if _, _, ok := j.Facts("company"); !ok {
		return 0, nil
	}
	back := j.Lookback
	if back <= 0 {
		back = 400 * 24 * time.Hour
	}
	qs, err := j.Questions.StewardQuestionsBetween(ctx, j.Tenant, now.Add(-back), now)
	if err != nil {
		return 0, fmt.Errorf("notify: questions: %w", err)
	}
	due := func(q ask.StewardQuestion) bool {
		return q.Space != "company" && q.Also == "" && q.Reply == nil && (q.Claim == nil || !now.Before(q.Claim.Until)) &&
			ask.WorkingDays(q.At, now, j.Zone) >= ask.RerouteAfter
	}
	n := 0
	for _, q := range qs {
		if !due(q) {
			continue
		}
		if _, err := j.Audit.Write(ctx, audit.Event{Actor: "sys_notify", Kind: "access", Subject: q.ID, State: "rerouted",
			Reason: "open 10 working days", Space: q.Space}); err != nil {
			return n, fmt.Errorf("notify: audit: %w", err)
		}
		moved := false
		if _, err := j.Questions.UpdateStewardQuestion(ctx, j.Tenant, q.ID, func(q *ask.StewardQuestion) error {
			if moved = due(*q); moved {
				q.Also, q.Waited = "company", ask.WorkingDays(q.At, now, j.Zone)
			}
			return nil
		}); err != nil {
			return n, fmt.Errorf("notify: questions: %w", err)
		}
		if moved {
			n++
		}
	}
	return n, nil
}

// waited tells the company-wide answerers how many questions came to them this run.
func (j Job) waited(ctx context.Context, n int) ([]outgoing, error) {
	if n == 0 {
		return nil, nil
	}
	groups, _, ok := j.Facts("company")
	if !ok {
		return nil, nil
	}
	people, err := j.members(ctx, groups)
	if err != nil {
		return nil, err
	}
	text := "1 question waited 10 working days in its space"
	if n != 1 {
		text = fmt.Sprintf("%d questions waited 10 working days in their spaces", n)
	}
	var out []outgoing
	for _, p := range people {
		out = append(out, outgoing{p, Message{Kind: KindQuestions, Text: text, Link: j.stewardLink("company")}})
	}
	return out, nil
}

// reviews tells each open review's stewards once when it opens and once when it becomes
// overdue: counts and dates only. Each notice is marked in Clock after the run sends, so a
// review is never told twice.
func (j Job) reviews(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	if j.Reviews == nil {
		return nil, nil, nil
	}
	all, err := j.Reviews(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: reviews: %w", err)
	}
	sort.Slice(all, func(a, b int) bool { return all[a].ID < all[b].ID })
	var out []outgoing
	var marks []func() error
	for _, r := range all {
		if r.State != reviews.Open {
			continue
		}
		due, err := time.Parse(time.RFC3339, r.Due)
		if err != nil {
			continue
		}
		name := strings.TrimPrefix(r.Space, "spc_")
		notices := []struct{ part, text string }{
			{"review-" + r.ID + "-open", fmt.Sprintf("The %s access review is open: %s, due %s", name, plural(len(r.Items), "person", "people"), due.UTC().Format(time.DateOnly))},
		}
		if !now.Before(due) {
			notices = append(notices, struct{ part, text string }{"review-" + r.ID + "-overdue", "The " + name + " access review is overdue"})
		}
		for _, n := range notices {
			_, told, err := j.Clock.Last(ctx, n.part)
			if err != nil {
				return nil, nil, fmt.Errorf("notify: reviews: %w", err)
			}
			if told {
				continue
			}
			people, err := j.members(ctx, r.Stewards)
			if err != nil {
				return nil, nil, err
			}
			for _, p := range people {
				out = append(out, outgoing{p, Message{Kind: KindQuestions, Text: n.text, Link: j.stewardLink(r.Space)}})
			}
			part := n.part
			marks = append(marks, func() error { return j.Clock.Mark(ctx, part, now) })
		}
	}
	return out, marks, nil
}

// tokens tells each token's owner once, TokenWarn before it expires: the tokens whose
// warning time falls in this run's window, to owners still active.
func (j Job) tokens(ctx context.Context, from, now time.Time) ([]outgoing, error) {
	if j.Tokens == nil {
		return nil, nil
	}
	ts, err := j.Tokens(ctx, from.Add(TokenWarn), now.Add(TokenWarn))
	if err != nil {
		return nil, fmt.Errorf("notify: tokens: %w", err)
	}
	sort.Slice(ts, func(a, b int) bool { return ts[a].Expires.Before(ts[b].Expires) })
	var out []outgoing
	for _, t := range ts {
		p, ok, err := j.person(ctx, t.Owner)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		days := int((t.Expires.Sub(now) + 12*time.Hour) / (24 * time.Hour))
		out = append(out, outgoing{p, Message{Kind: KindTokens, Text: fmt.Sprintf("Your API token %s expires in %s", t.Name, plural(days, "day", "days")), Link: j.tab("answer")}})
	}
	return out, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func (j Job) link() string { return j.tab("questions") }

// stewardLink is the space's steward screen.
func (j Job) stewardLink(space string) string {
	return strings.TrimSuffix(j.Base, "/") + "/?steward=" + strings.TrimPrefix(space, "spc_")
}

// stewards is one message per space and steward, to the stewards there are now.
func (j Job) stewards(ctx context.Context, asked []ask.StewardQuestion) ([]outgoing, error) {
	count := map[string]int{}
	for _, q := range asked {
		count[q.Space]++
	}
	spaces := make([]string, 0, len(count))
	for sp := range count {
		spaces = append(spaces, sp)
	}
	sort.Strings(spaces)
	var out []outgoing
	for _, sp := range spaces {
		groups, _, ok := j.Facts(sp)
		if !ok {
			continue
		}
		where := "sent company-wide"
		if sp != "company" {
			where = "for the " + strings.TrimPrefix(sp, "spc_") + " space"
		}
		msg := Message{Kind: KindQuestions, Text: plural(count[sp], "new question", "new questions") + " " + where, Link: j.stewardLink(sp)}
		people, err := j.members(ctx, groups)
		if err != nil {
			return nil, err
		}
		for _, p := range people {
			out = append(out, outgoing{p, msg})
		}
	}
	return out, nil
}

// members are the active members of groups now, each once.
func (j Job) members(ctx context.Context, groups []string) ([]Person, error) {
	seen := map[string]bool{}
	var out []Person
	for _, g := range groups {
		name, ok := strings.CutPrefix(g, "group:")
		if !ok {
			continue
		}
		ids, err := j.Dir.Members(ctx, j.Tenant, name)
		if err != nil {
			continue // a group the directory doesn't know has no one to tell
		}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			if p, ok, err := j.person(ctx, id); err != nil {
				return nil, err
			} else if ok {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// person is an active directory user with an email; anyone else is no one.
func (j Job) person(ctx context.Context, id string) (Person, bool, error) {
	u, found, err := j.Dir.User(ctx, j.Tenant, id)
	if err != nil {
		return Person{}, false, fmt.Errorf("notify: directory: %w", err)
	}
	if !found || !u.Active || len(u.Emails) == 0 || u.Emails[0] == "" {
		return Person{}, false, nil
	}
	return Person{ID: u.ID, Email: u.Emails[0]}, true, nil
}

// askers is one message per asker whose questions got a reply or a note in the window.
func (j Job) askers(ctx context.Context, all []ask.StewardQuestion, from, now time.Time) ([]outgoing, error) {
	in := func(t time.Time) bool { return !t.IsZero() && !t.Before(from) && t.Before(now) }
	replies, notes := map[string]int{}, map[string]int{}
	for _, q := range all {
		if q.Reply == nil {
			continue
		}
		if in(q.Reply.At) {
			replies[q.Asker]++
		}
		if in(q.Reply.NoteAt) {
			notes[q.Asker]++
		}
	}
	var out []outgoing
	add := func(m map[string]int, one, many string) error {
		ids := make([]string, 0, len(m))
		for id := range m {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			p, ok, err := j.person(ctx, id)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			text := "1 of your questions " + one
			if m[id] != 1 {
				text = fmt.Sprintf("%d of your questions %s", m[id], many)
			}
			out = append(out, outgoing{p, Message{Kind: KindReplies, Text: text, Link: j.link()}})
		}
		return nil
	}
	if err := add(replies, "has a reply", "have replies"); err != nil {
		return nil, err
	}
	return out, add(notes, "was published as a note", "were published as notes")
}

// Monday is the latest Monday 08:00 in zone at or before now.
func Monday(now time.Time, zone *time.Location) time.Time {
	if zone == nil {
		zone = time.UTC
	}
	t := now.In(zone)
	d := time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, zone)
	for d.Weekday() != time.Monday || d.After(t) {
		d = d.AddDate(0, 0, -1)
	}
	return d.UTC()
}

// weeklyDue reports whether this week's part hasn't run yet. On a cell's first run it
// marks this week as done, so a new cell's first digest is next Monday's.
func (j Job) weeklyDue(ctx context.Context, now time.Time) (time.Time, bool, error) {
	week := Monday(now, j.Zone)
	last, found, err := j.Clock.Last(ctx, "weekly")
	if err != nil {
		return week, false, fmt.Errorf("notify: last week: %w", err)
	}
	if !found {
		return week, false, j.Clock.Mark(ctx, "weekly", week)
	}
	return week, last.Before(week), nil
}

// weekly plans the digest and the watched questions' messages; the marks save the new
// hashes once the run has sent.
func (j Job) weekly(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	var out []outgoing
	if j.Digest && j.Askers != nil && j.Changes != nil {
		ids, err := j.Askers(ctx, now.Add(-DigestAskers), now)
		if err != nil {
			return nil, nil, fmt.Errorf("notify: askers: %w", err)
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			p, ok, err := j.person(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			if !ok {
				continue
			}
			titles, more, err := j.Changes(ctx, id)
			if err != nil || len(titles) == 0 {
				continue // nothing they can open changed, or it can't be read as them now
			}
			out = append(out, outgoing{p, Message{Kind: KindDigest, Text: digestText(titles, more), Link: j.tab("documents")}})
		}
		lines, people, err := j.stewardLines(ctx, now)
		if err != nil {
			return nil, nil, err
		}
		for i := range out {
			if ls := lines[out[i].to.ID]; len(ls) > 0 {
				out[i].msg.Text += "\n" + strings.Join(ls, "\n")
				delete(lines, out[i].to.ID)
			}
		}
		for _, p := range people {
			if ls := lines[p.ID]; len(ls) > 0 {
				out = append(out, outgoing{p, Message{Kind: KindDigest, Text: strings.Join(ls, "\n"), Link: j.link()}})
			}
		}
	}
	owned, err := j.owned(ctx, now)
	if err != nil {
		return nil, nil, err
	}
	out = append(out, owned...)
	if j.Watches == nil || j.Sources == nil {
		return out, nil, nil
	}
	qs, err := j.Watches.Watched(ctx, j.Tenant)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: watched: %w", err)
	}
	changed := map[string]int{}
	var order []string
	var marks []func() error
	for _, q := range qs {
		if _, ok, err := j.person(ctx, q.User); err != nil || !ok {
			continue
		}
		h, err := j.Sources(ctx, q.User, q.Space, q.Question)
		if err != nil || h == q.Sources {
			continue
		}
		if changed[q.User] == 0 {
			order = append(order, q.User)
		}
		changed[q.User]++
		user, id := q.User, q.ID
		marks = append(marks, func() error {
			_, err := j.Watches.UpdateSaved(ctx, j.Tenant, user, id, func(sq *ask.SavedQuestion) { sq.Sources = h })
			return err
		})
	}
	for _, u := range order {
		p, _, _ := j.person(ctx, u)
		text := "The sources of 1 of your saved questions changed"
		if n := changed[u]; n != 1 {
			text = fmt.Sprintf("The sources of %d of your saved questions changed", n)
		}
		out = append(out, outgoing{p, Message{Kind: KindWatch, Text: text, Link: j.tab("saved")}})
	}
	return out, marks, nil
}

// FirstMonday reports whether the week starting at Monday 08:00 in zone is the month's first.
func FirstMonday(now time.Time, zone *time.Location) bool {
	if zone == nil {
		zone = time.UTC
	}
	return Monday(now, zone).In(zone).Day() <= 7
}

// owned tells each owner, on the month's first weekly run, how many of their often-cited
// documents are past review (SB-2806): a count, to people still active, once a month.
func (j Job) owned(ctx context.Context, now time.Time) ([]outgoing, error) {
	if j.Owned == nil || !FirstMonday(now, j.Zone) {
		return nil, nil
	}
	by, err := j.Owned(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("notify: owned documents: %w", err)
	}
	helped := map[string]int{}
	if j.Helped != nil {
		if helped, err = j.Helped(ctx, now); err != nil {
			return nil, fmt.Errorf("notify: helpful answers: %w", err)
		}
	}
	seen := map[string]bool{}
	var ids []string
	for _, m := range []map[string]int{by, helped} {
		for id, n := range m {
			if n > 0 && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	var out []outgoing
	for _, id := range ids {
		p, ok, err := j.person(ctx, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		var lines []string
		switch n := by[id]; {
		case n == 1:
			lines = append(lines, "1 of your documents that answers often cite is past review")
		case n > 1:
			lines = append(lines, fmt.Sprintf("%d of your documents that answers often cite are past review", n))
		}
		if n := helped[id]; n > 0 {
			lines = append(lines, "Your documents were cited in "+plural(n, "answer", "answers")+" people found helpful last month")
		}
		out = append(out, outgoing{p, Message{Kind: KindOwned, Text: strings.Join(lines, "\n"), Link: j.tab("documents")}})
	}
	return out, nil
}

// stewardLines are the digest's lines for stewards (SB-2507): per space with questions,
// answered this week, still open and the oldest open one's age, counts only, to each active
// steward of the space now. people are the stewards, in order, each once.
func (j Job) stewardLines(ctx context.Context, now time.Time) (map[string][]string, []Person, error) {
	back := j.Lookback
	if back <= 0 {
		back = 400 * 24 * time.Hour
	}
	qs, err := j.Questions.StewardQuestionsBetween(ctx, j.Tenant, now.Add(-back), now)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: questions: %w", err)
	}
	type tally struct {
		answered, open, nothing, overdue int
		oldest                           time.Time
	}
	by := map[string]*tally{}
	week := now.Add(-7 * 24 * time.Hour)
	if j.FoundNothing != nil { // SB-3101: counts only, never a question
		found, err := j.FoundNothing(ctx, week, now)
		if err != nil {
			return nil, nil, fmt.Errorf("notify: traces: %w", err)
		}
		for sp, n := range found {
			if n > 0 {
				by[sp] = &tally{nothing: n}
			}
		}
	}
	for _, q := range qs {
		t := by[q.Space]
		if t == nil {
			t = &tally{}
		}
		switch {
		case q.Reply == nil:
			t.open++
			if ask.WorkingDays(q.At, now, j.Zone) >= ask.OverdueAfter {
				t.overdue++
			}
			if t.oldest.IsZero() || q.At.Before(t.oldest) {
				t.oldest = q.At
			}
		case !q.Reply.At.Before(week):
			t.answered++
		default:
			continue
		}
		by[q.Space] = t
	}
	spaces := make([]string, 0, len(by))
	for sp := range by {
		spaces = append(spaces, sp)
	}
	sort.SliceStable(spaces, func(a, b int) bool { // spaces with overdue questions first (SB-3404)
		if (by[spaces[a]].overdue > 0) != (by[spaces[b]].overdue > 0) {
			return by[spaces[a]].overdue > 0
		}
		return spaces[a] < spaces[b]
	})
	lines := map[string][]string{}
	var people []Person
	seen := map[string]bool{}
	for _, sp := range spaces {
		groups, _, ok := j.Facts(sp)
		if !ok {
			continue
		}
		t := by[sp]
		name := strings.TrimPrefix(sp, "spc_")
		if sp == "company" {
			name = "Company-wide"
		}
		line := fmt.Sprintf("%s: %d answered this week, %d still open", name, t.answered, t.open)
		if t.overdue > 0 {
			line = fmt.Sprintf("%s: %d overdue, %d answered this week, %d still open", name, t.overdue, t.answered, t.open)
		}
		if t.open > 0 {
			line += ", the oldest " + plural(int(now.Sub(t.oldest).Hours()/24), "day", "days") + " old"
		}
		if t.nothing > 0 {
			line += ", " + plural(t.nothing, "question", "questions") + " found nothing"
		}
		ps, err := j.members(ctx, groups)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range ps {
			lines[p.ID] = append(lines[p.ID], line)
			if !seen[p.ID] {
				seen[p.ID] = true
				people = append(people, p)
			}
		}
	}
	return lines, people, nil
}

func (j Job) tab(name string) string {
	return strings.TrimSuffix(j.Base, "/") + "/?tab=" + name
}

func digestText(titles []string, more int) string {
	if len(titles) > MaxDigest {
		more += len(titles) - MaxDigest
		titles = titles[:MaxDigest]
	}
	n := len(titles) + more
	text := plural(n, "document", "documents") + " changed in your spaces this week: " + strings.Join(titles, "; ")
	if more > 0 {
		text += fmt.Sprintf(" and %d more", more)
	}
	return text
}
