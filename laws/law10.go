package laws

import (
	"fmt"
	"math/rand"
	"time"
)

// LAW-10 (SB-403): within 10 seconds of a kill switch being set, nothing new starts in
// its scope; and a cell that can't reach the kill store for 30 seconds starts no
// consequential tool call. Time is a fake clock the world advances.

// KillWorld is the control plane's kill store and one cell's checker, on a fake clock.
type KillWorld interface {
	Kill(scope string) error                                           // "agent:ask", "agent:draft-reply", "connector:docs", "connector:tickets", "tenant"
	Unkill(scope string) error                                         // clears it
	Advance(d time.Duration)                                           // moves the clock; the cell's background work runs as it would
	StoreDown(down bool)                                               // the cell can (not) reach the kill store
	Allowed(agent, connector string, consequential bool) (bool, error) // would the cell start this now?
}

// NewKillWorld builds the system under test.
var NewKillWorld = func() KillWorld { return pendingKill{} }

type pendingKill struct{}

func (pendingKill) Kill(string) error     { return nil }
func (pendingKill) Unkill(string) error   { return nil }
func (pendingKill) Advance(time.Duration) {}
func (pendingKill) StoreDown(bool)        {}
func (pendingKill) Allowed(string, string, bool) (bool, error) {
	return false, fmt.Errorf("%w: kill switches arrive with SB-403", ErrPending)
}

var killScopes = []string{"agent:ask", "agent:draft-reply", "connector:docs", "connector:tickets", "tenant"}

func law10(r *rand.Rand, w KillWorld) error {
	now := time.Duration(0)
	killedAt := map[string]time.Duration{} // scope -> when it was set (absent: not killed)
	downSince := time.Duration(-1)
	w.Advance(3 * time.Second) // let the cell load the list once
	now += 3 * time.Second
	for step := 0; step < 60; step++ {
		switch r.Intn(6) {
		case 0:
			s := killScopes[r.Intn(len(killScopes))]
			if err := w.Kill(s); err != nil {
				return err
			}
			if _, ok := killedAt[s]; !ok {
				killedAt[s] = now
			}
		case 1:
			if r.Intn(3) == 0 && len(killedAt) > 0 {
				for s := range killedAt {
					if err := w.Unkill(s); err != nil {
						return err
					}
					delete(killedAt, s)
					break
				}
			}
		case 2:
			down := r.Intn(4) == 0
			w.StoreDown(down)
			if down && downSince < 0 {
				downSince = now
			} else if !down && downSince >= 0 {
				// A switch the cell couldn't see while the store was down counts from now:
				// during an outage only the 30-second fail-safe applies (sb-kill-switches(7)).
				for k, at := range killedAt {
					if at < now {
						killedAt[k] = now
					}
				}
				downSince = -1
			}
		default:
			d := time.Duration(r.Intn(6000)) * time.Millisecond
			w.Advance(d)
			now += d
		}
		agent := []string{"ask", "draft-reply"}[r.Intn(2)]
		conn := []string{"docs", "tickets", "chat"}[r.Intn(3)]
		conseq := r.Intn(2) == 0
		ok, err := w.Allowed(agent, conn, conseq)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		for _, s := range []string{"agent:" + agent, "connector:" + conn, "tenant"} {
			if at, killed := killedAt[s]; killed && downSince < 0 && now-at >= 10*time.Second {
				return fmt.Errorf("%s/%s (consequential=%v) started %v after %s was killed", agent, conn, conseq, now-at, s)
			}
		}
		if conseq && downSince >= 0 && now-downSince >= 30*time.Second {
			return fmt.Errorf("a consequential call started %v after the kill store became unreachable", now-downSince)
		}
	}
	return nil
}
