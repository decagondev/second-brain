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

type once struct{ seen map[string]bool }

func (o *once) First(_ context.Context, key string) (bool, error) {
	if o.seen == nil {
		o.seen = map[string]bool{}
	}
	if o.seen[key] {
		return false, nil
	}
	o.seen[key] = true
	return true, nil
}

// SB-1608: one budget_warning a month, from the call that passes 80%, before EBUDGET; a
// new month warns again.
func TestBudgetWarningOnceAMonth(t *testing.T) {
	now := time.Date(2027, 3, 1, 9, 0, 0, 0, time.UTC)
	var warned []string
	o := &once{}
	g := &Guard{Tenant: "acme-logistics", MonthlyUSD: 100, Price: Price{InputPerM: 1e6, OutputPerM: 0}, Meter: &MemoryMeter{},
		Now: func() time.Time { return now }, Once: o, Warn: func(_ context.Context, month string, _ float64) { warned = append(warned, month) }}
	ctx := context.Background()
	for i := 0; i < 79; i++ { // $79
		if err := g.Record(ctx, 1, 0); err != nil {
			t.Fatal(err)
		}
	}
	if len(warned) != 0 {
		t.Fatalf("warned at 79%%: %v", warned)
	}
	for i := 0; i < 15; i++ { // $94: still under budget, warned once
		_ = g.Record(ctx, 1, 0)
	}
	if len(warned) != 1 || warned[0] != "2027-03" {
		t.Fatalf("warnings %v, want one for 2027-03", warned)
	}
	if err := g.Allow(ctx, "usr_a", 0); err != nil {
		t.Fatalf("answers still work after the warning: %v", err)
	}
	now = time.Date(2027, 4, 2, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 81; i++ {
		_ = g.Record(ctx, 1, 0)
	}
	if len(warned) != 2 || warned[1] != "2027-04" {
		t.Fatalf("a new month: %v", warned)
	}
	// A second guard (another service) over the same marker doesn't warn again.
	g2 := &Guard{Tenant: g.Tenant, MonthlyUSD: g.MonthlyUSD, Price: g.Price, Meter: g.Meter, Now: g.Now, Once: o, Warn: g.Warn}
	_ = g2.Record(ctx, 1, 0)
	if len(warned) != 2 {
		t.Fatalf("another service warned again: %v", warned)
	}
	// A budget of 0 never warns.
	z := &Guard{Tenant: "beta-freight", Meter: &MemoryMeter{}, Now: g.Now, Once: &once{}, Price: g.Price,
		Warn: func(context.Context, string, float64) { t.Fatal("a zero budget warned") }}
	_ = z.Record(ctx, 5, 0)
}

// SB-3505: a month-end forecast over the budget sends the one warning early, well under
// 80%; the forecast is read at most once an hour, and none (too early) warns nothing.
func TestForecastWarnsEarly(t *testing.T) {
	now := time.Date(2027, 3, 10, 9, 0, 0, 0, time.UTC)
	var warned []string
	reads, forecast, ok := 0, 90.0, true
	g := &Guard{Tenant: "acme-logistics", MonthlyUSD: 100, Price: Price{InputPerM: 1e6, OutputPerM: 0}, Meter: &MemoryMeter{},
		Now: func() time.Time { return now }, Once: &once{}, Warn: func(_ context.Context, month string, _ float64) { warned = append(warned, month) },
		Forecast: func(context.Context) (float64, bool) { reads++; return forecast, ok }}
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_ = g.Record(ctx, 1, 0)
	}
	if len(warned) != 0 || reads != 1 {
		t.Fatalf("a forecast under budget: warned %v, read %d times", warned, reads)
	}
	forecast = 130
	_ = g.Record(ctx, 1, 0)
	if len(warned) != 0 || reads != 1 {
		t.Fatalf("read again within the hour: %v, %d", warned, reads)
	}
	now = now.Add(time.Hour)
	_ = g.Record(ctx, 1, 0)
	if len(warned) != 1 || warned[0] != "2027-03" || reads != 2 {
		t.Fatalf("over budget by forecast: %v, %d", warned, reads)
	}
	now = now.Add(2 * time.Hour)
	_ = g.Record(ctx, 1, 0)
	if len(warned) != 1 {
		t.Fatalf("warned twice: %v", warned)
	}
	g2 := &Guard{Tenant: "beta-freight", MonthlyUSD: 100, Price: g.Price, Meter: &MemoryMeter{}, Now: g.Now, Once: &once{},
		Warn:     func(context.Context, string, float64) { t.Fatal("no forecast warned") },
		Forecast: func(context.Context) (float64, bool) { return 500, false }}
	_ = g2.Record(ctx, 1, 0)
}

// SB-4105: a call that ran in a space is metered for the tenant and for the space.
func TestRecordInMetersTheSpace(t *testing.T) {
	m := &MemoryMeter{}
	g := &Guard{Tenant: "acme-logistics", MonthlyUSD: 100, Price: Price{InputPerM: 1, OutputPerM: 2}, Meter: m,
		Now: func() time.Time { return time.Date(2027, 3, 2, 0, 0, 0, 0, time.UTC) }}
	ctx := context.Background()
	if err := g.RecordIn(ctx, "spc_finance", 1_000_000, 1_000_000); err != nil {
		t.Fatal(err)
	}
	if err := g.Record(ctx, 1_000_000, 0); err != nil {
		t.Fatal(err)
	}
	spent, _ := m.Spent(ctx, "acme-logistics", "2027-03")
	by, _ := m.SpentBySpace(ctx, "acme-logistics", "2027-03")
	if spent != 4 || len(by) != 1 || by["spc_finance"] != 3 {
		t.Fatalf("spent %v, by space %v", spent, by)
	}
}
