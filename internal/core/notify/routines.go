package notify

import (
	"context"
	"fmt"

	"example.com/second-brain/internal/core/ask"
)

// The routines' message (sb-ask(3), My routines; SB-4302): a routine's answer is ready, and
// how many of its sources are new since the last run. A count only, never the question or
// the answer, which the person opens on the Legwork tab, checked again as them.
const KindRoutine = "routine"

func init() { Kinds[KindRoutine] = true }

// RoutineText is the message for a routine's answer with changed new sources.
func RoutineText(changed int) string {
	switch changed {
	case 0:
		return "Your routine answer is ready: nothing new since last time"
	case 1:
		return "Your routine answer is ready: 1 source is new since last time"
	}
	return fmt.Sprintf("Your routine answer is ready: %d sources are new since last time", changed)
}

// routines asks the routines due, as their owners, and tells each owner of an answer kept.
func (j Job) routines(ctx context.Context) ([]outgoing, error) {
	if j.Routines == nil {
		return nil, nil
	}
	done, err := j.Routines(ctx)
	if err != nil {
		return nil, fmt.Errorf("notify: routines: %w", err)
	}
	var out []outgoing
	for _, d := range done {
		if d.State != ask.StateDone {
			continue // paused or failed: the inbox lists it, nothing is sent
		}
		p, ok, err := j.person(ctx, d.User)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, outgoing{p, Message{Kind: KindRoutine, Text: RoutineText(d.Changed), Link: j.tab("legwork")}})
		}
	}
	return out, nil
}
