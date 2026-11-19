package budget

import (
	"context"
	"errors"
	"testing"
	"time"
)

type counts map[string]float64

func (c counts) Add(_, _ string, l map[string]string, v float64) { c[l["reason"]] += v }

func TestGuard(t *testing.T) {
	now := time.Date(2027, 3, 31, 23, 0, 0, 0, time.UTC)
	m, c := &MemoryMeter{}, counts{}
	g := &Guard{Tenant: "acme-logistics", MonthlyUSD: 10, Price: Price{InputPerM: 3, OutputPerM: 15}, Meter: m, Metrics: c, Now: func() time.Time { return now }}
	ctx := context.Background()
	if err := g.Allow(ctx, "u1", 1000); err != nil {
		t.Fatal(err)
	}
	_ = g.Record(ctx, 1_000_000, 400_000) // $3 + $6 = $9
	if err := g.Allow(ctx, "u1", 100_000); !errors.Is(err, ErrBudget) || c["budget"] != 1 {
		t.Fatalf("worst case over budget: %v %v", err, c)
	}
	now = now.Add(2 * time.Hour) // a new month
	if err := g.Allow(ctx, "u1", 100_000); err != nil {
		t.Fatalf("new month: %v", err)
	}
	for i := 0; i < PerUserPerMinute; i++ {
		_ = g.Allow(ctx, "u2", 10)
	}
	if err := g.Allow(ctx, "u2", 10); !errors.Is(err, ErrBudget) || c["rate"] != 1 {
		t.Fatalf("rate limit: %v", err)
	}
	if err := g.Allow(ctx, "u3", 10); err != nil {
		t.Fatalf("another user is not limited: %v", err)
	}
	now = now.Add(time.Minute)
	if err := g.Allow(ctx, "u2", 10); err != nil {
		t.Fatalf("refill: %v", err)
	}
	m.Fail = true
	if err := g.Allow(ctx, "u4", 10); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("meter down must refuse: %v", err)
	}
}
