package ask_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
)

// SB-3001: a passage is the cited chunk's text, for the asker only, for a cite in the
// answer only, checked live each time: revoked access refuses it, and a version the index
// lost says gone.
func TestPassage(t *testing.T) {
	ctx := context.Background()
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	resp, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Passage(ctx, scope(t, "alice"), resp.Trace, "docs:refunds@2", -1)
	if err != nil || p.Text != "refund limit is 500" || p.Cut || p.Gone {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := a.Passage(ctx, scope(t, "alice"), resp.Trace, "docs:salaries@2", -1); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a cite not in the answer: %v", err)
	}
	if _, err := a.Passage(ctx, scope(t, "bob"), resp.Trace, "docs:refunds@2", -1); !errors.Is(err, ask.ErrNotFound) {
		t.Fatalf("someone else's trace: %v", err)
	}
	if _, err := a.Passage(ctx, scope(t, "alice"), resp.Trace, "docs:refunds@2", 7); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a span not in the answer: %v", err)
	}
	src := a.Auth.Connectors["docs"].(source)
	src["refunds|alice"] = ports.VerdictDeny
	if _, err := a.Passage(ctx, scope(t, "alice"), resp.Trace, "docs:refunds@2", -1); !errors.Is(err, ask.ErrSourceDenied) {
		t.Fatalf("revoked after the answer: %v", err)
	}
	src["refunds|alice"] = ports.VerdictUnknown
	if _, err := a.Passage(ctx, scope(t, "alice"), resp.Trace, "docs:refunds@2", -1); !errors.Is(err, ask.ErrSourceDenied) {
		t.Fatalf("unknown is never allow: %v", err)
	}
	src["refunds|alice"] = ports.VerdictAllow
	a.Index = index{}
	if p, err := a.Passage(ctx, scope(t, "alice"), resp.Trace, "docs:refunds@2", -1); err != nil || !p.Gone || p.Text != "" {
		t.Fatalf("gone: %+v %v", p, err)
	}
}

func TestExcerpt(t *testing.T) {
	long := strings.Repeat("Refunds need approval. ", 40)
	got, cut := ask.Excerpt(long, ask.MaxPassage)
	if !cut || len([]rune(got)) > ask.MaxPassage || !strings.HasSuffix(got, "approval.") {
		t.Fatalf("%v %q", cut, got)
	}
	got, cut = ask.Excerpt(strings.Repeat("word ", 200), 600)
	if !cut || !strings.HasSuffix(got, "…") || len([]rune(got)) > 601 {
		t.Fatalf("%v %q", cut, got)
	}
	if got, cut := ask.Excerpt("short.", 600); cut || got != "short." {
		t.Fatalf("%v %q", cut, got)
	}
}

// SB-3003: a language adds one line to the prompt and changes no cite; an unknown code
// changes nothing.
func TestAnswerLanguage(t *testing.T) {
	ctx := context.Background()
	m := &model{reply: `{"claims":[{"text":"Das Limit ist 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	de, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?", Language: "de"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(m.got[0].System, "Write each claim's text in German. Keep every label exactly as given.") {
		t.Fatalf("system prompt: %q", m.got[0].System)
	}
	none, err := a.Ask(ctx, scope(t, "alice"), ask.Request{Question: "refund limit?", Language: "xx"})
	if err != nil {
		t.Fatal(err)
	}
	if m.got[1].System == m.got[0].System || strings.Contains(m.got[1].System, "Write each claim") {
		t.Fatalf("unknown code: %q", m.got[1].System)
	}
	if de.Answer.Claims[0].Cites[0].Cite != none.Answer.Claims[0].Cites[0].Cite || de.Answer.Claims[0].Uncited {
		t.Fatalf("%+v", de.Answer.Claims)
	}
}
