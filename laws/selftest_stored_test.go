package laws

import (
	"math/rand"
	"testing"
)

// LAW-26's test goes red when a stored report is shown as it was stored.
func TestSelf_LAW26CatchesAStaleReport(t *testing.T) {
	caught := false
	for seed := int64(1); seed <= runs && !caught; seed++ {
		caught = law26(rand.New(rand.NewSource(seed)), &storedCell{stale: true}) != nil
	}
	if !caught {
		t.Fatal("LAW-26 stayed green with reports shown as they were stored")
	}
}
