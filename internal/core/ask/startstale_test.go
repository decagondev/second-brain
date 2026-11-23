package ask_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

type cases []ask.EvalCase

func (c *cases) PutEvalCase(_ context.Context, x ask.EvalCase) error { *c = append(*c, x); return nil }
func (c *cases) EvalCases(context.Context, string) ([]ask.EvalCase, error) {
	return *c, nil
}
func (c *cases) DeleteEvalCase(context.Context, string, string) error { return nil }

// byEmail answers ACL by the member's email, as a source that knows people by email does.
type byEmail map[string]ports.Verdict

func (byEmail) APIVersion() int { return ports.ConnectorAPIVersion }
func (byEmail) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, nil
}
func (s byEmail) ACL(_ context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	for i, r := range refs {
		out[i] = s[r.ID+"|"+u.Email]
	}
	return out, nil
}
func (byEmail) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", nil
}
func (byEmail) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, nil
}

// SB-2904, SB-2907: a case whose expected document has a new version or is gone is to check;
// start_here items gone, stale or unreadable to most members are listed with why, and one
// the steward can't open only counted.
func TestEvalsAndStartHereToCheck(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	a := freshAgent(t, `{"claims":[]}`)
	x := stewardsOf(a, now)
	cs := &cases{
		{ID: "ev_1", Space: "spc_ops", Question: "refund?", Expect: "docs:c", ExpectVersion: 3}, // the same version: fine
		{ID: "ev_2", Space: "spc_ops", Question: "rota?", Expect: "docs:a", ExpectVersion: 2},   // a is at 3 now
		{ID: "ev_3", Space: "spc_ops", Question: "old?", Expect: "docs:nope"},                   // gone
		{ID: "ev_4", Space: "spc_hr", Question: "hr?", Expect: "docs:nope"},                     // another space's
	}
	x.Evals = cs
	acl := a.Auth.Connectors["docs"].(source)
	members := byEmail{}
	for _, e := range []string{"m1", "m2", "m3", "m4"} {
		members["c|"+e] = ports.VerdictAllow // everyone opens c
		members["b|"+e] = ports.VerdictAllow
	}
	members["a|m1"] = ports.VerdictAllow   // only one in four opens a: unreadable
	members["b|m4"] = ports.VerdictUnknown // Unknown is can't, but one in four isn't more than a quarter
	conns := map[string]ports.Connector{"docs": dual{acl, members}}
	x.Agent.Auth.Connectors = conns
	x.Members = func(context.Context, string) ([]string, error) { return []string{"m1", "m2", "m3", "m4"}, nil }
	x.StartItems = func(string) []string { return []string{"docs:c", "docs:a", "docs:gone.md", "docs:salaries", "docs:b"} }
	res, err := x.Review(ctx, scope(t, "alice"), "ops", reviewSrc{}, 365)
	if err != nil {
		t.Fatal(err)
	}
	want := []ask.EvalCheck{{ID: "ev_2", Question: "rota?", Check: ask.CheckNewVersion}, {ID: "ev_3", Question: "old?", Check: ask.CheckGone}}
	if !reflect.DeepEqual(res.EvalsToCheck, want) {
		t.Fatalf("evals to check: %+v", res.EvalsToCheck)
	}
	got := map[string][]string{}
	for _, it := range res.StartHere {
		got[it.Item] = it.Why
		if it.Item == "docs:a" && (it.Result == nil || it.MembersWithout != 3 || it.Members != 4) {
			t.Fatalf("a: %+v", it)
		}
	}
	// a is old (stale) and three of four members can't open it; b is old and stale only;
	// the gone item is listed; salaries, which the steward can't open, is counted only.
	wantWhy := map[string][]string{"docs:a": {ask.StartStale, ask.StartUnreadable}, "docs:b": {ask.StartStale}, "docs:gone.md": {ask.StartGone}}
	if !reflect.DeepEqual(got, wantWhy) || res.StartHidden != 1 {
		t.Fatalf("start here: %+v hidden %d", got, res.StartHidden)
	}
	// A failing case to check is no regression.
	run := func(ask.EvalCase) (bool, error) { return false, nil }
	passed := []ask.EvalCase{{ID: "ev_2", PassedOn: "r1", LastRelease: "r1", LastPassed: true}}
	if _, hold, _ := ask.RunCases(passed, "r2", run, map[string]string{"ev_2": ask.CheckNewVersion}); hold != nil {
		t.Fatalf("a case to check held a release: %+v", hold)
	}
	if _, hold, _ := ask.RunCases(passed, "r2", run, nil); hold == nil {
		t.Fatal("a regression held nothing")
	}
}

// dual answers the steward's ACL by id and a member's by email.
type dual struct {
	byID    source
	byEmail byEmail
}

func (dual) APIVersion() int { return ports.ConnectorAPIVersion }
func (dual) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, nil
}
func (d dual) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	if u.ID != "" {
		return d.byID.ACL(ctx, u, refs)
	}
	return d.byEmail.ACL(ctx, u, refs)
}
func (dual) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", nil
}
func (dual) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, nil
}
