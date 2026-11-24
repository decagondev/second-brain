package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

// SB-4005: a brief runs once on its day, from 08:00; one that can't run tells the stewards.
func TestBriefsRunWeekly(t *testing.T) {
	tue := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC) // a Tuesday
	if got := BriefDay(tue, nil, time.Tuesday); !got.Equal(time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("day %v", got)
	}
	if got := BriefDay(tue.Add(-2*time.Hour), nil, time.Tuesday); !got.Equal(time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("before 08:00 %v", got)
	}
	j, s, c := world(tue)
	j.Questions = questions{}
	c.weekly["weekly"] = Monday(tue, nil)
	ran := map[string]int{}
	why := map[string]string{"spc_finance": "", "spc_ops": BriefBudget}
	j.Briefs = map[string]time.Weekday{"spc_finance": time.Tuesday, "spc_ops": time.Tuesday, "spc_later": time.Friday}
	j.RunBrief = func(_ context.Context, sp string) (string, error) {
		ran[sp]++
		return why[sp], nil
	}
	j.Facts = func(sp string) ([]string, string, bool) {
		if sp == "spc_ops" || sp == "spc_finance" {
			return []string{"group:finance-leads"}, "", true
		}
		return nil, "", false
	}
	// spc_later ran last Friday; its day hasn't come this week.
	c.weekly["brief-spc_later"] = BriefDay(tue, nil, time.Friday)
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ran["spc_finance"] != 1 || ran["spc_ops"] != 1 || ran["spc_later"] != 0 {
		t.Fatalf("ran %v", ran)
	}
	var got []string
	for _, m := range s.got {
		got = append(got, m.msg.Kind+" "+m.to.ID+": "+m.msg.Text)
	}
	if strings.Join(got, "\n") != "questions usr_sam: This week's brief didn't run: the monthly model budget is spent" {
		t.Fatalf("sent %v", got)
	}
	// The same week again runs nothing.
	s.got = nil
	j.Now = func() time.Time { return tue.Add(3 * time.Hour) }
	if _, err := j.Run(context.Background()); err != nil || ran["spc_finance"] != 1 || len(s.got) != 0 {
		t.Fatalf("again: ran %v sent %v err %v", ran, s.got, err)
	}
}
