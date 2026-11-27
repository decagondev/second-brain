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
type Price struct{ InputPerM, OutputPerM, CachedPerM float64 }

// Cost is one call's dollars: cached input tokens at CachedPerM (InputPerM when unset), the
// rest at InputPerM, output at OutputPerM (SB-5605).
func (p Price) Cost(input, cached, output int) float64 {
	cp := p.CachedPerM
	if cp == 0 {
		cp = p.InputPerM
	}
	if cached > input {
		cached = input
	}
	return float64(input-cached)/1e6*p.InputPerM + float64(cached)/1e6*cp + float64(output)/1e6*p.OutputPerM
}

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
	// Warn is told once a month when spend passes WarnAt of the budget (SB-1608); Once
	// decides which call is the first, across every service and restart. Either nil sends
	// none.
	Warn func(ctx context.Context, month string, spent float64)
	Once Once
	// Forecast is the month-end forecast and whether there is one (SB-3505); a forecast
	// over the budget sends the warning early. It is read at most once an hour; nil reads none.
	Forecast func(ctx context.Context) (float64, bool)

	forecastAt time.Time

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

// WarnAt is the share of the monthly budget that sends the budget warning.
const WarnAt = 0.8

// Once reports whether this is the first time key is claimed; only one caller ever wins.
type Once interface {
	First(ctx context.Context, key string) (bool, error)
}

// Record meters a finished call from the provider's reported usage, and sends the month's
// budget warning if this call took spend past WarnAt of the budget.
func (g *Guard) Record(ctx context.Context, input, output int) error {
	return g.RecordIn(ctx, "", input, output)
}

// SpaceMeter keeps a tenant's spend per space and month too (SB-4105). A meter without it
// meters the tenant only.
type SpaceMeter interface {
	AddSpace(ctx context.Context, tenant, space, month string, usd float64) error
	SpentBySpace(ctx context.Context, tenant, month string) (map[string]float64, error)
}

// PersonMeter keeps a tenant's spend per person and month too (SB-5003). A meter without
// it limits no one.
type PersonMeter interface {
	AddPerson(ctx context.Context, tenant, user, month string, usd float64) error
	SpentByPerson(ctx context.Context, tenant, month string) (map[string]float64, error)
}

// RecordFor is RecordIn for a call made for user: with a PersonMeter, it is metered for them
// too (SB-5003).
func (g *Guard) RecordFor(ctx context.Context, space, user string, input, output int) error {
	if err := g.RecordIn(ctx, space, input, output); err != nil {
		return err
	}
	if pm, ok := g.Meter.(PersonMeter); ok && user != "" {
		usd := float64(input)/1e6*g.Price.InputPerM + float64(output)/1e6*g.Price.OutputPerM
		return pm.AddPerson(ctx, g.Tenant, user, g.month(), usd)
	}
	return nil
}

// RecordCost meters usd of one call made in space for user (SB-5604, SB-5605: a cached
// or batched call costs less than its tokens at list price).
func (g *Guard) RecordCost(ctx context.Context, space, user string, usd float64) error {
	if err := g.recordUSD(ctx, space, usd); err != nil {
		return err
	}
	if pm, ok := g.Meter.(PersonMeter); ok && user != "" {
		return pm.AddPerson(ctx, g.Tenant, user, g.month(), usd)
	}
	return nil
}

// RecordIn is Record for a call that ran in space (spc_NAME, or "" for none): the spend is
// metered for the tenant and, with a SpaceMeter, for the space.
func (g *Guard) RecordIn(ctx context.Context, space string, input, output int) error {
	return g.recordUSD(ctx, space, float64(input)/1e6*g.Price.InputPerM+float64(output)/1e6*g.Price.OutputPerM)
}

func (g *Guard) recordUSD(ctx context.Context, space string, usd float64) error {
	month := g.month()
	if err := g.Meter.Add(ctx, g.Tenant, month, usd); err != nil {
		return err
	}
	if sm, ok := g.Meter.(SpaceMeter); ok && space != "" {
		if err := sm.AddSpace(ctx, g.Tenant, space, month, usd); err != nil {
			return err
		}
	}
	g.warn(ctx, month)
	return nil
}

// warn sends the budget warning once a month. A meter or marker it can't read sends
// nothing now; the next metered call tries again.
func (g *Guard) warn(ctx context.Context, month string) {
	if g.Warn == nil || g.Once == nil || g.MonthlyUSD <= 0 {
		return
	}
	spent, err := g.Meter.Spent(ctx, g.Tenant, month)
	if err != nil || spent < WarnAt*g.MonthlyUSD && !g.forecastOver(ctx) {
		return
	}
	if first, err := g.Once.First(ctx, "budget-warning-"+g.Tenant+"-"+month); err == nil && first {
		g.Warn(ctx, month, spent)
	}
}

// forecastOver reports whether the month-end forecast passes the budget, reading it at
// most once an hour.
func (g *Guard) forecastOver(ctx context.Context) bool {
	if g.Forecast == nil {
		return false
	}
	g.mu.Lock()
	now := g.Now()
	due := g.forecastAt.IsZero() || now.Sub(g.forecastAt) >= time.Hour
	if due {
		g.forecastAt = now
	}
	g.mu.Unlock()
	if !due {
		return false
	}
	f, ok := g.Forecast(ctx)
	return ok && f > g.MonthlyUSD
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

// MemoryMeter is an in-process Meter and SpaceMeter (development and tests).
type MemoryMeter struct {
	mu      sync.Mutex
	m       map[string]float64
	spaces  map[string]map[string]float64
	persons map[string]map[string]float64
	Fail    bool
}

// AddPerson implements PersonMeter.
func (m *MemoryMeter) AddPerson(_ context.Context, tenant, user, month string, usd float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.persons == nil {
		m.persons = map[string]map[string]float64{}
	}
	k := tenant + "/" + month
	if m.persons[k] == nil {
		m.persons[k] = map[string]float64{}
	}
	m.persons[k][user] += usd
	return nil
}

// SpentByPerson implements PersonMeter.
func (m *MemoryMeter) SpentByPerson(_ context.Context, tenant, month string) (map[string]float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Fail {
		return nil, errors.New("meter down")
	}
	out := map[string]float64{}
	for u, v := range m.persons[tenant+"/"+month] {
		out[u] = v
	}
	return out, nil
}

// AddSpace implements SpaceMeter.
func (m *MemoryMeter) AddSpace(_ context.Context, tenant, space, month string, usd float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.spaces == nil {
		m.spaces = map[string]map[string]float64{}
	}
	k := tenant + "/" + month
	if m.spaces[k] == nil {
		m.spaces[k] = map[string]float64{}
	}
	m.spaces[k][space] += usd
	return nil
}

// SpentBySpace implements SpaceMeter.
func (m *MemoryMeter) SpentBySpace(_ context.Context, tenant, month string) (map[string]float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Fail {
		return nil, errors.New("meter down")
	}
	out := map[string]float64{}
	for s, v := range m.spaces[tenant+"/"+month] {
		out[s] = v
	}
	return out, nil
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
