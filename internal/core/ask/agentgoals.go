package ask

import (
	"sort"
	"strings"
	"time"
)

// Agent goals (sb-agents(7), Goals; SB-5004): an agent's job, one sentence, and up to three
// targets on its value numbers, from the tenant manifest. The registry shows each target
// against the last GoalWeeks weeks, a target missed two weeks running tells the owner once,
// and the record lists every target with its history. Meeting a target changes nothing.
const GoalWeeks = 4

// The value numbers a target may use (SB-4701).
const (
	MetricPublishedAsIs = "published_as_is"
	MetricDiscardRate   = "discard_rate"
	MetricHours         = "hours_to_publish"
	MetricHelpful       = "helpful_rate"
	MetricCostPerPub    = "cost_per_published_usd"
)

// GoalDef is an agent's job and targets as the manifest sets them.
type GoalDef struct {
	Job     string
	Targets []TargetDef
}

// TargetDef is one target: Metric at least AtLeast, or at most AtMost.
type TargetDef struct {
	Metric  string   `json:"metric"`
	AtLeast *float64 `json:"at_least,omitempty"`
	AtMost  *float64 `json:"at_most,omitempty"`
}

// TargetWeek is one week's value of a target's metric, nil with nothing to count, and
// whether it met the target.
type TargetWeek struct {
	Week  string   `json:"week"`
	Value *float64 `json:"value,omitempty"`
	Met   *bool    `json:"met,omitempty"`
}

// TargetView is a target with its weeks, oldest first.
type TargetView struct {
	TargetDef
	Weeks []TargetWeek `json:"weeks"`
}

// GoalView is an agent's job and its targets' weeks.
type GoalView struct {
	Agent   string       `json:"agent"`
	Job     string       `json:"job,omitempty"`
	Targets []TargetView `json:"targets"`
}

// weekNumbers are an agent's value numbers over one window, counted as AgentValue counts.
type weekNumbers struct {
	drafts, asIs, edited, discarded, rated, up int
	hours                                      []float64
	cost                                       float64
}

func (w weekNumbers) value(metric string) *float64 {
	ratio := func(a, b int) *float64 {
		if b == 0 {
			return nil
		}
		x := float64(a) / float64(b)
		return &x
	}
	switch metric {
	case MetricPublishedAsIs:
		return ratio(w.asIs, w.drafts)
	case MetricDiscardRate:
		return ratio(w.discarded, w.drafts)
	case MetricHours:
		return median(w.hours)
	case MetricHelpful:
		return ratio(w.up, w.rated)
	case MetricCostPerPub:
		if n := w.asIs + w.edited; n > 0 {
			x := w.cost / float64(n)
			return &x
		}
	}
	return nil
}

// numbersOf counts agent's drafts made, traces answered and feedback given in [from, to).
func numbersOf(agent string, qs []StewardQuestion, traces []Trace, fb []Feedback, from, to time.Time) weekNumbers {
	in := func(t time.Time) bool { return !t.Before(from) && t.Before(to) }
	var w weekNumbers
	for _, q := range qs {
		if draftAgent(q) != agent || !in(q.At) {
			continue
		}
		w.drafts++
		switch {
		case q.Thread.Discarded:
			w.discarded++
		case q.Reply != nil:
			if strings.TrimSpace(q.Reply.Text) == strings.TrimSpace(q.Thread.Text) {
				w.asIs++
			} else {
				w.edited++
			}
			w.hours = append(w.hours, q.Reply.At.Sub(q.At).Hours())
		}
	}
	ids := map[string]bool{}
	for _, t := range traces {
		if traceAgent(t) == agent && in(t.At) {
			w.cost += t.CostUSD
			ids[t.ID] = true
		}
	}
	for _, f := range fb {
		if ids[f.Trace] {
			w.rated++
			if f.Rating == "up" {
				w.up++
			}
		}
	}
	return w
}

// met says v meets t; nil for no value.
func (t TargetDef) met(v *float64) *bool {
	if v == nil {
		return nil
	}
	ok := (t.AtLeast != nil && *v >= *t.AtLeast) || (t.AtMost != nil && *v <= *t.AtMost)
	return &ok
}

// GoalViews are the goals of the agents in goals (only those in only, or all when nil),
// each target counted over the weeks complete weeks before now's, oldest first.
func GoalViews(goals map[string]GoalDef, qs []StewardQuestion, traces []Trace, fb []Feedback, now time.Time, weeks int, only []string) []GoalView {
	out := []GoalView{}
	names := make([]string, 0, len(goals))
	for a := range goals {
		if only == nil || contains(only, a) {
			names = append(names, a)
		}
	}
	sort.Strings(names)
	this := WeekStart(now)
	for _, a := range names {
		g := goals[a]
		v := GoalView{Agent: a, Job: g.Job, Targets: []TargetView{}}
		nums := make([]weekNumbers, weeks)
		for i := range nums {
			from := this.AddDate(0, 0, -7*(weeks-i))
			nums[i] = numbersOf(a, qs, traces, fb, from, from.AddDate(0, 0, 7))
		}
		for _, t := range g.Targets {
			tv := TargetView{TargetDef: t, Weeks: make([]TargetWeek, weeks)}
			for i := range nums {
				val := nums[i].value(t.Metric)
				tv.Weeks[i] = TargetWeek{Week: this.AddDate(0, 0, -7*(weeks-i)).Format(time.DateOnly), Value: val, Met: t.met(val)}
			}
			v.Targets = append(v.Targets, tv)
		}
		out = append(out, v)
	}
	return out
}

// Missed is a target missed two complete weeks running: its agent, metric, and the week the
// run of misses began, so the owner is told once a run.
type Missed struct {
	Agent, Metric, Since string
}

// MissedTwice are the targets whose last two weeks in vs both missed.
func MissedTwice(vs []GoalView) []Missed {
	var out []Missed
	for _, v := range vs {
		for _, t := range v.Targets {
			n := len(t.Weeks)
			miss := func(i int) bool { return i >= 0 && t.Weeks[i].Met != nil && !*t.Weeks[i].Met }
			if n < 2 || !miss(n-1) || !miss(n-2) {
				continue
			}
			start := n - 2
			for miss(start - 1) {
				start--
			}
			out = append(out, Missed{Agent: v.Agent, Metric: t.Metric, Since: t.Weeks[start].Week})
		}
	}
	return out
}
