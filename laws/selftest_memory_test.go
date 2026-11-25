package laws

import (
	"math/rand"
	"testing"
)

// LAW-25's test goes red when everyone's memory is sent with everyone's ask.
func TestSelf_LAW25CatchesASharedMemory(t *testing.T) {
	caught := false
	for seed := int64(1); seed <= runs && !caught; seed++ {
		caught = law25(rand.New(rand.NewSource(seed)), sharedMemory{}) != nil
	}
	if !caught {
		t.Fatal("LAW-25 stayed green with every memory sent with every ask")
	}
}
