package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Daily brief (sb-ask(3), Notifications; SB-3805): a person who sets digest to daily gets
// their digest and watched questions each working morning at 08:00 instead of Monday's.

// The digest settings.
const (
	DigestWeekly = "weekly"
	DigestDaily  = "daily"
)

// ErrDigest is a digest setting that is neither daily nor weekly.
var ErrDigest = errors.New("notify: EINVAL digest is daily or weekly")

// Daily reports whether the person chose the daily brief.
func (p Prefs) Daily() bool { return p.Digest == DigestDaily }

// WorkingMorning is the latest Monday-to-Friday 08:00 in zone at or before now.
func WorkingMorning(now time.Time, zone *time.Location) time.Time {
	if zone == nil {
		zone = time.UTC
	}
	t := now.In(zone)
	d := time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, zone)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday || d.After(t) {
		d = d.AddDate(0, 0, -1)
	}
	return d.UTC()
}

// dailyPerson reports whether user chose the daily brief; settings that can't be read are
// the weekly default.
func (j Job) dailyPerson(ctx context.Context, user string) bool {
	if j.Prefs == nil {
		return false
	}
	p, err := j.Prefs.NotifyPrefs(ctx, j.Tenant, user)
	return err == nil && p.Daily()
}

// dailyDue reports whether this working morning's part hasn't run yet. On the first run
// it marks this morning as done.
func (j Job) dailyDue(ctx context.Context, now time.Time) (time.Time, bool, error) {
	day := WorkingMorning(now, j.Zone)
	last, found, err := j.Clock.Last(ctx, "daily")
	if err != nil {
		return day, false, fmt.Errorf("notify: last morning: %w", err)
	}
	if !found {
		return day, false, j.Clock.Mark(ctx, "daily", day)
	}
	return day, last.Before(day), nil
}

// daily plans the daily people's digests and watched questions: what changed since the
// last working morning, as them, now. A day with nothing sends nothing.
func (j Job) daily(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	out, err := j.dailyDigests(ctx, now)
	if err != nil {
		return nil, nil, err
	}
	watch, marks, err := j.watched(ctx, func(user string) bool { return j.dailyPerson(ctx, user) })
	if err != nil {
		return nil, nil, err
	}
	return append(out, watch...), marks, nil
}

func (j Job) zone() *time.Location {
	if j.Zone == nil {
		return time.UTC
	}
	return j.Zone
}

// dailyDigests are the daily people's digests: what changed since the last working
// morning, as them, now.
func (j Job) dailyDigests(ctx context.Context, now time.Time) ([]outgoing, error) {
	var out []outgoing
	if !j.Digest || j.Askers == nil || j.Changes == nil {
		return nil, nil
	}
	days := 1
	if now.In(j.zone()).Weekday() == time.Monday {
		days = 3 // since Friday's
	}
	ids, err := j.Askers(ctx, now.Add(-DigestAskers), now)
	if err != nil {
		return nil, fmt.Errorf("notify: askers: %w", err)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] || !j.dailyPerson(ctx, id) {
			continue
		}
		seen[id] = true
		p, ok, err := j.person(ctx, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		titles, more, err := j.Changes(ctx, id, days)
		if err != nil || len(titles) == 0 {
			continue
		}
		text := strings.Replace(digestText(titles, more), " this week: ", " since the last working morning: ", 1)
		out = append(out, outgoing{p, Message{Kind: KindDigest, Text: text, Link: j.tab("documents")}})
	}
	return out, nil
}
