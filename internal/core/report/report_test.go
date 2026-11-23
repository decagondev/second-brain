package report

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/notify"
)

type src struct{}

func notice(s string) *string { return &s }

func (src) Traces(context.Context, string, time.Time, time.Time) ([]ask.Trace, error) {
	cited := ask.Response{Answer: ask.Answer{Claims: []answer.Claim{{Text: "SECRET-CONTENT", Cites: []answer.Cite{{Cite: "docs:a@1"}}}}}}
	stale := cited
	stale.NoticeStale = &ask.StaleNotice{Stale: 1, Of: 1, OldestDays: 400} // SB-2807
	none := ask.Response{Notice: notice("no_access"), Answer: ask.Answer{Claims: []answer.Claim{{Text: "nothing", Uncited: true}}}}
	return []ask.Trace{{User: "u1", Response: cited}, {User: "u1", Response: cited}, {User: "u2", Response: none}, {User: "u3", Response: stale},
		{User: "u3", Response: ask.Response{Agent: ask.AgentSearch}},                                                                      // a search counts apart (SB-1701)
		{User: "u4", Response: ask.Response{Agent: ask.AgentLimited}}, {User: "u4", Response: ask.Response{Agent: ask.AgentLimited}}}, nil // refusals too (SB-2305)
}
func (src) Drafts(context.Context, string, time.Time, time.Time) (int, int, error) { return 6, 2, nil }
func (src) Actions(context.Context, string, time.Time, time.Time) ([]approvals.Action, error) {
	return []approvals.Action{{State: approvals.Ran}, {State: approvals.Rejected}, {State: approvals.PendingApproval}}, nil
}
func (src) Seats(context.Context, string) (int, error) { return 10, nil }

// InputTokens: this week's calls twice last week's (SB-2906).
func (src) InputTokens(_ context.Context, _ string, from, _ time.Time) ([]int, error) {
	if from.Format("2006-01-02") == "2027-03-08" {
		return []int{4000, 4000, 4100}, nil
	}
	return []int{2000, 2000, 2050}, nil
}
func (src) Tokens(context.Context, string, time.Time, time.Time) (int, int, error) {
	return 2_000_000, 500_000, nil
}

func TestBuildAndMarkdown(t *testing.T) {
	from, to, err := ParseWeek("2027-W10")
	if err != nil || from.Weekday() != time.Monday || from.Format("2006-01-02") != "2027-03-08" {
		t.Fatalf("%v %v", from, err)
	}
	r, err := Build(context.Background(), src{}, "acme-logistics", from, to, Price{InputPerM: 3, OutputPerM: 15})
	if err != nil {
		t.Fatal(err)
	}
	if r.Week != "2027-W10" || r.ActiveUsers != 3 || r.ActivePerSeat != 0.3 || r.Asks != 4 || r.CitedAnswerRate != 0.75 || r.Gaps["no_access"] != 1 || r.Searches != 1 || r.Limited != 2 {
		t.Fatalf("%+v", r)
	}
	if r.Actions["proposed"] != 3 || r.Actions["ran_with_receipt"] != 1 || r.Actions["rejected"] != 1 || r.SpendUSD != 13.5 {
		t.Fatalf("%+v", r)
	}
	md := r.Markdown()
	for _, want := range []string{"3 of 10 (30%)", "| Cited-answer rate | 75% |", "no_access 1", "6 / 2 (3.0 to 1)", "$13.50", "| Searches | 1 |", "| Asks refused by the per-person limit | 2 |", "| Answers resting on stale sources | 1 of 3 cited answers (33%) |", "| Input tokens per model call (95th percentile) | 4100 (last week 2050, growing) |"} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q:\n%s", want, md)
		}
	}
	if strings.Contains(md, "SECRET-CONTENT") || strings.Contains(md, "docs:a") {
		t.Fatalf("the report carries content:\n%s", md)
	}
	if w, f, _ := WeekOf(time.Date(2027, 3, 10, 15, 0, 0, 0, time.UTC)); w != "2027-W10" || !f.Equal(from) {
		t.Fatalf("WeekOf: %s %v", w, f)
	}
	if _, _, err := ParseWeek("2027-10"); err == nil {
		t.Error("a bad week was accepted")
	}
}

type withNotify struct{ src }

func (withNotify) LastNotify(context.Context) (notify.RunRecord, bool, error) {
	return notify.RunRecord{At: time.Date(2027, 3, 14, 9, 5, 0, 0, time.UTC), Via: "email", Sent: 4, Skipped: 1, Failed: 2, Error: true}, true, nil
}

// SB-2607: the report says how the notify job last ran, counts only; a source without it has no line.
func TestNotifyLine(t *testing.T) {
	from, to, _ := ParseWeek("2027-W10")
	r, err := Build(context.Background(), withNotify{}, "acme-logistics", from, to, Price{})
	if err != nil {
		t.Fatal(err)
	}
	if want := "| Notifications | last run 2027-03-14 09:05 UTC by email, failed: 4 sent, 1 skipped, 2 failed |"; !strings.Contains(r.Markdown(), want) {
		t.Fatalf("lacks %q:\n%s", want, r.Markdown())
	}
	r, _ = Build(context.Background(), src{}, "acme-logistics", from, to, Price{})
	if strings.Contains(r.Markdown(), "Notifications") {
		t.Fatal("a cell that keeps no notify runs has a notifications line")
	}
}
