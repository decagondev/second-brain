package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"example.com/second-brain/internal/core/ask"
)

// The legwork agents' messages (sb-ask(3), Notifications; SB-4201 to SB-4203): a report is
// ready, a meeting's brief is ready, and action items a steward confirmed. Counts, a
// space's name and a time only, never a question, an agenda or an item's text.
const (
	KindResearch = "research"
	KindPrep     = "prep"
	KindAction   = "action"
)

func init() {
	for _, k := range []string{KindResearch, KindPrep, KindAction} {
		Kinds[k] = true
	}
}

// PrepWindow is when, before an event starts, its brief is made: 25 to 35 minutes.
const (
	PrepFrom = 25 * time.Minute
	PrepTo   = 35 * time.Minute
)

// legwork runs the queued research reports, makes the briefs due before meetings and tells
// owners of confirmed action items, each once.
func (j Job) legwork(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	var out []outgoing
	var marks []func() error
	if j.Research != nil {
		done, err := j.Research(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("notify: research: %w", err)
		}
		for _, d := range done {
			if d.State != ask.StateDone {
				continue
			}
			if p, ok, err := j.person(ctx, d.User); err != nil {
				return nil, nil, err
			} else if ok {
				out = append(out, outgoing{p, Message{Kind: KindResearch, Text: "Your research report is ready", Link: j.tab("legwork")}})
			}
		}
	}
	prep, prepMarks, err := j.preps(ctx, now)
	if err != nil {
		return nil, nil, err
	}
	out, marks = append(out, prep...), append(marks, prepMarks...)
	if j.Actions == nil {
		return out, marks, nil
	}
	back := j.Lookback
	if back <= 0 || back > 30*24*time.Hour {
		back = 30 * 24 * time.Hour
	}
	as, err := j.Actions(ctx, now.Add(-back))
	if err != nil {
		return nil, nil, fmt.Errorf("notify: actions: %w", err)
	}
	type key struct{ user, space string }
	counts := map[key]int{}
	for _, a := range as {
		if a.User == "" || a.DoneAt != nil {
			continue
		}
		k := "action-" + a.ID
		if _, done, err := j.Clock.Last(ctx, k); err != nil || done {
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		counts[key{a.User, a.Space}]++
		marks = append(marks, func() error { return j.Clock.Mark(ctx, k, now) })
	}
	keys := make([]key, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		return keys[a].user < keys[b].user || keys[a].user == keys[b].user && keys[a].space < keys[b].space
	})
	for _, k := range keys {
		p, ok, err := j.person(ctx, k.user)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			continue
		}
		n, word := counts[k], "action items"
		if n == 1 {
			word = "action item"
		}
		out = append(out, outgoing{p, Message{Kind: KindAction, Text: fmt.Sprintf("You have %d %s from a meeting in %s", n, word, k.space), Link: j.tab("legwork")}})
	}
	return out, marks, nil
}

// preps makes, as each person who turned prep on, the briefs of their events starting
// PrepFrom to PrepTo from now, once per event.
func (j Job) preps(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	if j.PrepDue == nil || j.Askers == nil || j.Prefs == nil {
		return nil, nil, nil
	}
	users, err := j.Askers(ctx, now.Add(-DigestAskers), now)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: askers: %w", err)
	}
	var out []outgoing
	var marks []func() error
	for _, u := range users {
		p, err := j.Prefs.NotifyPrefs(ctx, j.Tenant, u)
		if err != nil || !p.Prep || !p.Wants(KindPrep) || p.Paused(now) {
			continue
		}
		person, ok, err := j.person(ctx, u)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			continue
		}
		made, err := j.PrepDue(ctx, u, now.Add(PrepFrom), now.Add(PrepTo), func(event string) bool {
			_, done, err := j.Clock.Last(ctx, prepKey(u, event))
			return err != nil || done
		})
		if err != nil {
			return nil, nil, fmt.Errorf("notify: prep: %w", err)
		}
		for _, m := range made {
			k := prepKey(u, m.Event)
			marks = append(marks, func() error { return j.Clock.Mark(ctx, k, now) })
			out = append(out, outgoing{person, Message{Kind: KindPrep, Text: "Your brief for the meeting at " + m.Start.In(j.zone()).Format("15:04") + " is ready", Link: j.tab("legwork")}})
		}
	}
	return out, marks, nil
}

func prepKey(user, event string) string { return "prep-" + hashKey(user+"|"+event) }

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}
