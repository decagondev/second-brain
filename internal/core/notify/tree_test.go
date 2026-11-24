package notify

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

// SB-3904: a team's question reaches its department's stewards after 10 working days and
// the company-wide answerers after 20, staying in each queue it reached; a claim holds it,
// and the messages carry counts only.
func TestQuestionsClimbTheTree(t *testing.T) {
	day10 := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC) // a Monday
	asked := day10.AddDate(0, 0, -14)                      // 10 working days before
	j, s, _ := world(day10)
	q := &mutq{questions{[]ask.StewardQuestion{
		{ID: "qs_1", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_ap", At: asked},
		{ID: "qs_2", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_ap", At: asked,
			Claim: &ask.StewardClaim{By: "usr_sam", Until: day10.Add(time.Hour)}},
	}}}
	rows := &auditRows{}
	j.Questions, j.Audit, j.Zone = q, rows, time.UTC
	j.Parent = func(sp string) string {
		if sp == "spc_ap" {
			return "spc_finance"
		}
		return ""
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(q.qs[0].Up, ","); got != "spc_finance" || q.qs[0].Also != "" || q.qs[0].Waited != 10 {
		t.Fatalf("day 10: %+v", q.qs[0])
	}
	if len(q.qs[1].Up) != 0 {
		t.Fatal("a claimed question climbed")
	}
	var told []string
	for _, m := range s.got {
		if strings.Contains(m.msg.Text, "SECRET") {
			t.Fatal("a question in a message")
		}
		if strings.Contains(m.msg.Text, "waited") {
			told = append(told, m.to.ID+": "+m.msg.Text+" "+m.msg.Link)
		}
	}
	if strings.Join(told, "|") != "usr_sam: 1 question waited 10 working days in a team https://brain.acme.example/?steward=finance" {
		t.Fatalf("day 10 told %q", told)
	}
	// Day 20: on to company, still in the department's queue.
	day20 := day10.AddDate(0, 0, 14)
	j.Now = func() time.Time { return day20 }
	s.got, rows.got = nil, nil
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(q.qs[0].Up, ","); got != "spc_finance,company" || q.qs[0].Also != "company" || q.qs[0].Waited != 20 {
		t.Fatalf("day 20: %+v", q.qs[0])
	}
	if len(rows.got) < 1 || rows.got[0].Subject != "qs_1" || !strings.Contains(rows.got[0].Reason, "to company") {
		t.Fatalf("audit %+v", rows.got)
	}
	// The claim lapsed on day 10, so qs_2 has now waited 20 days: it takes one step a run.
	if got := strings.Join(q.qs[1].Up, ","); got != "spc_finance" {
		t.Fatalf("qs_2 %+v", q.qs[1])
	}
}

// SB-4104: a request climbs once it is past its due days, not after 10, and its message
// says so with counts only.
func TestRequestsClimbWhenOverdue(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC) // a Monday
	asked := now.AddDate(0, 0, -3)                       // the Friday before: 1 working day
	j, s, _ := world(now)
	meta := func(days int) *ask.RequestMeta {
		return &ask.RequestMeta{Category: "Laptop", DueDays: days, Due: ask.AddWorkingDays(asked, days, time.UTC)}
	}
	q := &mutq{questions{[]ask.StewardQuestion{
		{ID: "qs_1", Asker: "usr_ann", Question: "SECRETREQUEST", Space: "spc_ap", At: asked, Request: meta(1)},
		{ID: "qs_2", Asker: "usr_ann", Question: "SECRETREQUEST", Space: "spc_ap", At: asked, Request: meta(2)},
	}}}
	j.Questions, j.Audit, j.Zone = q, &auditRows{}, time.UTC
	j.Parent = func(sp string) string {
		if sp == "spc_ap" {
			return "spc_finance"
		}
		return ""
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(q.qs[0].Up, ",") != "spc_finance" || len(q.qs[1].Up) != 0 {
		t.Fatalf("climbed: %+v %+v", q.qs[0].Up, q.qs[1].Up)
	}
	var told []string
	for _, m := range s.got {
		if strings.Contains(m.msg.Text, "SECRET") || strings.Contains(m.msg.Text, "Laptop") {
			t.Fatal("a request's text or category in a message")
		}
		if strings.Contains(m.msg.Text, "due date") {
			told = append(told, m.to.ID+": "+m.msg.Text)
		}
	}
	if strings.Join(told, "|") != "usr_sam: 1 request went past its due date in a team" {
		t.Fatalf("told %q", told)
	}
	if d := ask.AddWorkingDays(time.Date(2026, 11, 20, 15, 0, 0, 0, time.UTC), 1, time.UTC); d.Weekday() != time.Monday || d.Day() != 23 {
		t.Fatalf("a Friday's 1 working day is due %v", d)
	}
}

// SB-4105: a space's stewards are told once at 80% of its budget and once when it is spent.
func TestSpaceBudgetWarnings(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, s, _ := world(now)
	share := 0.85
	j.SpaceBudgets = func(context.Context) ([]SpaceBudget, error) {
		return []SpaceBudget{{Space: "spc_finance", Month: "2026-11", Share: share}}, nil
	}
	run := func() []string {
		t.Helper()
		s.got = nil
		if _, err := j.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, m := range s.got {
			if strings.Contains(m.msg.Text, "budget") {
				out = append(out, m.to.ID+": "+m.msg.Text)
			}
		}
		return out
	}
	if got := strings.Join(run(), "|"); got != "usr_sam: The finance space has used 85% of its monthly budget" {
		t.Fatalf("80%%: %q", got)
	}
	if got := run(); len(got) != 0 {
		t.Fatalf("told twice: %q", got)
	}
	share = 1.2
	if got := strings.Join(run(), "|"); !strings.HasPrefix(got, "usr_sam: The finance space has spent its monthly budget") {
		t.Fatalf("100%%: %q", got)
	}
}
