// Package rollout is the release controller (sb-release(1), How the controller decides;
// SB-507, SB-508). Its state and moves are those of proofs/tla/Rollout.tla, and it checks
// the model's invariants after every move, refusing any move that would break one.
package rollout

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Soak is how long a release must run on ring 0 and ring 1 before promotion.
const Soak = 48 * time.Hour

// Channel is a tenant's channel.
type Channel string

const (
	Stable  Channel = "stable"
	Testing Channel = "testing"
)

// Release is one registered release.
type Release struct {
	ID       string    `json:"id"`
	Order    int       `json:"order"` // build order; later releases are greater
	Gated    bool      `json:"gated"` // signed and passed the gate (admission)
	Revoked  bool      `json:"revoked"`
	Ring0At  time.Time `json:"ring0_at,omitempty"` // first deployed to ring 0
	Ring1At  time.Time `json:"ring1_at,omitempty"` // first deployed to ring 1
	Promoted time.Time `json:"promoted,omitempty"`
}

// Tenant is one tenant on the release train.
type Tenant struct {
	Name     string  `json:"name"`
	Channel  Channel `json:"channel"`
	Ring     int     `json:"ring"` // 0 internal, 1 friendly testing, 2 stable
	Running  string  `json:"running"`
	Draining bool    `json:"draining"`
}

// State is the controller's whole state (DIR/releases/state.json for a local control plane).
type State struct {
	Releases   map[string]*Release `json:"releases"`
	Tenants    map[string]*Tenant  `json:"tenants"`
	Stable     string              `json:"stable"`
	PrevStable string              `json:"prev_stable"`
	next       int
}

// ErrRefused is a move the controller won't make.
var ErrRefused = errors.New("rollout: refused")

// New starts with one gated initial stable that every tenant runs.
func New(initial string, tenants []Tenant) *State {
	s := &State{Releases: map[string]*Release{initial: {ID: initial, Gated: true}}, Tenants: map[string]*Tenant{}, Stable: initial, PrevStable: initial, next: 1}
	for _, t := range tenants {
		tt := t
		tt.Running = initial
		s.Tenants[t.Name] = &tt
	}
	return s
}

func refuse(f string, a ...any) error { return fmt.Errorf("%w: %s", ErrRefused, fmt.Sprintf(f, a...)) }

// Register records a built release. Only gated (admitted) releases can ever be deployed.
func (s *State) Register(id string, gated bool) error {
	if _, dup := s.Releases[id]; dup {
		return refuse("%s is already registered", id)
	}
	if s.next == 0 {
		for _, r := range s.Releases {
			if r.Order >= s.next {
				s.next = r.Order + 1
			}
		}
	}
	s.Releases[id] = &Release{ID: id, Order: s.next, Gated: gated}
	s.next++
	return nil
}

func (s *State) deployable(id string) (*Release, error) {
	r, ok := s.Releases[id]
	switch {
	case !ok:
		return nil, refuse("unknown release %s", id)
	case !r.Gated:
		return nil, refuse("%s didn't pass admission (LAW-06)", id)
	case r.Revoked:
		return nil, refuse("%s is revoked", id)
	}
	return r, nil
}

// DeployTesting puts a testing-channel tenant on a release at or after stable.
func (s *State) DeployTesting(tenant, id string, now time.Time) error {
	t, ok := s.Tenants[tenant]
	if !ok || t.Channel != Testing || t.Draining {
		return refuse("%s isn't a serving testing tenant", tenant)
	}
	r, err := s.deployable(id)
	if err != nil {
		return err
	}
	if r.Order < s.Releases[s.Stable].Order {
		return refuse("%s is older than stable", id)
	}
	return s.commit(func() {
		t.Running = id
		if t.Ring == 0 && r.Ring0At.IsZero() {
			r.Ring0At = now
		}
		if t.Ring == 1 && r.Ring1At.IsZero() {
			r.Ring1At = now
		}
	})
}

// Promote makes id the new stable. The promote guard: every stable tenant runs stable.
func (s *State) Promote(id string, now time.Time) error {
	r, err := s.deployable(id)
	if err != nil {
		return err
	}
	if r.Order <= s.Releases[s.Stable].Order {
		return refuse("%s isn't newer than stable %s", id, s.Stable)
	}
	if r.Ring0At.IsZero() || r.Ring1At.IsZero() || now.Sub(r.Ring0At) < Soak || now.Sub(r.Ring1At) < Soak {
		return refuse("%s hasn't soaked 48h on ring 0 and ring 1", id)
	}
	for _, t := range s.sorted() {
		if t.Channel == Stable && t.Running != s.Stable {
			return refuse("promote guard: %s runs %s, not current stable %s (LAW-08)", t.Name, t.Running, s.Stable)
		}
	}
	return s.commit(func() {
		s.PrevStable, s.Stable = s.Stable, id
		r.Promoted = now
	})
}

// DeployStable moves a serving stable tenant to the current stable.
func (s *State) DeployStable(tenant string) error {
	t, ok := s.Tenants[tenant]
	if !ok || t.Channel != Stable || t.Draining || t.Running == s.Stable {
		return refuse("%s needs no stable deploy", tenant)
	}
	return s.commit(func() { t.Running = s.Stable })
}

// Rollback revokes stable and restores the previous stable at once; cells on the revoked
// release drain until redeployed (BUG-0001).
func (s *State) Rollback() (revoked string, err error) {
	if s.Releases[s.PrevStable].Order >= s.Releases[s.Stable].Order {
		return "", refuse("there's no previous stable to roll back to")
	}
	bad := s.Stable
	return bad, s.commit(func() {
		s.Releases[bad].Revoked = true
		s.Stable = s.PrevStable
		for _, t := range s.Tenants {
			if t.Running == bad {
				t.Draining = true
			}
		}
	})
}

// Redeploy moves a draining tenant to the current stable.
func (s *State) Redeploy(tenant string) error {
	t, ok := s.Tenants[tenant]
	if !ok || !t.Draining {
		return refuse("%s isn't draining", tenant)
	}
	return s.commit(func() { t.Running, t.Draining = s.Stable, false })
}

// RevokeTesting revokes a release that is only on the testing channel.
func (s *State) RevokeTesting(id string) error {
	r, err := s.deployable(id)
	if err != nil {
		return err
	}
	if r.Order <= s.Releases[s.Stable].Order {
		return refuse("%s isn't testing-only; roll back instead", id)
	}
	return s.commit(func() {
		r.Revoked = true
		for _, t := range s.Tenants {
			if t.Running == id {
				t.Draining = true
			}
		}
	})
}

// Violations lists the Rollout.tla invariants the state breaks.
func (s *State) Violations() []string {
	var out []string
	stable := s.Releases[s.Stable]
	for _, t := range s.sorted() {
		r := s.Releases[t.Running]
		if r == nil || !r.Gated {
			out = append(out, "OnlyGatedRuns:"+t.Name)
			continue
		}
		if r.Revoked && !t.Draining {
			out = append(out, "RevokedOnlyWhileDraining:"+t.Name)
		}
		if t.Draining && !r.Revoked {
			out = append(out, "DrainingOnlyRevoked:"+t.Name)
		}
		if t.Channel == Stable && !t.Draining {
			if t.Running != s.Stable && t.Running != s.PrevStable {
				out = append(out, "OneBehindMax:"+t.Name)
			}
			if r.Order > stable.Order {
				out = append(out, "StableNotAhead:"+t.Name)
			}
		}
	}
	if stable.Revoked {
		out = append(out, "StableNotRevoked")
	}
	return out
}

// commit applies a move on a copy and keeps it only if no invariant breaks.
func (s *State) commit(move func()) error {
	before := s.clone()
	move()
	if v := s.Violations(); len(v) > 0 {
		*s = *before
		return refuse("the move would break %v", v)
	}
	return nil
}

func (s *State) clone() *State {
	c := &State{Releases: map[string]*Release{}, Tenants: map[string]*Tenant{}, Stable: s.Stable, PrevStable: s.PrevStable, next: s.next}
	for k, v := range s.Releases {
		vv := *v
		c.Releases[k] = &vv
	}
	for k, v := range s.Tenants {
		vv := *v
		c.Tenants[k] = &vv
	}
	return c
}

func (s *State) sorted() []*Tenant {
	out := make([]*Tenant, 0, len(s.Tenants))
	for _, t := range s.Tenants {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// RollbackTarget is the bound for a rollback drill (sb-release(1)).
const RollbackTarget = 10 * time.Minute

// DrillReport is one rollback drill's outcome.
type DrillReport struct {
	Revoked  string
	Restored string
	Cells    int
	Took     time.Duration
	Failed   []string // tenants whose redeploy failed
	Passed   bool
}

// Drill rolls back and redeploys every draining cell through deploy, timing it against
// RollbackTarget (SB-508). A failed redeploy leaves that cell draining, never serving the
// revoked release.
func (s *State) Drill(deploy func(tenant, release string) error, now func() time.Time) (DrillReport, error) {
	start := now()
	bad, err := s.Rollback()
	if err != nil {
		return DrillReport{}, err
	}
	rep := DrillReport{Revoked: bad, Restored: s.Stable}
	for _, t := range s.sorted() {
		if !t.Draining {
			continue
		}
		rep.Cells++
		if err := deploy(t.Name, s.Stable); err != nil {
			rep.Failed = append(rep.Failed, t.Name)
			continue
		}
		if err := s.Redeploy(t.Name); err != nil {
			rep.Failed = append(rep.Failed, t.Name)
		}
	}
	rep.Took = now().Sub(start)
	rep.Passed = len(rep.Failed) == 0 && rep.Took < RollbackTarget
	return rep, nil
}
