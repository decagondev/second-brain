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

// SB-3306: owner= lists the owner's documents the asker can open, or narrows a search; an
// owner who isn't one active user matches nothing.
func TestTheirDocuments(t *testing.T) {
	ctx := context.Background()
	a, _ := setup(t, &model{})
	ix := a.Index.(index)
	ix[0].Owner, ix[1].Owner, ix[0].Title, ix[1].Title = "ann@acme.example", "ann@acme.example", "Refunds", "Salaries"
	ix = append(ix, ports.Chunk{ID: "chk_old", Tenant: tenant, Kind: "docs", SourceID: "refunds", Version: 1, Cite: "docs:refunds@1", Text: "old", Owner: "ann@acme.example", Title: "Refunds"})
	a.Index = ix
	a.Owner = func(_ context.Context, email string) (answer.Person, bool) {
		return answer.Person{Email: email}, email == "ann@acme.example"
	}
	got, err := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Owner: "Ann@acme.example"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || got.Results[0].Cite != "docs:refunds@2" || got.Refused != 1 {
		t.Fatalf("listed %+v refused %d", got.Results, got.Refused)
	}
	if got, _ := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Query: "refund limit", Owner: "bob@acme.example"}); len(got.Results) != 0 {
		t.Fatalf("someone else's: %+v", got.Results)
	}
	a.Owner = func(context.Context, string) (answer.Person, bool) { return answer.Person{}, false }
	if got, _ := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Owner: "ann@acme.example"}); len(got.Results) != 0 {
		t.Fatal("an owner who isn't one active user matches nothing")
	}
	if _, err := a.Search(ctx, scope(t, "alice"), ask.SearchRequest{Owner: "not-an-email"}); err == nil {
		t.Fatal("owner must be an email")
	}
}
