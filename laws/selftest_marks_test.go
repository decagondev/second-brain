package laws

import (
	"math/rand"
	"testing"
)

// LAW-28's test goes red when a note is published without its mark.
func TestSelf_LAW28CatchesAnUnmarkedNote(t *testing.T) {
	dir := t.TempDir()
	caught := false
	for seed := int64(1); seed <= 30 && !caught; seed++ {
		caught = law28(rand.New(rand.NewSource(seed)), dir, RealMarks{Strip: true}) != nil
	}
	if !caught {
		t.Fatal("LAW-28 stayed green with every note published unmarked")
	}
}
