package ask_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
)

// tickets is a source that can fetch, answers ACL from a table, and counts Act calls.
type tickets struct {
	source
	docs  map[string]ports.Document
	acted *int
}

func (t tickets) Fetch(_ context.Context, r ports.SourceRef) (ports.Document, error) {
	d, ok := t.docs[r.ID]
	if !ok {
		return ports.Document{}, errors.New("404")
	}
	return d, nil
}

func (t tickets) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	*t.acted++
	return ports.Receipt{}, errors.New("drafts never act")
}

// projectPolicy forbids drafting on FIN tickets for everyone, like the example tenant
// policy does for non-finance staff.
type projectPolicy struct{}

func (projectPolicy) Evaluate(_ context.Context, r policy.Request) (policy.Result, error) {
	if r.Action == "use_agent" && r.Context["target_project"] == "FIN" {
		return policy.Result{Decision: policy.Deny, Reason: "forbidden by acme.cedar#2"}, nil
	}
	return policy.Result{Decision: policy.Allow, Reason: "test"}, nil
}

type counter map[string]float64

func (c counter) Add(name, _ string, l map[string]string, v float64) { c[name+"/"+l["outcome"]] += v }

func TestDraft(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Thanks for flagging this.","cites":[]},{"text":"Refunds up to 500 need no approval.","cites":["C2"]},{"text":"The ticket says the order was 300.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	acted := 0
	tk := tickets{source: source{"OPS-7|alice": ports.VerdictAllow, "refunds|alice": ports.VerdictAllow, "FIN-2|alice": ports.VerdictAllow, "OPS-8|alice": ports.VerdictDeny},
		docs: map[string]ports.Document{
			"OPS-7": {Ref: ports.SourceRef{Kind: "tickets", ID: "OPS-7"}, Version: 4, Title: "Refund request", Text: "Customer asks for a refund of 300."},
			"FIN-2": {Ref: ports.SourceRef{Kind: "tickets", ID: "FIN-2"}, Version: 1, Title: "Payroll", Text: "x"},
			"OPS-8": {Ref: ports.SourceRef{Kind: "tickets", ID: "OPS-8"}, Version: 1, Title: "Secret", Text: "y"},
		}, acted: &acted}
	a.Auth.Connectors["tickets"] = tk
	c := counter{}
	d := ask.Drafts{Agent: a, Killed: func(context.Context) bool { return false }, Policy: policy.Kernel{Engine: projectPolicy{}, Recorder: rec{}}, Metrics: c}
	ctx := context.Background()

	r, err := d.Draft(ctx, scope(t, "alice"), ask.DraftRequest{Context: "tickets:OPS-7", Instruction: "Reply politely with our refund rule."})
	if err != nil {
		t.Fatal(err)
	}
	cl := r.Answer.Claims
	if len(cl) != 3 || !cl[0].Uncited || cl[1].Cites[0].Cite != "docs:refunds@2" || cl[2].Cites[0].Cite != "tickets:OPS-7@4" || r.Context != "tickets:OPS-7" {
		t.Fatalf("draft %+v", r)
	}
	if !strings.Contains(m.got[0].System, "never send") || !strings.Contains(m.got[0].Messages[0].Content, "Customer asks for a refund") {
		t.Errorf("model request: %+v", m.got[0])
	}
	if acted != 0 {
		t.Fatal("the draft agent called Act")
	}

	bad := []struct {
		req  ask.DraftRequest
		want error
	}{
		{ask.DraftRequest{Context: "email:x", Instruction: "hi"}, ask.ErrInvalid},
		{ask.DraftRequest{Context: "tickets:OPS-7", Instruction: " "}, ask.ErrInvalid},
		{ask.DraftRequest{Context: "tickets:OPS-8", Instruction: "hi"}, ask.ErrNotFound}, // the item is denied
		{ask.DraftRequest{Context: "tickets:OPS-9", Instruction: "hi"}, ask.ErrNotFound}, // unknown: refused
		{ask.DraftRequest{Context: "tickets:FIN-2", Instruction: "hi"}, ask.ErrNotFound}, // policy narrows by project
	}
	for _, b := range bad {
		if _, err := d.Draft(ctx, scope(t, "alice"), b.req); !errors.Is(err, b.want) {
			t.Errorf("%+v: got %v, want %v", b.req, err, b.want)
		}
	}
	d.Killed = func(context.Context) bool { return true }
	if _, err := d.Draft(ctx, scope(t, "alice"), ask.DraftRequest{Context: "tickets:OPS-7", Instruction: "hi"}); !errors.Is(err, ask.ErrKilled) {
		t.Errorf("killed: %v", err)
	}

	if err := d.Outcome(ctx, scope(t, "alice"), r.Trace, "used"); err != nil || c["sb_drafts_total/used"] != 1 {
		t.Fatalf("outcome: %v %v", err, c)
	}
	if err := d.Outcome(ctx, scope(t, "bob"), r.Trace, "used"); !errors.Is(err, ask.ErrNotFound) {
		t.Errorf("someone else's draft: %v", err)
	}
	if err := d.Outcome(ctx, scope(t, "alice"), r.Trace, "sent"); !errors.Is(err, ask.ErrInvalid) {
		t.Errorf("bad outcome: %v", err)
	}
}
