package laws

import (
	"math/rand"
	"testing"
)

func TestLAW06_EveryTemplateSuiteGatesTheRelease(t *testing.T) {
	Check(t, "LAW-06", 20, func(r *rand.Rand) error { return law06Templates(r, realGate) })
}

// LAW-06's template test catches a gate that leaves one template's suite out.
func TestSelf_LAW06TemplateGateMutationIsCaught(t *testing.T) {
	if !violates(20, func(r *rand.Rand) error { return law06Templates(r, dropOne(r)) }) {
		t.Error("LAW-06 (templates): a gate missing a template suite was not caught")
	}
}
