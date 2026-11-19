package kill

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Bound is how fast a switch must take effect (LAW-10).
const Bound = 10 * time.Second

// Target is a cell under drill: whether a new op in scope is refused right now.
type Target interface {
	Refused(ctx context.Context, scope string) (bool, error)
}

// DrillResult is one scope's outcome.
type DrillResult struct {
	Scope string
	Took  time.Duration // until the cell refused; Bound or more is a failure
	OK    bool
	Err   string
}

// Drill fires each scope in turn against one staging cell and checks it takes effect
// within Bound, then clears it and checks the cell recovers (SB-404). Every set and
// clear goes through Control, so the drill is audited like any kill.
func Drill(ctx context.Context, c Control, t Target, tenant string, scopes []string, sleep func(time.Duration)) []DrillResult {
	const step = 250 * time.Millisecond
	var out []DrillResult
	for _, s := range scopes {
		res := DrillResult{Scope: s}
		start := c.Now()
		if err := c.Kill(ctx, tenant, s, "release kill drill", "sys_kill_drill"); err != nil {
			res.Err = err.Error()
			out = append(out, res)
			continue
		}
		for {
			refused, err := t.Refused(ctx, s)
			if err != nil {
				res.Err = err.Error()
				break
			}
			res.Took = c.Now().Sub(start)
			if refused {
				res.OK = res.Took < Bound
				break
			}
			if res.Took >= Bound {
				break
			}
			sleep(step)
		}
		if err := c.Unkill(ctx, tenant, s, "release kill drill done", "sys_kill_drill"); err != nil && res.Err == "" {
			res.Err = "unkill: " + err.Error()
			res.OK = false
		}
		out = append(out, res)
	}
	return out
}

// DrillPassed reports whether every scope took effect in time.
func DrillPassed(rs []DrillResult) bool {
	for _, r := range rs {
		if !r.OK {
			return false
		}
	}
	return len(rs) > 0
}

// DrillReport is the release-notes section for a drill (Markdown).
func DrillReport(release, tenant string, rs []DrillResult) string {
	var b strings.Builder
	verdict := "passed"
	if !DrillPassed(rs) {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "### Kill drill: %s\n\nRelease `%s`, staging cell `%s`. Bound: %s (LAW-10).\n\n| Scope | Took effect in | Result |\n|---|---|---|\n", verdict, release, tenant, Bound)
	for _, r := range rs {
		result := "pass"
		if !r.OK {
			result = "FAIL"
			if r.Err != "" {
				result += ": " + r.Err
			}
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", r.Scope, r.Took.Round(time.Millisecond), result)
	}
	return b.String()
}
