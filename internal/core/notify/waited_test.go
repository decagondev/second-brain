package notify

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
)

// mutq is a question store whose updates are kept.
type mutq struct{ questions }

func (q *mutq) StewardQuestionsBetween(ctx context.Context, t string, from, to time.Time) ([]ask.StewardQuestion, error) {
	return q.questions.StewardQuestionsBetween(ctx, t, from, to)
}
func (q *mutq) UpdateStewardQuestion(_ context.Context, _, id string, f func(*ask.StewardQuestion) error) (ask.StewardQuestion, error) {
	for i := range q.qs {
		if q.qs[i].ID == id {
			err := f(&q.qs[i])
			return q.qs[i], err
		}
	}
	return ask.StewardQuestion{}, nil
}

type auditRows struct{ got []audit.Event }

func (r *auditRows) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	r.got = append(r.got, e)
	return audit.Row{}, nil
}

// SB-3404: a question open 10 working days, with no live claim, is shown to the company-wide
// answerers once, audited, and told as a count; one at day 9, a claimed one and an answered
// one aren't.
func TestOldQuestionsWaitThenGoCompanyWide(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC) // a Monday
	j, s, _ := world(now)
	twoWeeks := now.AddDate(0, 0, -14) // 10 working days
	q := &mutq{questions{[]ask.StewardQuestion{
		{ID: "qs_1", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_finance", At: twoWeeks},
		{ID: "qs_2", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_finance", At: now.AddDate(0, 0, -13)},
		{ID: "qs_3", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_finance", At: twoWeeks,
			Claim: &ask.StewardClaim{By: "usr_sam", Until: now.Add(time.Hour)}},
		{ID: "qs_4", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "spc_finance", At: twoWeeks,
			Reply: &ask.StewardReply{Text: "SECRETREPLY", At: twoWeeks}},
		{ID: "qs_5", Asker: "usr_ann", Question: "SECRETQUESTION", Space: "company", At: twoWeeks},
	}}}
	rows := &auditRows{}
	j.Questions, j.Audit, j.Zone = q, rows, time.UTC
	if ask.WorkingDays(twoWeeks, now, time.UTC) != 10 || ask.WorkingDays(now.AddDate(0, 0, -13), now, time.UTC) != 9 {
		t.Fatal("working days")
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if q.qs[0].Also != "company" || q.qs[0].Waited != 10 || q.qs[0].Space != "spc_finance" {
		t.Fatalf("qs_1 %+v", q.qs[0])
	}
	for _, i := range []int{1, 2, 3, 4} {
		if q.qs[i].Also != "" {
			t.Fatalf("%s went company-wide", q.qs[i].ID)
		}
	}
	if len(rows.got) != 1 || rows.got[0].Subject != "qs_1" || rows.got[0].State != "rerouted" || rows.got[0].Actor != "sys_notify" {
		t.Fatalf("audit %+v", rows.got)
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
	if strings.Join(told, "|") != "usr_hal: 1 question waited 10 working days in its space https://brain.acme.example/?steward=company" {
		t.Fatalf("told %q", told)
	}
	// The next run tells nobody again; the claim on qs_3 lapses and it goes too.
	s.got, rows.got = nil, nil
	q.qs[2].Claim.Until = now.Add(-time.Minute)
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(rows.got) != 1 || rows.got[0].Subject != "qs_3" || q.qs[2].Also != "company" {
		t.Fatalf("after the claim lapsed %+v", rows.got)
	}
	// Without an auditor nothing goes company-wide.
	j.Audit = nil
	q.qs[1].At = twoWeeks
	if _, err := j.Run(context.Background()); err != nil || q.qs[1].Also != "" {
		t.Fatalf("rerouted without an audit: %v", err)
	}
}

// SB-3404: the steward's digest line counts overdue questions first, and lists the spaces
// that have them first.
func TestDigestOverdueFirst(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC) // a Monday
	j, s, c := world(now)
	c.weekly["weekly"] = Monday(now, nil).AddDate(0, 0, -7)
	j.Questions = questions{[]ask.StewardQuestion{
		{ID: "qs_1", Space: "spc_finance", At: now.Add(-time.Hour)},
		{ID: "qs_2", Space: "spc_ops", At: now.AddDate(0, 0, -7)}, // 5 working days
	}}
	j.Facts = func(sp string) ([]string, string, bool) {
		if sp == "company" {
			return nil, "", false
		}
		return []string{"group:helpdesk"}, "", true
	}
	j.Clock.(*clock).t = now.Add(-time.Minute)
	j.Digest, j.Zone = true, time.UTC
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) { return nil, nil }
	j.Changes = func(context.Context, string, int) ([]string, int, error) { return nil, 0, nil }
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "ops: 1 overdue, 0 answered this week, 1 still open, the oldest 7 days old\nfinance: 0 answered this week, 1 still open, the oldest 0 days old"
	if len(s.got) != 1 || s.got[0].msg.Text != want {
		t.Fatalf("%+v", s.got)
	}
}

func TestWorkingDays(t *testing.T) {
	mon := time.Date(2026, 11, 23, 17, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		from, to time.Time
		want     int
	}{
		{mon, mon.AddDate(0, 0, 7), 5},
		{mon.AddDate(0, 0, -3), mon, 1}, // Friday to Monday
		{mon.AddDate(0, 0, -2), mon.AddDate(0, 0, -1), 0},
		{mon, mon.Add(time.Hour), 0},
	} {
		if got := ask.WorkingDays(c.from, c.to, nil); got != c.want {
			t.Errorf("%v to %v: %d, want %d", c.from, c.to, got, c.want)
		}
	}
}
