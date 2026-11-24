package ask_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

// SB-4005, SB-4006: an agent's answer waits in the space's queue as a draft, its text
// seen only by the steward it ran as; a draft with nothing cited is never put.
func TestAgentDraftWaitsForItsSteward(t *testing.T) {
	ctx := context.Background()
	a := freshAgent(t, `{"claims":[{"text":"Refund limit is 500.","cites":["C1"]},{"text":"Uncited guess.","cites":[]}]}`)
	ix := a.Index.(index)
	for i := range ix {
		ix[i].Title = strings.ToUpper(ix[i].SourceID)
		ix[i].Spaces = []string{"spc_ops"}
	}
	x := stewardsOf(a, time.Now())
	if _, err := x.AgentDraft(ctx, scope(t, "alice"), ask.DraftBrief, "Weekly brief: ask", ask.Request{Question: ask.BriefQuestion}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("company-wide: %v", err)
	}
	if _, err := x.AgentDraft(ctx, inOps(t, "alice"), "memo", "Memo", ask.Request{Question: ask.BriefQuestion}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a kind that isn't: %v", err)
	}
	sq, err := x.AgentDraft(ctx, inOps(t, "alice"), ask.DraftBrief, "Weekly brief: ask", ask.Request{Question: ask.BriefQuestion, Agent: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if sq.Question != "Weekly brief: ask" || sq.Space != "spc_ops" || sq.Thread == nil || sq.Thread.Kind != "brief" || sq.Trace == "" {
		t.Fatalf("draft %+v", sq)
	}
	if strings.Contains(sq.Thread.Text, "Uncited") || !strings.HasPrefix(sq.Thread.Text, "- Refund limit is 500. [") {
		t.Fatalf("text %q", sq.Thread.Text)
	}
	mine, err := x.Queue(ctx, scope(t, "alice"), "ops")
	if err != nil || len(mine) != 1 || mine[0].Thread == nil || mine[0].Thread.Text == "" || mine[0].Thread.Hidden {
		t.Fatalf("alice's queue %+v %v", mine, err)
	}
	theirs, err := x.Queue(ctx, scope(t, "sam"), "ops")
	if err != nil || len(theirs) != 1 || theirs[0].Thread.Text != "" || !theirs[0].Thread.Hidden || theirs[0].Thread.Agent != "ask" {
		t.Fatalf("sam's queue %+v %v", theirs, err)
	}
	none := freshAgent(t, `{"claims":[]}`)
	x.Agent = none
	if _, err := x.AgentDraft(ctx, inOps(t, "alice"), ask.DraftDecisions, "Decisions: standup.vtt", ask.Request{Question: ask.DecisionsQuestion}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("nothing cited: %v", err)
	}
}
