package ask_test

import (
	"context"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

// SB-3303: a short ask adds the line and keeps at most three claims; SB-3304: a disagreement
// with one document is dropped.
func TestShortAnswerAndOneDocumentDisagreement(t *testing.T) {
	ctx := context.Background()
	m := &model{reply: `{"claims":[{"text":"a","cites":["C1"],"disagree":true},{"text":"b","cites":["C1"]},{"text":"c","cites":["C1"]},{"text":"d","cites":[]},{"text":"e","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	short, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?", Length: "short"})
	if err != nil {
		t.Fatal(err)
	}
	if len(short.Answer.Claims) != ask.MaxShort || !strings.HasSuffix(m.got[0].System, ask.ShortLine) {
		t.Fatalf("short: %d claims, system %q", len(short.Answer.Claims), m.got[0].System)
	}
	if short.Answer.Claims[0].Disagree {
		t.Fatal("a disagreement citing one document is dropped")
	}
	full, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?", Length: "long"})
	if err != nil || len(full.Answer.Claims) != 5 || strings.Contains(m.got[1].System, ask.ShortLine) {
		t.Fatalf("any other length changes nothing: %v %d", err, len(full.Answer.Claims))
	}
}

// SB-3304: the composer keeps disagree only when two documents survive.
func TestDisagreeNeedsTwoDocuments(t *testing.T) {
	c := answer.NewContext([]ports.Chunk{{Cite: "docs:a@1"}, {Cite: "docs:a@1", Start: 5}, {Cite: "docs:b@3"}})
	res := answer.Compose(`{"claims":[{"text":"x","cites":["C1","C3"],"disagree":true},{"text":"y","cites":["C1","C2"],"disagree":true},{"text":"z","cites":["C1","C9"],"disagree":true}]}`, c, nil)
	if !res.Claims[0].Disagree || res.Claims[1].Disagree || res.Claims[2].Disagree {
		t.Fatalf("%+v", res.Claims)
	}
}
