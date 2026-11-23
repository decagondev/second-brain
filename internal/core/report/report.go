// Package report is the weekly leadership report and the usage API's numbers
// (sb-observability(7), Weekly leadership report; SB-606). Counts only, never content.
package report

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
)

// Source reads one week of a cell's records.
type Source interface {
	Traces(ctx context.Context, tenant string, from, to time.Time) ([]ask.Trace, error)
	Drafts(ctx context.Context, tenant string, from, to time.Time) (used, discarded int, err error)
	Actions(ctx context.Context, tenant string, from, to time.Time) ([]approvals.Action, error)
	Seats(ctx context.Context, tenant string) (int, error)
	Tokens(ctx context.Context, tenant string, from, to time.Time) (input, output int, err error)
}

// QuestionSource is a source that also has questions to stewards (SB-1704).
type QuestionSource interface {
	StewardQuestions(ctx context.Context, tenant string, from, to time.Time) ([]ask.StewardQuestion, error)
}

// ChangeSource is a source that knows which documents changed, by space (SB-1708).
type ChangeSource interface {
	ChangedDocs(ctx context.Context, tenant string, from, to time.Time) (map[string]int, error)
}

// Price is the release's price per million tokens.
type Price struct{ InputPerM, OutputPerM float64 }

// Report is one week.
type Report struct {
	Tenant        string    `json:"tenant"`
	Week          string    `json:"week"` // 2027-W10
	From, To      time.Time `json:"-"`
	ActiveUsers   int       `json:"active_users"`
	Seats         int       `json:"seats"`
	ActivePerSeat float64   `json:"active_per_seat"`
	Asks          int       `json:"asks"`
	Searches      int       `json:"searches"` // searches without the model (SB-1701)
	// Limited is asks refused for asks_per_minute (SB-2305), never counted as asks.
	Limited int `json:"limited"`
	// Questions to stewards asked and answered this week, by space (SB-1704), never their text.
	QuestionsAsked    map[string]int `json:"questions_asked,omitempty"`
	QuestionsAnswered map[string]int `json:"questions_answered,omitempty"`
	// Changed documents this week by space, never rebuilt versions (SB-1708).
	Changed         map[string]int `json:"changed,omitempty"`
	CitedAnswerRate float64        `json:"cited_answer_rate"`
	Gaps            map[string]int `json:"gaps"`
	DraftsUsed      int            `json:"drafts_used"`
	DraftsDiscarded int            `json:"drafts_discarded"`
	Actions         map[string]int `json:"actions"`
	TokensIn        int            `json:"tokens_in"`
	TokensOut       int            `json:"tokens_out"`
	SpendUSD        float64        `json:"spend_usd"`
}

// WeekOf is the ISO week label and its Monday-to-Monday UTC bounds.
func WeekOf(t time.Time) (string, time.Time, time.Time) {
	d := t.UTC()
	from := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	for from.Weekday() != time.Monday {
		from = from.AddDate(0, 0, -1)
	}
	y, w := from.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w), from, from.AddDate(0, 0, 7)
}

// ParseWeek reads "2027-W10".
func ParseWeek(s string) (time.Time, time.Time, error) {
	var y, w int
	if _, err := fmt.Sscanf(s, "%d-W%d", &y, &w); err != nil || w < 1 || w > 53 {
		return time.Time{}, time.Time{}, fmt.Errorf("report: week must be like 2027-W10")
	}
	jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC) // always in ISO week 1
	mon := jan4
	for mon.Weekday() != time.Monday {
		mon = mon.AddDate(0, 0, -1)
	}
	from := mon.AddDate(0, 0, 7*(w-1))
	return from, from.AddDate(0, 0, 7), nil
}

// Build computes one week.
func Build(ctx context.Context, src Source, tenant string, from, to time.Time, price Price) (Report, error) {
	label, _, _ := WeekOf(from)
	r := Report{Tenant: tenant, Week: label, From: from, To: to, Gaps: map[string]int{}, Actions: map[string]int{}}
	traces, err := src.Traces(ctx, tenant, from, to)
	if err != nil {
		return r, err
	}
	active := map[string]bool{}
	cited := 0
	for _, t := range traces {
		if t.Response.Agent == ask.AgentSearch {
			r.Searches++
			active[t.User] = true
			continue
		}
		if t.Response.Agent == ask.AgentLimited {
			r.Limited++
			continue
		}
		r.Asks++
		active[t.User] = true
		any := false
		for _, c := range t.Response.Answer.Claims {
			if !c.Uncited && len(c.Cites) > 0 {
				any = true
			}
		}
		if any {
			cited++
		}
		if t.Response.Notice != nil {
			r.Gaps[*t.Response.Notice]++
		}
	}
	if r.Asks > 0 {
		r.CitedAnswerRate = float64(cited) / float64(r.Asks)
	}
	if qs, ok := src.(QuestionSource); ok {
		got, err := qs.StewardQuestions(ctx, tenant, from, to)
		if err != nil {
			return r, err
		}
		r.QuestionsAsked, r.QuestionsAnswered = map[string]int{}, map[string]int{}
		for _, q := range got {
			sp := strings.TrimPrefix(q.Space, "spc_")
			r.QuestionsAsked[sp]++
			if q.Reply != nil {
				r.QuestionsAnswered[sp]++
			}
		}
	}
	if cs, ok := src.(ChangeSource); ok {
		if r.Changed, err = cs.ChangedDocs(ctx, tenant, from, to); err != nil {
			return r, err
		}
	}
	if r.DraftsUsed, r.DraftsDiscarded, err = src.Drafts(ctx, tenant, from, to); err != nil {
		return r, err
	}
	acts, err := src.Actions(ctx, tenant, from, to)
	if err != nil {
		return r, err
	}
	for _, a := range acts {
		r.Actions["proposed"]++
		switch a.State {
		case approvals.Ran:
			r.Actions["ran_with_receipt"]++
			r.Actions["approved"]++
		case approvals.DidNotRun:
			r.Actions["did_not_run"]++
			r.Actions["approved"]++
		case approvals.Rejected:
			r.Actions["rejected"]++
		case approvals.Expired:
			r.Actions["expired"]++
		case approvals.Drafted, approvals.PendingApproval, approvals.Approved, approvals.Executing:
			// still in flight: counted as proposed only
		}
	}
	if r.Seats, err = src.Seats(ctx, tenant); err != nil {
		return r, err
	}
	r.ActiveUsers = len(active)
	if r.Seats > 0 {
		r.ActivePerSeat = float64(r.ActiveUsers) / float64(r.Seats)
	}
	if r.TokensIn, r.TokensOut, err = src.Tokens(ctx, tenant, from, to); err != nil {
		return r, err
	}
	r.SpendUSD = float64(r.TokensIn)/1e6*price.InputPerM + float64(r.TokensOut)/1e6*price.OutputPerM
	return r, nil
}

// Markdown is the report as sent: numbers only, no links to content.
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Second Brain weekly report: %s, %s\n\n", r.Tenant, r.Week)
	fmt.Fprintf(&b, "| Measure | This week |\n|---|---|\n")
	fmt.Fprintf(&b, "| Weekly active users per seat | %d of %d (%.0f%%) |\n", r.ActiveUsers, r.Seats, 100*r.ActivePerSeat)
	fmt.Fprintf(&b, "| Asks | %d |\n", r.Asks)
	fmt.Fprintf(&b, "| Searches | %d |\n", r.Searches)
	fmt.Fprintf(&b, "| Asks refused by the per-person limit | %d |\n", r.Limited)
	fmt.Fprintf(&b, "| Cited-answer rate | %.0f%% |\n", 100*r.CitedAnswerRate)
	kinds := make([]string, 0, len(r.Gaps))
	for k := range r.Gaps {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool {
		return r.Gaps[kinds[i]] > r.Gaps[kinds[j]] || (r.Gaps[kinds[i]] == r.Gaps[kinds[j]] && kinds[i] < kinds[j])
	})
	gaps := make([]string, len(kinds))
	for i, k := range kinds {
		gaps[i] = fmt.Sprintf("%s %d", k, r.Gaps[k])
	}
	if len(gaps) == 0 {
		gaps = []string{"none"}
	}
	fmt.Fprintf(&b, "| Gaps | %s |\n", strings.Join(gaps, ", "))
	if len(r.QuestionsAsked) > 0 {
		sps := make([]string, 0, len(r.QuestionsAsked))
		for k := range r.QuestionsAsked {
			sps = append(sps, k)
		}
		sort.Strings(sps)
		parts := make([]string, len(sps))
		for i, k := range sps {
			parts[i] = fmt.Sprintf("%s %d of %d", k, r.QuestionsAnswered[k], r.QuestionsAsked[k])
		}
		fmt.Fprintf(&b, "| Questions to stewards answered | %s |\n", strings.Join(parts, ", "))
	}
	if len(r.Changed) > 0 {
		sps := make([]string, 0, len(r.Changed))
		for k := range r.Changed {
			sps = append(sps, k)
		}
		sort.Strings(sps)
		parts := make([]string, len(sps))
		for i, k := range sps {
			parts[i] = fmt.Sprintf("%s %d", k, r.Changed[k])
		}
		fmt.Fprintf(&b, "| Documents changed | %s |\n", strings.Join(parts, ", "))
	}
	ratio := "n/a"
	if r.DraftsDiscarded > 0 {
		ratio = fmt.Sprintf("%.1f to 1", float64(r.DraftsUsed)/float64(r.DraftsDiscarded))
	}
	fmt.Fprintf(&b, "| Drafts used / discarded | %d / %d (%s) |\n", r.DraftsUsed, r.DraftsDiscarded, ratio)
	fmt.Fprintf(&b, "| Actions | %d proposed, %d approved, %d rejected, %d expired, %d ran with a receipt, %d did not run |\n",
		r.Actions["proposed"], r.Actions["approved"], r.Actions["rejected"], r.Actions["expired"], r.Actions["ran_with_receipt"], r.Actions["did_not_run"])
	fmt.Fprintf(&b, "| Spend | $%.2f (%d input, %d output tokens) |\n", r.SpendUSD, r.TokensIn, r.TokensOut)
	return b.String()
}

// Weekly builds last week's report and delivers it (Monday 07:00 UTC, to the approver
// groups through the cell's notification adapter).
func Weekly(ctx context.Context, src Source, tenant string, now time.Time, price Price, deliver func(ctx context.Context, markdown string) error) (Report, error) {
	_, thisMonday, _ := WeekOf(now)
	r, err := Build(ctx, src, tenant, thisMonday.AddDate(0, 0, -7), thisMonday, price)
	if err != nil {
		return r, err
	}
	return r, deliver(ctx, r.Markdown())
}
