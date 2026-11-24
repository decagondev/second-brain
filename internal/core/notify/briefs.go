package notify

import (
	"context"
	"fmt"
	"sort"
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
)

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

// briefs runs the briefs that are due, and tells a space's stewards each one that didn't
// run. Each is marked in Clock when it has run or been told, so it runs once a week.
func (j Job) briefs(ctx context.Context, now time.Time) ([]outgoing, error) {
	if j.RunBrief == nil {
		return nil, nil
	}
	spaces := make([]string, 0, len(j.Briefs))
	for s := range j.Briefs {
		spaces = append(spaces, s)
	}
	sort.Strings(spaces)
	var out []outgoing
	for _, sp := range spaces {
		day := BriefDay(now, j.Zone, j.Briefs[sp])
		last, found, err := j.Clock.Last(ctx, "brief-"+sp)
		if err != nil {
			return nil, fmt.Errorf("notify: last brief: %w", err)
		}
		if found && !last.Before(day) {
			continue
		}
		why, err := j.RunBrief(ctx, sp)
		if err != nil {
			return nil, fmt.Errorf("notify: brief of %s: %w", sp, err)
		}
		if err := j.Clock.Mark(ctx, "brief-"+sp, day); err != nil {
			return nil, err
		}
		if why == "" || j.Send == nil {
			continue
		}
		groups, _, ok := j.Facts(sp)
		if !ok {
			continue
		}
		people, err := j.members(ctx, groups)
		if err != nil {
			return nil, err
		}
		for _, p := range people {
			out = append(out, outgoing{p, Message{Kind: KindQuestions, Text: briefText(why), Link: j.stewardLink(sp)}})
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
	case BriefKilled:
		return "This week's brief didn't run: its agent is switched off"
	default:
		return "This week's brief didn't run: the space has no active steward to run it as"
	}
}
