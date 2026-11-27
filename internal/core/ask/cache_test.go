package ask_test

import (
	"context"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
)

// SB-5604, SB-5605: the system prompt, the cached prefix, holds no question, passage or
// memory and is the same for two asks; a routine's call may batch, an ask's never does.
func TestStablePrefixAndBatching(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	ctx := context.Background()
	if _, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Ask(ask.WithFeature(ctx, ask.FeatureRoutine), scope(t, "alice"), ask.Request{Question: "what is the refund policy?"}); err != nil {
		t.Fatal(err)
	}
	first, second := m.got[0], m.got[1]
	if !first.CacheSystem || first.System != second.System {
		t.Fatalf("the prefix differs or isn't cached: %v", first.CacheSystem)
	}
	for _, leak := range []string{"refund limit", "refund policy", "refund limit is 500"} {
		if strings.Contains(first.System, leak) {
			t.Fatalf("the cached prefix holds %q", leak)
		}
	}
	if first.Batch || !second.Batch {
		t.Fatalf("batch: ask %v, routine %v", first.Batch, second.Batch)
	}
}
