package ask

import (
	"context"
	"strings"
	"testing"
	"time"
)

// SB-5002, SB-5003: economy sends everything but research to small, frontier everything to
// main, a held small tier wins over both, and a person at their limit gets small for their
// own asks but not for a brief.
func TestRoutingPoliciesAndLimits(t *testing.T) {
	held := false
	policy := map[string]string{"spc_eco": RoutingEconomy, "spc_top": RoutingFrontier}
	a := Agent{Small: smallStub{}, SmallHeld: func() bool { return held },
		Routing: func(s string) string {
			if p, ok := policy[s]; ok {
				return p
			}
			return RoutingBalanced
		},
		PersonOver: func(_ context.Context, u string) (bool, error) { return u == "usr_over", nil }}
	long := strings.Repeat("a", SmallMaxRunes+1)
	tier := func(user, space, feature, q string) string {
		ctx := WithFeature(a.atLimit(context.Background(), user), feature)
		return a.tierFor(ctx, space, Request{}, q, 0, 0)
	}
	for _, c := range []struct{ user, space, feature, q, want string }{
		{"usr_a", "spc_eco", FeatureAsk, long, TierSmall},
		{"usr_a", "spc_eco", FeatureBrief, long, TierSmall},
		{"usr_a", "spc_eco", FeatureResearch, "q", TierMain},
		{"usr_a", "spc_top", FeatureAsk, "q", TierMain},
		{"usr_a", "spc_ops", FeatureAsk, "q", TierSmall},
		{"usr_a", "spc_ops", FeatureAsk, long, TierMain},
		{"usr_over", "spc_ops", FeatureAsk, long, TierSmall},
		{"usr_over", "spc_top", FeatureResearch, long, TierSmall},
		{"usr_over", "spc_ops", FeatureBrief, long, TierMain},
	} {
		if got := tier(c.user, c.space, c.feature, c.q); got != c.want {
			t.Errorf("%+v: %q", c, got)
		}
	}
	held = true
	if got := tier("usr_over", "spc_eco", FeatureAsk, "q"); got != TierMain {
		t.Fatalf("held small tier: %q", got)
	}
	if f := featureFor(context.Background(), Request{Agent: AgentPrep}, false); f != FeatureResearch {
		t.Fatalf("prep: %q", f)
	}
	if f := featureFor(context.Background(), Request{}, true); f != FeatureEval {
		t.Fatalf("probe: %q", f)
	}
}

// SB-5001: rows of fewer than five answers hide their count and people but keep their spend
// in the totals; evals and answers older than twelve weeks don't count.
func TestAIUsageMasksSmallRows(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	var ts []Trace
	for i := 0; i < 6; i++ {
		ts = append(ts, Trace{User: "usr_" + string(rune('a'+i%3)), Space: "spc_ops", At: now.Add(-time.Hour), CostUSD: 0.5, Tier: TierSmall})
	}
	ts = append(ts,
		Trace{User: "usr_a", Space: "spc_fin", At: now.Add(-time.Hour), CostUSD: 1, Feature: FeatureResearch},
		Trace{User: "usr_a", Space: "spc_fin", At: now.Add(-time.Hour), CostUSD: 9, Feature: FeatureEval},
		Trace{User: "usr_a", Space: "spc_fin", At: now.AddDate(0, 0, -7*UsageWeeks-1), CostUSD: 9})
	u := BuildAIUsage(ts, now)
	if len(u.Rows) != 2 || u.Total.Uses != 7 || u.Total.USD != 4 {
		t.Fatalf("usage: %+v", u)
	}
	fin, ops := u.Rows[0], u.Rows[1]
	if !fin.Masked || fin.Uses != 0 || fin.USD != 1 || fin.Feature != FeatureResearch || fin.Tier != TierMain {
		t.Fatalf("fin row: %+v", fin)
	}
	if ops.Masked || ops.Uses != 6 || ops.People != 3 || ops.Tier != TierSmall {
		t.Fatalf("ops row: %+v", ops)
	}
	csv := string(u.CSV())
	if !strings.Contains(csv, "research,ask,main,fewer than 5,fewer than 5,1.0000") || !strings.Contains(csv, "total,,,,,7,,4.0000") {
		t.Fatalf("csv:\n%s", csv)
	}
	b := Bands(map[string]float64{"a": 1, "b": 9, "c": 10}, 10)
	if b.Under80 != FewerThan5 || b.AtLimit != FewerThan5 {
		t.Fatalf("bands: %+v", b)
	}
}

// SB-5004: each target is counted per week, and two missed weeks in a row alert once, from
// the week the run began.
func TestGoalViewsAndMissedTwice(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC) // a Wednesday
	this := WeekStart(now)
	draft := func(week int, discarded bool) StewardQuestion {
		at := this.AddDate(0, 0, -7*week+1)
		q := StewardQuestion{At: at, Thread: &SavedThread{Agent: "month-end", Text: "draft", Discarded: discarded}}
		if !discarded {
			q.Reply = &StewardReply{Text: "draft", At: at.Add(2 * time.Hour)}
		}
		return q
	}
	qs := []StewardQuestion{draft(4, false), draft(3, false), draft(2, true), draft(1, true), draft(0, true)}
	half := 0.5
	goals := map[string]GoalDef{"month-end": {Job: "Close the month", Targets: []TargetDef{{Metric: MetricPublishedAsIs, AtLeast: &half}}},
		"other": {Job: "Other"}}
	vs := GoalViews(goals, qs, nil, nil, now, GoalWeeks, []string{"month-end"})
	if len(vs) != 1 || len(vs[0].Targets) != 1 || len(vs[0].Targets[0].Weeks) != GoalWeeks {
		t.Fatalf("views: %+v", vs)
	}
	w := vs[0].Targets[0].Weeks
	if !*w[0].Met || !*w[1].Met || *w[2].Met || *w[3].Met || *w[3].Value != 0 {
		t.Fatalf("weeks: %+v", w)
	}
	m := MissedTwice(vs)
	if len(m) != 1 || m[0].Since != w[2].Week || m[0].Metric != MetricPublishedAsIs {
		t.Fatalf("missed: %+v", m)
	}
	qs[3] = draft(1, false)
	if m := MissedTwice(GoalViews(goals, qs, nil, nil, now, GoalWeeks, nil)); len(m) != 0 {
		t.Fatalf("met last week: %+v", m)
	}
}
