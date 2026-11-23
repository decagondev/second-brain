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
	"example.com/second-brain/internal/core/ports"
)

// Kinds a person can turn off.
const (
	KindQuestions = "questions" // new questions, to stewards (SB-2101)
	KindReplies   = "replies"   // replies and notes, to askers (SB-2102)
	KindDigest    = "digest"    // the weekly digest (SB-2103)
	KindWatch     = "watch"     // a watched saved question's sources changed (SB-2104)
	KindAccess    = "access"    // someone asked for access to a document (SB-2205)
)

// Kinds are every kind, for validating a person's settings.
var Kinds = map[string]bool{KindQuestions: true, KindReplies: true, KindDigest: true, KindWatch: true, KindAccess: true}

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

// Prefs are a person's settings: the kinds they turned off.
type Prefs struct {
	Off []string `json:"off"`
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
	out := Prefs{Off: []string{}}
	for _, k := range p.Off {
		if !Kinds[k] {
			return Prefs{}, fmt.Errorf("%w: %q (one of questions, replies, digest, watch, access)", ErrKind, k)
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

// Clock keeps when the job last read everything.
type Clock interface {
	Last(ctx context.Context) (t time.Time, found bool, err error)
	Mark(ctx context.Context, t time.Time) error
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
	Prefs     PrefStore
	Send      Sender // nil sends nothing
	Clock     Clock
	Now       func() time.Time
	// Lookback is how far back a question can be and still have its reply told; the
	// tenant's retention, since older questions are gone.
	Lookback time.Duration
	// Failed counts a send that failed, by kind; nil counts nothing.
	Failed func(kind string)
}

// FirstLookback is how far back the job reads on its first run.
const FirstLookback = 10 * time.Minute

// Result is what one run did.
type Result struct {
	Sent, Skipped, Failed int
}

// ErrSend is a run in which some sends failed; the events are not sent again.
var ErrSend = errors.New("notify: some messages couldn't be sent")

// Run tells what happened in [last run, now).
func (j Job) Run(ctx context.Context) (Result, error) {
	now := j.Now().UTC()
	from, found, err := j.Clock.Last(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("notify: last run: %w", err)
	}
	if !found || from.After(now) {
		from = now.Add(-FirstLookback)
	}
	var out []outgoing
	if j.Send != nil {
		if out, err = j.plan(ctx, from, now); err != nil {
			return Result{}, err
		}
	}
	var res Result
	for _, o := range out {
		if !j.wants(ctx, o.to.ID, o.msg.Kind) {
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
	if err := j.Clock.Mark(ctx, now); err != nil {
		return res, fmt.Errorf("notify: mark: %w", err)
	}
	if res.Failed > 0 {
		return res, fmt.Errorf("%w (%d)", ErrSend, res.Failed)
	}
	return res, nil
}

// wants reads the person's settings; settings that can't be read send nothing.
func (j Job) wants(ctx context.Context, user, kind string) bool {
	if j.Prefs == nil {
		return true
	}
	p, err := j.Prefs.NotifyPrefs(ctx, j.Tenant, user)
	return err == nil && p.Wants(kind)
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
	return append(out, more...), err
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func (j Job) link() string {
	return strings.TrimSuffix(j.Base, "/") + "/?tab=questions"
}

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
