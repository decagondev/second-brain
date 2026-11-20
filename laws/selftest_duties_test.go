package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/toolbus"
)

func TestLAW15_SegregationOfDuties(t *testing.T) {
	dutyRuns.one, dutyRuns.two = 0, 0
	Check(t, "LAW-15", runs, func(r *rand.Rand) error { return law15Duties(r, RealDutyBus) })
	if dutyRuns.one == 0 || dutyRuns.two == 0 {
		t.Fatalf("actions ran after one approval %d times and after two %d times: the law test checked nothing for one of them", dutyRuns.one, dutyRuns.two)
	}
}

// The duty law test catches a bus that ignores reporting lines, and one without the duty.
func TestSelf_DutyMutationsAreCaught(t *testing.T) {
	noLines := func(map[string]string) (*toolbus.Bus, error) { return dutyBus() }
	noDuty := func(m map[string]string) (*toolbus.Bus, error) {
		b, err := RealDutyBus(m)
		if err == nil {
			tool := b.Tools["update"]
			tool.Duty = nil
			b.Tools["update"] = tool
		}
		return b, err
	}
	for name, mk := range map[string]DutyBus{"no reporting lines": noLines, "no duty": noDuty} {
		if !violates(runs, func(r *rand.Rand) error { return law15Duties(r, mk) }) {
			t.Errorf("LAW-15 (duties): a bus with %s was not caught", name)
		}
	}
}
