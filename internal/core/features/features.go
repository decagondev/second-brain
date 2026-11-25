// Package features is feature rollouts (sb-release(7), Feature rollouts; SB-5106): each
// release lists its features, and one marked pilot is on only for the tenant's pilot group
// until PilotDays have passed since the cell first ran it, then for everyone, unless an
// admin holds it.
package features

import (
	"errors"
	"sort"
	"time"
)

// Rollouts of a feature.
const (
	Pilot = "pilot"
	On    = "on"
)

// PilotDays is how long a pilot feature stays with the pilot group, and PilotGroup the
// IdP group that has it first.
const (
	PilotDays  = 14
	PilotGroup = "pilot"
)

// Feature is one feature of the release.
type Feature struct {
	Name    string `json:"name"`
	Ticket  string `json:"ticket"`
	Rollout string `json:"rollout"`
}

// List is this release's features with a rollout, by name.
var List = []Feature{
	{Name: "branch-moderators", Ticket: "SB-5105", Rollout: Pilot},
	{Name: "pause-link", Ticket: "SB-5107", Rollout: Pilot},
}

// Known says name is one of List's.
func Known(name string) (Feature, bool) {
	for _, f := range List {
		if f.Name == name {
			return f, true
		}
	}
	return Feature{}, false
}

// ErrUnknown is a feature this release doesn't have.
var ErrUnknown = errors.New("features: no such feature in this release")

// State is what the cell keeps: when it first ran each pilot feature, and which are held.
type State struct {
	Since map[string]time.Time `json:"since"`
	Held  map[string]bool      `json:"held"`
}

// Seen is s with now as the first run of every pilot feature it hadn't seen, and whether
// anything changed.
func (s State) Seen(now time.Time) (State, bool) {
	changed := false
	if s.Since == nil {
		s.Since = map[string]time.Time{}
	}
	for _, f := range List {
		if _, ok := s.Since[f.Name]; !ok && f.Rollout == Pilot {
			s.Since[f.Name], changed = now.UTC(), true
		}
	}
	return s, changed
}

// Status is a feature's state now: on, pilot (with the time it goes on for everyone) or
// held.
type Status struct {
	Feature
	State  string    `json:"state"`
	Since  time.Time `json:"since,omitempty"`
	OnAt   time.Time `json:"on_at,omitempty"`
	Pilots string    `json:"pilot_group,omitempty"`
}

// Of is f's status in s now.
func (s State) Of(f Feature, now time.Time) Status {
	st := Status{Feature: f, State: On}
	if f.Rollout != Pilot {
		return st
	}
	since, ok := s.Since[f.Name]
	if !ok {
		since = now.UTC()
	}
	st.Since, st.Pilots = since, PilotGroup
	switch {
	case s.Held[f.Name]:
		st.State = "held"
	case now.Before(since.AddDate(0, 0, PilotDays)):
		st.State, st.OnAt = Pilot, since.AddDate(0, 0, PilotDays)
	}
	return st
}

// All are every feature's status, by name.
func (s State) All(now time.Time) []Status {
	out := make([]Status, 0, len(List))
	for _, f := range List {
		out = append(out, s.Of(f, now))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// For says feature name is on for a person in groups now: always for a feature without a
// pilot, for the pilot group while it is in pilot or held, and for everyone after.
// An unknown feature is off.
func (s State) For(name string, groups []string, now time.Time) bool {
	f, ok := Known(name)
	if !ok {
		return false
	}
	if st := s.Of(f, now); st.State == On {
		return true
	}
	for _, g := range groups {
		if g == PilotGroup {
			return true
		}
	}
	return false
}
