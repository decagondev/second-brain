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
	"net/url"
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
	KindWelcome   = "welcome"   // a new member's welcome brief (SB-4006)
	KindPolicy    = "policy"    // a policy to read, once per version (SB-4103)
	KindLimit     = "limit"     // 80% of the personal limit, once a month (SB-5003)
	KindSuspect   = "suspect"   // an owner's document has a suspect passage, once per version (SB-5101)
	KindWidened   = "widened"   // a document's sharing widened at its source, once per change (SB-5403)
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
var Kinds = map[string]bool{KindQuestions: true, KindReplies: true, KindDigest: true, KindWatch: true, KindAccess: true, KindTokens: true, KindOwned: true, KindWelcome: true, KindPolicy: true, KindLimit: true, KindSuspect: true, KindWidened: true}

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
	// Agent and Space are the agent run a message is about, and Pause the link that pauses
	// that agent there, set only for someone allowed to (SB-5107).
	Agent, Space string
	Pause        string
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
	// Digest is "daily" for the daily brief (SB-3805); "" is the weekly default.
	Digest string `json:"digest,omitempty"`
	// Prep makes a brief before each meeting with an agenda (SB-4202).
	Prep bool `json:"prep,omitempty"`
	// PausedAt is when the last pause began, and CatchUpOffered whether the inbox has
	// offered Catch me up since it ended (SB-4506).
	PausedAt       *time.Time `json:"paused_at,omitempty"`
	CatchUpOffered bool       `json:"catch_up_offered,omitempty"`
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
		u, at := now.UTC().Add(time.Duration(days)*24*time.Hour), now.UTC()
		p.PausedUntil, p.PausedAt, p.CatchUpOffered = &u, &at, false
	}
	return p, nil
}

// CatchUpDue reports whether a pause has ended at now and the inbox hasn't offered Catch me
// up since (SB-4506), with when the pause began.
func (p Prefs) CatchUpDue(now time.Time) (time.Time, bool) {
	if p.PausedAt == nil || p.CatchUpOffered || p.Paused(now) || now.Sub(*p.PausedAt) > 90*24*time.Hour {
		return time.Time{}, false
	}
	return *p.PausedAt, true
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
	out := Prefs{Off: []string{}, PausedUntil: p.PausedUntil, Prep: p.Prep, PausedAt: p.PausedAt, CatchUpOffered: p.CatchUpOffered}
	switch p.Digest {
	case "", DigestWeekly:
	case DigestDaily:
		out.Digest = DigestDaily
	default:
		return Prefs{}, ErrDigest
	}
	for _, k := range p.Off {
		if !Kinds[k] {
			return Prefs{}, fmt.Errorf("%w: %q (one of questions, replies, digest, watch, access, tokens, owned, welcome, policy, research, prep, action, incident, fix, checkin, agent, routine, limit, check, adminweek)", ErrKind, k)
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
	// Parent is the space key of a space's department, "" for none (SB-3904); nil is a
	// cell without a tree, where a question goes straight to company.
	Parent func(space string) string
	Dir    Directory
	// FoundNothing counts, per space, the asks in a window that found nothing, for the
	// stewards' line (SB-3101); nil adds nothing.
	FoundNothing func(ctx context.Context, from, to time.Time) (map[string]int, error)
	// Trends are, per space, the terms asked most in a window, each with at least 5 asks
	// (SB-3704); nil adds none.
	Trends func(ctx context.Context, from, to time.Time) (map[string][]ask.TermCount, error)
	Prefs  PrefStore
	Send   Sender // nil sends nothing
	Clock  Clock
	Now    func() time.Time
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
	// Changes are the titles of the documents that changed in the last days in user's
	// spaces, as user, now (GET /v1/changes), at most MaxDigest, and how many more.
	Changes func(ctx context.Context, user string, days int) (titles []string, more int, err error)
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
	// Briefs are the spaces' scheduled agents' days, by space key, and RunBrief runs one,
	// returning why it didn't ("" when it did) (SB-4005); nil runs none.
	Briefs   map[string]time.Weekday
	RunBrief func(ctx context.Context, space string) (why string, err error)
	// Updates are the spaces with updates: true, by key, and RunUpdate drafts one's weekly
	// status update each Monday (SB-4102); nil drafts none.
	Updates   []string
	RunUpdate func(ctx context.Context, space string) (why string, err error)
	// Leaders are the spaces that get the leader's week, by key, and RunLeader drafts one's
	// each Monday (SB-4503).
	Leaders   []string
	RunLeader func(ctx context.Context, space string) (why string, err error)
	// SpaceBudgets are the spaces with a budget_usd_month and the share of it their branch
	// has spent this month (SB-4105); their stewards are told at WarnShare and at 1.
	SpaceBudgets func(ctx context.Context) ([]SpaceBudget, error)
	// Welcomes are the welcome briefs waiting, and Welcomed marks one sent (SB-4006); nil
	// sends none. Transcripts are the transcripts first seen in a window, and RunDecisions
	// drafts one's decisions to its space's stewards; nil drafts none.
	Welcomes     func(ctx context.Context) ([]Welcome, error)
	Welcomed     func(ctx context.Context, w Welcome) error
	Transcripts  func(ctx context.Context, from, to time.Time) ([]NewDoc, error)
	RunDecisions func(ctx context.Context, d NewDoc) (why string, err error)
	// PolicyNotices are the policies' current versions and who to tell (SB-4103); each is
	// told once. nil tells no one.
	PolicyNotices func(ctx context.Context) ([]ask.PolicyNotice, error)
	// SourceChanged notes the documents with a new version in [from, now) on the agent
	// drafts that drew on them, and makes the day's redrafts (SB-4905); nil does neither.
	SourceChanged func(ctx context.Context, from, now time.Time) error
	// NearLimit are the people at 80% or more of their personal limit this month (SB-5003),
	// each told once a month; nil tells no one.
	NearLimit func(ctx context.Context, now time.Time) ([]string, error)
	// WorkEvents are the work events first seen in a window, and RunWork drafts one or a
	// batch of one space and kind as its trigger's approver (SB-5301); nil drafts none.
	WorkEvents func(ctx context.Context, from, to time.Time) ([]NewDoc, error)
	// Mail drafts a reply note for each new message in the shared mailboxes spaces name
	// (SB-5907); nil reads none.
	Mail    func(ctx context.Context) error
	RunWork func(ctx context.Context, docs []NewDoc) (why string, err error)
	// MayPause says a person may pause an agent in a space and has the pause link (SB-5107);
	// nil adds the link for no one.
	MayPause func(ctx context.Context, user, space, agent string) bool
	// Suspects are the documents with a suspect passage, each with its owner's user id
	// (SB-5101); each version is told once. nil tells no one.
	Suspects func(ctx context.Context) ([]SuspectDoc, error)
	// Notices are one-off notices worked out elsewhere (SB-5403 and later), each sent once
	// by its key; nil sends none.
	Notices func(ctx context.Context, now time.Time) ([]Notice, error)
	// Research runs the queued research reports and returns those it finished (SB-4201);
	// Actions are the action items confirmed since a time (SB-4203); PrepDue makes a
	// person's briefs of the events starting in a window (SB-4202). nil does none.
	Research func(ctx context.Context) ([]ask.Finished, error)
	Actions  func(ctx context.Context, since time.Time) ([]ask.Action, error)
	PrepDue  func(ctx context.Context, user string, from, to time.Time, done func(event string) bool) ([]ask.PrepMade, error)
	// Incidents are the incident tickets first seen in a window, by space, and RunIncident
	// drafts one's brief (SB-4204); Checkins drafts the week's goal check-ins and returns
	// the goals gone three weeks without one (SB-4207). nil does none.
	Incidents   func(ctx context.Context, from, to time.Time) ([]NewDoc, error)
	RunIncident func(ctx context.Context, d NewDoc) (why string, err error)
	// RunReview drafts a closed incident's review (SB-4604); nil drafts none.
	RunReview func(ctx context.Context, d NewDoc) (why string, err error)
	Checkins  func(ctx context.Context) ([]SilentGoal, error)
	// Routines asks the saved questions due as routines, as their owners, and returns those
	// it ran (SB-4302); nil asks none.
	Routines func(ctx context.Context) ([]ask.Finished, error)
	// Waits are the waits a working day past their date (SB-4504); nil tells nobody.
	Waits func(ctx context.Context) ([]ask.OverdueWait, error)
	// AgentAlerts are what to tell agents' owners now (SB-4701, SB-4703), each once by
	// its key; nil tells none.
	AgentAlerts func(ctx context.Context, now time.Time) ([]AgentAlert, error)
	// SweepShadows deletes the shadow runs of every window that has ended (SB-4704).
	SweepShadows func(ctx context.Context) (int, error)
	// SweepBatches cancels the batches past their morning deadline that no step came back
	// for (SB-5701).
	SweepBatches func(ctx context.Context) error
	// Rhythms starts onboarding for new members, stops rhythms whose subject left and
	// offers the steps whose day has come (SB-4801); nil runs none.
	Rhythms func(ctx context.Context) (int, error)
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

// agentWork is the agents' alerts, after the shadow runs whose window ended are deleted:
// a shadow run never outlives its window (SB-4704).
func (j Job) agentWork(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	if j.SweepShadows != nil {
		if _, err := j.SweepShadows(ctx); err != nil {
			return nil, nil, fmt.Errorf("notify: shadows: %w", err)
		}
	}
	if j.Rhythms != nil {
		if _, err := j.Rhythms(ctx); err != nil {
			return nil, nil, fmt.Errorf("notify: rhythms: %w", err)
		}
	}
	return j.agentAlerts(ctx, now)
}

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
	briefs, err := j.briefs(ctx, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, briefs...)
	spend, err := j.budgets(ctx, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, spend...)
	trig, trigMarks, err := j.triggers(ctx, from, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, trig...)
	leg, legMarks, err := j.legwork(ctx, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, leg...)
	trigMarks = append(trigMarks, legMarks...)
	inc, err := j.incidents(ctx, from, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, inc...)
	chk, err := j.checkins(ctx, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, chk...)
	rts, err := j.routines(ctx)
	if err != nil {
		return Result{}, err
	}
	out = append(out, rts...)
	wts, wtMarks, err := j.waits(ctx, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, wts...)
	agt, agtMarks, err := j.agentWork(ctx, now)
	if err != nil {
		return Result{}, err
	}
	out = append(out, agt...)
	wtMarks = append(wtMarks, agtMarks...)
	week, due, err := j.weeklyDue(ctx, now)
	if err != nil {
		return Result{}, err
	}
	marks := revMarks
	marks = append(marks, trigMarks...)
	marks = append(marks, wtMarks...)
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
	day, dailyDue, err := j.dailyDue(ctx, now)
	if err != nil {
		return Result{}, err
	}
	if dailyDue {
		more, mark, err := j.daily(ctx, now)
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
		o.msg.Pause = j.pauseLink(ctx, o.to.ID, o.msg) // SB-5107
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
	if dailyDue {
		if err := j.Clock.Mark(ctx, "daily", day); err != nil {
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
	asked = notDrafts(asked) // a steward's own draft tells no one (SB-3705)
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

// reroute climbs the tree (SB-3404, SB-3904): a question open RerouteAfter working days in
// its space, with no live claim, goes to the queue above it, the department's for a team
// and the company-wide answerers' for a department, and RerouteAfter more takes a team's
// on to company. It stays in every queue it reached. A step whose queue has no stewards
// waits. Each step is audited, counts and ids only, and the counts are by queue reached.
func (j Job) reroute(ctx context.Context, now time.Time) (map[string]int, error) {
	if j.Audit == nil || j.Facts == nil {
		return nil, nil
	}
	back := j.Lookback
	if back <= 0 {
		back = 400 * 24 * time.Hour
	}
	qs, err := j.Questions.StewardQuestionsBetween(ctx, j.Tenant, now.Add(-back), now)
	if err != nil {
		return nil, fmt.Errorf("notify: questions: %w", err)
	}
	n := map[string]int{}
	for _, q := range qs {
		to := j.nextUp(q, now)
		if to == "" {
			continue
		}
		if _, err := j.Audit.Write(ctx, audit.Event{Actor: "sys_notify", Kind: "access", Subject: q.ID, State: "rerouted",
			Reason: fmt.Sprintf("open %d working days; to %s", climbAfter(q)*(len(q.Up)+1), to), Space: q.Space}); err != nil {
			return n, fmt.Errorf("notify: audit: %w", err)
		}
		moved := false
		if _, err := j.Questions.UpdateStewardQuestion(ctx, j.Tenant, q.ID, func(q *ask.StewardQuestion) error {
			if moved = j.nextUp(*q, now) == to; moved {
				q.Up, q.Waited = append(q.Up, to), ask.WorkingDays(q.At, now, j.Zone)
				if to == "company" {
					q.Also = "company"
				}
			}
			return nil
		}); err != nil {
			return n, fmt.Errorf("notify: questions: %w", err)
		}
		if moved && q.Request != nil {
			n[requestKey+to]++
		} else if moved {
			n[to]++
		}
	}
	return n, nil
}

// nextUp is the queue a question goes to now, or "" for none: open, unclaimed, past
// RerouteAfter working days for each step it has taken, with stewards in the next queue.
func (j Job) nextUp(q ask.StewardQuestion, now time.Time) string {
	if q.Space == "company" || ask.IsPerson(q.Space) || q.Also == "company" || q.Reply != nil || (q.Claim != nil && now.Before(q.Claim.Until)) {
		return ""
	}
	if ask.WorkingDays(q.At, now, j.Zone) < climbAfter(q)*(len(q.Up)+1) {
		return ""
	}
	to := "company"
	if j.Parent != nil && len(q.Up) == 0 {
		if p := j.Parent(q.Space); p != "" {
			to = p
		}
	}
	if _, _, ok := j.Facts(to); !ok {
		return ""
	}
	return to
}

// climbAfter is how many working days a question waits in each queue before it climbs: a
// request its due days (SB-4104), any other question RerouteAfter.
func climbAfter(q ask.StewardQuestion) int {
	if q.Request != nil && q.Request.DueDays > 0 {
		return q.Request.DueDays
	}
	return ask.RerouteAfter
}

// requestKey prefixes a queue's count of requests that climbed to it, in reroute's counts.
const requestKey = "request:"

// waited tells each queue's answerers how many questions came up to them this run: the
// company-wide answerers, and a department's stewards for its teams' (SB-3904). Counts only.
func (j Job) waited(ctx context.Context, moved map[string]int) ([]outgoing, error) {
	keys := make([]string, 0, len(moved))
	for k := range moved {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []outgoing
	for _, to := range keys {
		n := moved[to]
		groups, _, ok := j.Facts(strings.TrimPrefix(to, requestKey))
		if !ok || n == 0 {
			continue
		}
		people, err := j.members(ctx, groups)
		if err != nil {
			return nil, err
		}
		text := waitedText(to, n)
		for _, p := range people {
			out = append(out, outgoing{p, Message{Kind: KindQuestions, Text: text, Link: j.stewardLink(strings.TrimPrefix(to, requestKey))}})
		}
	}
	return out, nil
}

// waitedText is what a queue's answerers are told about the n questions, or requests, that
// came up to them. Counts only.
func waitedText(to string, n int) string {
	if q, ok := strings.CutPrefix(to, requestKey); ok {
		switch {
		case n == 1 && q == "company":
			return "1 request went past its due date in its space"
		case n == 1:
			return "1 request went past its due date in a team"
		case q == "company":
			return fmt.Sprintf("%d requests went past their due dates in their spaces", n)
		}
		return fmt.Sprintf("%d requests went past their due dates in your teams", n)
	}
	text := "1 question waited 10 working days in its space"
	if n != 1 {
		text = fmt.Sprintf("%d questions waited 10 working days in their spaces", n)
	}
	if to != "company" {
		text = "1 question waited 10 working days in a team"
		if n != 1 {
			text = fmt.Sprintf("%d questions waited 10 working days in your teams", n)
		}
	}
	return text
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
		opened, what := fmt.Sprintf("The %s access review is open: %s, due %s", name, plural(len(r.Items), "person", "people"), due.UTC().Format(time.DateOnly)), "access"
		if r.Kind == reviews.KindAgents { // SB-4705
			unused := 0
			for _, it := range r.Items {
				if it.Unused {
					unused++
				}
			}
			opened = fmt.Sprintf("The %s agent review is open: %s, %d unused, due %s", name, plural(len(r.Items), "agent or trigger", "agents and triggers"), unused, due.UTC().Format(time.DateOnly))
			what = "agent"
		}
		notices := []struct{ part, text string }{{"review-" + r.ID + "-open", opened}}
		if !now.Before(due) {
			notices = append(notices, struct{ part, text string }{"review-" + r.ID + "-overdue", "The " + name + " " + what + " review is overdue"})
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
	if ask.IsPerson(space) {
		space = "me" // the person's own queue (SB-3706)
	}
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
		switch {
		case ask.IsPerson(sp):
			where = "sent to you as a document's owner" // SB-3706
		case sp != "company":
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
		if id, ok := strings.CutPrefix(g, "user:"); ok && !seen[id] { // one person's queue (SB-3706)
			seen[id] = true
			if p, ok, err := j.person(ctx, id); err != nil {
				return nil, err
			} else if ok {
				out = append(out, p)
			}
			continue
		}
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
		if q.Reply == nil || q.Draft != nil {
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
			if j.dailyPerson(ctx, id) {
				continue // told each working morning instead (SB-3805)
			}
			p, ok, err := j.person(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			if !ok {
				continue
			}
			titles, more, err := j.Changes(ctx, id, 7)
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
	watch, marks, err := j.watched(ctx, func(user string) bool { return !j.dailyPerson(ctx, user) })
	if err != nil {
		return nil, nil, err
	}
	return append(out, watch...), marks, nil
}

// watched plans the watch messages of the people for whom ok is true; the marks save the
// new hashes once the run has sent.
func (j Job) watched(ctx context.Context, ok func(user string) bool) ([]outgoing, []func() error, error) {
	if j.Watches == nil || j.Sources == nil {
		return nil, nil, nil
	}
	var out []outgoing
	qs, err := j.Watches.Watched(ctx, j.Tenant)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: watched: %w", err)
	}
	changed := map[string]int{}
	var order []string
	var marks []func() error
	for _, q := range qs {
		if !ok(q.User) {
			continue
		}
		if _, active, err := j.person(ctx, q.User); err != nil || !active {
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
		trends                           []ask.TermCount
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
	if j.Trends != nil { // SB-3704: counts and the space's own terms, never a question
		trends, err := j.Trends(ctx, week, now)
		if err != nil {
			return nil, nil, fmt.Errorf("notify: traces: %w", err)
		}
		for sp, tc := range trends {
			if by[sp] == nil {
				by[sp] = &tally{}
			}
			by[sp].trends = tc
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
		if !ok || ask.IsPerson(sp) {
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
		if len(t.trends) > 0 {
			var parts []string
			for _, tc := range t.trends {
				parts = append(parts, fmt.Sprintf("%s %d", tc.Term, tc.Count))
			}
			line += ", asked most: " + strings.Join(parts, ", ")
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

// pauseLink is the web UI's Pause this agent here for a message about an agent run in a
// space, when to may pause it there and has the feature (SB-5106, SB-5107); "" otherwise.
func (j Job) pauseLink(ctx context.Context, to string, m Message) string {
	if m.Agent == "" || m.Space == "" || j.MayPause == nil || !j.MayPause(ctx, to, m.Space, m.Agent) {
		return ""
	}
	return strings.TrimSuffix(j.Base, "/") + "/?pause=" + url.QueryEscape(m.Agent) + "&space=" + url.QueryEscape(m.Space)
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

// notDrafts leaves out the drafts stewards started themselves.
func notDrafts(qs []ask.StewardQuestion) []ask.StewardQuestion {
	out := qs[:0:0]
	for _, q := range qs {
		if q.Draft == nil {
			out = append(out, q)
		}
	}
	return out
}
