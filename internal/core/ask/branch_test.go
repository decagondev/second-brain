package ask_test

import (
	"context"
	"errors"
	"testing"

	"example.com/second-brain/internal/core/ask"
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
