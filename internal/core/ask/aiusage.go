package ask

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// The AI command centre (sb-admin(3), AI command centre; SB-5001): 12 weeks of answers and
// their model spend by week, space, feature, agent and tier, counted from the traces, with
// each space's budget, each agent's cap and the personal limit beside them. People appear
// only as counts, and a row of fewer than MinUsageRow answers shows as "fewer than 5".
const (
	UsageWeeks  = 12
	MinUsageRow = 5
	FewerThan5  = "fewer than 5"
)

// UsageRow is one week of one space, feature, agent and tier. A row of fewer than
// MinUsageRow answers shows neither its count nor its people (Masked); its spend still
// counts toward the totals.
type UsageRow struct {
	Week    string  `json:"week"` // the Monday it starts, YYYY-MM-DD (UTC)
	Space   string  `json:"space"`
	Feature string  `json:"feature"`
	Agent   string  `json:"agent"`
	Tier    string  `json:"tier"`
	Uses    int     `json:"uses"`
	People  int     `json:"people"`
	USD     float64 `json:"usd"`
	Masked  bool    `json:"fewer_than_5,omitempty"`
	users   map[string]bool
}

// UsageTotal is a feature's, or everything's, answers and spend over the weeks.
type UsageTotal struct {
	Feature string  `json:"feature,omitempty"`
	Uses    int     `json:"uses"`
	USD     float64 `json:"usd"`
}

// SpaceBudget is one space's monthly budget and this month's spend (SB-4105).
type SpaceBudget struct {
	Space      string  `json:"space"`
	MonthlyUSD int     `json:"monthly_usd"`
	SpentUSD   float64 `json:"spent_usd"`
}

// LimitBands counts the people of this month against the personal limit (SB-5003): under
// 80%, 80% to the limit, and at it, each "fewer than 5" under MinUsageRow, never who.
type LimitBands struct {
	MonthlyUSD float64 `json:"monthly_usd_per_person"`
	Under80    string  `json:"under_80"`
	From80     string  `json:"from_80"`
	AtLimit    string  `json:"at_limit"`
}

// AIUsage is GET /v1/admin/ai/usage.
type AIUsage struct {
	Weeks    int           `json:"weeks"`
	From     string        `json:"from"`
	Rows     []UsageRow    `json:"rows"`
	Features []UsageTotal  `json:"features"`
	Total    UsageTotal    `json:"total"`
	Budgets  []SpaceBudget `json:"budgets"`
	Caps     []AgentSpend  `json:"caps"`
	Limit    *LimitBands   `json:"limit,omitempty"`
}

// WeekStart is the Monday 00:00 UTC of t's week.
func WeekStart(t time.Time) time.Time {
	t = t.UTC()
	d := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-d, 0, 0, 0, 0, time.UTC)
}

// usageFeature is the feature a trace counts under, "" for one the centre doesn't count.
func usageFeature(t Trace) string {
	if traceAgent(t) == "" || t.Feature == FeatureEval {
		return ""
	}
	if t.Feature == "" {
		return FeatureAsk
	}
	return t.Feature
}

// BuildAIUsage counts traces from the UsageWeeks weeks to now's.
func BuildAIUsage(traces []Trace, now time.Time) AIUsage {
	from := WeekStart(now).AddDate(0, 0, -7*(UsageWeeks-1))
	rows := map[[5]string]*UsageRow{}
	feats := map[string]*UsageTotal{}
	out := AIUsage{Weeks: UsageWeeks, From: from.Format(time.DateOnly), Rows: []UsageRow{}, Features: []UsageTotal{}, Budgets: []SpaceBudget{}, Caps: []AgentSpend{}}
	for _, t := range traces {
		f := usageFeature(t)
		if f == "" || t.At.Before(from) || t.At.After(now) {
			continue
		}
		tier := t.Tier
		if tier == "" {
			tier = TierMain
		}
		space := spaceName(t.Space)
		if space == "" {
			space = "company"
		}
		k := [5]string{WeekStart(t.At).Format(time.DateOnly), space, f, traceAgent(t), tier}
		r := rows[k]
		if r == nil {
			r = &UsageRow{Week: k[0], Space: k[1], Feature: k[2], Agent: k[3], Tier: k[4], users: map[string]bool{}}
			rows[k] = r
		}
		r.Uses++
		r.USD += t.CostUSD
		r.users[t.User] = true
		if feats[f] == nil {
			feats[f] = &UsageTotal{Feature: f}
		}
		feats[f].Uses++
		feats[f].USD += t.CostUSD
		out.Total.Uses++
		out.Total.USD += t.CostUSD
	}
	for _, r := range rows {
		r.People = len(r.users)
		if r.Uses < MinUsageRow {
			r.Uses, r.People, r.Masked = 0, 0, true
		}
		out.Rows = append(out.Rows, *r)
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		a, b := out.Rows[i], out.Rows[j]
		for _, p := range [][2]string{{a.Week, b.Week}, {a.Space, b.Space}, {a.Feature, b.Feature}, {a.Agent, b.Agent}, {a.Tier, b.Tier}} {
			if p[0] != p[1] {
				return p[0] < p[1]
			}
		}
		return false
	})
	for _, f := range Features {
		if t, ok := feats[f]; ok {
			out.Features = append(out.Features, *t)
		}
	}
	return out
}

// Band is a count shown as itself, or "fewer than 5" under MinUsageRow.
func Band(n int) string {
	if n < MinUsageRow {
		return FewerThan5
	}
	return strconv.Itoa(n)
}

// Bands counts spent (each person's spend this month) against limit.
func Bands(spent map[string]float64, limit float64) *LimitBands {
	var under, from, at int
	for _, usd := range spent {
		switch {
		case usd >= limit:
			at++
		case usd >= 0.8*limit:
			from++
		default:
			under++
		}
	}
	return &LimitBands{MonthlyUSD: limit, Under80: Band(under), From80: Band(from), AtLimit: Band(at)}
}

// CSV is the rows as CSV, the page's numbers exactly: a masked row says "fewer than 5".
func (u AIUsage) CSV() []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"week", "space", "feature", "agent", "tier", "uses", "people", "usd"})
	for _, r := range u.Rows {
		uses, people := strconv.Itoa(r.Uses), strconv.Itoa(r.People)
		if r.Masked {
			uses, people = FewerThan5, FewerThan5
		}
		_ = w.Write([]string{r.Week, r.Space, r.Feature, r.Agent, r.Tier, uses, people, fmt.Sprintf("%.4f", r.USD)})
	}
	_ = w.Write([]string{"total", "", "", "", "", strconv.Itoa(u.Total.Uses), "", fmt.Sprintf("%.4f", u.Total.USD)})
	w.Flush()
	return b.Bytes()
}
