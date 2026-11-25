package ask_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

// SB-4901: an ambiguous term gets one question and no answer; the asker's choice answers
// with it, a second question is refused, a caller that can't ask states its assumption,
// and a question that isn't ambiguous is answered as asked.
func TestAskBeforeGuessing(t *testing.T) {
	ctx := context.Background()
	m := &model{reply: `{"claims":[{"text":"x","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	a.Departmental = func(context.Context, tenancy.Scope) []ask.DeptSpace {
		return []ask.DeptSpace{
			{Name: "finance", Agent: "close-checklist", Glossary: map[string]string{"AP": "accounts payable"}},
			{Name: "eng", Agent: "incident-summary", Glossary: map[string]string{"AP": "access point"}},
		}
	}
	alice := scope(t, "alice")
	resp, err := a.Ask(ctx, alice, ask.Request{Question: "Is the AP refund done?", Clarify: true})
	if err != nil {
		t.Fatal(err)
	}
	c := resp.Clarify
	if c == nil || c.Kind != ask.ClarifyTerm || len(c.Choices) != 3 || c.Choices[2] != ask.JustAnswer || len(m.got) != 0 || resp.Trace != "" {
		t.Fatalf("one question, nothing composed: %+v, %d model calls", c, len(m.got))
	}
	if !strings.Contains(c.Choices[0], "AP as eng uses it: access point") || !strings.Contains(c.Choices[1], "accounts payable") {
		t.Fatalf("choices %q", c.Choices)
	}
	resp, err = a.Ask(ctx, alice, ask.Request{Question: "Is the AP refund done?", Clarify: true, Choice: c.Choices[1]})
	if err != nil || resp.Clarify != nil || len(m.got) != 1 || !strings.Contains(fmt.Sprint(m.got[0].Messages), ask.ChoiceLead+c.Choices[1]) {
		t.Fatalf("the choice answers with it: %v %+v", err, resp.Clarify)
	}
	if _, err := a.Ask(ctx, alice, ask.Request{Question: "Is the AP refund done?", Clarify: true, Choice: "something else"}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a choice it didn't offer: %v", err)
	}
	if _, err := a.Ask(ctx, alice, ask.Request{Question: "Where is the refund policy?", Choice: c.Choices[0]}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a choice for a question with nothing to ask: %v", err)
	}
	// A later question of the conversation is never asked about.
	if resp, err := a.Ask(ctx, alice, ask.Request{Question: "And the AP refund?", Clarify: true, Conversation: resp.Conversation}); err != nil || resp.Clarify != nil {
		t.Fatalf("a second question in a conversation: %v %+v", err, resp.Clarify)
	}
	// A caller that can't show a question takes the first choice and says so.
	resp, err = a.Ask(ctx, alice, ask.Request{Question: "Is the AP refund done?"})
	if err != nil || resp.Clarify != nil || resp.Assumed != ask.AssumedLead+c.Choices[0] || ask.Warnings(resp)[0] != resp.Assumed {
		t.Fatalf("assumed: %v %q", err, resp.Assumed)
	}
	if resp, _ := a.Ask(ctx, alice, ask.Request{Question: "How much did refunds cost?", Clarify: true}); resp.Clarify == nil || resp.Clarify.Kind != ask.ClarifyPeriod {
		t.Fatalf("no period: %+v", resp.Clarify)
	}
	if resp, _ := a.Ask(ctx, alice, ask.Request{Question: "How much did refunds cost in 2026?", Clarify: true}); resp.Clarify != nil {
		t.Fatalf("a period given: %+v", resp.Clarify)
	}
	a.Departmental = func(context.Context, tenancy.Scope) []ask.DeptSpace {
		return []ask.DeptSpace{
			{Name: "finance", Glossary: map[string]string{"close": "month-end close"}},
			{Name: "sales", Glossary: map[string]string{"pipeline": "open deals"}},
		}
	}
	if resp, _ := a.Ask(ctx, alice, ask.Request{Question: "Does the close slow the pipeline refund?", Clarify: true}); resp.Clarify == nil || resp.Clarify.Kind != ask.ClarifySpace || resp.Clarify.Choices[0] != "About finance" {
		t.Fatalf("several spaces: %+v", resp.Clarify)
	}
}
