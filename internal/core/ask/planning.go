package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// Quarterly planning (sb-departments(7), Quarterly planning; SB-4805): two weeks before a
// quarter ends, each space's quarterly-planning rhythm drafts next quarter's goals for its
// stewards from its goals carried over and at risk and the requests it took, every goal
// cited. A department's stewards see their teams' proposed goals rolled up, the ones they
// can't open counted, with possible overlaps flagged. A goal only starts when a steward
// publishes it; nothing here merges or publishes.

// The rhythm and agents of planning and 1:1 prep.
const (
	QuarterlyPlanning = "quarterly-planning"
	OneOnOne          = "one-on-one"
	AgentGoalPlan     = "goal-plan"
	AgentOneOnOne     = "one-on-one"
	// PlanningLead is how long before a quarter ends planning starts.
	PlanningLead = 14 * 24 * time.Hour
	// MaxProposedGoals is the most goals one planning draft proposes.
	MaxProposedGoals = 10
)

// ProposedGoal is one goal a planning draft proposes: its title, the goal it carries over
// (its cite), and why.
type ProposedGoal struct {
	Title string `json:"title"`
	Cite  string `json:"cite"`
	Why   string `json:"why"` // carry over, at risk
}

// quarterEnd is the first instant of the quarter after t's.
func quarterEnd(t time.Time) time.Time {
	q := (int(t.Month()) - 1) / 3
	return time.Date(t.Year(), time.Month(q*3+1), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 3, 0)
}

// nextQuarter is the quarter after t's, YYYY-Qn.
func nextQuarter(t time.Time) string { return quarterOf(quarterEnd(t)) }

// quarterly starts each space's planning rhythm once, two weeks before the quarter ends.
func (x Rhythms) quarterly(ctx context.Context) error {
	if x.Spaces == nil {
		return nil
	}
	now := x.now()
	end := quarterEnd(now)
	if end.Sub(now) > PlanningLead {
		return nil
	}
	runs, err := x.Store.RhythmRuns(ctx, x.Stewards.Agent.Tenant)
	if err != nil {
		return fmt.Errorf("%w: rhythms: %v", ErrUnavailable, err)
	}
	for _, space := range x.Spaces() {
		d, ok := x.Defs(space)[QuarterlyPlanning]
		if !ok {
			continue
		}
		started := false
		for _, r := range runs {
			if r.Space == space && r.Rhythm == QuarterlyPlanning && !r.At.Before(end.Add(-PlanningLead-24*time.Hour)) {
				started = true
			}
		}
		if started {
			continue
		}
		if _, err := x.start(ctx, RhythmSystem, space, QuarterlyPlanning, d, space); err != nil && !isInvalid(err) {
			return err
		}
	}
	return nil
}

func isInvalid(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrInvalid.Error())
}

// firstSteward is the user id of the space's first steward by email, "" for none: a
// system-started steward step is drafted as them, and only they read its text.
func (x Rhythms) firstSteward(ctx context.Context, space string) string {
	if x.Stewards.StewardPeople == nil || x.Stewards.Owner == nil {
		return ""
	}
	ps, err := x.Stewards.StewardPeople(ctx, space)
	if err != nil {
		return ""
	}
	var emails []string
	for _, p := range ps {
		emails = append(emails, p.Email)
	}
	sort.Strings(emails)
	for _, e := range emails {
		if id, ok, err := x.Stewards.Owner(ctx, e); err == nil && ok {
			return id
		}
	}
	return ""
}

func flatGoals(qs []GoalQuarter) []Goal {
	var out []Goal
	var walk func(gs []Goal)
	walk = func(gs []Goal) {
		for _, g := range gs {
			out = append(out, g)
			walk(g.Goals)
		}
	}
	for _, q := range qs {
		walk(q.Goals)
	}
	return out
}

// goalPlan drafts next quarter's goals as a steward: this quarter's goals not done carry
// over (at risk or off track ones flagged), each cited, and the request categories the
// space took this quarter, counted.
func (x Rhythms) goalPlan(ctx context.Context, s tenancy.Scope, r RhythmRun) (string, []ProposedGoal, error) {
	gr, err := x.Stewards.Goals(ctx, s, r.Space)
	if err != nil {
		return "", nil, err
	}
	now := x.now()
	this := quarterOf(now)
	var b strings.Builder
	fmt.Fprintf(&b, "Proposed goals for %s:\n", nextQuarter(now))
	var out []ProposedGoal
	for _, g := range flatGoals(gr.Quarters) {
		if g.Space != r.Space || g.Quarter != this || g.Status == "done" || len(out) == MaxProposedGoals {
			continue
		}
		why := "carry over"
		if g.Status == "at risk" || g.Status == "off track" {
			why = "at risk"
		}
		doc, _, _ := strings.Cut(g.Cite, "@")
		out = append(out, ProposedGoal{Title: g.Title, Cite: doc, Why: why})
		fmt.Fprintf(&b, "- %s (%s, %s now) [%s]\n", g.Title, why, orDash(g.Status), doc)
	}
	if qs, err := x.Stewards.queue(ctx, r.Space); err == nil {
		cats := map[string]int{}
		start := quarterEnd(now).AddDate(0, -3, 0)
		for _, q := range qs {
			if q.Request != nil && !q.At.Before(start) {
				cats[q.Request.Category]++
			}
		}
		var names []string
		for c := range cats {
			names = append(names, c)
		}
		sort.Strings(names)
		for _, c := range names {
			fmt.Fprintf(&b, "- Requests this quarter in %s: %d\n", c, cats[c])
		}
	}
	if len(out) == 0 {
		return "", nil, fmt.Errorf("%w: no goal to carry over", ErrInvalid)
	}
	return strings.TrimRight(b.String(), "\n"), out, nil
}

// PlanningTeam is one team's proposed goals as a department steward sees them.
type PlanningTeam struct {
	Space    string         `json:"space"`
	Proposed []ProposedGoal `json:"proposed"`
	Hidden   int            `json:"hidden"`
}

// PlanningOverlap flags two teams' proposed goals that may be the same work: a flag, never
// a merge.
type PlanningOverlap struct {
	// Teams are the two spaces; Goals their two titles; Term what they share.
	Teams []string `json:"teams"`
	Goals []string `json:"goals"`
	Term  string   `json:"term"`
}

// PlanningResponse is GET /v1/spaces/NAME/planning.
type PlanningResponse struct {
	Space    string            `json:"space"`
	Quarter  string            `json:"quarter"`
	Teams    []PlanningTeam    `json:"teams"`
	Overlaps []PlanningOverlap `json:"overlaps"`
}

// Planning is the proposed goals of a space and its teams, for its stewards: each team's
// latest planning run, a goal shown only when the steward can open its cite, the rest
// counted, and overlaps flagged where two teams' goals share a glossary term or two words.
func (x Rhythms) Planning(ctx context.Context, s tenancy.Scope, space string) (PlanningResponse, error) {
	if _, _, err := x.Stewards.steward(ctx, s, space); err != nil {
		return PlanningResponse{}, err
	}
	spaces := []string{space}
	if x.Stewards.Children != nil {
		spaces = append(spaces, x.Stewards.Children(space)...)
	}
	runs, err := x.Store.RhythmRuns(ctx, x.Stewards.Agent.Tenant)
	if err != nil {
		return PlanningResponse{}, fmt.Errorf("%w: rhythms: %v", ErrUnavailable, err)
	}
	out := PlanningResponse{Space: space, Quarter: nextQuarter(x.now()), Teams: []PlanningTeam{}, Overlaps: []PlanningOverlap{}}
	for _, sp := range spaces {
		var latest *RhythmRun
		for i := range runs {
			if runs[i].Space == sp && runs[i].Rhythm == QuarterlyPlanning && len(runs[i].Proposed) > 0 {
				latest = &runs[i]
			}
		}
		if latest == nil {
			continue
		}
		var cites []string
		for _, g := range latest.Proposed {
			cites = append(cites, g.Cite+"@0")
		}
		open, err := x.Stewards.Agent.opensCites(ctx, s, cites)
		if err != nil {
			return PlanningResponse{}, err
		}
		t := PlanningTeam{Space: sp, Proposed: []ProposedGoal{}}
		for _, g := range latest.Proposed {
			if open[g.Cite+"@0"] {
				t.Proposed = append(t.Proposed, g)
			} else {
				t.Hidden++
			}
		}
		out.Teams = append(out.Teams, t)
	}
	var terms []string
	if x.Glossary != nil {
		for k := range x.Glossary(space) {
			terms = append(terms, k)
		}
		sort.Strings(terms)
	}
	for i := 0; i < len(out.Teams); i++ {
		for j := i + 1; j < len(out.Teams); j++ {
			for _, a := range out.Teams[i].Proposed {
				for _, b := range out.Teams[j].Proposed {
					if term := shared(a.Title, b.Title, terms); term != "" {
						out.Overlaps = append(out.Overlaps, PlanningOverlap{Teams: []string{out.Teams[i].Space, out.Teams[j].Space}, Goals: []string{a.Title, b.Title}, Term: term})
					}
				}
			}
		}
	}
	return out, nil
}

// shared is a glossary term both titles use, or two words of four letters both use, "" for
// neither.
func shared(a, b string, terms []string) string {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	for _, t := range terms {
		if lt := strings.ToLower(t); strings.Contains(la, lt) && strings.Contains(lb, lt) {
			return t
		}
	}
	wa, wb := nameWords(a), nameWords(b)
	var both []string
	for w := range wa {
		if wb[w] {
			both = append(both, w)
		}
	}
	if len(both) >= 2 {
		sort.Strings(both)
		return strings.Join(both, " ")
	}
	return ""
}
