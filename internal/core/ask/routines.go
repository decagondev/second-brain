package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/tenancy"
)

// My routines (sb-ask(3), My routines; SB-4302): a saved question asked for its owner each
// working morning or once a week, as them, now. Its answer is kept like a report and checked
// again as its reader when it is opened (LAW-26).

// The routines' limits and kinds.
const (
	MaxRoutines    = 5
	RoutinesPerRun = 10
	RoutineHour    = 8
	KindRoutine    = "routine"
	EveryDaily     = "daily"
	EveryWeekly    = "weekly"
	EveryOff       = "off"
	// RoutinePaused is a run when the budget is spent.
	RoutinePaused = "paused: the budget for this month is spent"
)

// Routine is a saved question's schedule, and what its last run cited (each cite's hash).
type Routine struct {
	Every string    `json:"every"`
	Day   string    `json:"day,omitempty"`
	Last  time.Time `json:"last,omitzero"`
	Seen  []string  `json:"seen,omitempty"`
}

// RoutineStore lists and changes the saved questions that are routines.
type RoutineStore interface {
	Routines(ctx context.Context, tenant string) ([]SavedQuestion, error)
	UpdateSaved(ctx context.Context, tenant, user, id string, f func(*SavedQuestion)) (found bool, err error)
}

var routineDays = map[string]time.Weekday{"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
	"thursday": time.Thursday, "friday": time.Friday}

// SetRoutine makes one of the caller's saved questions a routine, or stops it (every off).
func (x Questions) SetRoutine(ctx context.Context, s tenancy.Scope, id, every, day string) (SavedQuestion, error) {
	if err := s.Check(); err != nil || s.Tenant() != x.Tenant {
		return SavedQuestion{}, ErrNoShare
	}
	if x.Watches == nil {
		return SavedQuestion{}, fmt.Errorf("%w: this cell can't keep routines", ErrUnavailable)
	}
	every, day = strings.ToLower(strings.TrimSpace(every)), strings.ToLower(strings.TrimSpace(day))
	var r *Routine
	switch every {
	case EveryOff, "":
	case EveryDaily:
		r = &Routine{Every: EveryDaily}
	case EveryWeekly:
		if _, ok := routineDays[day]; !ok {
			return SavedQuestion{}, fmt.Errorf("%w: a weekly routine needs a day, monday to friday", ErrInvalid)
		}
		r = &Routine{Every: EveryWeekly, Day: day}
	default:
		return SavedQuestion{}, fmt.Errorf("%w: every is daily, weekly or off", ErrInvalid)
	}
	have, err := x.Store.SavedOf(ctx, x.Tenant, s.User())
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	var q *SavedQuestion
	n := 0
	for i := range have {
		if have[i].ID == id {
			q = &have[i]
		} else if have[i].Routine != nil {
			n++
		}
	}
	if q == nil {
		return SavedQuestion{}, ErrNoShare
	}
	if r != nil && n >= MaxRoutines {
		return SavedQuestion{}, fmt.Errorf("%w: at most %d routines; stop one first", ErrInvalid, MaxRoutines)
	}
	if r != nil {
		r.Last = x.Now().UTC() // the first run is the next slot after now
		if q.Routine != nil {
			r.Seen = q.Routine.Seen
		}
	}
	found, err := x.Watches.UpdateSaved(ctx, x.Tenant, s.User(), id, func(sq *SavedQuestion) { sq.Routine = r })
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	if !found {
		return SavedQuestion{}, ErrNoShare
	}
	q.Routine = r
	return *q, nil
}

// RoutineSlot is a routine's latest run time at or before now: 08:00 on the latest working
// day (daily) or its day (weekly), in zone.
func RoutineSlot(r Routine, now time.Time, zone *time.Location) time.Time {
	if zone == nil {
		zone = time.UTC
	}
	t := now.In(zone)
	d := time.Date(t.Year(), t.Month(), t.Day(), RoutineHour, 0, 0, 0, zone)
	want, weekly := routineDays[r.Day]
	weekly = weekly && r.Every == EveryWeekly
	for i := 0; i < 8; i++ {
		ok := !d.After(t)
		if weekly {
			ok = ok && d.Weekday() == want
		} else {
			ok = ok && d.Weekday() != time.Saturday && d.Weekday() != time.Sunday
		}
		if ok {
			break
		}
		d = d.AddDate(0, 0, -1)
	}
	return d.UTC()
}

// RoutineDue says whether a routine's slot has passed since its last run.
func RoutineDue(r Routine, now time.Time, zone *time.Location) bool {
	return r.Last.Before(RoutineSlot(r, now, zone))
}

func citeHash(c string) string { return hashOf("routine\x00" + c)[:16] }

// RunRoutines asks up to RoutinesPerRun due routines, oldest first, each as its owner now,
// and keeps each answer for them. A person who isn't active now is never asked for (LAW-12).
func (l Legwork) RunRoutines(ctx context.Context) ([]Finished, error) {
	if l.Routines == nil || l.ScopeIn == nil || l.Research == nil {
		return nil, nil
	}
	qs, err := l.Routines.Routines(ctx, l.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: routines: %v", ErrUnavailable, err)
	}
	now := l.now()
	var due []SavedQuestion
	for _, q := range qs {
		if q.Routine == nil || !RoutineDue(*q.Routine, now, l.Zone) {
			continue
		}
		if q.From != "" && !stillShared(qs, q) { // SB-4603: its steward stopped sharing it
			if err := l.stopRoutine(ctx, q); err != nil {
				return nil, err
			}
			continue
		}
		due = append(due, q)
	}
	sort.SliceStable(due, func(i, j int) bool { return due[i].Routine.Last.Before(due[j].Routine.Last) })
	if len(due) > RoutinesPerRun {
		due = due[:RoutinesPerRun]
	}
	var out []Finished
	for _, q := range due {
		f, err := l.runRoutine(ctx, q, now)
		if err != nil {
			return out, err
		}
		if f != nil {
			out = append(out, *f)
		}
	}
	return out, nil
}

func (l Legwork) runRoutine(ctx context.Context, q SavedQuestion, now time.Time) (*Finished, error) {
	mark := func(seen []string) error {
		_, err := l.Routines.UpdateSaved(ctx, l.Agent.Tenant, q.User, q.ID, func(sq *SavedQuestion) {
			if sq.Routine != nil {
				sq.Routine.Last = now
				if seen != nil {
					sq.Routine.Seen = seen
				}
			}
		})
		if err != nil {
			return fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
		}
		return nil
	}
	s, err := l.ScopeIn(ctx, q.User, q.Space)
	if err != nil { // not active now: nothing is asked or kept (LAW-12)
		return nil, mark(nil)
	}
	if left, err := l.leftSpace(ctx, s, q.Space); err != nil {
		return nil, err
	} else if left { // no longer a member: nothing is asked, and a shared copy stops (SB-4603)
		if q.From != "" {
			return nil, l.stopRoutine(ctx, q)
		}
		return nil, mark(nil)
	}
	if l.Agent.runsStop(ctx, s, "ask") != nil { // paused here or at its cap: nothing is asked (SB-4702, SB-4703)
		return nil, mark(nil)
	}
	r := Research{ID: l.NewID("rs_"), Tenant: l.Agent.Tenant, Release: l.Agent.Release, User: q.User, Kind: KindRoutine, Question: q.Question, Space: q.Space,
		Steps: []string{q.Question}, State: StateDone, At: now}
	resp, err := l.Agent.Ask(ctx, s, Request{Question: q.Question})
	switch {
	case errors.Is(err, ErrBudget):
		r.State, r.Why, r.Stopped = StateFailed, RoutinePaused, "budget"
	case errors.Is(err, ErrKilled):
		r.State, r.Why = StateFailed, "answers are turned off"
	case err != nil:
		r.State, r.Why = StateFailed, "the question couldn't be answered"
	}
	var seen []string
	if err == nil {
		claims, _ := citedOnly(resp.Answer.Claims)
		old := map[string]bool{}
		for _, h := range q.Routine.Seen {
			old[h] = true
		}
		isNew := map[string]bool{}
		for _, c := range claims {
			for _, ci := range c.Cites {
				h := citeHash(ci.Cite)
				if !contains(seen, h) {
					seen = append(seen, h)
					if !old[h] {
						isNew[ci.Cite] = true
					}
				}
			}
		}
		claims = newFirst(claims, isNew)
		n := len(isNew)
		r.Changed = &n
		if !claims[0].Uncited {
			r.Sections = []Section{{Question: q.Question, Trace: resp.Trace, Claims: claims, Assumed: resp.Assumed}}
		}
		if seen == nil {
			seen = []string{}
		}
	}
	r.DoneAt = now
	if err := l.audit(ctx, q.User, r, KindRoutine, hashOf(q.Question)); err != nil {
		return nil, err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return nil, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	if err := mark(seen); err != nil {
		return nil, err
	}
	f := Finished{ID: r.ID, User: r.User, State: r.State}
	if r.Changed != nil {
		f.Changed = *r.Changed
	}
	return &f, nil
}

// newFirst puts the claims that cite a new source first, in their order, then the rest.
func newFirst(claims []answer.Claim, isNew map[string]bool) []answer.Claim {
	has := func(c answer.Claim) bool {
		for _, ci := range c.Cites {
			if isNew[ci.Cite] {
				return true
			}
		}
		return false
	}
	out := make([]answer.Claim, 0, len(claims))
	for _, c := range claims {
		if has(c) {
			out = append(out, c)
		}
	}
	for _, c := range claims {
		if !has(c) {
			out = append(out, c)
		}
	}
	return out
}

// stillShared says whether a member's copy's routine is still shared with its space.
func stillShared(qs []SavedQuestion, cp SavedQuestion) bool {
	for _, q := range qs {
		if q.ID == cp.From {
			return q.Shared != "" && SpaceKey(q.Shared) == cp.Space && q.Routine != nil
		}
	}
	return false
}

// stopRoutine stops a shared routine's copy: its schedule goes, and nothing is asked.
func (l Legwork) stopRoutine(ctx context.Context, q SavedQuestion) error {
	if _, err := l.Routines.UpdateSaved(ctx, l.Agent.Tenant, q.User, q.ID, func(sq *SavedQuestion) { sq.Routine = nil }); err != nil {
		return fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	return nil
}

// leftSpace says whether s is no longer a member of space (spc_NAME) now; a company-wide
// routine, or a cell without spaces, never left one.
func (l Legwork) leftSpace(ctx context.Context, s tenancy.Scope, space string) (bool, error) {
	if !strings.HasPrefix(space, "spc_") || l.Stewards.Agent.Auth.Spaces == nil {
		return false, nil
	}
	err := l.Stewards.member(ctx, s, strings.TrimPrefix(space, "spc_"))
	if errors.Is(err, ErrNotMember) {
		return true, nil
	}
	return false, err
}
