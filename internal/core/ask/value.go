package ask

import (
	"sort"
	"strings"
	"time"
)

// Agent value (sb-agents(7), Value; SB-4701): what each agent's drafts came to over
// ValueWeeks, per agent and space, counted from the steward queues, the traces and the
// feedback. Counts only: never a draft's text, a question or who asked.

// ValueWeeks are the weeks value counts back.
const ValueWeeks = 12

// MinDoubled is the fewest drafts in a week whose discard share can be called doubled.
const MinDoubled = 4

// ValueRow is one agent in one space.
type ValueRow struct {
	Agent           string   `json:"agent"`
	Space           string   `json:"space"`
	Drafts          int      `json:"drafts"`
	Published       int      `json:"published"`
	PublishedEdited int      `json:"published_edited"`
	Discarded       int      `json:"discarded"`
	Rounds          int      `json:"rounds"`
	MedianHours     *float64 `json:"median_hours,omitempty"`
	Rated           int      `json:"rated"`
	HelpfulRate     *float64 `json:"helpful_rate,omitempty"`
	CostUSD         float64  `json:"cost_usd"`
	CostPerPublish  *float64 `json:"cost_per_published_usd,omitempty"`
	Paused          bool     `json:"paused,omitempty"`
	hours           []float64
	up              int
}

// AgentSpend is one agent's model spend this month, against its cap (SB-4703).
type AgentSpend struct {
	Agent      string   `json:"agent"`
	Month      string   `json:"month"`
	SpentUSD   float64  `json:"spent_usd"`
	MonthlyUSD *float64 `json:"monthly_usd,omitempty"`
	Capped     bool     `json:"capped,omitempty"`
}

// ValueReport is GET /v1/admin/agents/value.
type ValueReport struct {
	Weeks int          `json:"weeks"`
	From  string       `json:"from"`
	Rows  []ValueRow   `json:"rows"`
	Spend []AgentSpend `json:"spend"`
}

// draftAgent is the agent of an agent draft, or "" for anything else in a queue.
func draftAgent(q StewardQuestion) string {
	if q.Thread == nil {
		return ""
	}
	return q.Thread.Agent
}

// traceAgent is the agent a trace counts for: "" is ask; searches, evals, refused rate
// limits and welcomes are no agent's.
func traceAgent(t Trace) string {
	switch a := t.Response.Agent; a {
	case "":
		return "ask"
	case AgentSearch, AgentEval, AgentLimited, AgentWelcome:
		return ""
	default:
		return a
	}
}

// AgentValue counts qs (the steward questions made since from), traces and fb (the
// feedback since from) for the agents only names, or every agent when only is nil.
func AgentValue(qs []StewardQuestion, traces []Trace, fb []Feedback, from time.Time, only []string) []ValueRow {
	keep := func(a string) bool { return a != "" && (only == nil || contains(only, a)) }
	rows := map[[2]string]*ValueRow{}
	row := func(agent, space string) *ValueRow {
		k := [2]string{agent, spaceName(space)}
		if rows[k] == nil {
			rows[k] = &ValueRow{Agent: agent, Space: spaceName(space)}
		}
		return rows[k]
	}
	for _, q := range qs {
		a := draftAgent(q)
		if !keep(a) || q.At.Before(from) {
			continue
		}
		r := row(a, q.Space)
		r.Drafts++
		r.Rounds += q.Thread.Rounds
		switch {
		case q.Thread.Discarded:
			r.Discarded++
		case q.Reply != nil:
			if strings.TrimSpace(q.Reply.Text) == strings.TrimSpace(q.Thread.Text) {
				r.Published++
			} else {
				r.PublishedEdited++
			}
			r.hours = append(r.hours, q.Reply.At.Sub(q.At).Hours())
		}
	}
	by := map[string]*ValueRow{}
	for _, t := range traces {
		a := traceAgent(t)
		if !keep(a) || t.At.Before(from) || t.Space == "" {
			continue
		}
		r := row(a, t.Space)
		r.CostUSD += t.CostUSD
		by[t.ID] = r
	}
	for _, f := range fb {
		if r, ok := by[f.Trace]; ok && !f.At.Before(from) {
			r.Rated++
			if f.Rating == "up" {
				r.up++
			}
		}
	}
	out := []ValueRow{}
	for _, r := range rows {
		if r.Drafts == 0 && r.Rated == 0 && r.CostUSD == 0 {
			continue // nothing to say
		}
		r.MedianHours = median(r.hours) // to a tenth of an hour, as service levels
		if r.Rated > 0 {
			h := float64(r.up) / float64(r.Rated)
			r.HelpfulRate = &h
		}
		if n := r.Published + r.PublishedEdited; n > 0 {
			c := r.CostUSD / float64(n)
			r.CostPerPublish = &c
		}
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Agent != out[j].Agent {
			return out[i].Agent < out[j].Agent
		}
		return out[i].Space < out[j].Space
	})
	return out
}

// Doubled is an agent whose share of drafts discarded last week is at least twice the week
// before's (SB-4701).
type Doubled struct {
	Agent, Week          string // Week is last week, YYYY-Www
	Discarded, Drafts    int
	BeforeDiscarded, Was int
}

// DoubledDiscards are the agents of qs whose discard share in the ISO week before now's, with
// at least MinDoubled drafts, is at least twice the share of the week before that.
func DoubledDiscards(qs []StewardQuestion, now time.Time, zone *time.Location) []Doubled {
	last := isoWeek(now.AddDate(0, 0, -7), zone)
	before := isoWeek(now.AddDate(0, 0, -14), zone)
	type tally struct{ drafts, discarded int }
	count := map[string]map[string]*tally{}
	for _, q := range qs {
		a := draftAgent(q)
		if a == "" {
			continue
		}
		w := isoWeek(q.At, zone)
		if w != last && w != before {
			continue
		}
		if count[a] == nil {
			count[a] = map[string]*tally{last: {}, before: {}}
		}
		c := count[a][w]
		c.drafts++
		if q.Thread.Discarded {
			c.discarded++
		}
	}
	var out []Doubled
	for a, c := range count {
		l, b := c[last], c[before]
		if l.drafts < MinDoubled || b.drafts == 0 || b.discarded == 0 {
			continue
		}
		// l.discarded/l.drafts >= 2 * b.discarded/b.drafts, in whole numbers
		if l.discarded*b.drafts >= 2*b.discarded*l.drafts {
			out = append(out, Doubled{Agent: a, Week: last, Discarded: l.discarded, Drafts: l.drafts, BeforeDiscarded: b.discarded, Was: b.drafts})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Agent < out[j].Agent })
	return out
}

// MonthSpend is each agent's cost_usd in traces of month (UTC, YYYY-MM), with the caps
// given; an agent with a cap and no spend is listed too.
func MonthSpend(traces []Trace, month string, caps map[string]float64, only []string) []AgentSpend {
	spent := map[string]float64{}
	for _, t := range traces {
		if a := traceAgent(t); a != "" && t.At.UTC().Format("2006-01") == month {
			spent[a] += t.CostUSD
		}
	}
	for a := range caps {
		if _, ok := spent[a]; !ok {
			spent[a] = 0
		}
	}
	out := []AgentSpend{}
	for a, usd := range spent {
		if only != nil && !contains(only, a) {
			continue
		}
		s := AgentSpend{Agent: a, Month: month, SpentUSD: usd}
		if c, ok := caps[a]; ok {
			c := c
			s.MonthlyUSD, s.Capped = &c, usd >= c
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Agent < out[j].Agent })
	return out
}
