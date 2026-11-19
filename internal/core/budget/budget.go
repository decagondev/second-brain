// Package budget is the monthly model budget and the per-user rate limit (sb-observability(7),
// Budgets and rate limits; SB-609). Both are checked before a model call; over either, the
// call is refused with EBUDGET, never made unmetered.
package budget

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// PerUserPerMinute is the rate limit.
const PerUserPerMinute = 30

// Price is per million tokens, from the release's price table.
type Price struct{ InputPerM, OutputPerM float64 }

// Meter keeps a tenant's spend per calendar month (UTC), e.g. "2027-03".
type Meter interface {
	Spent(ctx context.Context, tenant, month string) (float64, error)
	Add(ctx context.Context, tenant, month string, usd float64) error
}

// Errors.
var (
	ErrBudget      = errors.New("budget: EBUDGET")
	ErrUnavailable = errors.New("budget: EUNAVAIL meter unreadable")
)

// Counter counts refusals.
type Counter interface {
	Add(name, help string, labels map[string]string, v float64)
}

// Guard is one cell's budget and rate limit.
type Guard struct {
	Tenant     string
	MonthlyUSD float64
	Price      Price
	Meter      Meter
	Metrics    Counter
	Now        func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	at     time.Time
}

func (g *Guard) month() string { return g.Now().UTC().Format("2006-01") }

// Allow checks the rate limit and the budget for one call of up to maxTokens output.
func (g *Guard) Allow(ctx context.Context, user string, maxTokens int) error {
	if !g.take(user) {
		g.count("rate")
		return fmt.Errorf("%w: rate limit (%d calls a minute)", ErrBudget, PerUserPerMinute)
	}
	spent, err := g.Meter.Spent(ctx, g.Tenant, g.month())
	if err != nil {
		return ErrUnavailable // no meter, no call
	}
	worst := float64(maxTokens) / 1e6 * g.Price.OutputPerM
	if spent+worst > g.MonthlyUSD {
		g.count("budget")
		return fmt.Errorf("%w: monthly budget of $%.0f reached", ErrBudget, g.MonthlyUSD)
	}
	return nil
}

// Record meters a finished call from the provider's reported usage.
func (g *Guard) Record(ctx context.Context, input, output int) error {
	usd := float64(input)/1e6*g.Price.InputPerM + float64(output)/1e6*g.Price.OutputPerM
	return g.Meter.Add(ctx, g.Tenant, g.month(), usd)
}

func (g *Guard) take(user string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.buckets == nil {
		g.buckets = map[string]*bucket{}
	}
	now := g.Now()
	b, ok := g.buckets[user]
	if !ok {
		b = &bucket{tokens: PerUserPerMinute, at: now}
		g.buckets[user] = b
	}
	b.tokens += now.Sub(b.at).Minutes() * PerUserPerMinute
	if b.tokens > PerUserPerMinute {
		b.tokens = PerUserPerMinute
	}
	b.at = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (g *Guard) count(reason string) {
	if g.Metrics != nil {
		g.Metrics.Add("sb_budget_refusals_total", "Model calls refused by the budget or rate limit.", map[string]string{"tenant": g.Tenant, "reason": reason}, 1)
	}
}

// MemoryMeter is an in-process Meter (development and tests).
type MemoryMeter struct {
	mu   sync.Mutex
	m    map[string]float64
	Fail bool
}

// Spent implements Meter.
func (m *MemoryMeter) Spent(_ context.Context, tenant, month string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Fail {
		return 0, errors.New("meter down")
	}
	return m.m[tenant+"/"+month], nil
}

// Add implements Meter.
func (m *MemoryMeter) Add(_ context.Context, tenant, month string, usd float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.m == nil {
		m.m = map[string]float64{}
	}
	m.m[tenant+"/"+month] += usd
	return nil
}
