package manifest

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Run AI like a business (sb-tenant(5); SB-5003, SB-5004): a monthly limit per person, and
// an agent's job and targets in the agents map.

// Limits are the tenant's per-person limits (SB-5003).
type Limits struct {
	MonthlyUSDPerPerson float64 `json:"monthly_usd_per_person"`
}

// Bounds of a personal limit.
const (
	MinPersonUSD = 1.0
	MaxPersonUSD = 10000.0
)

// AgentGoal is an agent's job, one sentence, and its targets (SB-5004).
type AgentGoal struct {
	Job     string   `json:"job,omitempty"`
	Targets []Target `json:"targets,omitempty"`
}

// Target is one value number an agent aims at: at least AtLeast or at most AtMost, one of them.
type Target struct {
	Metric  string   `json:"metric"`
	AtLeast *float64 `json:"at_least,omitempty"`
	AtMost  *float64 `json:"at_most,omitempty"`
}

// TargetMetrics are the value numbers a target may use (sb-agents(7), Value; SB-4701).
var TargetMetrics = []string{"published_as_is", "discard_rate", "hours_to_publish", "helpful_rate", "cost_per_published_usd"}

// Goal bounds.
const (
	MaxTargets = 3
	MaxJob     = 200
)

// limits reads limits: { monthly_usd_per_person: N }.
func (c *checker) limits(n *node) {
	if !c.mapOf(n, "limits", "{ monthly_usd_per_person: 20 }") {
		return
	}
	l := &Limits{}
	for i, k := range n.keys {
		f := "limits." + k.value
		if k.value != "monthly_usd_per_person" {
			c.add(k, f, "unknown field; use monthly_usd_per_person")
			continue
		}
		if usd, ok := c.number(n.vals[i], f, MinPersonUSD, MaxPersonUSD); ok {
			l.MonthlyUSDPerPerson = usd
		}
	}
	if l.MonthlyUSDPerPerson == 0 {
		c.add(n, "limits.monthly_usd_per_person", "required field is missing")
		return
	}
	c.t.Limits = l
}

// number is a scalar number in [min, max], unquoted.
func (c *checker) number(v *node, f string, lo, hi float64) (float64, bool) {
	x, ok := c.scalar(v, f)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(x, 64)
	if v.quoted || err != nil || n < lo || n > hi {
		c.add(v, f, "%q must be a number from %g to %g", x, lo, hi)
		return 0, false
	}
	return n, true
}

// agentGoalField reads one field of an agent in the agents map other than monthly_usd: job
// or targets; false for a field it doesn't know.
func (c *checker) agentGoalField(agent string, k, v *node, f string) bool {
	g := c.t.AgentGoals[agent]
	switch k.value {
	case "job":
		j, ok := c.scalar(v, f)
		if !ok {
			return true
		}
		j = strings.TrimSpace(j)
		if j == "" || len(j) > MaxJob || strings.ContainsAny(j, "\n\r") || strings.Count(strings.TrimRight(j, "."), ". ") > 0 {
			c.add(v, f, "the job is one sentence of at most %d characters", MaxJob)
			return true
		}
		g.Job = j
	case "targets":
		g.Targets = c.targets(v, f)
	default:
		return false
	}
	if c.t.AgentGoals == nil {
		c.t.AgentGoals = map[string]AgentGoal{}
	}
	c.t.AgentGoals[agent] = g
	return true
}

// targets reads [{ metric: M, at_least: N } …], at most MaxTargets, each metric once.
func (c *checker) targets(v *node, f string) []Target {
	if v.kind != seqNode {
		c.add(v, f, "must be a list, e.g. [{ metric: published_as_is, at_least: 0.6 }]")
		return nil
	}
	if len(v.items) == 0 || len(v.items) > MaxTargets {
		c.add(v, f, "1 to %d targets", MaxTargets)
	}
	var out []Target
	for i, it := range v.items {
		fi := fmt.Sprintf("%s[%d]", f, i)
		if it.kind != mapNode {
			c.add(it, fi, "must be a map, e.g. { metric: helpful_rate, at_least: 0.7 }")
			continue
		}
		var t Target
		for j, kk := range it.keys {
			fk := fi + "." + kk.value
			switch kk.value {
			case "metric":
				t.Metric = c.oneOf(it.vals[j], fk, TargetMetrics)
			case "at_least", "at_most":
				if x, ok := c.number(it.vals[j], fk, 0, MaxAgentUSD); ok {
					if kk.value == "at_least" {
						t.AtLeast = &x
					} else {
						t.AtMost = &x
					}
				}
			default:
				c.add(kk, fk, "unknown field; use metric, at_least or at_most")
			}
		}
		switch {
		case t.Metric == "":
			c.add(it, fi+".metric", "required field is missing")
		case (t.AtLeast == nil) == (t.AtMost == nil):
			c.add(it, fi, "a target has exactly one of at_least and at_most")
		default:
			for _, o := range out {
				if o.Metric == t.Metric {
					c.add(it, fi+".metric", "%q has a target already", t.Metric)
				}
			}
			out = append(out, t)
		}
	}
	return out
}

// goalPart is an agent's job and targets in the agents map's flow form, "" for none.
func goalPart(g AgentGoal) string {
	var parts []string
	if g.Job != "" {
		parts = append(parts, "job: "+strconv.Quote(g.Job))
	}
	if len(g.Targets) > 0 {
		ts := make([]string, 0, len(g.Targets))
		for _, t := range g.Targets {
			if t.AtLeast != nil {
				ts = append(ts, "{ metric: "+t.Metric+", at_least: "+strconv.FormatFloat(*t.AtLeast, 'f', -1, 64)+" }")
			} else {
				ts = append(ts, "{ metric: "+t.Metric+", at_most: "+strconv.FormatFloat(*t.AtMost, 'f', -1, 64)+" }")
			}
		}
		parts = append(parts, "targets: ["+strings.Join(ts, ", ")+"]")
	}
	return strings.Join(parts, ", ")
}

// limitsLine is the limits row, "" for none.
func limitsLine(t Tenant) string {
	if t.Limits == nil {
		return ""
	}
	return "{ monthly_usd_per_person: " + strconv.FormatFloat(t.Limits.MonthlyUSDPerPerson, 'f', -1, 64) + " }"
}

// GoalAgents are the agents with a job or targets, sorted.
func (t Tenant) GoalAgents() []string {
	out := make([]string, 0, len(t.AgentGoals))
	for a := range t.AgentGoals {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
