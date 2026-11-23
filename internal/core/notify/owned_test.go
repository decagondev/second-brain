package notify

import (
	"context"
	"testing"
	"time"
)

// SB-2806: on the month's first weekly run each active owner hears a count, once; someone
// gone hears nothing, and the kind can be turned off.
func TestOwnersHearMonthly(t *testing.T) {
	run := func(now time.Time, off bool) []sent {
		t.Helper()
		j, s, c := world(now)
		j.Questions = questions{}
		c.weekly["weekly"] = Monday(now, nil).AddDate(0, 0, -7)
		j.Owned = func(context.Context, time.Time) (map[string]int, error) {
			return map[string]int{"usr_ann": 3, "usr_bob": 1, "usr_gone": 2, "usr_cat": 0}, nil
		}
		if off {
			j.Prefs = prefs{"usr_ann": {Off: []string{KindOwned}}}
		}
		if _, err := j.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		var out []sent
		for _, m := range s.got {
			if m.msg.Kind == KindOwned {
				out = append(out, m)
			}
		}
		// The same week again: the weekly part has run, so nobody is told twice.
		before := len(s.got)
		if _, err := j.Run(context.Background()); err != nil || len(s.got) != before {
			t.Fatalf("told twice: %v %d", err, len(s.got)-before)
		}
		return out
	}
	first := time.Date(2026, 12, 7, 9, 0, 0, 0, time.UTC) // the first Monday of December
	got := run(first, false)
	if len(got) != 2 || got[0].to.ID != "usr_ann" || got[0].msg.Text != "3 of your documents that answers often cite are past review" ||
		got[1].msg.Text != "1 of your documents that answers often cite is past review" || got[0].msg.Link != "https://brain.acme.example/?tab=documents" {
		t.Fatalf("%+v", got)
	}
	if got := run(first, true); len(got) != 1 || got[0].to.ID != "usr_bob" {
		t.Fatalf("a person who turned it off was told: %+v", got)
	}
	if got := run(time.Date(2026, 12, 14, 9, 0, 0, 0, time.UTC), false); len(got) != 0 {
		t.Fatalf("told on a later Monday: %+v", got)
	}
	// Tuesday 1 December is in the week of Monday 30 November, so December waits for the 7th.
	if !FirstMonday(first, nil) || FirstMonday(time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC), nil) {
		t.Fatal("FirstMonday")
	}
}
