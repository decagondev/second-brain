package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
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
// [from, now), once per ticket and space, and a review when a later version of one that had
// a brief is closed (SB-4604), once; it tells the space's stewards each waits.
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
		_, briefed, err := j.Clock.Last(ctx, key)
		if err != nil {
			return nil, err
		}
		run, kind, text := j.RunIncident, "brief", "An incident brief is waiting in "
		if briefed {
			if !d.Closed || j.RunReview == nil {
				continue
			}
			key = "review-" + hex.EncodeToString(sum[:8])
			if _, done, err := j.Clock.Last(ctx, key); err != nil || done {
				if err != nil {
					return nil, err
				}
				continue
			}
			run, kind, text = j.RunReview, "review", "An incident review is waiting in "
		}
		why, err := run(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("notify: incident %s: %w", kind, err)
		}
		if err := j.Clock.Mark(ctx, key, now); err != nil {
			return nil, err
		}
		told, err := j.tellStewards(ctx, d.Space, why, text)
		if err != nil {
			return nil, err
		}
		out = append(out, told...)
	}
	return out, nil
}

// tellStewards tells a space's stewards one incident draft waits there, unless why says it
// didn't run.
func (j Job) tellStewards(ctx context.Context, space, why, text string) ([]outgoing, error) {
	if why != "" || j.Send == nil || j.Facts == nil {
		return nil, nil
	}
	groups, _, ok := j.Facts(space)
	if !ok {
		return nil, nil
	}
	people, err := j.members(ctx, groups)
	if err != nil {
		return nil, err
	}
	var out []outgoing
	for _, p := range people {
		out = append(out, outgoing{p, Message{Kind: KindIncident, Text: text + strings.TrimPrefix(space, "spc_"), Link: j.stewardLink(space)}})
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

// waitKinds are how a wait's item is named in a message: a kind, never its text.
var waitKinds = map[string]string{ask.WaitRequest: "a request", ask.WaitAction: "an action item", ask.WaitGoal: "a goal"}

// waits tells, once per wait and date, the stewards of the spaces each overdue wait goes to
// (SB-4504): who waits on whom since when, never the item's text or reason.
func (j Job) waits(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	if j.Waits == nil {
		return nil, nil, nil
	}
	ws, err := j.Waits(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: waits: %w", err)
	}
	var out []outgoing
	var marks []func() error
	for _, w := range ws {
		k := "wait-" + w.Key
		if _, done, err := j.Clock.Last(ctx, k); err != nil || done {
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		marks = append(marks, func() error { return j.Clock.Mark(ctx, k, now) })
		if j.Send == nil {
			continue
		}
		text := fmt.Sprintf("%s has waited on %s since %s (%s)", w.From, w.On, w.Until, waitKinds[w.Kind])
		for _, to := range w.To {
			key := ask.SpaceKey(to)
			groups, _, ok := j.Facts(key)
			if !ok {
				continue
			}
			people, err := j.members(ctx, groups)
			if err != nil {
				return nil, nil, err
			}
			for _, p := range people {
				out = append(out, outgoing{p, Message{Kind: KindQuestions, Text: text, Link: j.stewardLink(key)}})
			}
		}
	}
	return out, marks, nil
}
