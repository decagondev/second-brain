package ask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/prompts"
	"example.com/second-brain/internal/core/tenancy"
)

// Research reports (sb-ask(3), Research reports; SB-4201): the research agent plans at most
// MaxSteps sub-questions from the question alone; nothing runs until the person presses
// Run; the notify job then asks each one as them and keeps a cited report, which is checked
// again as its reader whenever it is shown (LAW-26).

// The research limits.
const (
	MaxSteps          = 10
	MaxStep           = 300 // characters of a sub-question, and of a research question
	ResearchKeep      = 30 * 24 * time.Hour
	ResearchPerRun    = 3 // reports the notify job runs in one run
	DefaultResearchUS = 2.0
)

// Kinds and states of a stored result.
const (
	KindResearch = "research"
	KindPrep     = "prep"

	StatePlanned = "planned"
	StateQueued  = "queued"
	StateRunning = "running"
	StateDone    = "done"
	StateFailed  = "failed"
)

// Research is a research report or a meeting brief as the cell keeps it, for its owner.
type Research struct {
	ID        string    `json:"id"`
	Tenant    string    `json:"tenant"`
	User      string    `json:"user"`
	Kind      string    `json:"kind"`
	Question  string    `json:"question"` // a report's question, a brief's title
	Space     string    `json:"space"`    // spc_NAME or company
	Documents []string  `json:"documents,omitempty"`
	Steps     []string  `json:"steps"`
	State     string    `json:"state"`
	Why       string    `json:"why,omitempty"`
	Stopped   string    `json:"stopped,omitempty"` // cap or budget
	CostUSD   float64   `json:"cost_usd"`
	At        time.Time `json:"at"`
	DoneAt    time.Time `json:"done_at,omitempty"`
	Sections  []Section `json:"sections,omitempty"`
	// Attendees are a brief's attendees (SB-4202).
	Attendees []Attendee `json:"attendees,omitempty"`
	// Changed is how many of a routine's sources are new since its last run (SB-4302).
	Changed *int `json:"changed,omitempty"`
}

// Section is one step's answer: its question, trace and cited claims, and for a brief the
// decisions behind the item and, for a steward, how many open questions share its words.
type Section struct {
	Question  string         `json:"question"`
	Trace     string         `json:"trace"`
	Claims    []answer.Claim `json:"claims"`
	Decisions []answer.Cite  `json:"decisions,omitempty"`
	Questions *int           `json:"questions,omitempty"`
}

// Attendee is one attendee of a brief: known is one active user now, and owns the cited
// documents whose owner at the source they are.
type Attendee struct {
	Email string        `json:"email"`
	Known bool          `json:"known"`
	Owns  []answer.Cite `json:"owns,omitempty"`
}

// ResearchStore keeps reports and briefs.
type ResearchStore interface {
	PutResearch(ctx context.Context, r Research) error
	GetResearch(ctx context.Context, tenant, id string) (Research, bool, error)
	ResearchOf(ctx context.Context, tenant, user string, since time.Time) ([]Research, error)
	QueuedResearch(ctx context.Context, tenant string) ([]Research, error)
}

// Legwork is the legwork agents' service for one cell (SB-4201 to SB-4203).
type Legwork struct {
	Agent    Agent
	Stewards Stewards
	Research ResearchStore
	Actions  ActionStore
	Audit    Auditor
	NewID    func(prefix string) string
	Now      func() time.Time
	// ResearchUSD is the tenant's research_usd; nil or 0 is DefaultResearchUS.
	ResearchUSD func() float64
	// ScopeIn is an active user as a scope in a space (spc_NAME or company) now; an error is
	// someone who isn't active.
	ScopeIn func(ctx context.Context, user, space string) (tenancy.Scope, error)
	// Calendar reads a person's own calendar, as them (SB-4202); nil is a cell without one.
	Calendar ports.Calendar
	// Known is whether an email is one active user now (LAW-12); nil knows nobody.
	Known func(ctx context.Context, email string) bool
	// Users are the tenant's active users now, to match an action item's owner (SB-4203).
	Users func(ctx context.Context) ([]ports.User, error)
	// Tell sends one person one notification of kind, as the notify job would; nil tells
	// no one (SB-4206).
	Tell func(ctx context.Context, user, kind, text string) error
	// Routines are the saved questions asked on a schedule, and Zone their time zone
	// (SB-4302); nil runs none, and nil Zone is UTC.
	Routines RoutineStore
	Zone     *time.Location
	// Approvals are the actions waiting for the person's approval group, as they may see
	// them now, for the inbox (SB-4301); nil lists none.
	Approvals func(ctx context.Context, s tenancy.Scope) ([]InboxItem, error)
	// CatchUpOffer is, once after a person's notification pause ended, the day it began, and
	// marks it offered (SB-4506); nil offers none.
	CatchUpOffer func(ctx context.Context, user string) (since time.Time, ok bool, err error)
}

// ResearchItem is one report or brief as its owner's list shows it.
type ResearchItem struct {
	ID       string     `json:"id"`
	Kind     string     `json:"kind"`
	Question string     `json:"question"`
	Space    string     `json:"space"`
	State    string     `json:"state"`
	Steps    int        `json:"steps"`
	At       time.Time  `json:"at"`
	DoneAt   *time.Time `json:"done_at,omitempty"`
}

// ResearchView is a whole report as its owner sees it now (LAW-26).
type ResearchView struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Question  string         `json:"question"`
	Space     string         `json:"space"`
	State     string         `json:"state"`
	Why       string         `json:"why,omitempty"`
	Steps     []string       `json:"steps"`
	At        time.Time      `json:"at"`
	DoneAt    *time.Time     `json:"done_at,omitempty"`
	CostUSD   float64        `json:"cost_usd"`
	Stopped   string         `json:"stopped,omitempty"`
	Summary   []answer.Claim `json:"summary"`
	Sections  []Section      `json:"sections"`
	Attendees []Attendee     `json:"attendees,omitempty"`
	Dropped   int            `json:"dropped"`
	Changed   *int           `json:"changed,omitempty"`
}

func (l Legwork) now() time.Time { return l.Now().UTC() }

func (l Legwork) researchUSD() float64 {
	if l.ResearchUSD != nil {
		if v := l.ResearchUSD(); v > 0 {
			return v
		}
	}
	return DefaultResearchUS
}

func (l Legwork) check(s tenancy.Scope) error {
	if err := s.Check(); err != nil || s.Tenant() != l.Agent.Tenant {
		return fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if l.Research == nil {
		return fmt.Errorf("%w: this cell keeps no reports", ErrUnavailable)
	}
	return nil
}

func cut(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > n {
		s = string([]rune(s)[:n])
	}
	return s
}

// Plan asks the research agent for a plan of q, in s's space, and keeps it, unrun.
func (l Legwork) Plan(ctx context.Context, s tenancy.Scope, q string, docs []string) (ResearchView, error) {
	if err := l.check(s); err != nil {
		return ResearchView{}, err
	}
	q = strings.TrimSpace(q)
	if q == "" || utf8.RuneCountInString(q) > MaxStep {
		return ResearchView{}, fmt.Errorf("%w: a research question is 1 to %d characters", ErrInvalid, MaxStep)
	}
	keys, err := documentKeys(docs)
	if err != nil {
		return ResearchView{}, err
	}
	if l.Agent.Killed == nil || l.Agent.Killed(ctx) || spaceKilled(l.Agent.SpaceKilled, s) {
		return ResearchView{}, ErrKilled
	}
	if _, err := l.Agent.legworkDept(ctx, s, AgentResearch); err != nil {
		return ResearchView{}, err
	}
	_, plain, _, err := l.Agent.loadSkill(ctx, s, q) // SB-4601: planned from the question alone
	if err != nil {
		return ResearchView{}, err
	}
	steps, err := l.steps(ctx, s, plain)
	if err != nil {
		return ResearchView{}, err
	}
	r := Research{ID: l.NewID("rs_"), Tenant: l.Agent.Tenant, User: s.User(), Kind: KindResearch, Question: q, Space: s.Space(),
		Documents: keys, Steps: steps, State: StatePlanned, At: l.now()}
	if err := l.audit(ctx, s.User(), r, StatePlanned, hashOf(q)); err != nil {
		return ResearchView{}, err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return ResearchView{}, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	return view(r, nil, 0), nil
}

// steps is the model's plan for q: at most MaxSteps lines, each cut to MaxStep; no usable
// line plans q itself. The model sees the question only, never a document.
func (l Legwork) steps(ctx context.Context, s tenancy.Scope, q string) ([]string, error) {
	p, err := prompts.Get("research")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	out, err := l.Agent.Model.Complete(ctx, ports.ModelRequest{System: p.Text, JSON: true, User: s.User(), Space: s.Space(),
		Messages: []ports.Message{{Role: "user", Content: "Question: " + q}}})
	if err != nil {
		if errors.Is(err, budget.ErrBudget) {
			return nil, fmt.Errorf("%w: %v", ErrBudget, err)
		}
		return nil, fmt.Errorf("%w: model: %v", ErrUnavailable, err)
	}
	return planSteps(out.Text, q), nil
}

// planSteps reads a plan: {"steps": [...]}, at most MaxSteps non-empty lines, each cut.
func planSteps(text, q string) []string {
	var v struct {
		Steps []string `json:"steps"`
	}
	var steps []string
	if json.Unmarshal([]byte(strings.TrimSpace(text)), &v) == nil {
		seen := map[string]bool{}
		for _, st := range v.Steps {
			st = cut(st, MaxStep)
			if st == "" || seen[strings.ToLower(st)] {
				continue
			}
			seen[strings.ToLower(st)] = true
			steps = append(steps, st)
			if len(steps) == MaxSteps {
				break
			}
		}
	}
	if len(steps) == 0 {
		steps = []string{q}
	}
	return steps
}

func (l Legwork) audit(ctx context.Context, actor string, r Research, state, sum string) error {
	if l.Audit == nil {
		return fmt.Errorf("%w: no audit log", ErrUnavailable)
	}
	if _, err := l.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "access", Subject: r.ID, State: state, PayloadSHA256: sum,
		Reason: r.Kind + " " + state, Space: auditSpace(r.Space)}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

// own is the caller's own report, younger than ResearchKeep; anything else is ErrNotFound.
func (l Legwork) own(ctx context.Context, s tenancy.Scope, id string) (Research, error) {
	if err := l.check(s); err != nil {
		return Research{}, err
	}
	r, found, err := l.Research.GetResearch(ctx, l.Agent.Tenant, id)
	if err != nil {
		return Research{}, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	if !found || r.User != s.User() || l.now().Sub(r.At) > ResearchKeep {
		return Research{}, ErrNotFound
	}
	return r, nil
}

// Run queues the caller's planned report for the notify job.
func (l Legwork) Run(ctx context.Context, s tenancy.Scope, id string) (ResearchView, error) {
	r, err := l.own(ctx, s, id)
	if err != nil {
		return ResearchView{}, err
	}
	if r.Kind != KindResearch || r.State != StatePlanned {
		return ResearchView{}, fmt.Errorf("%w: only a planned report runs", ErrInvalid)
	}
	r.State = StateQueued
	if err := l.audit(ctx, s.User(), r, StateQueued, hashOf(r.Question)); err != nil {
		return ResearchView{}, err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return ResearchView{}, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	return view(r, nil, 0), nil
}

// List is the caller's reports and briefs of the last ResearchKeep, newest first.
func (l Legwork) List(ctx context.Context, s tenancy.Scope) ([]ResearchItem, error) {
	if err := l.check(s); err != nil {
		return nil, err
	}
	rs, err := l.Research.ResearchOf(ctx, l.Agent.Tenant, s.User(), l.now().Add(-ResearchKeep))
	if err != nil {
		return nil, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].At.After(rs[j].At) })
	out := []ResearchItem{}
	for _, r := range rs {
		it := ResearchItem{ID: r.ID, Kind: r.Kind, Question: r.Question, Space: r.Space, State: l.fixState(ctx, r), Steps: len(r.Steps), At: r.At}
		if !r.DoneAt.IsZero() {
			d := r.DoneAt
			it.DoneAt = &d
		}
		out = append(out, it)
	}
	return out, nil
}

// Open is the caller's report as they may see it now: every cite checked again as them, in
// the report's space (LAW-26).
func (l Legwork) Open(ctx context.Context, s tenancy.Scope, id string) (ResearchView, error) {
	r, err := l.own(ctx, s, id)
	if err != nil {
		return ResearchView{}, err
	}
	r.State = l.fixState(ctx, r) // SB-4206: a suggestion is applied once its source changes
	return l.recheck(ctx, s, r)
}

func (l Legwork) recheck(ctx context.Context, s tenancy.Scope, r Research) (ResearchView, error) {
	as := s
	if l.ScopeIn != nil && r.Space != s.Space() {
		in, err := l.ScopeIn(ctx, s.User(), r.Space)
		if err != nil {
			return ResearchView{}, ErrNotFound //nolint:nilerr // not active now: nothing to show
		}
		as = in
	}
	var secs []Section
	dropped := 0
	for _, sec := range r.Sections {
		claims, d, err := l.Agent.Recheck(ctx, as, sec.Claims)
		if err != nil {
			return ResearchView{}, err
		}
		dropped += d
		sec.Claims = claims
		if sec.Decisions, err = l.keepCites(ctx, as, sec.Decisions); err != nil {
			return ResearchView{}, err
		}
		secs = append(secs, sec)
	}
	var atts []Attendee
	for _, a := range r.Attendees {
		owns, err := l.keepCites(ctx, as, a.Owns)
		if err != nil {
			return ResearchView{}, err
		}
		a.Owns = owns
		atts = append(atts, a)
	}
	r.Sections, r.Attendees = secs, atts
	return view(r, secs, dropped), nil
}

// keepCites is the cites s can open now.
func (l Legwork) keepCites(ctx context.Context, s tenancy.Scope, cs []answer.Cite) ([]answer.Cite, error) {
	if len(cs) == 0 {
		return cs, nil
	}
	var ids []string
	for _, c := range cs {
		ids = append(ids, c.Cite)
	}
	open, err := l.Agent.opensCites(ctx, s, ids)
	if err != nil {
		return nil, err
	}
	var out []answer.Cite
	for _, c := range cs {
		if open[c.Cite] {
			out = append(out, c)
		}
	}
	return out, nil
}

func view(r Research, secs []Section, dropped int) ResearchView {
	v := ResearchView{ID: r.ID, Kind: r.Kind, Question: r.Question, Space: r.Space, State: r.State, Why: r.Why, Steps: r.Steps, At: r.At,
		CostUSD: r.CostUSD, Stopped: r.Stopped, Summary: []answer.Claim{}, Sections: []Section{}, Attendees: r.Attendees, Dropped: dropped, Changed: r.Changed}
	if v.Steps == nil {
		v.Steps = []string{}
	}
	if !r.DoneAt.IsZero() {
		d := r.DoneAt
		v.DoneAt = &d
	}
	for _, sec := range secs {
		v.Sections = append(v.Sections, sec)
		for _, c := range sec.Claims {
			if !c.Uncited && len(c.Cites) > 0 {
				v.Summary = append(v.Summary, c)
				break
			}
		}
	}
	return v
}

// Finished is one report the notify job finished, to tell its owner about.
type Finished struct {
	ID, User, State string
	// Changed is how many of a routine's sources are new since its last run (SB-4302).
	Changed int
}

// RunQueued runs up to ResearchPerRun queued reports, oldest first, each as its owner.
func (l Legwork) RunQueued(ctx context.Context) ([]Finished, error) {
	if l.Research == nil || l.ScopeIn == nil {
		return nil, nil
	}
	qs, err := l.Research.QueuedResearch(ctx, l.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	sort.SliceStable(qs, func(i, j int) bool { return qs[i].At.Before(qs[j].At) })
	if len(qs) > ResearchPerRun {
		qs = qs[:ResearchPerRun]
	}
	var out []Finished
	for _, r := range qs {
		r, err := l.runOne(ctx, r)
		if err != nil {
			return out, err
		}
		out = append(out, Finished{ID: r.ID, User: r.User, State: r.State})
	}
	return out, nil
}

func (l Legwork) runOne(ctx context.Context, r Research) (Research, error) {
	r.State = StateRunning
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return r, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	s, err := l.ScopeIn(ctx, r.User, r.Space)
	if err != nil {
		r.State, r.Why = StateFailed, "its owner is no longer active"
		return r, l.finish(ctx, r)
	}
	tctx, tally := budget.WithTally(ctx)
	limit := l.researchUSD()
	skill := ""
	if r.Kind == KindResearch {
		skill, _ = SkillName(r.Question) // SB-4601: each step is asked with the report's skill
	}
	for _, step := range r.Steps {
		if tally.USD() >= limit {
			r.Stopped = "cap"
			break
		}
		asked := step
		if skill != "" {
			asked = "/" + skill + " " + step
		}
		resp, err := l.Agent.Ask(tctx, s, Request{Question: asked, Agent: AgentResearch, Documents: r.Documents})
		switch {
		case errors.Is(err, ErrBudget):
			r.Stopped = "budget"
		case errors.Is(err, ErrKilled):
			r.State, r.Why = StateFailed, "the research agent is turned off"
		case err != nil:
			r.State, r.Why = StateFailed, "a step couldn't be answered"
		}
		if err != nil {
			break
		}
		if claims, _ := citedOnly(resp.Answer.Claims); !claims[0].Uncited {
			r.Sections = append(r.Sections, Section{Question: step, Trace: resp.Trace, Claims: claims})
		}
	}
	r.CostUSD = tally.USD()
	if r.State == StateRunning {
		r.State = StateDone
	}
	return r, l.finish(ctx, r)
}

func (l Legwork) finish(ctx context.Context, r Research) error {
	r.DoneAt = l.now()
	if err := l.audit(ctx, r.User, r, r.State, hashOf(r.Question)); err != nil {
		return err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	return nil
}
