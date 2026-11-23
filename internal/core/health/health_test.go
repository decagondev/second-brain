package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/box"
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
