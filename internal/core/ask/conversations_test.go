package ask_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// memRecords is a MemoryStore, one record per person.
type memRecords map[string]ask.Memory

func (m memRecords) MemoryOf(_ context.Context, _, user string) (ask.Memory, bool, error) {
	r, ok := m[user]
	return r, ok, nil
}
func (m memRecords) PutMemory(_ context.Context, r ask.Memory) error {
	if r.Empty() {
		delete(m, r.User)
		return nil
	}
	m[r.User] = r
	return nil
}

func conversationAgent(t *testing.T) (ask.Agent, memRecords, source) {
	m := &model{reply: `{"claims":[{"text":"500","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	mem := memRecords{}
	a.Exclusions = mem
	st := a.Store.(*store)
	a.TracesOf = func(_ context.Context, user string, _ time.Time) ([]ask.Trace, error) {
		var out []ask.Trace
		for _, tr := range st.traces {
			if tr.User == user {
				out = append(out, tr)
			}
		}
		return out, nil
	}
	return a, mem, a.Auth.Connectors["docs"].(source)
}

// SB-5802, SB-5803: a chat conversation is listed for its person alone and opens with each
// answer checked as them now; a web one is listed only once they keep conversations on
// every device, which keeps no answer, and turning it off deletes every entry.
func TestYourConversations(t *testing.T) {
	a, mem, src := conversationAgent(t)
	ctx := context.Background()
	chat := tenancy.WithClient(ctx, tenancy.ClientChat)
	r1, err := a.Ask(chat, scope(t, "alice"), ask.Request{Question: "refund limit?", Thread: "slack/T1/C1/1.1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit on the web?"}); err != nil {
		t.Fatal(err)
	}
	list, err := a.ListConversations(ctx, scope(t, "alice"))
	if err != nil || len(list.Conversations) != 1 || list.Conversations[0].ID != r1.Conversation || list.Conversations[0].Client != "chat" || list.Conversations[0].First != "refund limit?" || list.Sync {
		t.Fatalf("list %+v %v", list, err)
	}
	if other, _ := a.ListConversations(ctx, scope(t, "bob")); len(other.Conversations) != 0 {
		t.Fatalf("bob sees %+v", other)
	}
	if _, err := a.OpenConversation(ctx, scope(t, "bob"), r1.Conversation); !errors.Is(err, ask.ErrNoConversation) {
		t.Fatalf("bob opened alice's: %v", err)
	}
	v, err := a.OpenConversation(ctx, scope(t, "alice"), r1.Conversation)
	if err != nil || len(v.Turns) != 1 || v.Turns[0].Unavailable || v.Turns[0].Question != "refund limit?" || len(v.Turns[0].Claims) != 1 {
		t.Fatalf("open %+v %v", v, err)
	}
	src["refunds|alice"] = ports.VerdictDeny // access removed since
	if v, _ = a.OpenConversation(ctx, scope(t, "alice"), r1.Conversation); !v.Turns[0].Unavailable || len(v.Turns[0].Claims) != 0 {
		t.Fatalf("a cite she can't open now shows: %+v", v)
	}
	src["refunds|alice"] = ports.VerdictAllow

	if _, err := a.SetSync(ctx, scope(t, "alice"), true); err != nil {
		t.Fatal(err)
	}
	r3, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "and on Fridays?"})
	if err != nil {
		t.Fatal(err)
	}
	rec := mem["alice"]
	if len(rec.Conversations) != 1 || rec.Conversations[0].ID != r3.Conversation || rec.Conversations[0].Traces[0] != r3.Trace || rec.Conversations[0].Questions[0] != "and on Fridays?" {
		t.Fatalf("synced %+v", rec)
	}
	if list, _ = a.ListConversations(ctx, scope(t, "alice")); len(list.Conversations) != 2 || !list.Sync {
		t.Fatalf("synced list %+v", list)
	}
	if _, err := a.SetSync(ctx, scope(t, "alice"), false); err != nil {
		t.Fatal(err)
	}
	if _, kept := mem["alice"]; kept {
		t.Fatalf("off kept %+v", mem["alice"])
	}
	if list, _ = a.ListConversations(ctx, scope(t, "alice")); len(list.Conversations) != 1 {
		t.Fatalf("after off %+v", list)
	}
}

// SB-5804: a shared conversation shows the reader only the answers whose every cite they
// can open now, counts the rest, carries the highest label, and ends when revoked.
func TestShareAConversation(t *testing.T) {
	a, _, src := conversationAgent(t)
	ctx := context.Background()
	r1, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "and over that?", Conversation: r1.Conversation}); err != nil {
		t.Fatal(err)
	}
	st := &qstore{saved: map[string]ask.SavedQuestion{}, shares: map[string]ask.Share{}}
	x := ask.ConversationShares{Agent: a, Questions: ask.Questions{Tenant: tenant, Store: st, Audit: &rows{}, Now: time.Now}}
	if _, err := x.Share(ctx, scope(t, "bob"), r1.Conversation); err == nil {
		t.Fatal("bob shared alice's conversation")
	}
	sh, err := x.Share(ctx, scope(t, "alice"), r1.Conversation)
	if err != nil || len(sh.Traces) != 2 {
		t.Fatalf("share %+v %v", sh, err)
	}
	v, err := x.Open(ctx, scope(t, "bob"), sh.ID)
	if err != nil || len(v.Turns) != 0 || v.Left != 2 {
		t.Fatalf("bob can't open the cites: %+v %v", v, err)
	}
	src["refunds|bob"] = ports.VerdictAllow
	if v, err = x.Open(ctx, scope(t, "bob"), sh.ID); err != nil || len(v.Turns) != 2 || v.Left != 0 || v.Turns[1].Question != "and over that?" {
		t.Fatalf("bob can: %+v %v", v, err)
	}
	if err := x.Revoke(ctx, scope(t, "bob"), sh.ID); !errors.Is(err, ask.ErrNoShare) {
		t.Fatalf("bob revoked: %v", err)
	}
	if err := x.Revoke(ctx, scope(t, "alice"), sh.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := x.Open(ctx, scope(t, "bob"), sh.ID); !errors.Is(err, ask.ErrNoShare) {
		t.Fatalf("a revoked share opened: %v", err)
	}
	q := ask.Questions{Tenant: tenant, Store: st, Audit: &rows{}, Now: time.Now}
	if _, err := q.Open(ctx, scope(t, "bob"), sh.ID); !errors.Is(err, ask.ErrNoShare) {
		t.Fatal("a conversation share opened as a question share")
	}
}
