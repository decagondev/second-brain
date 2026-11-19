package kill

import (
	"context"
	"strings"
	"testing"
	"time"
)

// checkerTarget is a cell whose checker polls every Poll on the drill's clock.
type checkerTarget struct {
	ch       *Checker
	now      *time.Time
	lastPoll time.Time
	frozen   bool // a cell that never reloads
}

func (t *checkerTarget) Refused(ctx context.Context, scope string) (bool, error) {
	if !t.frozen && t.now.Sub(t.lastPoll) >= Poll {
		_ = t.ch.Refresh(ctx)
		t.lastPoll = *t.now
	}
	op := Op{}
	switch {
	case strings.HasPrefix(scope, "agent:"):
		op.Agent = strings.TrimPrefix(scope, "agent:")
	case strings.HasPrefix(scope, "connector:"):
		op.Connector = strings.TrimPrefix(scope, "connector:")
	default:
		op.Agent = "ask"
	}
	return t.ch.Check(op) != nil, nil
}

func TestDrill(t *testing.T) {
	scopes := []string{"agent:ask", "agent:draft-reply", "connector:docs", "connector:tickets", "tenant"}
	for _, frozen := range []bool{false, true} {
		c, ch, _, rs, now := setup()
		_ = ch.Refresh(context.Background())
		tgt := &checkerTarget{ch: ch, now: now, lastPoll: *now, frozen: frozen}
		res := Drill(context.Background(), *c, tgt, "acme-logistics", scopes, func(d time.Duration) { *now = now.Add(d) })
		if len(res) != len(scopes) {
			t.Fatalf("%d results", len(res))
		}
		if DrillPassed(res) == frozen {
			t.Fatalf("frozen=%v: passed=%v\n%s", frozen, DrillPassed(res), DrillReport("rel_x", "staging-1", res))
		}
		if !frozen {
			for _, r := range res {
				if r.Took > Poll+250*time.Millisecond {
					t.Errorf("%s took %s", r.Scope, r.Took)
				}
			}
			if len(rs.rs) != 2*len(scopes) {
				t.Errorf("%d audit rows, want a kill and an unkill per scope", len(rs.rs))
			}
			if rep := DrillReport("rel_x", "staging-1", res); !strings.Contains(rep, "Kill drill: passed") || !strings.Contains(rep, "| `tenant` |") {
				t.Errorf("report:\n%s", rep)
			}
		}
	}
	if DrillPassed(nil) {
		t.Error("an empty drill passed")
	}
}
