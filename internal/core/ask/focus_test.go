package ask_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

func prompt(r ports.ModelRequest) string {
	var b strings.Builder
	for _, m := range r.Messages {
		b.WriteString(m.Content + "\n")
	}
	return b.String()
}

// SB-2901: a later turn sends only the last 8 earlier questions and says how many it left
// out; the conversation keeps them all.
func TestShortConversations(t *testing.T) {
	ctx := context.Background()
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	resp, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund question 1?"})
	if err != nil || resp.EarlierDropped != 0 {
		t.Fatalf("%+v %v", resp, err)
	}
	cv := resp.Conversation
	for i := 2; i <= 10; i++ {
		if resp, err = a.Ask(ctx, scope(t, "alice"), ask.Request{Question: fmt.Sprintf("refund question %d?", i), Conversation: cv}); err != nil {
			t.Fatal(err)
		}
	}
	// The tenth question has nine earlier ones: the first is left out.
	last := prompt(m.got[len(m.got)-1])
	if resp.EarlierDropped != 1 || strings.Contains(last, "refund question 1?") || !strings.Contains(last, "refund question 2?") || !strings.Contains(last, "refund question 9?") {
		t.Fatalf("dropped %d, prompt:\n%s", resp.EarlierDropped, last)
	}
	qs, _, _ := a.Store.Questions(ctx, tenant, "alice", cv)
	if len(qs) != 10 {
		t.Fatalf("the stored conversation lost questions: %d", len(qs))
	}
}

// SB-2902: a long document matching every chunk gives the model 3 of them; other documents
// keep their place.
func TestThreeChunksPerDocument(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"x","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	ix := a.Index.(index)
	for i := 0; i < 6; i++ {
		ix = append(ix, ports.Chunk{ID: fmt.Sprintf("chk_long%d", i), Tenant: tenant, Kind: "docs", SourceID: "refunds", Version: 2,
			Cite: "docs:refunds@2", Text: fmt.Sprintf("refund limit part %d", i)})
	}
	a.Index = ix
	if _, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"}); err != nil {
		t.Fatal(err)
	}
	p := prompt(m.got[0])
	if n := strings.Count(p, "docs:refunds@2"); n != 3 {
		t.Fatalf("%d chunks of one document in the prompt:\n%s", n, p)
	}
}
