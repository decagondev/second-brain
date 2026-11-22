package ask_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

func (ix index) ChunksOf(_ context.Context, _ string, cites []string) ([]ports.Chunk, error) {
	var out []ports.Chunk
	for _, c := range ix {
		for _, w := range cites {
			if c.Cite == w {
				out = append(out, c)
			}
		}
	}
	return out, nil
}

type stq struct {
	m map[string]ask.StewardQuestion
}

func (s *stq) PutStewardQuestion(_ context.Context, q ask.StewardQuestion) error {
	s.m[q.ID] = q
	return nil
}
func (s *stq) GetStewardQuestion(_ context.Context, _, id string) (ask.StewardQuestion, bool, error) {
	q, ok := s.m[id]
	return q, ok, nil
}
func (s *stq) StewardQuestionsOf(_ context.Context, _, asker string) ([]ask.StewardQuestion, error) {
	var out []ask.StewardQuestion
	for _, q := range s.m {
		if q.Asker == asker {
			out = append(out, q)
		}
	}
	return out, nil
}
func (s *stq) StewardQueue(_ context.Context, _, space string) ([]ask.StewardQuestion, error) {
	var out []ask.StewardQuestion
	for _, q := range s.m {
		if q.Space == space {
			out = append(out, q)
		}
	}
	return out, nil
}
func (s *stq) UpdateStewardQuestion(_ context.Context, _, id string, f func(*ask.StewardQuestion) error) (ask.StewardQuestion, error) {
	q := s.m[id]
	if err := f(&q); err != nil {
		return q, err
	}
	s.m[id] = q
	return q, nil
}
func (s *stq) StewardQuestionsBetween(context.Context, string, time.Time, time.Time) ([]ask.StewardQuestion, error) {
	return nil, nil
}

type fakeBus struct{ calls []toolbus.Call }

func (b *fakeBus) Invoke(_ context.Context, _ tenancy.Scope, c toolbus.Call) (approvals.Action, error) {
	b.calls = append(b.calls, c)
	return approvals.Action{ID: "tc_1", State: approvals.Ran, Receipt: c.Target + "@sha256:x"}, nil
}

// SB-1704, SB-1705: only an askable answer is sent; readers only narrow; a note needs the
// steward to confirm the count of cites some readers can't open.
func TestStewardsAndNotes(t *testing.T) {
	ctx := context.Background()
	a, _ := setup(t, &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`})
	a.Emails = func(_ context.Context, u string) ([]string, error) { return []string{u + "@acme.example"}, nil }
	src := a.Auth.Connectors["docs"].(source)
	src["refunds|alice@acme.example"] = ports.VerdictAllow
	src["refunds|sam@acme.example"] = ports.VerdictAllow
	src["refunds|sam"] = ports.VerdictAllow
	bus := &fakeBus{}
	n := 0
	x := ask.Stewards{Agent: a, Store: &stq{m: map[string]ask.StewardQuestion{}}, Audit: &rows{}, Bus: bus,
		Policy: policy.Kernel{Engine: allowAll{}, Recorder: rec{}},
		Facts:  func(string) ([]string, string, bool) { return []string{"group:answerers"}, "", true },
		Readers: func(context.Context, string) ([]string, error) {
			return []string{"alice@acme.example", "bob@acme.example", "sam@acme.example"}, nil
		},
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}
	alice := scope(t, "alice")
	resp, err := a.Ask(ctx, alice, ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := x.Send(ctx, alice, "refund limit?", resp.Trace); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a question from an answer that answered was sent: %v", err)
	}
	notice := ask.NoticeNoAccess
	_ = a.Store.PutTrace(ctx, ask.Trace{ID: "tr_q", Tenant: tenant, User: "alice", Space: "company", Response: ask.Response{Notice: &notice}, At: time.Now()})
	if _, err := x.Send(ctx, scope(t, "bob"), "refund limit?", "tr_q"); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("bob sent from alice's answer: %v", err)
	}
	sq, err := x.Send(ctx, alice, "refund limit?", "tr_q")
	if err != nil {
		t.Fatal(err)
	}
	sam := scope(t, "sam")
	if _, err := x.Note(ctx, sam, "company", sq.ID, ask.NoteRequest{Title: "Refunds"}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a note before a reply: %v", err)
	}
	if _, err := x.Reply(ctx, sam, "company", sq.ID, "500, see the policy.", []string{"docs:salaries@2"}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a cite the steward can't open was accepted: %v", err)
	}
	if _, err := x.Reply(ctx, sam, "company", sq.ID, "500, see the policy.", []string{"docs:refunds@2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := x.Note(ctx, sam, "company", sq.ID, ask.NoteRequest{Title: "Refunds", Readers: []string{"eve@acme.example"}}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a note's readers were widened: %v", err)
	}
	p, err := x.Note(ctx, sam, "company", sq.ID, ask.NoteRequest{Title: "Refunds"})
	if err != nil || p.Readers != 3 || p.Narrower != 1 || len(bus.calls) != 0 {
		t.Fatalf("preview %+v %v (bob can't open the refunds policy)", p, err)
	}
	zero := 0
	if _, err := x.Note(ctx, sam, "company", sq.ID, ask.NoteRequest{Title: "Refunds", Confirm: &zero}); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a confirmation of the wrong count published: %v", err)
	}
	one := 1
	res, err := x.Note(ctx, sam, "company", sq.ID, ask.NoteRequest{Title: "Refunds", Confirm: &one})
	if err != nil || res.State != "Ran" || len(bus.calls) != 1 || bus.calls[0].Tool != ask.NoteTool || bus.calls[0].Target != "notes/company/"+sq.ID+".md" {
		t.Fatalf("%+v %v %+v", res, err, bus.calls)
	}
	mine, _ := x.Mine(ctx, alice)
	if len(mine) != 1 || mine[0].Reply == nil || !mine[0].Reply.Note || len(mine[0].Reply.Cites) != 1 {
		t.Fatalf("alice's questions %+v", mine)
	}
	bobMine, _ := x.Mine(ctx, scope(t, "bob"))
	if len(bobMine) != 0 {
		t.Fatal("bob sees alice's questions")
	}
}
