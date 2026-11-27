package budget

import (
	"context"
	"sync"
)

// Tally adds up what the model calls made with one context cost (SB-4201): a research
// report's asks carry one, so the report stops at its cap whatever else the cell spends.
// A tally made inside another adds to it too, so one answer's cost (SB-4606) still counts
// toward its report's cap.
type Tally struct {
	mu     sync.Mutex
	usd    float64
	saved  map[string]float64 // what reuse, batching, caching and the economy tier saved (SB-5606)
	parent *Tally
}

type tallyKey struct{}

// WithTally returns ctx carrying a new tally, and the tally.
func WithTally(ctx context.Context) (context.Context, *Tally) {
	t := &Tally{parent: TallyOf(ctx)}
	return context.WithValue(ctx, tallyKey{}, t), t
}

// TallyOf is ctx's tally, or nil.
func TallyOf(ctx context.Context) *Tally {
	t, _ := ctx.Value(tallyKey{}).(*Tally)
	return t
}

// Add counts one call's tokens at p.
func (t *Tally) Add(p Price, input, output int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.usd += float64(input)*p.InputPerM/1e6 + float64(output)*p.OutputPerM/1e6
	t.mu.Unlock()
	t.parent.Add(p, input, output)
}

// AddUSD counts one call's dollars.
func (t *Tally) AddUSD(usd float64) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.usd += usd
	t.mu.Unlock()
	t.parent.AddUSD(usd)
}

// Save counts dollars a call didn't cost, by why (reuse, batch, cache, tier).
func (t *Tally) Save(why string, usd float64) {
	if t == nil || usd <= 0 {
		return
	}
	t.mu.Lock()
	if t.saved == nil {
		t.saved = map[string]float64{}
	}
	t.saved[why] += usd
	t.mu.Unlock()
	t.parent.Save(why, usd)
}

// Saved is what the calls saved so far, by why; nil for nothing.
func (t *Tally) Saved() map[string]float64 {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.saved) == 0 {
		return nil
	}
	out := map[string]float64{}
	for k, v := range t.saved {
		out[k] = v
	}
	return out
}

// USD is what the calls cost so far.
func (t *Tally) USD() float64 {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.usd
}
