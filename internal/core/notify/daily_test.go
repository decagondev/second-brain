package notify

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

func TestWorkingMorning(t *testing.T) {
	for at, want := range map[string]string{
		"2026-11-24T09:00:00Z": "2026-11-24T08:00:00Z", // Tuesday after 08:00
		"2026-11-24T07:00:00Z": "2026-11-23T08:00:00Z", // Tuesday before: Monday's
		"2026-11-23T07:00:00Z": "2026-11-20T08:00:00Z", // Monday before: Friday's
		"2026-11-22T12:00:00Z": "2026-11-20T08:00:00Z", // Sunday: Friday's
	} {
		now, _ := time.Parse(time.RFC3339, at)
		if got := WorkingMorning(now, nil).Format(time.RFC3339); got != want {
			t.Errorf("WorkingMorning(%s) = %s, want %s", at, got, want)
		}
	}
}

func TestPrefsDigest(t *testing.T) {
	for in, want := range map[string]string{"": "", "weekly": "", "daily": "daily"} {
		p, err := Prefs{Digest: in}.Check()
		if err != nil || p.Digest != want {
			t.Errorf("%q: %+v %v", in, p, err)
		}
	}
	if _, err := (Prefs{Digest: "hourly"}).Check(); !errors.Is(err, ErrDigest) {
		t.Errorf("hourly: %v", err)
	}
}

// SB-3805: a daily person gets their digest and watches each working morning and not on
// Monday's weekly run; a weekly person is unchanged; a quiet day sends nothing.
func TestDailyBrief(t *testing.T) {
	tue := time.Date(2026, 11, 24, 8, 30, 0, 0, time.UTC)
	j, s, c := world(tue)
	j.Questions = questions{}
	c.weekly["weekly"] = Monday(tue, nil)
	c.weekly["daily"] = WorkingMorning(tue, nil).AddDate(0, 0, -1)
	j.Digest, j.Zone = true, time.UTC
	j.Prefs = prefs{"usr_ann": {Digest: DigestDaily}, "usr_gone": {Digest: DigestDaily}, "usr_cat": {Digest: DigestDaily}}
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) {
		return []string{"usr_ann", "usr_bob", "usr_gone", "usr_cat"}, nil
	}
	var days []int
	j.Changes = func(_ context.Context, u string, d int) ([]string, int, error) {
		days = append(days, d)
		if u == "usr_cat" {
			return nil, 0, nil // a quiet day
		}
		return []string{"Refund policy"}, 0, nil
	}
	w := &watches{[]ask.SavedQuestion{
		{ID: "sq_1", User: "usr_ann", Question: "SECRETQUESTION a", Watch: true, Sources: "old"},
		{ID: "sq_2", User: "usr_bob", Question: "SECRETQUESTION b", Watch: true, Sources: "old"},
	}}
	j.Watches = w
	j.Sources = func(context.Context, string, string, string) (string, error) { return "new", nil }
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range s.got {
		got = append(got, m.msg.Kind+" "+m.to.ID+": "+m.msg.Text)
	}
	want := []string{
		"digest usr_ann: 1 document changed in your spaces since the last working morning: Refund policy",
		"watch usr_ann: The sources of 1 of your saved questions changed",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("sent:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if w.qs[0].Sources != "new" || w.qs[1].Sources != "old" {
		t.Fatalf("hashes: %+v", w.qs)
	}
	if len(days) != 2 || days[0] != 1 {
		t.Fatalf("days: %v", days)
	}
	// The same morning again sends nothing.
	s.got = nil
	j.Now = func() time.Time { return tue.Add(time.Hour) }
	if _, err := j.Run(context.Background()); err != nil || len(s.got) != 0 {
		t.Fatalf("again: %v %v", s.got, err)
	}

	// Monday: the weekly people get Monday's, the daily ones three days' worth.
	mon := time.Date(2026, 11, 30, 8, 30, 0, 0, time.UTC)
	j, s, c = world(mon)
	j.Questions = questions{}
	c.weekly["weekly"] = Monday(mon, nil).AddDate(0, 0, -7)
	c.weekly["daily"] = WorkingMorning(mon, nil).AddDate(0, 0, -3)
	j.Digest, j.Zone = true, time.UTC
	j.Prefs = prefs{"usr_ann": {Digest: DigestDaily}}
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) {
		return []string{"usr_ann", "usr_bob"}, nil
	}
	days = nil
	j.Changes = func(_ context.Context, u string, d int) ([]string, int, error) {
		days = append(days, d)
		return []string{"Refund policy"}, 0, nil
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, m := range s.got {
		got = append(got, m.to.ID+": "+m.msg.Text)
	}
	want = []string{
		"usr_bob: 1 document changed in your spaces this week: Refund policy",
		"usr_ann: 1 document changed in your spaces since the last working morning: Refund policy",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") || len(days) != 2 || days[0] != 7 || days[1] != 3 {
		t.Fatalf("monday:\n%s\ndays %v", strings.Join(got, "\n"), days)
	}

	// Turned off or paused: nothing, as for the weekly one.
	p, _ := Prefs{Digest: DigestDaily, Off: []string{KindDigest}}.Check()
	j, s, c = world(tue)
	j.Questions = questions{}
	c.weekly["weekly"] = Monday(tue, nil)
	c.weekly["daily"] = WorkingMorning(tue, nil).AddDate(0, 0, -1)
	j.Digest, j.Prefs = true, prefs{"usr_ann": p}
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) { return []string{"usr_ann"}, nil }
	j.Changes = func(context.Context, string, int) ([]string, int, error) { return []string{"x"}, 0, nil }
	if _, err := j.Run(context.Background()); err != nil || len(s.got) != 0 {
		t.Fatalf("off: %v %v", s.got, err)
	}
}
