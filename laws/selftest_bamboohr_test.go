package laws

import (
	"math/rand"
	"testing"
)

func TestLAW03_BambooHRUnknownIsNeverAllow(t *testing.T) {
	w := NewBambooWorld()
	Check(t, "LAW-03", 30, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW07_BambooHRSensitiveFieldsOnlyWhenAllowed(t *testing.T) {
	Check(t, "LAW-07", 40, func(r *rand.Rand) error { return law07BambooHR(r, allowFromSpaces) })
}

// The BambooHR law test catches a connector that reads every sensitive field.
func TestSelf_BambooHRMutationIsCaught(t *testing.T) {
	greedy := func([]string) func() []string {
		return func() []string { return []string{"payRate", "ssn", "dateOfBirth", "homePhone"} }
	}
	if !violates(40, func(r *rand.Rand) error { return law07BambooHR(r, greedy) }) {
		t.Error("LAW-07 (bamboohr): a connector that reads every sensitive field was not caught")
	}
}

func TestLAW01_BambooHRAccessIsLive(t *testing.T) {
	Check(t, "LAW-01", 20, law01BambooHR)
}
