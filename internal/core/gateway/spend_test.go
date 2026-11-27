package gateway

import (
	"context"
	"math"
	"testing"
	"time"

	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/ports"
)

type batcher struct {
	model
	submitted, collected int
	fail                 int
}

func (b *batcher) SubmitBatch(context.Context, ports.ModelRequest) (string, error) {
	b.submitted++
	return "b1", nil
}

func (b *batcher) CollectBatch(context.Context, string) (ports.ModelResponse, bool, error) {
	b.collected++
	if b.fail > 0 {
		b.fail--
		return ports.ModelResponse{}, true, ErrUnavailable
	}
	return ports.ModelResponse{Text: "batched", Usage: ports.Usage{Input: 1000, Output: 1000}}, true, nil
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// SB-5604 to SB-5606: a batchable call goes through the batch API at half price and says
// what that saved; a failed batch is tried once more; an ask never batches; cached input is
// charged at the cached price; the small tier says what it saved against the main one.
func TestBatchCacheAndTierSavings(t *testing.T) {
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	guard := func(p budget.Price) *budget.Guard {
		return &budget.Guard{Tenant: "acme-logistics", MonthlyUSD: 1e6, Price: p, Meter: &budget.MemoryMeter{}, Now: func() time.Time { return now }}
	}
	b := &batcher{fail: 1}
	g := Gateway{Model: b, Secrets: scrub{}, Budget: guard(budget.Price{InputPerM: 1000, OutputPerM: 1000}), Sleep: func(context.Context, time.Duration) error { return nil }}
	ctx, tally := budget.WithTally(context.Background())
	resp, err := g.Complete(ctx, ports.ModelRequest{Batch: true})
	if err != nil || resp.Text != "batched" || b.submitted != 2 || b.calls != 0 {
		t.Fatalf("batch: %+v %v submitted %d calls %d", resp, err, b.submitted, b.calls)
	}
	if !near(tally.USD(), 1.0) || !near(tally.Saved()["batch"], 1.0) {
		t.Fatalf("batch cost %v saved %v", tally.USD(), tally.Saved())
	}
	if _, err := g.Complete(context.Background(), ports.ModelRequest{}); err != nil || b.calls != 1 || b.submitted != 2 {
		t.Fatal("an ask was batched")
	}

	cached := &model{}
	cached.usage = ports.Usage{Input: 1000, Cached: 800, Output: 0}
	g = Gateway{Model: cached, Secrets: scrub{}, Budget: guard(budget.Price{InputPerM: 1000, CachedPerM: 100})}
	ctx, tally = budget.WithTally(context.Background())
	if _, err := g.Complete(ctx, ports.ModelRequest{CacheSystem: true}); err != nil {
		t.Fatal(err)
	}
	if !near(tally.USD(), 0.2+0.08) || !near(tally.Saved()["cache"], 0.72) {
		t.Fatalf("cache cost %v saved %v", tally.USD(), tally.Saved())
	}

	main := budget.Price{InputPerM: 1000, OutputPerM: 1000}
	small := &model{}
	small.usage = ports.Usage{Input: 1000, Output: 1000}
	g = Gateway{Model: small, Secrets: scrub{}, Budget: guard(budget.Price{InputPerM: 100, OutputPerM: 100}), MainPrice: &main}
	ctx, tally = budget.WithTally(context.Background())
	if _, err := g.Complete(ctx, ports.ModelRequest{}); err != nil {
		t.Fatal(err)
	}
	if !near(tally.Saved()["tier"], 1.8) {
		t.Fatalf("tier saved %v", tally.Saved())
	}
}
