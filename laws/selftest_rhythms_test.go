package laws

import (
	"math/rand"
	"testing"
)

// LAW-29's test goes red when a rhythm keeps running after its subject leaves, and when a
// rhythm answers its own steps.
func TestSelf_LAW29CatchesARhythmThatActsOrOutlivesItsSubject(t *testing.T) {
	for name, sys := range map[string]RealRhythms{"ignores leaving": {IgnoreLeave: true}, "answers itself": {AutoReply: true}} {
		dir := t.TempDir()
		caught := false
		for seed := int64(1); seed <= 30 && !caught; seed++ {
			caught = law29(rand.New(rand.NewSource(seed)), dir, sys) != nil
		}
		if !caught {
			t.Errorf("LAW-29 stayed green with a rhythm that %s", name)
		}
	}
}
