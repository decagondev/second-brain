package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Rhythms (sb-departments(7), Rhythms; SB-4801): a space's named playbooks for a
// recurring moment, started for a person, a collection or the space. Each step is offered
// on its day to one named approver as an item in their queue and waits there: a rhythm
// only drafts, and it stops at once when its subject leaves the space or its starter or a
// steward stops it (LAW-29).

// The states of a rhythm and of its steps.
const (
	RhythmProposed = "proposed" // waiting for its plan's approver
	RhythmRunning  = "running"
	RhythmStopped  = "stopped"
	RhythmDone     = "done"

	StepWaiting = "waiting"
	StepOffered = "offered"
	StepDone    = "done"
	StepSkipped = "skipped"

	// DraftRhythm is the kind of a rhythm step's draft.
	DraftRhythm = "rhythm"
	// RhythmSystem starts the onboarding rhythm when SCIM adds a person (SB-4802).
	RhythmSystem = "system"
)

// RhythmDef is one rhythm as the manifest defines it (sb-space(5), rhythms).
type RhythmDef struct {
	Subject string          `json:"subject"`
	Plan    string          `json:"plan,omitempty"`
	Steps   []RhythmStepDef `json:"steps"`
}

// RhythmStepDef is one dated step of a rhythm.
type RhythmStepDef struct {
	Day      int    `json:"day"`
	Kind     string `json:"kind"`
	Ask      string `json:"ask"`
	Approver string `json:"approver"`
	Agent    string `json:"agent,omitempty"`
}

// RhythmRun is one started rhythm as the cell keeps it: ids only.
type RhythmRun struct {
	ID      string `json:"id"`
	Tenant  string `json:"tenant"`
	Space   string `json:"space"` // a space name
	Rhythm  string `json:"rhythm"`
	Subject string `json:"subject"` // a user id, a collection id or the space name
	Kind    string `json:"subject_kind"`
	Starter string `json:"starter"` // a user id, or RhythmSystem
	// PlanBy is who approves the plan before it starts: a user id, or "steward".
	PlanBy   string          `json:"plan_by,omitempty"`
	PlanItem string          `json:"plan_item,omitempty"`
	State    string          `json:"state"`
	Why      string          `json:"why,omitempty"`
	At       time.Time       `json:"at"`
	Started  time.Time       `json:"started,omitzero"`
	Ended    time.Time       `json:"ended,omitzero"`
	Steps    []RhythmRunStep `json:"steps"`
	// ShareName is the subject's yes to their name going with the gaps the stewards get
	// (SB-4802); without it a gap reaches them unnamed.
	ShareName bool `json:"share_name,omitempty"`
	// Proposed are a quarterly planning run's proposed goals, for the parent's roll-up
	// (SB-4805): titles and cites only.
	Proposed []ProposedGoal `json:"proposed,omitempty"`
}

// RhythmRunStep is one step of a run: its definition, when it is due, and the queue item
// it was offered as.
type RhythmRunStep struct {
	RhythmStepDef
	Due   time.Time `json:"due,omitzero"`
	State string    `json:"state"`
	Queue string    `json:"queue,omitempty"` // PersonKey(user) or a space key
	Item  string    `json:"item,omitempty"`
	Why   string    `json:"why,omitempty"`
	// Copy is the subject's own copy of a one-on-one brief (SB-4807, LAW-30).
	Copy string `json:"copy,omitempty"`
}

// RhythmRef marks a queue item a rhythm offered: its run and step.
type RhythmRef struct {
	Run    string `json:"run"`
	Rhythm string `json:"rhythm"`
	Step   int    `json:"step"` // -1 is the plan
	Day    int    `json:"day"`
}

// RhythmRoster is the members of a space the onboarding rhythm has seen, so a person SCIM
// adds later starts one (SB-4802).
type RhythmRoster struct {
	Tenant  string    `json:"tenant"`
	Space   string    `json:"space"`
	Members []string  `json:"members"`
	At      time.Time `json:"at"`
}

// RhythmStore keeps started rhythms and the onboarding rosters.
type RhythmStore interface {
	PutRhythmRun(ctx context.Context, r RhythmRun) error
	GetRhythmRun(ctx context.Context, tenant, id string) (RhythmRun, bool, error)
	RhythmRuns(ctx context.Context, tenant string) ([]RhythmRun, error)
	PutRoster(ctx context.Context, r RhythmRoster) error
	GetRoster(ctx context.Context, tenant, space string) (RhythmRoster, bool, error)
}

// Rhythms is the rhythms service for one cell.
type Rhythms struct {
	Stewards Stewards
	Legwork  Legwork
	Store    RhythmStore
	Projects ProjectStore
	// Collections makes a project's collection (SB-4803).
	Collections CollectionService
	// Defs are the rhythms a space runs now (built in, its department's and its own); nil
	// is a space that doesn't exist.
	Defs func(space string) map[string]RhythmDef
	// Spaces are the manifest's space names.
	Spaces func() []string
	// ScopeIn is an active user as a scope in a space now; an error is someone who isn't an
	// active member of it.
	ScopeIn func(ctx context.Context, user, space string) (tenancy.Scope, error)
	// Manager is a person's manager's user id from SCIM, "" for none.
	Manager func(ctx context.Context, user string) (string, error)
	// Members are a space's members now, by user id (SB-4802).
	Members func(ctx context.Context, space string) ([]string, error)
	// TracesOf are a person's answers from a time on (SB-4802's gaps).
	TracesOf func(ctx context.Context, user string, from time.Time) ([]Trace, error)
	// Glossary is a space's glossary as it takes effect (SB-4805's overlaps).
	Glossary func(space string) map[string]string
}

func (x Rhythms) now() time.Time { return x.Stewards.Now().UTC() }

// RhythmView is a run as one reader sees it.
type RhythmView struct {
	ID      string           `json:"id"`
	Rhythm  string           `json:"rhythm"`
	Space   string           `json:"space"`
	Subject string           `json:"subject"`
	Kind    string           `json:"subject_kind"`
	State   string           `json:"state"`
	Why     string           `json:"why,omitempty"`
	Started time.Time        `json:"started,omitzero"`
	Mine    bool             `json:"mine,omitempty"` // the reader started it, or it is about them
	Steps   []RhythmStepView `json:"steps"`
}

// RhythmStepView is one step as a reader sees it.
type RhythmStepView struct {
	Day      int    `json:"day"`
	Kind     string `json:"kind"`
	Ask      string `json:"ask"`
	Approver string `json:"approver"`
	Due      string `json:"due,omitempty"`
	State    string `json:"state"`
	Why      string `json:"why,omitempty"` // why a step was skipped
	Yours    bool   `json:"yours,omitempty"`
}

// RhythmStart is POST /v1/spaces/NAME/rhythms.
type RhythmStart struct {
	Rhythm  string `json:"rhythm"`
	Subject string `json:"subject"` // an email for a person, a collection id, or "" for the space
}

func (x Rhythms) def(space, name string) (RhythmDef, error) {
	defs := x.Defs(space)
	if defs == nil {
		return RhythmDef{}, fmt.Errorf("%w: no space %q", ErrInvalid, space)
	}
	d, ok := defs[name]
	if !ok {
		return RhythmDef{}, fmt.Errorf("%w: space %s runs no rhythm %q", ErrInvalid, space, name)
	}
	return d, nil
}

func (x Rhythms) audit(ctx context.Context, actor string, r RhythmRun, state, why string) error {
	if actor == "" {
		actor = RhythmSystem
	}
	if _, err := x.Stewards.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "access", Subject: r.ID, State: state,
		Reason: "rhythm " + r.Rhythm + ": " + why, Space: auditSpace(SpaceKey(r.Space))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

// Start starts rhythm req.Rhythm in space for its subject, as one of the space's stewards.
func (x Rhythms) Start(ctx context.Context, s tenancy.Scope, space string, req RhythmStart) (RhythmView, error) {
	if req.Rhythm != OneOnOne { // a manager starts one-on-one prep for a report (SB-4807)
		if _, _, err := x.Stewards.steward(ctx, s, space); err != nil {
			return RhythmView{}, err
		}
	} else if err := s.Check(); err != nil {
		return RhythmView{}, ErrNotSteward
	}
	d, err := x.def(space, req.Rhythm)
	if err != nil {
		return RhythmView{}, err
	}
	subject := strings.TrimSpace(req.Subject)
	switch d.Subject {
	case "space":
		subject = space
	case "person":
		if x.Stewards.Owner == nil {
			return RhythmView{}, fmt.Errorf("%w: this cell has no directory", ErrInvalid)
		}
		id, ok, err := x.Stewards.Owner(ctx, strings.ToLower(subject))
		if err != nil {
			return RhythmView{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		if !ok {
			return RhythmView{}, fmt.Errorf("%w: %s isn't one active person", ErrInvalid, subject)
		}
		subject = id
		if req.Rhythm == OneOnOne {
			if err := x.isManager(ctx, s.User(), id); err != nil {
				return RhythmView{}, err
			}
		}
	case "collection":
		if _, err := x.Stewards.Agent.readCollection(ctx, s, x.Stewards.Agent.Collections, subject); err != nil {
			return RhythmView{}, err
		}
	}
	r, err := x.start(ctx, s.User(), space, req.Rhythm, d, subject)
	if err != nil {
		return RhythmView{}, err
	}
	return x.view(r, s.User()), nil
}

// start starts a rhythm whose starter may start it: the plan goes to its approver first, or
// the steps due today are offered now.
func (x Rhythms) start(ctx context.Context, starter, space, name string, d RhythmDef, subject string) (RhythmRun, error) {
	runs, err := x.Store.RhythmRuns(ctx, x.Stewards.Agent.Tenant)
	if err != nil {
		return RhythmRun{}, fmt.Errorf("%w: rhythms: %v", ErrUnavailable, err)
	}
	for _, r := range runs {
		if r.Space == space && r.Rhythm == name && r.Subject == subject && (r.State == RhythmRunning || r.State == RhythmProposed) {
			return RhythmRun{}, fmt.Errorf("%w: rhythm %s already runs for this subject", ErrInvalid, name)
		}
	}
	if d.Subject == "person" && x.ScopeIn != nil {
		if _, err := x.ScopeIn(ctx, subject, space); err != nil {
			return RhythmRun{}, fmt.Errorf("%w: the person isn't a member of %s", ErrInvalid, space)
		}
	}
	now := x.now()
	r := RhythmRun{ID: x.Stewards.NewID("rh_"), Tenant: x.Stewards.Agent.Tenant, Space: space, Rhythm: name, Subject: subject,
		Kind: d.Subject, Starter: starter, State: RhythmRunning, At: now}
	for _, st := range d.Steps {
		r.Steps = append(r.Steps, RhythmRunStep{RhythmStepDef: st, State: StepWaiting})
	}
	switch d.Plan {
	case "manager":
		if x.Manager != nil && d.Subject == "person" {
			m, err := x.Manager(ctx, subject)
			if err != nil {
				return RhythmRun{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
			}
			r.PlanBy = m
		}
		if r.PlanBy == "" {
			r.PlanBy = "steward" // no manager in SCIM: the space's stewards approve it
		}
	case "steward":
		r.PlanBy = "steward"
	}
	if r.PlanBy != "" {
		r.State = RhythmProposed
	}
	if err := x.audit(ctx, starter, r, "rhythm_started", "started for a "+d.Subject); err != nil {
		return RhythmRun{}, err
	}
	if r.State == RhythmProposed {
		item, err := x.put(ctx, r, -1, SpaceKeyOrPerson(r.PlanBy, space), "Approve the "+name+" plan ("+fmt.Sprint(len(r.Steps))+" steps over "+fmt.Sprint(lastDay(r))+" days)", nil, starterOf(r))
		if err != nil {
			return RhythmRun{}, err
		}
		r.PlanItem = item
		return r, x.save(ctx, r)
	}
	r.Started = now
	x.schedule(&r)
	if err := x.save(ctx, r); err != nil {
		return RhythmRun{}, err
	}
	return x.offerDue(ctx, r)
}

func starterOf(r RhythmRun) string {
	if r.Starter == RhythmSystem {
		return ""
	}
	return r.Starter
}

func lastDay(r RhythmRun) int {
	if len(r.Steps) == 0 {
		return 0
	}
	return r.Steps[len(r.Steps)-1].Day
}

func (x Rhythms) schedule(r *RhythmRun) {
	for i := range r.Steps {
		r.Steps[i].Due = r.Started.Add(time.Duration(r.Steps[i].Day) * 24 * time.Hour)
	}
}

func (x Rhythms) save(ctx context.Context, r RhythmRun) error {
	if err := x.Store.PutRhythmRun(ctx, r); err != nil {
		return fmt.Errorf("%w: rhythms: %v", ErrUnavailable, err)
	}
	return nil
}

// put puts one item in queue q (a space name or a person's key): a step's prompt, or its draft's text when there is one.
func (x Rhythms) put(ctx context.Context, r RhythmRun, step int, q, title string, thread *SavedThread, asker string) (string, error) {
	ref := &RhythmRef{Run: r.ID, Rhythm: r.Rhythm, Step: step}
	if step >= 0 {
		ref.Day = r.Steps[step].Day
	}
	sq := StewardQuestion{ID: x.Stewards.NewID("qs_"), Tenant: r.Tenant, Asker: asker, Question: firstLine(title), Space: SpaceKey(q),
		At: x.now(), Thread: thread, Rhythm: ref}
	if err := x.Stewards.Store.PutStewardQuestion(ctx, sq); err != nil {
		return "", fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
	}
	return sq.ID, nil
}

// Approve approves a proposed rhythm's plan, as the person it waits for (or a steward of
// its space, when it waits for the stewards): its days count from now.
func (x Rhythms) Approve(ctx context.Context, s tenancy.Scope, id string) (RhythmView, error) {
	r, err := x.get(ctx, id)
	if err != nil {
		return RhythmView{}, err
	}
	if r.State != RhythmProposed {
		return RhythmView{}, fmt.Errorf("%w: the rhythm isn't waiting for its plan", ErrInvalid)
	}
	if !x.mayApprovePlan(ctx, s, r) {
		return RhythmView{}, ErrNotSteward
	}
	if r, err = x.live(ctx, r); err != nil || r.State != RhythmProposed {
		return x.view(r, s.User()), err
	}
	r.State, r.Started = RhythmRunning, x.now()
	x.schedule(&r)
	x.closeItem(ctx, SpaceKeyOrPerson(r.PlanBy, r.Space), r.PlanItem, s.User())
	if err := x.audit(ctx, s.User(), r, "rhythm_approved", "plan approved"); err != nil {
		return RhythmView{}, err
	}
	if err := x.save(ctx, r); err != nil {
		return RhythmView{}, err
	}
	r, err = x.offerDue(ctx, r)
	return x.view(r, s.User()), err
}

// SpaceKeyOrPerson is the queue a plan waits in, by name: the person's own, or the space's.
func SpaceKeyOrPerson(by, space string) string {
	if by == "steward" {
		return space
	}
	return PersonKey(by)
}

func (x Rhythms) mayApprovePlan(ctx context.Context, s tenancy.Scope, r RhythmRun) bool {
	if r.PlanBy == "steward" {
		_, _, err := x.Stewards.steward(ctx, s, r.Space)
		return err == nil
	}
	return s.Check() == nil && s.User() == r.PlanBy
}

// closeItem marks a plan's item answered once its plan is approved or the rhythm stops.
func (x Rhythms) closeItem(ctx context.Context, queue, id, by string) {
	if id == "" {
		return
	}
	q, err := x.Stewards.question(ctx, queue, id)
	if err != nil || q.Reply != nil {
		return
	}
	q.Reply = &StewardReply{By: by, Text: "Done.", Cites: []string{}, At: x.now()}
	_ = x.Stewards.Store.PutStewardQuestion(ctx, q)
}

// Stop stops a rhythm, as its starter, its subject or a steward of its space.
func (x Rhythms) Stop(ctx context.Context, s tenancy.Scope, id string) (RhythmView, error) {
	r, err := x.get(ctx, id)
	if err != nil {
		return RhythmView{}, err
	}
	if err := s.Check(); err != nil {
		return RhythmView{}, ErrNotSteward
	}
	if s.User() != r.Starter && (r.Kind != "person" || s.User() != r.Subject) {
		if _, _, err := x.Stewards.steward(ctx, s, r.Space); err != nil {
			return RhythmView{}, err
		}
	}
	if r.State != RhythmRunning && r.State != RhythmProposed {
		return x.view(r, s.User()), nil
	}
	r, err = x.stop(ctx, s.User(), r, "stopped by "+who(s.User(), r))
	return x.view(r, s.User()), err
}

func who(user string, r RhythmRun) string {
	switch user {
	case r.Starter:
		return "its starter"
	case r.Subject:
		return "its subject"
	}
	return "a steward"
}

// stop ends a run now: steps not yet offered are skipped, and nothing more is offered.
func (x Rhythms) stop(ctx context.Context, actor string, r RhythmRun, why string) (RhythmRun, error) {
	r.State, r.Why, r.Ended = RhythmStopped, why, x.now()
	for i := range r.Steps {
		if r.Steps[i].State == StepWaiting {
			r.Steps[i].State = StepSkipped
		}
	}
	x.closeItem(ctx, SpaceKeyOrPerson(r.PlanBy, r.Space), r.PlanItem, actor)
	if err := x.audit(ctx, actor, r, "rhythm_stopped", why); err != nil {
		return r, err
	}
	return r, x.save(ctx, r)
}

// Consent records the subject's answer on whether their name goes with their gaps.
func (x Rhythms) Consent(ctx context.Context, s tenancy.Scope, id string, share bool) (RhythmView, error) {
	r, err := x.get(ctx, id)
	if err != nil {
		return RhythmView{}, err
	}
	if s.Check() != nil || r.Kind != "person" || s.User() != r.Subject {
		return RhythmView{}, fmt.Errorf("%w: only the person a rhythm is about answers this", ErrInvalid)
	}
	r.ShareName = share
	if err := x.audit(ctx, s.User(), r, "rhythm_consent", fmt.Sprintf("name with gaps: %t", share)); err != nil {
		return RhythmView{}, err
	}
	return x.view(r, s.User()), x.save(ctx, r)
}

func (x Rhythms) get(ctx context.Context, id string) (RhythmRun, error) {
	r, ok, err := x.Store.GetRhythmRun(ctx, x.Stewards.Agent.Tenant, strings.TrimSpace(id))
	if err != nil {
		return RhythmRun{}, fmt.Errorf("%w: rhythms: %v", ErrUnavailable, err)
	}
	if !ok || r.Tenant != x.Stewards.Agent.Tenant {
		return RhythmRun{}, fmt.Errorf("%w: no such rhythm", ErrInvalid)
	}
	return r, nil
}

// gone is why a run's subject has left, "" while it is still there.
func (x Rhythms) gone(ctx context.Context, r RhythmRun) (string, error) {
	if x.Defs(r.Space) == nil {
		return "its space is gone", nil
	}
	switch r.Kind {
	case "person":
		if x.ScopeIn == nil {
			return "", nil
		}
		if !x.isMember(ctx, r.Subject, r.Space) {
			return "its subject left the space", nil
		}
	case "collection":
		if x.Stewards.Agent.Collections == nil {
			return "", nil
		}
		c, ok, err := x.Stewards.Agent.Collections.GetCollection(ctx, r.Tenant, r.Subject)
		if err != nil {
			return "", fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
		}
		if !ok {
			return "its collection is gone", nil
		}
		if c.Closed != nil && r.Rhythm != "close-out" {
			return "its project closed", nil
		}
	}
	return "", nil
}

// isMember is whether a person is an active member of the space now.
func (x Rhythms) isMember(ctx context.Context, user, space string) bool {
	_, err := x.ScopeIn(ctx, user, space)
	return err == nil
}

// live checks a running or proposed run's subject is still there, stopping it at once when
// it isn't (LAW-29), and marks each offered step done once its item is answered or
// discarded.
func (x Rhythms) live(ctx context.Context, r RhythmRun) (RhythmRun, error) {
	if r.State != RhythmRunning && r.State != RhythmProposed {
		return r, nil
	}
	why, err := x.gone(ctx, r)
	if err != nil {
		return r, err
	}
	if why != "" {
		return x.stop(ctx, RhythmSystem, r, why)
	}
	changed, open := false, false
	for i, st := range r.Steps {
		if st.State != StepOffered {
			open = open || st.State == StepWaiting
			continue
		}
		q, err := x.Stewards.question(ctx, st.Queue, st.Item)
		if err != nil && !errors.Is(err, ErrNotSteward) {
			return r, err
		}
		if err != nil || q.Reply != nil || q.Thread != nil && q.Thread.Discarded {
			r.Steps[i].State, changed = StepDone, true
			continue
		}
		open = true
	}
	if r.State == RhythmRunning && !open {
		r.State, r.Ended, changed = RhythmDone, x.now(), true
		if err := x.audit(ctx, RhythmSystem, r, "rhythm_done", "every step done"); err != nil {
			return r, err
		}
	}
	if changed {
		return r, x.save(ctx, r)
	}
	return r, nil
}

// offerDue offers each waiting step whose day has come, to its approver.
func (x Rhythms) offerDue(ctx context.Context, r RhythmRun) (RhythmRun, error) {
	if r.State != RhythmRunning {
		return r, nil
	}
	now := x.now()
	for i := range r.Steps {
		if r.Steps[i].State != StepWaiting || r.Steps[i].Due.After(now) {
			continue
		}
		if err := x.offer(ctx, &r, i); err != nil {
			return r, err
		}
		if err := x.save(ctx, r); err != nil {
			return r, err
		}
	}
	return r, nil
}

// approver is the user a step goes to, or "steward" for the space's stewards.
func (x Rhythms) approver(ctx context.Context, r RhythmRun, st RhythmStepDef) (string, error) {
	switch st.Approver {
	case "subject":
		return r.Subject, nil
	case "starter":
		if r.Starter == RhythmSystem {
			return "steward", nil
		}
		return r.Starter, nil
	case "manager":
		subj := r.Subject
		if r.Kind != "person" {
			subj = r.Starter
		}
		if x.Manager == nil {
			return "steward", nil
		}
		m, err := x.Manager(ctx, subj)
		if err != nil {
			return "", fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		if m == "" {
			return "steward", nil
		}
		return m, nil
	}
	return "steward", nil
}

// offer drafts step i as its approver, from what they can open, and puts it in their queue,
// or in the space's for the stewards. It never publishes, sends or creates anything else.
func (x Rhythms) offer(ctx context.Context, r *RhythmRun, i int) error {
	st := &r.Steps[i]
	to, err := x.approver(ctx, *r, st.RhythmStepDef)
	if err != nil {
		return err
	}
	queue, as := r.Space, r.Starter // a queue by name; a steward's step is drafted as the steward who started it
	switch {
	case to != "steward":
		queue, as = PersonKey(to), to
	case r.Starter == RhythmSystem:
		as = x.firstSteward(ctx, r.Space) // a system start: drafted as one steward, theirs to read
	}
	var s tenancy.Scope
	if as != "" && x.ScopeIn != nil {
		if s, err = x.ScopeIn(ctx, as, r.Space); err != nil {
			st.State, st.Why = StepSkipped, "its approver isn't a member of the space now"
			return x.audit(ctx, RhythmSystem, *r, "rhythm_step_skipped", fmt.Sprintf("day %d: %s", st.Day, st.Why))
		}
	}
	title := fmt.Sprintf("%s, day %d: %s", r.Rhythm, st.Day, st.Ask)
	thread, err := x.draft(ctx, r, *st, s, as != "")
	if err != nil {
		st.State, st.Why = StepSkipped, "nothing to draft: "+err.Error()
		return x.audit(ctx, RhythmSystem, *r, "rhythm_step_skipped", fmt.Sprintf("day %d: nothing to draft", st.Day))
	}
	item, err := x.put(ctx, *r, i, queue, title, thread, as) // drafted as its approver: theirs to read
	if err != nil {
		return err
	}
	st.State, st.Queue, st.Item = StepOffered, queue, item
	if st.Agent == AgentOneOnOne && thread != nil { // LAW-30: the report gets the same brief, at the same time
		if st.Copy, err = x.put(ctx, *r, i, PersonKey(r.Subject), title, &SavedThread{Text: thread.Text, Kind: DraftRhythm}, r.Subject); err != nil {
			return err
		}
	}
	return x.audit(ctx, RhythmSystem, *r, "rhythm_step_offered", fmt.Sprintf("day %d %s to its approver", st.Day, st.Kind))
}

// draft is a step's draft text, or nil for a step that is only a prompt (a routine, a
// check-in, or a request with no agent).
func (x Rhythms) draft(ctx context.Context, rp *RhythmRun, st RhythmRunStep, s tenancy.Scope, scoped bool) (*SavedThread, error) {
	r := *rp
	if st.Kind != "draft" && st.Agent != "gaps" {
		return nil, nil
	}
	if st.Agent == "gaps" {
		text, err := x.gaps(ctx, r)
		if err != nil {
			return nil, err
		}
		return &SavedThread{Text: text, Kind: DraftRhythm}, nil
	}
	if !scoped {
		return nil, nil // a steward's step from a system start: a prompt only
	}
	var text, agent string
	var err error
	switch st.Agent {
	case "start-here":
		text, err = x.startHere(ctx, s, r.Space)
	case "catch-up":
		text, err = x.catchUp(ctx, s, r)
		agent = "ask"
	case "kickoff-brief":
		text, err = x.kickoffBrief(ctx, s, r)
	case "retrospective":
		text, err = x.retrospective(ctx, s, r)
	case AgentGoalPlan:
		text, rp.Proposed, err = x.goalPlan(ctx, s, r)
	case AgentOneOnOne:
		text, err = x.oneOnOne(ctx, s, r)
	default:
		var resp Response
		if resp, err = x.Stewards.Agent.Ask(ctx, s, Request{Question: st.Ask}); err == nil {
			text, agent = strings.TrimPrefix(resp.Assumed+"\n", "\n")+DraftText(resp.Answer), "ask"
		}
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("%w: nothing cited", ErrInvalid)
	}
	return &SavedThread{Text: text, Agent: agent, Kind: DraftRhythm}, nil
}

// startHere is a joiner's day-one draft: the start-here items they can open and the people
// who own them (SB-4802), nothing they can't (LAW-01).
func (x Rhythms) startHere(ctx context.Context, s tenancy.Scope, space string) (string, error) {
	var items []string
	if x.Stewards.StartItems != nil {
		items = x.Stewards.StartItems(space)
	}
	if len(items) == 0 {
		return "", fmt.Errorf("%w: the space has no start-here list", ErrInvalid)
	}
	st, err := x.Stewards.Agent.StartHere(ctx, s, items)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	people := map[string]bool{}
	for _, it := range st.Items {
		t := it.Title
		if t == "" {
			t = it.Cite
		}
		fmt.Fprintf(&b, "- Read %s [%s]\n", t, it.Cite)
		if it.Owner != nil && it.Owner.Email != "" && !it.Mine {
			people[it.Owner.Email] = true
		}
	}
	if len(people) > 0 {
		var ps []string
		for p := range people {
			ps = append(ps, p)
		}
		sort.Strings(ps)
		fmt.Fprintf(&b, "- People to meet: %s\n", strings.Join(ps, ", "))
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// catchUp is a catch-up since the run started, as its approver (SB-4506), narrowed to the
// collection for a project's run.
func (x Rhythms) catchUp(ctx context.Context, s tenancy.Scope, r RhythmRun) (string, error) {
	req := CatchUpRequest{Since: r.Started.In(zoneOr(x.Legwork.Zone)).Format(time.DateOnly)}
	if r.Kind == "collection" {
		req.Collection = r.Subject
	}
	if !r.Started.Before(x.now().Add(-24 * time.Hour)) {
		req.Since = x.now().Add(-24 * time.Hour).In(zoneOr(x.Legwork.Zone)).Format(time.DateOnly)
	}
	resp, err := x.Legwork.CatchUp(ctx, s, req)
	if err != nil {
		return "", err
	}
	return DraftText(resp.Answer), nil
}

// gaps is the stewards' note of a joiner's questions that found nothing since they joined:
// counts and glossary terms only, and their name only when they said yes (SB-4802).
func (x Rhythms) gaps(ctx context.Context, r RhythmRun) (string, error) {
	if r.Kind != "person" || x.TracesOf == nil {
		return "", fmt.Errorf("%w: gaps need a person", ErrInvalid)
	}
	ts, err := x.TracesOf(ctx, r.Subject, r.Started)
	if err != nil {
		return "", fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	n, terms := 0, map[string]bool{}
	for _, t := range ts {
		if t.User != r.Subject || t.Space != SpaceKey(r.Space) || !nothing(t) {
			continue
		}
		n++
		for _, term := range t.Response.Terms {
			terms[term.Term] = true
		}
	}
	whom := "A new joiner"
	if r.ShareName && x.Stewards.Agent.Emails != nil {
		if es, err := x.Stewards.Agent.Emails(ctx, r.Subject); err == nil && len(es) > 0 {
			whom = es[0]
		}
	}
	text := fmt.Sprintf("%s asked %d questions here that found nothing.", whom, n)
	if len(terms) > 0 {
		var ts []string
		for t := range terms {
			ts = append(ts, t)
		}
		sort.Strings(ts)
		text += " Terms they used: " + strings.Join(ts, ", ") + "."
	}
	return text, nil
}

// For is the rhythms of a space a reader may see, checked now: a steward sees all of them;
// anyone else the ones they started, are the subject of, or approve a step of.
func (x Rhythms) For(ctx context.Context, s tenancy.Scope, space string) ([]RhythmView, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Stewards.Agent.Tenant {
		return nil, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	_, _, notSteward := x.Stewards.steward(ctx, s, space)
	runs, err := x.Store.RhythmRuns(ctx, x.Stewards.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: rhythms: %v", ErrUnavailable, err)
	}
	out := []RhythmView{}
	for _, r := range runs {
		if r.Space != space {
			continue
		}
		if r, err = x.live(ctx, r); err != nil {
			return nil, err
		}
		v := x.view(r, s.User())
		if notSteward != nil && !v.Mine && !anyYours(v) {
			continue
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Started.After(out[j].Started) })
	return out, nil
}

func anyYours(v RhythmView) bool {
	for _, st := range v.Steps {
		if st.Yours {
			return true
		}
	}
	return false
}

func (x Rhythms) view(r RhythmRun, user string) RhythmView {
	v := RhythmView{ID: r.ID, Rhythm: r.Rhythm, Space: r.Space, Subject: r.Subject, Kind: r.Kind, State: r.State, Why: r.Why,
		Started: r.Started, Mine: user == r.Starter || r.Kind == "person" && user == r.Subject, Steps: []RhythmStepView{}}
	for _, st := range r.Steps {
		sv := RhythmStepView{Day: st.Day, Kind: st.Kind, Ask: st.Ask, Approver: st.Approver, State: st.State, Why: st.Why, Yours: st.Queue == PersonKey(user)}
		if !st.Due.IsZero() {
			sv.Due = st.Due.In(zoneOr(x.Legwork.Zone)).Format(time.DateOnly)
		}
		v.Steps = append(v.Steps, sv)
	}
	return v
}

// Tick is the rhythms' run in the jobs service: it starts onboarding for each person SCIM
// added to a space since the last run, stops each run whose subject left, and offers each
// step whose day has come. It returns how many steps it offered.
func (x Rhythms) Tick(ctx context.Context) (int, error) {
	if err := x.joiners(ctx); err != nil {
		return 0, err
	}
	if err := x.quarterly(ctx); err != nil {
		return 0, err
	}
	runs, err := x.Store.RhythmRuns(ctx, x.Stewards.Agent.Tenant)
	if err != nil {
		return 0, fmt.Errorf("%w: rhythms: %v", ErrUnavailable, err)
	}
	n := 0
	for _, r := range runs {
		if r, err = x.live(ctx, r); err != nil {
			return n, err
		}
		before := offered(r)
		if r, err = x.offerDue(ctx, r); err != nil {
			return n, err
		}
		n += offered(r) - before
	}
	return n, nil
}

func offered(r RhythmRun) int {
	n := 0
	for _, st := range r.Steps {
		if st.State != StepWaiting {
			n++
		}
	}
	return n
}

// joiners starts onboarding for each member of a space with it on who wasn't a member at
// the last run; a space's first run only notes who is there.
func (x Rhythms) joiners(ctx context.Context) error {
	if x.Members == nil || x.Spaces == nil {
		return nil
	}
	for _, space := range x.Spaces() {
		d, ok := x.Defs(space)["onboarding"]
		if !ok {
			continue
		}
		members, err := x.Members(ctx, space)
		if err != nil {
			return fmt.Errorf("%w: members: %v", ErrUnavailable, err)
		}
		sort.Strings(members)
		ro, found, err := x.Store.GetRoster(ctx, x.Stewards.Agent.Tenant, space)
		if err != nil {
			return fmt.Errorf("%w: rosters: %v", ErrUnavailable, err)
		}
		if found {
			for _, m := range members {
				if contains(ro.Members, m) {
					continue
				}
				if _, err := x.start(ctx, RhythmSystem, space, "onboarding", d, m); err != nil && !errors.Is(err, ErrInvalid) {
					return err
				}
			}
		}
		if err := x.Store.PutRoster(ctx, RhythmRoster{Tenant: x.Stewards.Agent.Tenant, Space: space, Members: members, At: x.now()}); err != nil {
			return fmt.Errorf("%w: rosters: %v", ErrUnavailable, err)
		}
	}
	return nil
}
