package budget

import (
	"context"
	"testing"
)

// SB-4201: a report's calls add up in its own tally; a context without one counts nothing.
func TestTallyCountsOneContextsCalls(t *testing.T) {
	p := Price{InputPerM: 3, OutputPerM: 15}
	ctx, tl := WithTally(context.Background())
	TallyOf(ctx).Add(p, 1_000_000, 0)
	TallyOf(ctx).Add(p, 0, 100_000)
	if got := tl.USD(); got < 4.499 || got > 4.501 {
		t.Fatalf("tally %v, want 4.5", got)
	}
	TallyOf(context.Background()).Add(p, 1_000_000, 0) // no tally: a no-op
	if TallyOf(context.Background()).USD() != 0 {
		t.Fatal("a context without a tally counted")
	}
}
