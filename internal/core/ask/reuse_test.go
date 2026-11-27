package ask_test

import (
	"context"
	"testing"

	"example.com/second-brain/internal/core/ask"
)

type reuses map[string]ask.ReuseEntry

func (r reuses) PutReuse(_ context.Context, e ask.ReuseEntry) error { r[e.Key] = e; return nil }
func (r reuses) Reuse(_ context.Context, _, key string) (ask.ReuseEntry, bool, error) {
	e, ok := r[key]
	return e, ok, nil
}

// SB-5603: the same question reuses the earlier answer without a model call, says so, and
// Ask fresh asks again; a changed document answers fresh.
func TestAnsweredBeforeCheckedNow(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	a.Reuse = reuses{}
	ctx := context.Background()
	first, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "Refund limit?"})
	if err != nil || first.Reused != "" {
		t.Fatalf("first: %+v %v", first, err)
	}
	again, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund   LIMIT"})
	if err != nil || again.Reused == "" || len(m.got) != 1 || again.Trace == first.Trace || len(again.Answer.Claims) != 1 {
		t.Fatalf("reuse: %+v %v calls %d", again, err, len(m.got))
	}
	if fresh, _ := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "Refund limit?", Fresh: true}); fresh.Reused != "" || len(m.got) != 2 {
		t.Fatalf("ask fresh reused: %+v", fresh)
	}
	ix := a.Index.(index)
	ix[0].Version, ix[0].Cite = 3, "docs:refunds@3"
	a.Index = ix
	if changed, _ := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "Refund limit?"}); changed.Reused != "" {
		t.Fatalf("a changed document was reused: %+v", changed)
	}
}
