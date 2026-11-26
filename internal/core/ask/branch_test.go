package ask_test

import (
	"context"
	"errors"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

// SB-5206: a branch is a new conversation of the asker's own with the first N questions;
// the first is left as it was, and someone else's conversation can't be branched.
func TestBranchAConversation(t *testing.T) {
	ctx := context.Background()
	a, _ := setup(t, &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`})
	alice := scope(t, "alice")
	r1, err := a.Ask(ctx, alice, ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Ask(ctx, alice, ask.Request{Question: "and for staff?", Conversation: r1.Conversation}); err != nil {
		t.Fatal(err)
	}
	next, err := a.Branch(ctx, alice, r1.Conversation, 1)
	if err != nil || next == r1.Conversation {
		t.Fatalf("branch: %q %v", next, err)
	}
	qs, ok, _ := a.Store.Questions(ctx, tenant, "alice", next)
	if !ok || len(qs) != 1 || qs[0] != "refund limit?" {
		t.Fatalf("the branch's questions: %v", qs)
	}
	if old, _, _ := a.Store.Questions(ctx, tenant, "alice", r1.Conversation); len(old) != 2 {
		t.Fatalf("the first conversation changed: %v", old)
	}
	for _, c := range []struct {
		who  string
		conv string
		at   int
	}{{"bob", r1.Conversation, 1}, {"alice", r1.Conversation, 3}, {"alice", r1.Conversation, 0}, {"alice", "cv_none", 1}} {
		if _, err := a.Branch(ctx, scope(t, c.who), c.conv, c.at); !errors.Is(err, ask.ErrInvalid) {
			t.Errorf("%+v: %v", c, err)
		}
	}
}

// SB-5301: a pull request on GitHub and a ticket are work events; a document isn't.
func TestIsWorkEvent(t *testing.T) {
	pr := ports.Chunk{Kind: "code", SourceID: "acme/app/pulls/12"}
	if !ask.IsWorkEvent(ask.OnPROpened, pr) || ask.IsWorkEvent(ask.OnTicketFiled, pr) ||
		!ask.IsWorkEvent(ask.OnTicketFiled, ports.Chunk{Kind: "tickets", SourceID: "LIN-7"}) ||
		ask.IsWorkEvent(ask.OnPROpened, ports.Chunk{Kind: "code", SourceID: "acme/app/blob/README.md"}) {
		t.Fatal("IsWorkEvent")
	}
}
