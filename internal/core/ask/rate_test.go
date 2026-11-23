package ask_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

// SB-2305: a burst of ten asks gets six answers and four ERATEs with the seconds to wait;
// someone else isn't held up, search isn't limited, and the refusals are traced with no
// question.
func TestAsksPerMinute(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	a.Rate = &ask.Rate{}
	ctx := context.Background()
	alice := scope(t, "alice")
	answers, limited := 0, []*ask.RateError{}
	for i := 0; i < 10; i++ {
		now = now.Add(time.Second)
		_, err := a.Ask(ctx, alice, ask.Request{Question: "refund limit?"})
		var re *ask.RateError
		switch {
		case err == nil:
			answers++
		case errors.As(err, &re) && errors.Is(err, ask.ErrRate):
			limited = append(limited, re)
		default:
			t.Fatal(err)
		}
	}
	if answers != 6 || len(limited) != 4 || len(m.got) != 6 {
		t.Fatalf("%d answers, %d refused, %d model calls; want 6, 4, 6", answers, len(limited), len(m.got))
	}
	// The first ask was at 09:00:01; at 09:00:10 it is a minute old in 51 seconds.
	if w := limited[len(limited)-1].Wait; w != 51*time.Second {
		t.Fatalf("wait %v, want 51s", w)
	}
	tr, _, _ := a.Store.GetTrace(ctx, "acme-logistics", limited[0].Trace)
	if tr.Response.Agent != ask.AgentLimited || tr.User != "alice" || tr.QuerySHA256 != "" || len(tr.Decisions) != 0 {
		t.Fatalf("the refusal's trace: %+v", tr)
	}
	if _, err := a.Ask(ctx, scope(t, "bob"), ask.Request{Question: "refund limit?"}); err != nil {
		t.Fatalf("bob was held up by alice: %v", err)
	}
	if _, err := a.Search(ctx, alice, ask.SearchRequest{Query: "refund"}); err != nil {
		t.Fatalf("search was limited: %v", err)
	}
	now = now.Add(51 * time.Second)
	if _, err := a.Ask(ctx, alice, ask.Request{Question: "refund limit?"}); err != nil {
		t.Fatalf("a minute on: %v", err)
	}
	a.AsksPerMinute = func() int { return 1 }
	if _, err := a.Ask(ctx, alice, ask.Request{Question: "refund limit?"}); !errors.Is(err, ask.ErrRate) {
		t.Fatalf("the manifest's limit: %v", err)
	}
}
