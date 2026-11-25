package budget

import (
	"context"
	"sync"
)

// Tally adds up what the model calls made with one context cost (SB-4201): a research
// report's asks carry one, so the report stops at its cap whatever else the cell spends.
type Tally struct {
	mu  sync.Mutex
	usd float64
}

type tallyKey struct{}

// WithTally returns ctx carrying a new tally, and the tally.
func WithTally(ctx context.Context) (context.Context, *Tally) {
	t := &Tally{}
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
	defer t.mu.Unlock()
	t.usd += float64(input)*p.InputPerM/1e6 + float64(output)*p.OutputPerM/1e6
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
