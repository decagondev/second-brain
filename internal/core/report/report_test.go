package report

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
)

type src struct{}

func notice(s string) *string { return &s }

func (src) Traces(context.Context, string, time.Time, time.Time) ([]ask.Trace, error) {
	cited := ask.Response{Answer: ask.Answer{Claims: []answer.Claim{{Text: "SECRET-CONTENT", Cites: []answer.Cite{{Cite: "docs:a@1"}}}}}}
	none := ask.Response{Notice: notice("no_access"), Answer: ask.Answer{Claims: []answer.Claim{{Text: "nothing", Uncited: true}}}}
	return []ask.Trace{{User: "u1", Response: cited}, {User: "u1", Response: cited}, {User: "u2", Response: none}, {User: "u3", Response: cited},
		{User: "u3", Response: ask.Response{Agent: ask.AgentSearch}},                                                                      // a search counts apart (SB-1701)
		{User: "u4", Response: ask.Response{Agent: ask.AgentLimited}}, {User: "u4", Response: ask.Response{Agent: ask.AgentLimited}}}, nil // refusals too (SB-2305)
}
func (src) Drafts(context.Context, string, time.Time, time.Time) (int, int, error) { return 6, 2, nil }
func (src) Actions(context.Context, string, time.Time, time.Time) ([]approvals.Action, error) {
	return []approvals.Action{{State: approvals.Ran}, {State: approvals.Rejected}, {State: approvals.PendingApproval}}, nil
}
func (src) Seats(context.Context, string) (int, error) { return 10, nil }
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
	for _, want := range []string{"3 of 10 (30%)", "| Cited-answer rate | 75% |", "no_access 1", "6 / 2 (3.0 to 1)", "$13.50", "| Searches | 1 |", "| Asks refused by the per-person limit | 2 |"} {
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
