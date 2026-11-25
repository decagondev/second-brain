package notify

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Scheduled briefs (sb-agents(7), Scheduled and triggered agents; SB-4005): each space's
// brief runs once a week, on the first run after 08:00 on its day in Zone. RunBrief runs it
// as the space's first active steward and puts its draft in the steward queue; when it
// can't, the space's stewards are told why.

// The reasons a brief doesn't run.
const (
	BriefNoSteward = "no_steward"
	BriefBudget    = "budget"
	BriefNothing   = "nothing" // the agent found nothing it could cite
	BriefKilled    = "killed"
	// BriefSpaceBudget is a space whose branch has spent its own budget_usd_month (SB-4105).
	BriefSpaceBudget = "space_budget"
)

// SpaceBudget is one space's budget this month: its key, the month (2027-03) and the
// share of its budget its branch has spent.
type SpaceBudget struct {
	Space, Month string
	Share        float64
}

// WarnShare is the share of a space's budget at which its stewards are told (SB-4105).
const WarnShare = 0.8

// budgets tells each space's stewards once a month when its branch passes WarnShare of its
// budget, and once when it has spent it. Counts only: never what it was spent on.
func (j Job) budgets(ctx context.Context, now time.Time) ([]outgoing, error) {
	if j.SpaceBudgets == nil || j.Send == nil {
		return nil, nil
	}
	bs, err := j.SpaceBudgets(ctx)
	if err != nil {
		return nil, fmt.Errorf("notify: budgets: %w", err)
	}
	var out []outgoing
	for _, b := range bs {
		var level, text string
		name := strings.TrimPrefix(b.Space, "spc_")
		switch {
		case b.Share >= 1:
			level, text = "100", fmt.Sprintf("The %s space has spent its monthly budget: its scheduled briefs and updates pause until next month, and asks carry on", name)
		case b.Share >= WarnShare:
			level, text = "80", fmt.Sprintf("The %s space has used %d%% of its monthly budget", name, int(b.Share*100+1e-9))
		default:
			continue
		}
		key := "budget-" + b.Space + "-" + b.Month + "-" + level
		if _, found, err := j.Clock.Last(ctx, key); err != nil {
			return nil, fmt.Errorf("notify: last budget: %w", err)
		} else if found {
			continue
		}
		groups, _, ok := j.Facts(b.Space)
		if !ok {
			continue
		}
		people, err := j.members(ctx, groups)
		if err != nil {
			return nil, err
		}
		if err := j.Clock.Mark(ctx, key, now); err != nil {
			return nil, err
		}
		for _, p := range people {
			out = append(out, outgoing{p, Message{Kind: KindQuestions, Text: text, Link: j.stewardLink(b.Space)}})
		}
	}
	return out, nil
}

// BriefDay is the latest day 08:00 in zone at or before now.
func BriefDay(now time.Time, zone *time.Location, day time.Weekday) time.Time {
	if zone == nil {
		zone = time.UTC
	}
	t := now.In(zone)
	d := time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, zone)
	for d.Weekday() != day || d.After(t) {
		d = d.AddDate(0, 0, -1)
	}
	return d.UTC()
}

// scheduled is one weekly run: a space's brief, or its status update (SB-4102).
type scheduled struct {
	space, clock string
	agent        string // the agent that drafts it, for the pause link (SB-5107); "" for none
	day          time.Weekday
	run          func(ctx context.Context, space string) (string, error)
}

// briefs runs the briefs and updates that are due, and tells a space's stewards each brief
// that didn't run. Each is marked in Clock when it has run or been told, so it runs once a
// week.
func (j Job) briefs(ctx context.Context, now time.Time) ([]outgoing, error) {
	var runs []scheduled
	if j.RunBrief != nil {
		for s, d := range j.Briefs {
			runs = append(runs, scheduled{space: s, clock: "brief-" + s, day: d, run: j.RunBrief})
		}
	}
	if j.RunUpdate != nil {
		for _, s := range j.Updates {
			runs = append(runs, scheduled{space: s, clock: "update-" + s, agent: "update", day: time.Monday, run: j.RunUpdate})
		}
	}
	if j.RunLeader != nil {
		for _, s := range j.Leaders {
			runs = append(runs, scheduled{space: s, clock: "leader-" + s, agent: "leader-week", day: time.Monday, run: j.RunLeader})
		}
	}
	sort.Slice(runs, func(a, b int) bool { return runs[a].clock < runs[b].clock })
	var out []outgoing
	for _, r := range runs {
		day := BriefDay(now, j.Zone, r.day)
		last, found, err := j.Clock.Last(ctx, r.clock)
		if err != nil {
			return nil, fmt.Errorf("notify: last brief: %w", err)
		}
		if found && !last.Before(day) {
			continue
		}
		why, err := r.run(ctx, r.space)
		if err != nil {
			return nil, fmt.Errorf("notify: %s: %w", r.clock, err)
		}
		if err := j.Clock.Mark(ctx, r.clock, day); err != nil {
			return nil, err
		}
		if why == "" || j.Send == nil {
			continue
		}
		groups, _, ok := j.Facts(r.space)
		if !ok {
			continue
		}
		people, err := j.members(ctx, groups)
		if err != nil {
			return nil, err
		}
		text := briefText(why)
		if strings.HasPrefix(r.clock, "update-") {
			text = strings.Replace(text, "This week's brief", "This week's update", 1)
		}
		if strings.HasPrefix(r.clock, "leader-") {
			text = strings.Replace(text, "This week's brief", "The leader's week", 1)
		}
		for _, p := range people {
			out = append(out, outgoing{p, Message{Kind: KindQuestions, Text: text, Link: j.stewardLink(r.space), Agent: r.agent, Space: strings.TrimPrefix(r.space, "spc_")}})
		}
	}
	return out, nil
}

func briefText(why string) string {
	switch why {
	case BriefBudget:
		return "This week's brief didn't run: the monthly model budget is spent"
	case BriefNothing:
		return "This week's brief found nothing it could cite, so there's no draft"
	case BriefSpaceBudget:
		return "This week's brief didn't run: the space's monthly budget is spent, so briefs pause until next month"
	case BriefKilled:
		return "This week's brief didn't run: its agent is switched off"
	default:
		return "This week's brief didn't run: the space has no active steward to run it as"
	}
}
