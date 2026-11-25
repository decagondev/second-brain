package ask

import (
	"context"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// smallStub is a model the test never calls.
type smallStub struct{ ports.Model }

// SB-4405: only a plain, short, first question, in a space that doesn't pin main, while the
// small tier isn't held, goes to the small model; an eval probe names its tier.
func TestTierFor(t *testing.T) {
	ctx := context.Background()
	a := Agent{}
	if got := a.tierFor(ctx, "spc_ops", Request{}, "q", 0, 0); got != "" {
		t.Fatalf("no small model: %q", got)
	}
	a = Agent{Small: smallStub{}, SmallHeld: func() bool { return false }, PinMain: func(s string) bool { return s == "spc_pinned" }}
	for _, c := range []struct {
		space string
		req   Request
		q     string
		docs  int
		prior int
		want  string
	}{
		{"spc_ops", Request{}, "When is the cutoff?", 0, 0, TierSmall},
		{"spc_ops", Request{Agent: "ask"}, "When is the cutoff?", 0, 0, TierSmall},
		{"spc_ops", Request{Agent: "incident-summary"}, "When is the cutoff?", 0, 0, TierMain},
		{"spc_ops", Request{File: &FileUpload{}}, "When is the cutoff?", 0, 0, TierMain},
		{"spc_ops", Request{Collection: "col_1"}, "When is the cutoff?", 0, 0, TierMain},
		{"spc_ops", Request{}, "When is the cutoff?", 1, 0, TierMain},
		{"spc_ops", Request{}, "When is the cutoff?", 0, 1, TierMain},
		{"spc_ops", Request{}, strings.Repeat("a", SmallMaxRunes+1), 0, 0, TierMain},
		{"spc_pinned", Request{}, "When is the cutoff?", 0, 0, TierMain},
	} {
		if got := a.tierFor(ctx, c.space, c.req, c.q, c.docs, c.prior); got != c.want {
			t.Errorf("%+v: %q", c, got)
		}
	}
	if got := a.tierFor(WithTier(ctx, TierSmall), "spc_pinned", Request{}, "q", 0, 0); got != TierSmall {
		t.Fatalf("a probe's tier: %q", got)
	}
	a.SmallHeld = func() bool { return true }
	if got := a.tierFor(ctx, "spc_ops", Request{}, "q", 0, 0); got != TierMain {
		t.Fatalf("held: %q", got)
	}
	a.SmallHeld = nil
	if got := a.tierFor(ctx, "spc_ops", Request{}, "q", 0, 0); got != TierMain {
		t.Fatalf("a hold no one can read: %q", got)
	}
}
