package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/ask"
)

// ignoredHold is an updater that never reads the hold.
type ignoredHold struct{}

func (ignoredHold) Hold(*ask.Hold) func(context.Context, string) (string, error) {
	return func(context.Context, string) (string, error) { return "", nil }
}

// The hold law passes on the cell's hold and catches an updater that ignores it.
func TestSelf_EvalHoldLaw(t *testing.T) {
	run := func(sys HoldSystem) func(r *rand.Rand) error {
		return func(r *rand.Rand) error {
			w, done := NewRuntimeWorld()
			defer done()
			return law06Evals(r, w.(*runtimeWorld), sys)
		}
	}
	if violates(30, run(RealHold{})) {
		t.Fatal("the real hold breaks the law")
	}
	if !violates(30, run(ignoredHold{})) {
		t.Error("an updater that ignores the hold was not caught")
	}
}
