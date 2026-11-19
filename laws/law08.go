package laws

import (
	"fmt"
	"math/rand"
)

// LAW-08 (SB-507, SB-508): no stable-channel tenant that is serving runs a release more
// than one behind the current stable, whatever mix of builds, deploys, promotions,
// rollbacks and revocations happens; and nothing ungated ever runs. The controller must
// keep Rollout.tla's invariants after every move it accepts.

// RolloutWorld drives the release controller.
type RolloutWorld interface {
	Build(gated bool) (id string, err error)
	DeployTesting(tenant, id string) error // refusals are errors the law ignores
	Promote(id string, soaked bool) error
	DeployStable(tenant string) error
	Rollback() error
	Redeploy(tenant string) error
	RevokeTesting(id string) error
	// View: the current stable and previous stable, what each tenant runs, who is draining,
	// and which releases are gated.
	View() (stable, prev string, running map[string]string, draining map[string]bool, gated map[string]bool, order map[string]int)
}

// NewRolloutWorld builds the system under test.
var NewRolloutWorld = func(stable, testing []string) RolloutWorld { return pendingRollout{} }

type pendingRollout struct{}

func (pendingRollout) Build(bool) (string, error) {
	return "", fmt.Errorf("%w: the release controller arrives with SB-507", ErrPending)
}
func (pendingRollout) DeployTesting(string, string) error { return nil }
func (pendingRollout) Promote(string, bool) error         { return nil }
func (pendingRollout) DeployStable(string) error          { return nil }
func (pendingRollout) Rollback() error                    { return nil }
func (pendingRollout) Redeploy(string) error              { return nil }
func (pendingRollout) RevokeTesting(string) error         { return nil }
func (pendingRollout) View() (string, string, map[string]string, map[string]bool, map[string]bool, map[string]int) {
	return "", "", nil, nil, nil, nil
}

var (
	stableTenants  = []string{"acme-logistics", "gamma-parts"}
	testingTenants = []string{"internal", "beta-freight"}
)

func law08(r *rand.Rand, w RolloutWorld) error {
	var ids []string
	first, err := w.Build(true)
	if err != nil {
		return err
	}
	ids = append(ids, first)
	all := append(append([]string{}, stableTenants...), testingTenants...)
	pick := func(xs []string) string { return xs[r.Intn(len(xs))] }
	for step := 0; step < 40; step++ {
		switch r.Intn(8) {
		case 0:
			id, err := w.Build(r.Intn(3) != 0)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		case 1:
			_ = w.DeployTesting(pick(testingTenants), pick(ids))
		case 2:
			_ = w.Promote(pick(ids), r.Intn(4) != 0)
		case 3, 4:
			_ = w.DeployStable(pick(stableTenants))
		case 5:
			if r.Intn(3) == 0 {
				_ = w.Rollback()
			}
		case 6:
			_ = w.Redeploy(pick(all))
		default:
			if r.Intn(3) == 0 {
				_ = w.RevokeTesting(pick(ids))
			}
		}
		stable, prev, running, draining, gated, order := w.View()
		for _, t := range all {
			if !gated[running[t]] {
				return fmt.Errorf("step %d: %s runs %s, which never passed the gate (LAW-06)", step, t, running[t])
			}
		}
		for _, t := range stableTenants {
			if draining[t] {
				continue
			}
			if running[t] != stable && running[t] != prev {
				return fmt.Errorf("step %d: stable tenant %s runs %s; stable is %s, previous %s (LAW-08)", step, t, running[t], stable, prev)
			}
			if order[running[t]] > order[stable] {
				return fmt.Errorf("step %d: stable tenant %s runs %s, ahead of stable %s", step, t, running[t], stable)
			}
		}
	}
	return nil
}
