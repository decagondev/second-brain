package laws

import (
	"math/rand"
	"testing"
)

// LAW-27's test goes red when a consulting ask consults every team, member or not.
func TestSelf_LAW27CatchesAWideConsultation(t *testing.T) {
	caught := false
	for seed := int64(1); seed <= runs && !caught; seed++ {
		caught = law27(rand.New(rand.NewSource(seed)), &consultCell{wide: true}) != nil
	}
	if !caught {
		t.Fatal("LAW-27 stayed green with every team consulted")
	}
}
