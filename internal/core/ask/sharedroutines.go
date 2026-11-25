package ask

import (
	"context"
	"fmt"
	"sort"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Shared routines (sb-ask(3), Shared routines; SB-4603): a steward shares one of their own
// routines with their space, and each member who opts in gets their own copy, asked as them
// and kept for them alone. The steward sees how many opted in, never who or their answers.

// SharedStore keeps saved questions and lists every routine of the tenant.
type SharedStore interface {
	PutSaved(ctx context.Context, q SavedQuestion) error
	SavedOf(ctx context.Context, tenant, user string) ([]SavedQuestion, error)
	DeleteSaved(ctx context.Context, tenant, user, id string) (found bool, err error)
	Routines(ctx context.Context, tenant string) ([]SavedQuestion, error)
	UpdateSaved(ctx context.Context, tenant, user, id string, f func(*SavedQuestion)) (found bool, err error)
}

// SharedRoutine is one routine shared with a space, as a member sees it. OptedIn is only on
// the routines the caller shared.
type SharedRoutine struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Every    string `json:"every"`
	Day      string `json:"day,omitempty"`
	Joined   bool   `json:"joined"`
	Mine     bool   `json:"mine,omitempty"`
	OptedIn  *int   `json:"opted_in,omitempty"`
}

func (x Stewards) savedStore() (SharedStore, error) {
	if x.Saved == nil {
		return nil, fmt.Errorf("%w: this cell can't share routines", ErrUnavailable)
	}
	return x.Saved, nil
}

// ShareRoutine shares one of s's own routines in space name, which they steward, with the
// space, or stops sharing it (on false).
func (x Stewards) ShareRoutine(ctx context.Context, s tenancy.Scope, name, id string, on bool) (SavedQuestion, error) {
	st, err := x.savedStore()
	if err != nil {
		return SavedQuestion{}, err
	}
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return SavedQuestion{}, err
	}
	have, err := st.SavedOf(ctx, x.Agent.Tenant, s.User())
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	var q *SavedQuestion
	for i := range have {
		if have[i].ID == id {
			q = &have[i]
		}
	}
	switch {
	case q == nil:
		return SavedQuestion{}, ErrNoShare
	case q.Routine == nil || q.Space != SpaceKey(name) || q.From != "":
		return SavedQuestion{}, fmt.Errorf("%w: share a routine of your own, asked in %s", ErrInvalid, name)
	}
	shared, state := "", "unshared"
	if on {
		shared, state = name, "shared"
	}
	if err := x.auditRoutine(ctx, s.User(), id, state, name); err != nil {
		return SavedQuestion{}, err
	}
	if _, err := st.UpdateSaved(ctx, x.Agent.Tenant, s.User(), id, func(sq *SavedQuestion) { sq.Shared = shared }); err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	q.Shared = shared
	return *q, nil
}

func (x Stewards) auditRoutine(ctx context.Context, user, id, state, name string) error {
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: user, Kind: "access", Subject: id, State: state,
		Reason: "shared routine " + state, Space: auditSpace(SpaceKey(name))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

// sharedIn are the routines shared with space name now, by id.
func sharedIn(all []SavedQuestion, name string) map[string]SavedQuestion {
	out := map[string]SavedQuestion{}
	for _, q := range all {
		if q.Shared == name && q.Space == SpaceKey(name) && q.Routine != nil && q.From == "" {
			out[q.ID] = q
		}
	}
	return out
}

// SharedRoutines are the routines shared with space name, for one of its members.
func (x Stewards) SharedRoutines(ctx context.Context, s tenancy.Scope, name string) ([]SharedRoutine, error) {
	st, err := x.savedStore()
	if err != nil {
		return nil, err
	}
	if err := x.member(ctx, s, name); err != nil {
		return nil, err
	}
	all, err := st.Routines(ctx, x.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: routines: %v", ErrUnavailable, err)
	}
	shared := sharedIn(all, name)
	joined, counts := map[string]bool{}, map[string]int{}
	for _, q := range all {
		if _, ok := shared[q.From]; ok {
			counts[q.From]++
			joined[q.From] = joined[q.From] || q.User == s.User()
		}
	}
	out := []SharedRoutine{}
	for id, q := range shared {
		r := SharedRoutine{ID: id, Question: q.Question, Every: q.Routine.Every, Day: q.Routine.Day, Joined: joined[id], Mine: q.User == s.User()}
		if r.Mine {
			n := counts[id]
			r.OptedIn = &n // how many, never who
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Question < out[j].Question || out[i].Question == out[j].Question && out[i].ID < out[j].ID
	})
	return out, nil
}

// JoinRoutine opts s in to a routine shared with space name, as their own copy counted
// toward their routines, or out (on false), which removes the copy.
func (x Stewards) JoinRoutine(ctx context.Context, s tenancy.Scope, name, id string, on bool) (SavedQuestion, error) {
	st, err := x.savedStore()
	if err != nil {
		return SavedQuestion{}, err
	}
	if err := x.member(ctx, s, name); err != nil {
		return SavedQuestion{}, err
	}
	all, err := st.Routines(ctx, x.Agent.Tenant)
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: routines: %v", ErrUnavailable, err)
	}
	orig, ok := sharedIn(all, name)[id]
	if !ok {
		return SavedQuestion{}, fmt.Errorf("%w: no routine %s is shared with %s", ErrInvalid, id, name)
	}
	if orig.User == s.User() {
		return SavedQuestion{}, fmt.Errorf("%w: it is your own routine", ErrInvalid)
	}
	have, err := st.SavedOf(ctx, x.Agent.Tenant, s.User())
	if err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	n := 0
	for _, q := range have {
		if q.From == id {
			if on {
				return q, nil // already in
			}
			if err := x.auditRoutine(ctx, s.User(), q.ID, "left", name); err != nil {
				return SavedQuestion{}, err
			}
			if _, err := st.DeleteSaved(ctx, x.Agent.Tenant, s.User(), q.ID); err != nil {
				return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
			}
			return q, nil
		}
		if q.Routine != nil {
			n++
		}
	}
	switch {
	case !on:
		return SavedQuestion{}, fmt.Errorf("%w: you haven't opted in to %s", ErrInvalid, id)
	case n >= MaxRoutines:
		return SavedQuestion{}, fmt.Errorf("%w: at most %d routines; stop one first", ErrInvalid, MaxRoutines)
	case len(have) >= MaxSaved:
		return SavedQuestion{}, fmt.Errorf("%w: at most %d saved questions; remove one first", ErrInvalid, MaxSaved)
	}
	now := x.Now().UTC()
	cp := SavedQuestion{ID: x.NewID("sq_"), Tenant: x.Agent.Tenant, User: s.User(), Question: orig.Question, Space: orig.Space, At: now,
		From: id, Routine: &Routine{Every: orig.Routine.Every, Day: orig.Routine.Day, Last: now}}
	if err := x.auditRoutine(ctx, s.User(), cp.ID, "joined", name); err != nil {
		return SavedQuestion{}, err
	}
	if err := st.PutSaved(ctx, cp); err != nil {
		return SavedQuestion{}, fmt.Errorf("%w: saved: %v", ErrUnavailable, err)
	}
	return cp, nil
}

// SharedRoutineCount is how many routines are shared with space name now (SB-5105): a count
// for a branch moderator, never a question or who shared it.
func (x Stewards) SharedRoutineCount(ctx context.Context, name string) (int, error) {
	st, err := x.savedStore()
	if err != nil {
		return 0, err
	}
	all, err := st.Routines(ctx, x.Agent.Tenant)
	if err != nil {
		return 0, fmt.Errorf("%w: routines: %v", ErrUnavailable, err)
	}
	return len(sharedIn(all, name)), nil
}
