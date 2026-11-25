package slackapp

import (
	"context"
	"strings"
	"testing"
)

// SB-5102: a reply is masked before it is posted, and the app says how many strings changed.
func TestReplyIsMasked(t *testing.T) {
	key := "AKIA" + strings.Repeat("Q7", 8)
	var got []Message
	n := 0
	a := &App{Post: func(_ context.Context, _ string, m Message) error { got = append(got, m); return nil },
		Masked: func(_ context.Context, k int) { n += k }}
	if err := a.post(context.Background(), hook, Message{ResponseType: "ephemeral", Text: "card 4111 1111 1111 1111, key " + key}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || strings.Contains(got[0].Text, key) || strings.Contains(got[0].Text, "4111 1111 1111 1111") || got[0].ResponseType != "ephemeral" || n != 1 {
		t.Fatalf("posted %+v, masked %d", got, n)
	}
}
