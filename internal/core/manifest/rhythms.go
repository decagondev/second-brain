package manifest

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Rhythms (sb-space(5), rhythms; sb-departments(7), Rhythms; SB-4801): a space's named
// playbooks for a recurring moment. Each is a few dated steps, every one a draft that
// waits for a named person (LAW-29). A space's own rhythm of a name wins over a built-in
// one and over its department's.
const (
	MaxRhythms     = 10
	MaxRhythmSteps = 12
	MaxRhythmDays  = 120
	MaxRhythmAsk   = 300
)

// The kinds of a rhythm step, its approvers, the subjects a rhythm runs for and the agents
// a draft step may use.
var (
	RhythmKinds     = []string{"draft", "routine", "checkin", "request"}
	RhythmApprovers = []string{"subject", "starter", "manager", "steward"}
	RhythmSubjects  = []string{"person", "collection", "space"}
	RhythmPlans     = []string{"manager", "steward"}
	RhythmAgents    = []string{"ask", "start-here", "catch-up", "kickoff-brief", "retrospective", "gaps", "goal-plan", "one-on-one"}
)

var rhythmNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)

// Rhythm is one playbook: who it runs for, who approves its plan before it starts ("" for
// no one beyond the starter), and its steps in day order.
type Rhythm struct {
	Subject string       `json:"subject"`
	Plan    string       `json:"plan,omitempty"`
	Steps   []RhythmStep `json:"steps"`
}

// RhythmStep is one dated step: a draft, a routine, a goal check-in or a request, offered
// to its approver Day days after the rhythm starts.
type RhythmStep struct {
	Day      int    `json:"day"`
	Kind     string `json:"kind"`
	Ask      string `json:"ask"`
	Approver string `json:"approver"`
	Agent    string `json:"agent,omitempty"`
}

// BuiltinRhythms are the rhythms every space has unless it defines one of the same name.
var BuiltinRhythms = map[string]Rhythm{
	"onboarding": {Subject: "person", Plan: "manager", Steps: []RhythmStep{ // SB-4802
		{Day: 0, Kind: "draft", Agent: "start-here", Approver: "subject", Ask: "Where to start in this space, and the people to meet"},
		{Day: 14, Kind: "draft", Agent: "catch-up", Approver: "subject", Ask: "Catch up on your first two weeks"},
		{Day: 14, Kind: "request", Agent: "gaps", Approver: "steward", Ask: "Questions a new joiner asked that had no answer"},
		{Day: 42, Kind: "draft", Agent: "catch-up", Approver: "subject", Ask: "Catch up on what changed since you joined"},
		{Day: 42, Kind: "request", Agent: "gaps", Approver: "steward", Ask: "Questions a new joiner asked that had no answer"},
		{Day: 90, Kind: "routine", Approver: "subject", Ask: "What changed in this space this week?"},
	}},
	"kickoff": {Subject: "collection", Steps: []RhythmStep{ // SB-4803
		{Day: 0, Kind: "draft", Agent: "kickoff-brief", Approver: "starter", Ask: "What did similar past projects decide, and who owned them?"},
		{Day: 14, Kind: "checkin", Approver: "starter", Ask: "Week-2 check-in on the project's goal"},
		{Day: 45, Kind: "draft", Agent: "catch-up", Approver: "starter", Ask: "Mid-point catch-up on the project"},
	}},
	"close-out": {Subject: "collection", Steps: []RhythmStep{ // SB-4804
		{Day: 0, Kind: "draft", Agent: "retrospective", Approver: "starter", Ask: "Retrospective: goals, decisions, open waits and lessons"},
		{Day: 7, Kind: "request", Approver: "steward", Ask: "Lessons the project proposed, for the stewards to review"},
	}},
	"quarterly-planning": {Subject: "space", Steps: []RhythmStep{ // SB-4805
		{Day: 0, Kind: "draft", Agent: "goal-plan", Approver: "steward", Ask: "Next quarter's goals: what carries over, what is at risk, and what requests asked for"},
		{Day: 7, Kind: "checkin", Approver: "steward", Ask: "Propose next quarter's goals"},
	}},
	"one-on-one": {Subject: "person", Steps: oneOnOneSteps()}, // SB-4807
}

// oneOnOneSteps are a weekly 1:1 brief for 12 weeks, each for the manager, the report
// seeing the same one (LAW-30).
func oneOnOneSteps() []RhythmStep {
	var out []RhythmStep
	for w := 0; w < MaxRhythmSteps; w++ {
		out = append(out, RhythmStep{Day: 7 * w, Kind: "draft", Agent: "one-on-one", Approver: "manager", Ask: "1:1 prep from what they published"})
	}
	return out
}

// RhythmProblem is why a rhythm is refused, "" when it isn't: more than MaxRhythmSteps
// steps, a step past MaxRhythmDays, a kind, approver, agent or subject not in the lists,
// steps out of day order, or an ask that names a tool, an action or a wider scope (the
// skills' content rules, LAW-06, LAW-09).
func RhythmProblem(r Rhythm) string {
	switch {
	case !contains(RhythmSubjects, r.Subject):
		return "subject must be " + orList(RhythmSubjects)
	case r.Plan != "" && !contains(RhythmPlans, r.Plan):
		return "plan must be " + orList(RhythmPlans)
	case len(r.Steps) == 0:
		return "a rhythm needs at least one step"
	case len(r.Steps) > MaxRhythmSteps:
		return fmt.Sprintf("at most %d steps", MaxRhythmSteps)
	}
	for i, st := range r.Steps {
		at := fmt.Sprintf("step %d ", i+1)
		switch {
		case st.Day < 0 || st.Day > MaxRhythmDays:
			return at + fmt.Sprintf("day must be 0 to %d", MaxRhythmDays)
		case i > 0 && st.Day < r.Steps[i-1].Day:
			return at + "comes before the step above it; list steps in day order"
		case !contains(RhythmKinds, st.Kind):
			return at + "kind must be " + orList(RhythmKinds)
		case !contains(RhythmApprovers, st.Approver):
			return at + "needs a named approver: " + orList(RhythmApprovers)
		case st.Approver == "subject" && r.Subject != "person":
			return at + "approver subject needs a person subject"
		case st.Agent != "" && !contains(RhythmAgents, st.Agent):
			return at + "agent must be " + orList(RhythmAgents)
		case st.Agent != "" && st.Agent != "gaps" && st.Kind != "draft":
			return at + "only a draft step uses an agent"
		case strings.TrimSpace(st.Ask) == "" || utf8.RuneCountInString(st.Ask) > MaxRhythmAsk || !printable(st.Ask) || strings.ContainsAny(st.Ask, "\n\r"):
			return at + fmt.Sprintf("ask is 1 to %d printable characters on one line", MaxRhythmAsk)
		case SkillProblem(st.Ask) != "":
			return at + "ask " + SkillProblem(st.Ask)
		}
	}
	return ""
}

// rhythmField reads a space's rhythms or onboarding.
func (c *checker) rhythmField(key string, v *node, f string, s *Space) {
	switch key {
	case "onboarding":
		s.Onboarding = c.boolOf(v, f)
		return
	case "on": // SB-4905
		s.On = c.spaceOn(v, f)
		return
	case "routing": // SB-5002
		s.Routing = c.oneOf(v, f, RoutingPolicies)
		return
	case "suspect": // SB-5101
		s.Suspect = c.oneOf(v, f, SuspectModes)
		return
	}
	s.Rhythms = c.rhythms(v, f)
}

// formatWays is a space's onboarding, rhythms and on lines, "" for none.
func formatWays(sp Space, nl string) string {
	out := ""
	if len(sp.On) > 0 { // SB-4905
		out += fmt.Sprintf("    on: %s%s", flowList(sp.On, false), nl)
	}
	if sp.Routing != "" { // SB-5002
		out += fmt.Sprintf("    routing: %s%s", sp.Routing, nl)
	}
	if sp.Suspect != "" { // SB-5101
		out += fmt.Sprintf("    suspect: %s%s", sp.Suspect, nl)
	}
	if sp.Onboarding != nil {
		out += fmt.Sprintf("    onboarding: %t%s", *sp.Onboarding, nl)
	}
	if len(sp.Rhythms) > 0 {
		out += fmt.Sprintf("    rhythms: %s%s", formatRhythms(sp.Rhythms), nl)
	}
	return out
}

// rhythms reads a space's rhythms, NAME: { subject, plan, steps: [ { day, kind, ask,
// approver, agent } ] }.
func (c *checker) rhythms(v *node, f string) map[string]Rhythm {
	return namedMap(c, v, f, `{ welcome: { subject: person, steps: [ { day: 0, kind: draft, ask: "Where to start", approver: subject } ] } }`,
		"rhythm", MaxRhythms, rhythmNameRe, c.rhythm)
}

func (c *checker) rhythm(v *node, f string) (Rhythm, bool) {
	if !c.mapOf(v, f, `{ subject: person, steps: [ { day: 0, kind: draft, ask: "…", approver: subject } ] }`) {
		return Rhythm{}, false
	}
	r := Rhythm{}
	ok := true
	for j, k := range v.keys {
		fk := f + "." + k.value
		switch k.value {
		case "subject":
			r.Subject, _ = c.scalar(v.vals[j], fk)
		case "plan":
			r.Plan, _ = c.scalar(v.vals[j], fk)
		case "steps":
			n := v.vals[j]
			if n.kind != seqNode {
				c.add(n, fk, "must be a list of steps, like [ { day: 0, kind: draft, ask: \"…\", approver: subject } ]")
				ok = false
				continue
			}
			for i, it := range n.items {
				st, good := c.rhythmStep(it, fmt.Sprintf("%s[%d]", fk, i))
				ok = ok && good
				r.Steps = append(r.Steps, st)
			}
		default:
			c.add(k, fk, "unknown field in a rhythm; use subject, plan or steps")
			ok = false
		}
	}
	if !ok {
		return Rhythm{}, false
	}
	if p := RhythmProblem(r); p != "" {
		c.add(v, f, "%s", p)
		return Rhythm{}, false
	}
	return r, true
}

func (c *checker) rhythmStep(v *node, f string) (RhythmStep, bool) {
	if !c.mapOf(v, f, `{ day: 0, kind: draft, ask: "…", approver: subject }`) {
		return RhythmStep{}, false
	}
	st := RhythmStep{}
	for j, k := range v.keys {
		fk := f + "." + k.value
		switch k.value {
		case "day":
			st.Day = c.intIn(v.vals[j], fk, 0, MaxRhythmDays)
		case "kind":
			st.Kind, _ = c.scalar(v.vals[j], fk)
		case "ask":
			st.Ask, _ = c.scalar(v.vals[j], fk)
		case "approver":
			st.Approver, _ = c.scalar(v.vals[j], fk)
		case "agent":
			st.Agent, _ = c.scalar(v.vals[j], fk)
		default:
			c.add(k, fk, "unknown field in a rhythm step; use day, kind, ask, approver or agent")
			return RhythmStep{}, false
		}
	}
	return st, true
}

// Rhythms are the rhythms space name runs: the built-in ones, then its department's, then
// its own, a later one of a name winning; onboarding is left out when the space turns it
// off (SB-4802).
func (t Tenant) Rhythms(name string) map[string]Rhythm {
	s := t.Effective(name)
	out := map[string]Rhythm{}
	for k, v := range BuiltinRhythms {
		out[k] = v
	}
	for k, v := range s.Rhythms {
		out[k] = v
	}
	if s.Onboarding != nil && !*s.Onboarding {
		delete(out, "onboarding")
	}
	return out
}

func formatRhythms(rs map[string]Rhythm) string {
	names := make([]string, 0, len(rs))
	for k := range rs {
		names = append(names, k)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, k := range names {
		r := rs[k]
		steps := make([]string, 0, len(r.Steps))
		for _, st := range r.Steps {
			s := fmt.Sprintf("{ day: %d, kind: %s, ask: %q, approver: %s", st.Day, st.Kind, st.Ask, st.Approver)
			if st.Agent != "" {
				s += ", agent: " + st.Agent
			}
			steps = append(steps, s+" }")
		}
		head := k + ": { subject: " + r.Subject
		if r.Plan != "" {
			head += ", plan: " + r.Plan
		}
		parts = append(parts, head+", steps: [ "+strings.Join(steps, ", ")+" ] }")
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}
