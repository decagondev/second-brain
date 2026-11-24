package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/notify"
)

type runs map[string]time.Time

func (r runs) LastFinish(_ context.Context, job string) (time.Time, bool, error) {
	at, ok := r[job]
	if ok && at.IsZero() {
		return time.Time{}, false, errors.New("unreadable")
	}
	return at, ok, nil
}

type syncs map[string]Sync

func (s syncs) Sync(_ context.Context, kind string) (Sync, bool, error) {
	if kind == "broken" {
		return Sync{}, false, errors.New("unreadable")
	}
	x, ok := s[kind]
	return x, ok, nil
}

func TestOverdueAndStaleFirst(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	r := runs{"sync": now.Add(-30 * time.Second), "anchor": now.Add(-3 * time.Hour), "backup": {}, "drift": now.Add(-10 * time.Minute)}
	s := syncs{"docs": {LastSuccess: now.Add(-time.Minute)}, "chat": {LastSuccess: now.Add(-time.Hour), LastFailure: now.Add(-time.Minute), Failures: 4}}
	rep, err := Build(context.Background(), r, s, []string{"docs", "chat", "tickets"}, now)
	if err != nil {
		t.Fatal(err)
	}
	over := map[string]bool{}
	seenFresh := false
	for _, j := range rep.Jobs {
		over[j.Name] = j.Overdue
		if !j.Overdue {
			seenFresh = true
		} else if seenFresh {
			t.Fatalf("an overdue job after a fresh one: %+v", rep.Jobs)
		}
	}
	if over["sync"] || !over["anchor"] || !over["backup"] || over["drift"] || !over["retention"] {
		t.Fatalf("overdue %v", over)
	}
	for _, j := range rep.Jobs {
		if j.Name == "backup" && (!j.Unreadable || j.LastFinish != nil) {
			t.Fatalf("an unreadable record: %+v", j)
		}
	}
	got := map[string]Connector{}
	for i, c := range rep.Connectors {
		got[c.Kind] = c
		if i == len(rep.Connectors)-1 && c.Stale {
			t.Fatalf("stale connectors not first: %+v", rep.Connectors)
		}
	}
	if got["docs"].Stale || !got["chat"].Stale || got["chat"].Failures != 4 || !got["tickets"].Stale || got["tickets"].LastSuccess != nil {
		t.Fatalf("connectors %+v", got)
	}
	if _, err := Build(context.Background(), r, s, []string{"broken"}, now); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("an unreadable connector record: %v, want EUNAVAIL, never healthy", err)
	}
}

func TestBoxFromNoFactsIsUnknown(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	for name, b := range map[string]*Box{
		"missing": BoxFrom(nil, false, now),
		"garbage": BoxFrom([]byte("{not json"), true, now),
		"old":     BoxFrom([]byte(`{"at":"2026-11-20T12:00:00Z","ready":200}`), true, now),
	} {
		if b.Verdict != box.Unknown || len(b.Checks) == 0 {
			t.Fatalf("%s: %+v", name, b)
		}
	}
	b := BoxFrom([]byte(`{"at":"2026-11-23T11:30:00Z","ready":503}`), true, now)
	if b.Verdict != box.Fail || b.FactsAt == nil {
		t.Fatalf("%+v", b)
	}
}

// SB-2607: the notify line warns on a failed run, a missing one or one over an hour old,
// and a cell with no channel has none.
func TestNotifyFrom(t *testing.T) {
	now := time.Date(2027, 3, 14, 12, 0, 0, 0, time.UTC)
	ok := notify.RunRecord{At: now.Add(-5 * time.Minute), Via: "slack", Sent: 3}
	if n := NotifyFrom(ok, true, "slack", now); n == nil || n.Warning || n.Sent != 3 || n.LastRun == nil {
		t.Fatalf("a good run: %+v", n)
	}
	for name, c := range map[string]struct {
		rec   notify.RunRecord
		found bool
	}{
		"failed sends":  {notify.RunRecord{At: ok.At, Failed: 1}, true},
		"a stopped run": {notify.RunRecord{At: ok.At, Error: true}, true},
		"stale":         {notify.RunRecord{At: now.Add(-NotifyStale - time.Minute)}, true},
		"never ran":     {notify.RunRecord{}, false},
	} {
		if n := NotifyFrom(c.rec, c.found, "slack", now); n == nil || !n.Warning || n.Reason == "" {
			t.Errorf("%s: %+v", name, n)
		}
	}
	if n := NotifyFrom(ok, true, "", now); n != nil {
		t.Fatalf("no channel: %+v", n)
	}
}

// SB-3201, SB-3203: a failed sync names its class and fix until a success; every line is a
// check, and an unreadable job is unknown.
func TestFailureAndChecks(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	s := syncs{
		"docs":    {LastSuccess: now.Add(-time.Hour), LastFailure: now.Add(-time.Minute), Failures: 2, Failure: "auth"},
		"chat":    {LastSuccess: now.Add(-time.Minute), Failures: 0, Failure: ""},
		"tickets": {LastSuccess: now.Add(-time.Minute), LastFailure: now, Failures: 1},
	}
	rep, err := Build(context.Background(), runs{}, s, []string{"docs", "chat", "tickets"}, now)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Connector{}
	for _, c := range rep.Connectors {
		by[c.Kind] = c
	}
	if by["docs"].Failure != "auth" || by["docs"].Fix != "docs/runbooks/connectors.md#auth" {
		t.Fatalf("docs: %+v", by["docs"])
	}
	if by["chat"].Failure != "" || by["chat"].Fix != "" {
		t.Fatalf("a healthy connector has a failure: %+v", by["chat"])
	}
	if by["tickets"].Failure != "source_error" {
		t.Fatalf("a failure with no class: %+v", by["tickets"])
	}
	rep.Unstewarded = &Unstewarded{Spaces: []string{}, Unknown: []string{"hr"}}
	checks := map[string]Check{}
	for _, c := range rep.Checks() {
		checks[c.Name] = c
	}
	if !checks["connector:docs"].Warning || checks["connector:chat"].Warning || !checks["stewards"].Unknown || checks["stewards"].Warning {
		t.Fatalf("checks: %+v", checks)
	}
	for _, c := range checks {
		if c.Name[:4] == "job:" && !c.Warning && !c.Unknown {
			t.Fatalf("a job that never ran is fine: %+v", c)
		}
	}
}
