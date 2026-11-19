package laws

import (
	"fmt"
	"time"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/rollout"
)

// The real release controller (SB-507, SB-508) under LAW-06 and LAW-08.
func init() {
	NewRolloutWorld = func(stable, testing []string) RolloutWorld { return newRolloutWorld(stable, testing, false) }
}

type rolloutWorld struct {
	s   *rollout.State
	n   int
	now time.Time
	// noGuard is a mutation: promotion skips the promote guard.
	noGuard bool
	// skipTenantCustody is a mutation: the guard ignores cells in a tenant's account.
	skipTenantCustody bool
}

// NewTenantCustodyRolloutWorld puts every other stable tenant in tenant custody (SB-707): they
// pull from their own accounts, and the law must hold for them exactly as for the others.
var NewTenantCustodyRolloutWorld = func(stable, testing []string) RolloutWorld {
	w := newRolloutWorld(stable, testing, false)
	for i, t := range stable {
		if i%2 == 0 {
			_ = w.s.SetCustody(t, cell.CustodyTenant)
		}
	}
	return w
}

func newRolloutWorld(stable, testing []string, noGuard bool) *rolloutWorld {
	var ts []rollout.Tenant
	for _, t := range stable {
		ts = append(ts, rollout.Tenant{Name: t, Channel: rollout.Stable, Ring: 2})
	}
	for i, t := range testing {
		ts = append(ts, rollout.Tenant{Name: t, Channel: rollout.Testing, Ring: i % 2}) // ring 0 internal, ring 1 friendly
	}
	return &rolloutWorld{s: rollout.New("rel_2027-01-01_000000", ts), now: time.Unix(1_800_000_000, 0), noGuard: noGuard}
}

func (w *rolloutWorld) Build(gated bool) (string, error) {
	w.n++
	id := fmt.Sprintf("rel_2027-01-01_%06x", w.n)
	return id, w.s.Register(id, gated)
}

func (w *rolloutWorld) DeployTesting(t, id string) error { return w.s.DeployTesting(t, id, w.now) }

func (w *rolloutWorld) Promote(id string, soaked bool) error {
	at := w.now
	if soaked {
		at = at.Add(rollout.Soak + time.Hour)
	}
	if w.skipTenantCustody {
		saved := map[string]string{}
		for name, t := range w.s.Tenants {
			if t.Channel == rollout.Stable && t.Custody == cell.CustodyTenant {
				saved[name], t.Running = t.Running, w.s.Stable
			}
		}
		err := w.s.Promote(id, at)
		for name, r := range saved {
			w.s.Tenants[name].Running = r
		}
		return err
	}
	if w.noGuard {
		// The mutation: make every stable tenant look current just for the check.
		saved := map[string]string{}
		for name, t := range w.s.Tenants {
			if t.Channel == rollout.Stable {
				saved[name], t.Running = t.Running, w.s.Stable
			}
		}
		err := w.s.Promote(id, at)
		for name, r := range saved {
			w.s.Tenants[name].Running = r
		}
		return err
	}
	return w.s.Promote(id, at)
}

func (w *rolloutWorld) DeployStable(t string) error   { return w.s.DeployStable(t) }
func (w *rolloutWorld) Rollback() error               { _, err := w.s.Rollback(); return err }
func (w *rolloutWorld) Redeploy(t string) error       { return w.s.Redeploy(t) }
func (w *rolloutWorld) RevokeTesting(id string) error { return w.s.RevokeTesting(id) }

func (w *rolloutWorld) View() (string, string, map[string]string, map[string]bool, map[string]bool, map[string]int) {
	running, draining, gated, order := map[string]string{}, map[string]bool{}, map[string]bool{}, map[string]int{}
	for name, t := range w.s.Tenants {
		running[name], draining[name] = t.Running, t.Draining
	}
	for id, r := range w.s.Releases {
		gated[id], order[id] = r.Gated, r.Order
	}
	return w.s.Stable, w.s.PrevStable, running, draining, gated, order
}
