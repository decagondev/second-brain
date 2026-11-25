package notify

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

// SB-4504: an overdue wait goes once to the stewards of the spaces it names, saying who
// waits on whom since when, never its reason or text.
func TestOverdueWaitsGoUpOnce(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	j, s, _ := world(now)
	j.Questions = questions{}
	j.Waits = func(context.Context) ([]ask.OverdueWait, error) {
		return []ask.OverdueWait{{Key: "request-qs_1-2026-11-20", Kind: ask.WaitRequest, From: "ap", On: "legal", Until: "2026-11-20", To: []string{"finance"}}}, nil
	}
	for run := 0; run < 2; run++ {
		s.got = nil
		if _, err := j.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		var told []string
		for _, m := range s.got {
			if strings.Contains(m.msg.Text, "waited on") {
				told = append(told, m.to.ID+": "+m.msg.Text)
			}
		}
		want := "usr_sam: ap has waited on legal since 2026-11-20 (a request)"
		if run == 1 {
			want = ""
		}
		if strings.Join(told, "|") != want {
			t.Fatalf("run %d told %q", run, told)
		}
	}
}

// SB-4506: a pause that ended offers Catch me up once, from when it began.
func TestCatchUpDueAfterAPause(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	p, err := Prefs{}.Pause(3, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.CatchUpDue(now.Add(time.Hour)); ok {
		t.Fatal("offered while paused")
	}
	if since, ok := p.CatchUpDue(now.Add(4 * 24 * time.Hour)); !ok || !since.Equal(now) {
		t.Fatalf("not offered after: %v %v", since, ok)
	}
	p.CatchUpOffered = true
	if _, ok := p.CatchUpDue(now.Add(4 * 24 * time.Hour)); ok {
		t.Fatal("offered twice")
	}
	if p, _ = p.Pause(0, now); p.PausedAt == nil {
		t.Fatal("ending a pause forgot when it began")
	}
}
