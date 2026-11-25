package notify

import (
	"context"
	"fmt"
	"time"
)

// KindAgent tells an agent's owner about it (sb-agents(7), Value, Pause here, Spend caps;
// SB-4701 to SB-4703): a doubled discard rate, a pause and its reason, and its monthly cap.
// Never a draft or a question.
const KindAgent = "agent"

func init() { Kinds[KindAgent] = true }

// AgentAlert is one thing to tell an agent's owner once: Key says once (per week, month or
// event), To are the owner's user ids now.
type AgentAlert struct {
	Key, Text string
	To        []string
	// Kind is the notification kind, KindAgent when empty; KindCheck and KindAdminWeek
	// are told the same way (SB-5005, SB-5006).
	Kind string
}

// The kinds of the checks that come back and the admin's week (SB-5005, SB-5006).
const (
	KindCheck     = "check"
	KindAdminWeek = "adminweek"
)

func init() { Kinds[KindCheck], Kinds[KindAdminWeek] = true, true }

// agentAlerts tells each alert not yet sent, marking it once sent.
func (j Job) agentAlerts(ctx context.Context, now time.Time) ([]outgoing, []func() error, error) {
	if j.AgentAlerts == nil {
		return nil, nil, nil
	}
	as, err := j.AgentAlerts(ctx, now)
	if err != nil {
		return nil, nil, fmt.Errorf("notify: agent alerts: %w", err)
	}
	var out []outgoing
	var marks []func() error
	for _, a := range as {
		k := "agent-" + a.Key
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
		for _, id := range a.To {
			p, ok, err := j.person(ctx, id)
			if err != nil {
				return nil, nil, err
			}
			if ok {
				kind, tab := KindAgent, "legwork"
				switch a.Kind {
				case KindCheck:
					kind = KindCheck
				case KindAdminWeek:
					kind, tab = KindAdminWeek, "admin"
				}
				out = append(out, outgoing{p, Message{Kind: kind, Text: a.Text, Link: j.tab(tab)}})
			}
		}
	}
	return out, marks, nil
}
