package ask

import (
	"context"
	"errors"
	"sync"
	"time"

	"example.com/second-brain/internal/core/answer"
)

// Fair share on a small box (sb-ask(3), Fair share; sb-tenant(5) asks_per_minute; SB-2305).

// DefaultAsksPerMinute is asks_per_minute when the manifest doesn't say.
const DefaultAsksPerMinute = 6

// AgentLimited is the agent name of the trace an ask over the limit leaves: no question,
// no decisions, only that it was refused, so the weekly report counts it.
const AgentLimited = "limited"

// ErrRate is an ask over the asker's asks_per_minute (ERATE).
var ErrRate = errors.New("ask: ERATE")

// RateError is ErrRate with the wait and the refusal's trace.
type RateError struct {
	Wait  time.Duration
	Trace string
}

func (e *RateError) Error() string        { return "ask: ERATE: over asks_per_minute" }
func (e *RateError) Is(target error) bool { return target == ErrRate }

// Rate counts each person's asks over a sliding minute, in the cell's memory: a restart
// forgets the minute, which can only let a few more through, never refuse more.
type Rate struct {
	mu   sync.Mutex
	seen map[string][]time.Time
}

// rateSweep is how many people Rate holds before it drops everyone idle for a minute.
const rateSweep = 1024

// Take counts one ask by user at now if fewer than limit were made in the minute before;
// otherwise it counts nothing and says how long until one more is allowed.
func (r *Rate) Take(user string, limit int, now time.Time) (time.Duration, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen == nil {
		r.seen = map[string][]time.Time{}
	}
	cut := now.Add(-time.Minute)
	if len(r.seen) >= rateSweep {
		for u, ts := range r.seen {
			if !ts[len(ts)-1].After(cut) {
				delete(r.seen, u)
			}
		}
	}
	ts := r.seen[user]
	for len(ts) > 0 && !ts[0].After(cut) {
		ts = ts[1:]
	}
	if len(ts) >= limit {
		r.seen[user] = ts
		return ts[0].Sub(cut), false
	}
	r.seen[user] = append(ts, now)
	return 0, true
}

// limited refuses an ask over the limit, leaving a trace with no question.
func (a Agent) limited(ctx context.Context, user, space string, wait time.Duration) error {
	id := a.NewID("tr_")
	t := Trace{ID: id, Tenant: a.Tenant, User: user, Space: space, At: a.Now(),
		Response: Response{Trace: id, Release: a.Release, Agent: AgentLimited, People: []PersonRef{}, Answer: Answer{Claims: []answer.Claim{}}}}
	if err := a.Store.PutTrace(ctx, t); err != nil {
		return err
	}
	return &RateError{Wait: wait, Trace: id}
}
