package laws

import (
	"math/rand"
	"testing"
)

// LAW-24's test goes red when a department's branch takes in every team's members.
func TestSelf_LAW24CatchesAWideTree(t *testing.T) {
	caught := false
	for seed := int64(1); seed <= runs && !caught; seed++ {
		sys := &treeSpaces{askSpaces: newAskSpaces(), wide: true}
		caught = law24(rand.New(rand.NewSource(seed)), sys) != nil
	}
	if !caught {
		t.Fatal("LAW-24 stayed green with branches open to every team")
	}
}
