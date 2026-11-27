package gateway

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// People first when the provider is busy (sb-runtime(7), When the provider is busy;
// SB-5703). Outside a rate limit calls run side by side as before. Once a provider answers
// 429, calls take turns until it has served one without a limit and nobody waits: a
// person's ask goes before any scheduled call, and among the waiting the space served
// least in this storm goes first, so one busy space can't starve the others. A scheduled
// step that runs again on a later tick doesn't wait at all. An ask that would wait longer
// than MaxBusyWait gets ErrBusy, and the ask shows passages instead.

// MaxBusyWait is the longest an ask waits for a busy provider.
const MaxBusyWait = 20 * time.Second

// ErrBusy is a call that couldn't run in time because the provider is rate-limiting. It is
// unavailable (EUNAVAIL), so an ask answers with passages.
var ErrBusy = fmt.Errorf("%w: the model is busy", ErrUnavailable)

// Busy is one provider's queue, shared by every gateway of that provider.
type Busy struct {
	// MaxWait is MaxBusyWait unless set (tests).
	MaxWait time.Duration
	Now     func() time.Time

	mu      sync.Mutex
	until   time.Time // limited until
	storm   bool      // calls take turns
	holder  bool      // a call holds the turn
	queue   []*waiter
	served  map[string]int
	limited map[string]int
	waited  map[string]int
	timer   *time.Timer
	seq     int
}

// BusyCounts is one space's count of rate limits and of calls that waited for a turn.
type BusyCounts struct {
	Limited int `json:"rate_limited"`
	Waited  int `json:"waited"`
}

type waiter struct {
	space   string
	ask     bool
	seq     int
	granted chan struct{}
}

func (b *Busy) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

func (b *Busy) maxWait() time.Duration {
	if b.MaxWait > 0 {
		return b.MaxWait
	}
	return MaxBusyWait
}

func spaceName(s string) string {
	if s == "" {
		return "company"
	}
	return s
}

func (b *Busy) bump(m *map[string]int, space string) {
	if *m == nil {
		*m = map[string]int{}
	}
	(*m)[spaceName(space)]++
}

// errNextTick is a scheduled call a storm sends to the next tick.
var errNextTick = errors.New("busy: the next tick")

// turn waits for the call's turn: at once outside a storm (held false), or once the queue
// grants it (held true). A scheduled call that can come back later is errNextTick at once;
// past wait, ErrBusy.
func (b *Busy) turn(ctx context.Context, space string, ask, later bool, wait time.Duration) (bool, error) {
	b.mu.Lock()
	if !b.storm {
		b.mu.Unlock()
		return false, nil
	}
	b.bump(&b.waited, space)
	if later {
		b.mu.Unlock()
		return false, errNextTick
	}
	b.seq++
	w := &waiter{space: spaceName(space), ask: ask, seq: b.seq, granted: make(chan struct{})}
	b.queue = append(b.queue, w)
	b.dispatchLocked()
	b.mu.Unlock()
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-w.granted:
		return true, nil
	case <-t.C:
	case <-ctx.Done():
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-w.granted: // granted as the wait ended: give the turn back
		b.holder = false
		b.dispatchLocked()
	default:
		b.drop(w)
	}
	return false, ErrBusy
}

func (b *Busy) drop(w *waiter) {
	for i, q := range b.queue {
		if q == w {
			b.queue = append(b.queue[:i], b.queue[i+1:]...)
			return
		}
	}
}

// dispatchLocked grants the turn to the next waiter once the limit has passed: asks first,
// then the space served least in this storm, then the earliest.
func (b *Busy) dispatchLocked() {
	if b.holder || len(b.queue) == 0 {
		return
	}
	if d := b.until.Sub(b.now()); d > 0 {
		if b.timer == nil {
			b.timer = time.AfterFunc(d, func() {
				b.mu.Lock()
				b.timer = nil
				b.dispatchLocked()
				b.mu.Unlock()
			})
		}
		return
	}
	sort.SliceStable(b.queue, func(i, j int) bool {
		x, y := b.queue[i], b.queue[j]
		if x.ask != y.ask {
			return x.ask
		}
		if b.served[x.space] != b.served[y.space] {
			return b.served[x.space] < b.served[y.space]
		}
		return x.seq < y.seq
	})
	w := b.queue[0]
	b.queue = b.queue[1:]
	b.holder = true
	b.bump(&b.served, w.space)
	close(w.granted)
}

// release ends a call: a rate limit starts or extends the storm for backoff; a call that
// held the turn hands it on, and the storm ends once one went through with nobody waiting.
func (b *Busy) release(space string, held, limited bool, backoff time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limited {
		b.bump(&b.limited, space)
		if !b.storm {
			b.storm, b.served = true, map[string]int{}
		}
		if u := b.now().Add(backoff); u.After(b.until) {
			b.until = u
		}
	}
	if held {
		b.holder = false
	}
	if !limited && !b.holder && len(b.queue) == 0 && !b.now().Before(b.until) {
		b.storm, b.served = false, nil
	}
	b.dispatchLocked()
}

// Counts is each space's rate limits and waits so far, for the health report.
func (b *Busy) Counts() map[string]BusyCounts {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := map[string]BusyCounts{}
	for s, n := range b.limited {
		c := out[s]
		c.Limited = n
		out[s] = c
	}
	for s, n := range b.waited {
		c := out[s]
		c.Waited = n
		out[s] = c
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// turns sends req through the provider's queue (SB-5703): an ask retries a rate limit until
// it has waited MaxWait, then is ErrBusy; a scheduled step that runs again later goes to the
// next tick (ErrBatchPending) rather than wait; other scheduled calls wait behind every ask
// and retry as many times as before.
func (g Gateway) turns(ctx context.Context, req ports.ModelRequest, retries int, backoff time.Duration) (ports.ModelResponse, error) {
	ask := !req.Batch
	later := req.Batch && ports.DeferredStep(ctx) != ""
	start, max := g.now(), g.Busy.maxWait()
	for attempt := 0; ; attempt++ {
		left := max
		if ask {
			left = max - g.now().Sub(start)
		}
		if left <= 0 {
			return ports.ModelResponse{}, ErrBusy
		}
		held, err := g.Busy.turn(ctx, req.Space, ask, later, left)
		if errors.Is(err, errNextTick) {
			return ports.ModelResponse{}, fmt.Errorf("%w: the model is busy", ports.ErrBatchPending)
		}
		if err != nil {
			return ports.ModelResponse{}, err
		}
		resp, err := g.Model.Complete(ctx, req)
		limited := errors.Is(err, ports.ErrRateLimited)
		shift := attempt
		if shift > 6 {
			shift = 6
		}
		g.Busy.release(req.Space, held, limited, backoff<<shift)
		if err == nil {
			g.spent(ctx, req, resp.Usage, false)
			return resp, nil
		}
		if !limited {
			return ports.ModelResponse{}, fmt.Errorf("%w: %s: %v", ErrUnavailable, g.Provider, err)
		}
		g.count("sb_model_rate_limited_total", "Rate limits a provider answered, by space (SB-5703).", map[string]string{"tenant": g.Tenant, "provider": g.Provider, "space": spaceName(req.Space)}, 1)
		if !ask && attempt >= retries {
			return ports.ModelResponse{}, fmt.Errorf("%w: %s: rate limited", ErrUnavailable, g.Provider)
		}
	}
}
