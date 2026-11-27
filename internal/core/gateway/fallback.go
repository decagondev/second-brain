package gateway

import (
	"context"
	"errors"
	"sync"
	"time"

	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/ports"
)

// Fall back to a gated model (sb-runtime(7), Fallback model; SB-5601, SB-5602): when the
// first provider fails, an ask goes to a second one only if that one passed the release's
// eval gate. Three failures in five minutes send every call to the fallback for ten minutes
// before the first is tried again. With no gated model left, calls fail as unavailable and
// the ask shows passages instead (LAW-06: an ungated model never answers).

// The fallback's windows.
const (
	FailWindow  = 5 * time.Minute
	FailCount   = 3
	SwitchedFor = 10 * time.Minute
)

// Fallback is a ports.Model over two gateways.
type Fallback struct {
	Primary, Secondary ports.Model
	// Gated says the secondary passed this release's eval gate now; nil or false is never.
	Gated func() bool
	Now   func() time.Time
	// Switched is told when calls move to the fallback and when they move back.
	Switched func(on bool)

	mu       sync.Mutex
	failures []time.Time
	until    time.Time
	down     bool
}

// NewFallback is a fallback ready to use.
func NewFallback(primary, secondary ports.Model, gated func() bool, now func() time.Time) *Fallback {
	return &Fallback{Primary: primary, Secondary: secondary, Gated: gated, Now: now}
}

type usedKey struct{}

// Used records which model answered the calls of one ask.
type Used struct {
	mu       sync.Mutex
	fallback bool
}

// WithUsed gives ctx a record of whether the fallback answered, or keeps the one it has.
func WithUsed(ctx context.Context) (context.Context, *Used) {
	if u, ok := ctx.Value(usedKey{}).(*Used); ok {
		return ctx, u
	}
	u := &Used{}
	return context.WithValue(ctx, usedKey{}, u), u
}

// Fallback says the fallback answered at least one call.
func (u *Used) Fallback() bool {
	if u == nil {
		return false
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.fallback
}

type forceKey struct{}

// WithFallbackOnly sends every call of ctx to the fallback, gated or not: the eval run that
// gates it uses this, and nothing else does.
func WithFallbackOnly(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceKey{}, true)
}

func (f *Fallback) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func (f *Fallback) gated() bool { return f.Secondary != nil && f.Gated != nil && f.Gated() }

// failover is a failure that moves a call to the fallback: unavailable, rate limited or out
// of time. A secret refused or a budget spent is the same on either model, so it never does.
func failover(err error) bool {
	if errors.Is(err, ErrSecretInPrompt) || errors.Is(err, budget.ErrBudget) {
		return false
	}
	return errors.Is(err, ErrUnavailable) || errors.Is(err, ports.ErrModelDown) || errors.Is(err, ports.ErrRateLimited) ||
		errors.Is(err, context.DeadlineExceeded)
}

// Complete answers on the primary, or on the gated fallback when the primary fails or is
// switched out.
func (f *Fallback) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	if forced, _ := ctx.Value(forceKey{}).(bool); forced {
		if f.Secondary == nil {
			return ports.ModelResponse{}, ErrUnavailable
		}
		return f.Secondary.Complete(ctx, req)
	}
	now := f.now()
	f.mu.Lock()
	switched := now.Before(f.until)
	f.mu.Unlock()
	if switched && f.gated() {
		return f.second(ctx, req)
	}
	resp, err := f.Primary.Complete(ctx, req)
	if err == nil {
		f.settle(false)
		return resp, nil
	}
	if !failover(err) {
		return resp, err
	}
	f.fail(now)
	if !f.gated() {
		f.settle(true)
		return resp, err
	}
	return f.second(ctx, req)
}

func (f *Fallback) second(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	resp, err := f.Secondary.Complete(ctx, req)
	if err != nil {
		f.settle(failover(err))
		return resp, err
	}
	if u, ok := ctx.Value(usedKey{}).(*Used); ok {
		u.mu.Lock()
		u.fallback = true
		u.mu.Unlock()
	}
	f.settle(false)
	return resp, nil
}

// fail records a primary failure; the third in FailWindow switches to the fallback.
func (f *Fallback) fail(now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	kept := f.failures[:0]
	for _, t := range f.failures {
		if now.Sub(t) < FailWindow {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	f.failures = kept
	if len(f.failures) >= FailCount && !now.Before(f.until) && f.gated() {
		f.until = now.Add(SwitchedFor)
		f.failures = nil
		if f.Switched != nil {
			f.Switched(true)
		}
	}
}

// settle records whether no model could answer the last call.
func (f *Fallback) settle(down bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.down = down
}

// State is the fallback as the health report shows it: switched to the fallback until
// a time, and whether the last call found no model to answer.
type State struct {
	Gated    bool      `json:"gated"`
	Switched bool      `json:"switched"`
	Until    time.Time `json:"until,omitzero"`
	Down     bool      `json:"down"`
}

// State is the fallback now.
func (f *Fallback) State() State {
	now := f.now()
	f.mu.Lock()
	defer f.mu.Unlock()
	st := State{Gated: f.Secondary != nil && f.Gated != nil && f.Gated(), Down: f.down}
	if now.Before(f.until) {
		st.Switched, st.Until = true, f.until
	}
	return st
}
