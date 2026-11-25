package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// The follow-up agents' messages (sb-ask(3), Notifications; SB-4204 to SB-4207): an
// incident brief waits for a space's stewards, a document's owner has suggested edits,
// and a goal has gone three weeks without a check-in. A space's name only, never a
// ticket, a document or a goal's text.
const (
	KindIncident = "incident"
	KindFix      = "fix"
	KindCheckin  = "checkin"
)

func init() {
	for _, k := range []string{KindIncident, KindFix, KindCheckin} {
		Kinds[k] = true
	}
}

// SilentGoal is a goal with no check-in published for three weeks, for its owner.
type SilentGoal struct {
	Owner, Space, Goal string
}

// incidents drafts an incident brief for each incident ticket first seen in a space in
// [from, now), once per ticket and space, and tells the space's stewards it waits.
func (j Job) incidents(ctx context.Context, from, now time.Time) ([]outgoing, error) {
	if j.Incidents == nil || j.RunIncident == nil {
		return nil, nil
	}
	docs, err := j.Incidents(ctx, from, now)
	if err != nil {
		return nil, fmt.Errorf("notify: incidents: %w", err)
	}
	var out []outgoing
	for _, d := range docs {
		sum := sha256.Sum256([]byte(d.Space + "|" + d.Ref))
		key := "incident-" + hex.EncodeToString(sum[:8])
		if _, done, err := j.Clock.Last(ctx, key); err != nil || done {
			if err != nil {
				return nil, err
			}
			continue
		}
		why, err := j.RunIncident(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("notify: incident brief: %w", err)
		}
		if err := j.Clock.Mark(ctx, key, now); err != nil {
			return nil, err
		}
		if why != "" || j.Send == nil || j.Facts == nil {
			continue
		}
		groups, _, ok := j.Facts(d.Space)
		if !ok {
			continue
		}
		people, err := j.members(ctx, groups)
		if err != nil {
			return nil, err
		}
		name := strings.TrimPrefix(d.Space, "spc_")
		for _, p := range people {
			out = append(out, outgoing{p, Message{Kind: KindIncident, Text: "An incident brief is waiting in " + name, Link: j.stewardLink(d.Space)}})
		}
	}
	return out, nil
}

// checkins drafts each goal's check-in on Monday, once a week, and tells the owner of each
// goal gone three weeks without one.
func (j Job) checkins(ctx context.Context, now time.Time) ([]outgoing, error) {
	if j.Checkins == nil {
		return nil, nil
	}
	day := BriefDay(now, j.Zone, time.Monday)
	last, found, err := j.Clock.Last(ctx, "checkins")
	if err != nil {
		return nil, fmt.Errorf("notify: last check-ins: %w", err)
	}
	if found && !last.Before(day) {
		return nil, nil
	}
	silent, err := j.Checkins(ctx)
	if err != nil {
		return nil, fmt.Errorf("notify: check-ins: %w", err)
	}
	if err := j.Clock.Mark(ctx, "checkins", day); err != nil {
		return nil, err
	}
	var out []outgoing
	for _, g := range silent {
		p, ok, err := j.person(ctx, g.Owner)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, outgoing{p, Message{Kind: KindCheckin,
				Text: "One of your goals in " + strings.TrimPrefix(g.Space, "spc_") + " has had no check-in for 3 weeks", Link: j.stewardLink(g.Space)}})
		}
	}
	return out, nil
}
