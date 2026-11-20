package laws

import (
	"math/rand"
	"testing"
)

// The LAW-22 law test is not vacuous: real drafts, actions and approvals are allowed.
func TestSelf_BarrierLawAllowsSomething(t *testing.T) {
	barrierCounts.drafts, barrierCounts.acts, barrierCounts.approvals = 0, 0, 0
	for seed := int64(1); seed <= runs; seed++ {
		if err := law22(rand.New(rand.NewSource(seed)), NewBarrierSystem()); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
	}
	if barrierCounts.drafts == 0 || barrierCounts.acts == 0 || barrierCounts.approvals == 0 {
		t.Fatalf("drafts %d, actions %d, approvals %d allowed: the barrier law test checked nothing for one of them",
			barrierCounts.drafts, barrierCounts.acts, barrierCounts.approvals)
	}
}

// The LAW-22 law test catches a cell that ignores ownership, gives the bus no target
// facts, or gives it no approval facts.
func TestSelf_BarrierMutationsAreCaught(t *testing.T) {
	for name, mk := range map[string]func() SpaceSystem{
		"ignore owners":     func() SpaceSystem { b := newBarrierSpaces(); b.noOwners = true; return b },
		"no target facts":   func() SpaceSystem { b := newBarrierSpaces(); b.noTarget = true; return b },
		"no approval facts": func() SpaceSystem { b := newBarrierSpaces(); b.noApprove = true; return b },
	} {
		if !violates(runs, func(r *rand.Rand) error { return law22(r, mk()) }) {
			t.Errorf("LAW-22: a cell with %s was not caught", name)
		}
	}
}
