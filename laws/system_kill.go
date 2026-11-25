package laws

import (
	"context"
	"strings"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	killmem "example.com/second-brain/internal/adapters/kill/memory"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
)

// The real kill switches (SB-403): the audited control, the memory store, and one cell's
// checker polling every 2 seconds on a fake clock.
func init() {
	NewKillWorld = func() KillWorld { return newKillWorld(false) }
}

type killWorld struct {
	now      time.Time
	store    *killmem.Store
	ctl      kill.Control
	checker  *kill.Checker
	sincePol time.Duration
	// stale is a mutation: the checker loads the list once and never again.
	stale  bool
	loaded bool
}

func newKillWorld(stale bool) *killWorld {
	w := &killWorld{now: time.Unix(1_800_000_000, 0), store: killmem.New(), stale: stale}
	clock := func() time.Time { return w.now }
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_law" }, Store: auditmem.New(), Now: clock}
	w.ctl = kill.Control{Store: w.store, Audit: log, Now: clock}
	w.checker = &kill.Checker{Tenant: "acme-logistics", Store: w.store, Now: clock}
	return w
}

func (w *killWorld) Kill(scope string) error {
	return w.ctl.Kill(context.Background(), "acme-logistics", scope, "law-10", "usr_operator")
}

func (w *killWorld) Unkill(scope string) error {
	return w.ctl.Unkill(context.Background(), "acme-logistics", scope, "law-10", "usr_operator")
}

func (w *killWorld) Advance(d time.Duration) {
	for d > 0 {
		step := kill.Poll - w.sincePol
		if step > d {
			w.now = w.now.Add(d)
			w.sincePol += d
			return
		}
		w.now = w.now.Add(step)
		d -= step
		w.sincePol = 0
		if !w.stale || !w.loaded {
			if w.checker.Refresh(context.Background()) == nil {
				w.loaded = true
			}
		}
	}
}

func (w *killWorld) StoreDown(down bool) { w.store.SetDown(down) }

func (w *killWorld) Allowed(agent, connector string, consequential bool) (bool, error) {
	return w.AllowedIn(agent, "", connector, consequential)
}

// AllowedIn is one agent's run in one space (SB-4702).
func (w *killWorld) AllowedIn(agent, space, connector string, consequential bool) (bool, error) {
	err := w.checker.Check(kill.Op{Agent: agent, Space: space, Connector: connector, Consequential: consequential})
	if err != nil && !strings.Contains(err.Error(), "EKILLED") {
		return false, err
	}
	return err == nil, nil
}
