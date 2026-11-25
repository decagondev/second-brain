package reorg

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/manifest"
)

// Reorg dry run (sb-space(1), tree; SB-4806): what moving a team under another department,
// or merging one space into another, would change, before anything does. It reads the
// manifest, makes the manifest the change would leave, checks it like sb tenant check, and
// compares every affected space's settings, escalation path and agents before and after. A
// change that would let anyone see more is refused with the path (LAW-24). Nothing is
// applied: the new manifest goes through the usual review.

// Tree change kinds.
const (
	OpMove  = "move"
	OpMerge = "merge"
)

// ErrWiden is a change that would widen access (LAW-24), and ErrTree one the manifest
// doesn't allow.
var (
	ErrWiden = errors.New("reorg: the change would widen access")
	ErrTree  = errors.New("reorg: not a change of this tree")
)

// InFlight is what is under way in one space a reorg touches: counts only.
type InFlight struct {
	Rhythms   int      `json:"rhythms"`   // running or proposed
	Open      int      `json:"open"`      // questions, requests and drafts waiting in its queue
	Waiting   int      `json:"waiting"`   // of those, waiting on another space
	Projects  int      `json:"projects"`  // open
	Routines  int      `json:"routines"`  // scheduled routines asked in it
	Triggered []string `json:"triggered"` // its agents with a schedule or trigger
}

// SettingChange is one setting of a space that changes: what it inherits or loses.
type SettingChange struct {
	Setting string   `json:"setting"` // glossary, skills, rhythms, start_here, agents, label, retention_days
	Gains   []string `json:"gains,omitempty"`
	Loses   []string `json:"loses,omitempty"`
}

// SpaceEffect is what a change does to one space.
type SpaceEffect struct {
	Space         string          `json:"space"`
	ParentBefore  string          `json:"parent_before,omitempty"`
	ParentAfter   string          `json:"parent_after,omitempty"`
	EscalateNow   []string        `json:"escalate_now"`   // the queues a question climbs, now
	EscalateAfter []string        `json:"escalate_after"` // and after
	RollsUpNow    string          `json:"rolls_up_now,omitempty"`
	RollsUpAfter  string          `json:"rolls_up_after,omitempty"`
	Settings      []SettingChange `json:"settings"`
	Gone          bool            `json:"gone,omitempty"` // merged into another space
}

// TreePlan is the dry run of one change.
type TreePlan struct {
	Op     string          `json:"op"`
	Space  string          `json:"space"`
	To     string          `json:"to"` // the new parent, or the space merged into
	Spaces []SpaceEffect   `json:"spaces"`
	After  manifest.Tenant `json:"-"`
}

// Move is the dry run of making space a team of parent ("" for a department of its own).
func Move(t manifest.Tenant, space, parent string) (TreePlan, error) {
	if _, ok := t.Spaces[space]; !ok {
		return TreePlan{}, fmt.Errorf("%w: no space %q", ErrTree, space)
	}
	if _, ok := t.Spaces[parent]; parent != "" && !ok {
		return TreePlan{}, fmt.Errorf("%w: no space %q", ErrTree, parent)
	}
	after := copyTenant(t)
	sp := after.Spaces[space]
	sp.Parent = parent
	after.Spaces[space] = sp
	return plan(t, after, OpMove, space, parent, nil)
}

// Merge is the dry run of merging space from into space into: from's scope joins into's,
// from's teams become into's, and from is gone. Members of the two must be the same groups,
// or one side would see the other's documents (LAW-24).
func Merge(t manifest.Tenant, from, into string) (TreePlan, error) {
	a, ok := t.Spaces[from]
	b, ok2 := t.Spaces[into]
	if !ok || !ok2 || from == into {
		return TreePlan{}, fmt.Errorf("%w: merge two different spaces of the manifest", ErrTree)
	}
	if !sameSet(a.Members, b.Members) {
		var gains []string
		if g := minus(a.Members, b.Members); len(g) > 0 {
			gains = append(gains, fmt.Sprintf("spaces.%s.members would gain %s", into, strings.Join(g, ", ")))
		}
		if g := minus(b.Members, a.Members); len(g) > 0 {
			gains = append(gains, fmt.Sprintf("%s's documents would reach %s", from, strings.Join(g, ", ")))
		}
		return TreePlan{}, fmt.Errorf("%w: %s", ErrWiden, strings.Join(gains, "; "))
	}
	if labelRank(a.Label) > labelRank(b.Label) {
		return TreePlan{}, fmt.Errorf("%w: spaces.%s.label %s is looser than spaces.%s.label %s", ErrWiden, into, b.Label, from, a.Label)
	}
	after := copyTenant(t)
	m := after.Spaces[into]
	m.Scope = map[string][]string{}
	for k, v := range b.Scope {
		m.Scope[k] = append([]string{}, v...)
	}
	for k, v := range a.Scope {
		for _, p := range v {
			if !slices.Contains(m.Scope[k], p) {
				m.Scope[k] = append(m.Scope[k], p)
			}
		}
	}
	after.Spaces[into] = m
	delete(after.Spaces, from)
	var moved []string
	for name, s := range after.Spaces {
		if s.Parent == from {
			s.Parent = into
			after.Spaces[name] = s
			moved = append(moved, name)
		}
	}
	return plan(t, after, OpMerge, from, into, moved)
}

func labelRank(l string) int { return slices.Index(manifest.Labels, l) }

func copyTenant(t manifest.Tenant) manifest.Tenant {
	out := t
	out.Spaces = map[string]manifest.Space{}
	for k, v := range t.Spaces {
		out.Spaces[k] = v
	}
	return out
}

func sameSet(a, b []string) bool { return len(minus(a, b)) == 0 && len(minus(b, a)) == 0 }

// plan checks the manifest after a change and compares each affected space: the space
// itself, its teams, and the spaces whose parent changed. A space whose label or retention
// would loosen is refused (LAW-24).
func plan(before, after manifest.Tenant, op, space, to string, moved []string) (TreePlan, error) {
	checked, ds := manifest.Check(manifest.Format(after))
	if len(ds) > 0 {
		return TreePlan{}, fmt.Errorf("%w: %v", ErrTree, ds[0])
	}
	affected := map[string]bool{space: true, to: to != "", before.Spaces[space].Parent: true}
	for _, m := range moved {
		affected[m] = true
	}
	for _, c := range before.Children(space) {
		affected[c] = true
	}
	p := TreePlan{Op: op, Space: space, To: to, After: checked, Spaces: []SpaceEffect{}}
	var names []string
	for n, ok := range affected {
		if ok && n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		b := before.Effective(n)
		e := SpaceEffect{Space: n, ParentBefore: before.Spaces[n].Parent, EscalateNow: path(before, n), RollsUpNow: before.Spaces[n].Parent}
		if _, ok := checked.Spaces[n]; !ok {
			e.Gone, e.EscalateAfter = true, []string{}
			p.Spaces = append(p.Spaces, e)
			continue
		}
		a := checked.Effective(n)
		e.ParentAfter, e.EscalateAfter, e.RollsUpAfter = checked.Spaces[n].Parent, path(checked, n), checked.Spaces[n].Parent
		if labelRank(a.Label) < labelRank(b.Label) {
			return TreePlan{}, fmt.Errorf("%w: spaces.%s.label would loosen from %s to %s", ErrWiden, n, b.Label, a.Label)
		}
		if b.RetentionDays > 0 && a.RetentionDays > b.RetentionDays {
			return TreePlan{}, fmt.Errorf("%w: spaces.%s.retention_days would grow from %d to %d", ErrWiden, n, b.RetentionDays, a.RetentionDays)
		}
		e.Settings = settings(before, checked, n)
		p.Spaces = append(p.Spaces, e)
	}
	return p, nil
}

// path is the queues a space's question climbs: the space, its department, company.
func path(t manifest.Tenant, n string) []string {
	out := []string{n}
	if p := t.Spaces[n].Parent; p != "" {
		out = append(out, p)
	}
	return append(out, "company")
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// settings are the inherited settings that change for space n.
func settings(before, after manifest.Tenant, n string) []SettingChange {
	b, a := before.Effective(n), after.Effective(n)
	var out []SettingChange
	add := func(name string, was, now []string) {
		if g, l := minus(now, was), minus(was, now); len(g) > 0 || len(l) > 0 {
			out = append(out, SettingChange{Setting: name, Gains: g, Loses: l})
		}
	}
	add("glossary", keys(b.Glossary), keys(a.Glossary))
	add("skills", keys(b.Skills), keys(a.Skills))
	add("rhythms", keys(before.Rhythms(n)), keys(after.Rhythms(n)))
	add("start_here", b.StartHere, a.StartHere)
	add("agents", b.Agents, a.Agents)
	if b.Label != a.Label {
		out = append(out, SettingChange{Setting: "label", Gains: []string{a.Label}, Loses: []string{b.Label}})
	}
	if b.RetentionDays != a.RetentionDays {
		out = append(out, SettingChange{Setting: "retention_days", Gains: []string{fmt.Sprint(a.RetentionDays)}, Loses: []string{fmt.Sprint(b.RetentionDays)}})
	}
	if out == nil {
		out = []SettingChange{}
	}
	return out
}
