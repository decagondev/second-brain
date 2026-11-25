package proposals

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/manifest"
)

// Agent and skill history (sb-agents(7), Agent history; SB-5104): every version of an
// agent's manifest settings, or of one space's skill, read from the approved proposals,
// each with who proposed and approved it, and the version applied now. A restore is a new
// proposal, never a change.

// Version is one version of an agent's settings or a skill.
type Version struct {
	N        int       `json:"n"`
	At       time.Time `json:"at"`
	Proposal string    `json:"proposal,omitempty"` // "" for the version applied now, when no approved proposal made it
	By       string    `json:"by,omitempty"`
	Approved string    `json:"approved,omitempty"`
	Current  bool      `json:"current"`
	Lines    []string  `json:"lines"`
	// Manifest is the whole manifest the version comes from, kept out of the JSON: a restore
	// reads it.
	Manifest manifest.Tenant `json:"-"`
}

// AgentLines are agent name's settings in t, one per line: whether it is enabled, its cap,
// job and targets, and the spaces that list it. Nil when t doesn't know it at all.
func AgentLines(t manifest.Tenant, name string) []string {
	var out []string
	for _, a := range t.Agents {
		if a == name {
			out = append(out, "enabled: true")
		}
	}
	if usd, ok := t.AgentCaps[name]; ok {
		out = append(out, fmt.Sprintf("monthly_usd: %g", usd))
	}
	if g, ok := t.AgentGoals[name]; ok {
		b, _ := json.Marshal(g)
		out = append(out, "goal: "+string(b))
	}
	var spaces []string
	for n, s := range t.Spaces {
		for _, a := range s.Agents {
			if a == name {
				spaces = append(spaces, n)
			}
		}
	}
	sort.Strings(spaces)
	if len(spaces) > 0 {
		out = append(out, "spaces: "+strings.Join(spaces, ", "))
	}
	return out
}

// SkillLines are skill name of space in t: its instruction and each case, one per line;
// nil when the space has no such skill.
func SkillLines(t manifest.Tenant, space, name string) []string {
	sk, ok := t.Spaces[space].Skills[name]
	if !ok {
		return nil
	}
	out := []string{"instruction: " + sk.Instruction}
	var qs []string
	for q := range sk.Cases {
		qs = append(qs, q)
	}
	sort.Strings(qs)
	for _, q := range qs {
		out = append(out, "case: "+q+" -> "+sk.Cases[q])
	}
	return out
}

// History is the versions lines makes of the approved proposals, oldest first, numbered from
// 1, with the applied manifest last when it differs from the newest approved one. A version
// is kept only when its lines differ from the one before.
func History(ps []Proposal, applied manifest.Tenant, lines func(manifest.Tenant) []string) []Version {
	var ap []Proposal
	for _, p := range ps {
		if p.State == Approved {
			ap = append(ap, p)
		}
	}
	sort.SliceStable(ap, func(i, j int) bool { return ap[i].At.Before(ap[j].At) })
	var out []Version
	add := func(v Version) {
		if len(out) > 0 && strings.Join(out[len(out)-1].Lines, "\n") == strings.Join(v.Lines, "\n") {
			return
		}
		v.N = len(out) + 1
		out = append(out, v)
	}
	for _, p := range ap {
		t, ds := manifest.Check([]byte(p.Manifest))
		if len(ds) > 0 {
			continue
		}
		add(Version{At: p.At, Proposal: p.ID, By: p.Requester, Approved: p.DecidedBy, Lines: nonNil(lines(t)), Manifest: t})
	}
	now := nonNil(lines(applied))
	if len(out) == 0 || strings.Join(out[len(out)-1].Lines, "\n") != strings.Join(now, "\n") {
		add(Version{Lines: now, Manifest: applied})
	}
	out[len(out)-1].Current = true
	return out
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// Changes are the lines removed from a and added in b, as "- line" and "+ line".
func Changes(a, b Version) []string {
	in := func(xs []string, x string) bool {
		for _, y := range xs {
			if y == x {
				return true
			}
		}
		return false
	}
	out := []string{}
	for _, l := range a.Lines {
		if !in(b.Lines, l) {
			out = append(out, "- "+l)
		}
	}
	for _, l := range b.Lines {
		if !in(a.Lines, l) {
			out = append(out, "+ "+l)
		}
	}
	return out
}

// RestoreAgent is applied with agent name's settings as old has them: enabled or not, its cap,
// goal and the spaces that list it.
func RestoreAgent(applied, old manifest.Tenant, name string) manifest.Tenant {
	next := clone(applied)
	next.Agents = without(next.Agents, name)
	if contains(old.Agents, name) {
		next.Agents = append(next.Agents, name)
	}
	delete(next.AgentCaps, name)
	if usd, ok := old.AgentCaps[name]; ok {
		if next.AgentCaps == nil {
			next.AgentCaps = map[string]float64{}
		}
		next.AgentCaps[name] = usd
	}
	delete(next.AgentGoals, name)
	if g, ok := old.AgentGoals[name]; ok {
		if next.AgentGoals == nil {
			next.AgentGoals = map[string]manifest.AgentGoal{}
		}
		next.AgentGoals[name] = g
	}
	for n, s := range next.Spaces {
		s.Agents = without(s.Agents, name)
		if contains(old.Spaces[n].Agents, name) {
			s.Agents = append(s.Agents, name)
		}
		next.Spaces[n] = s
	}
	return next
}

// clone is a copy of t deep enough for RestoreAgent: its own agent lists, caps, goals and
// spaces.
func clone(t manifest.Tenant) manifest.Tenant {
	out := t
	out.Agents = append([]string{}, t.Agents...)
	if t.AgentCaps != nil {
		out.AgentCaps = map[string]float64{}
		for k, v := range t.AgentCaps {
			out.AgentCaps[k] = v
		}
	}
	if t.AgentGoals != nil {
		out.AgentGoals = map[string]manifest.AgentGoal{}
		for k, v := range t.AgentGoals {
			out.AgentGoals[k] = v
		}
	}
	out.Spaces = map[string]manifest.Space{}
	for k, s := range t.Spaces {
		s.Agents = append([]string(nil), s.Agents...)
		out.Spaces[k] = s
	}
	return out
}

func without(xs []string, x string) []string {
	var out []string
	for _, y := range xs {
		if y != x {
			out = append(out, y)
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
