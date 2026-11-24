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
