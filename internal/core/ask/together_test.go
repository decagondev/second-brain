package ask_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
)

// SB-5202, SB-5204: the reviewer edits a section in place and is an editor; a rejected
// suggestion puts back what the section held; a draft with an open suggestion can't be
// published; the published note names who asked for it and who edited it, and says edited.
func TestEditADraftTogether(t *testing.T) {
	ctx := context.Background()
	a, _ := setup(t, &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`})
	a.Emails = func(_ context.Context, u string) ([]string, error) { return []string{u + "@acme.example"}, nil }
	src := a.Auth.Connectors["docs"].(source)
	src["refunds|sam@acme.example"] = ports.VerdictAllow
	src["refunds|sam"] = ports.VerdictAllow
	bus := &fakeBus{}
	n := 0
	store := &stq{m: map[string]ask.StewardQuestion{}}
	x := ask.Stewards{Agent: a, Store: store, Audit: &rows{}, Bus: bus,
		Policy:  policy.Kernel{Engine: allowAll{}, Recorder: rec{}},
		Facts:   func(string) ([]string, string, bool) { return []string{"group:answerers"}, "", true },
		Readers: func(context.Context, string) ([]string, error) { return []string{"sam@acme.example"}, nil },
		NewID:   func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Now: time.Now}
	sam, kim := scope(t, "sam"), scope(t, "kim")
	agentText := "# Shipped\nThe dock opens at 7.\n\n# Decided\nVans are washed on Fridays.\n"
	q := ask.StewardQuestion{ID: "qs_d", Tenant: tenant, Asker: "sam", Question: "Weekly brief", Space: "company", At: time.Now(),
		Thread: &ask.SavedThread{Agent: "case-brief", Kind: ask.DraftBrief, Release: "rel_7", Text: agentText, Asked: ask.BriefQuestion,
			Suggestions: []ask.DraftSuggestion{{N: 1, Section: 1, Old: "# Shipped\nThe dock opened.\n\n", New: "# Shipped\nThe dock opens at 7.\n\n", Round: 1, State: ask.SuggestionOpen},
				{N: 2, Section: 2, Old: "# Decided\nNothing.\n", New: "# Decided\nVans are washed on Fridays.\n", Round: 1, State: ask.SuggestionOpen}}}}
	if err := store.PutStewardQuestion(ctx, q); err != nil {
		t.Fatal(err)
	}
	if _, err := x.EditSection(ctx, kim, "company", "qs_d", 2, "# Decided\nOnly sam reviews."); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("another steward edited: %v", err)
	}
	th, err := x.Decide(ctx, sam, "company", "qs_d", 1, false)
	if err != nil || !strings.Contains(th.Text, "The dock opened.") || th.Suggestions[0].State != ask.SuggestionRejected {
		t.Fatalf("reject: %v %+v", err, th)
	}
	if _, err := x.Decide(ctx, sam, "company", "qs_d", 1, true); !errors.Is(err, ask.ErrInvalid) {
		t.Fatalf("a decided suggestion was decided again: %v", err)
	}
	if _, err := x.Reply(ctx, sam, "company", "qs_d", "The dock opened.", []string{"docs:refunds@2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := x.Note(ctx, sam, "company", "qs_d", ask.NoteRequest{Title: "Brief"}); err == nil || !strings.Contains(err.Error(), "open suggestions") {
		t.Fatalf("a draft with an open suggestion was published: %v", err)
	}
	// Back to an open draft: accept the other suggestion, edit a section, then publish.
	if err := store.PutStewardQuestion(ctx, func() ask.StewardQuestion {
		c, _, _ := store.GetStewardQuestion(ctx, tenant, "qs_d")
		c.Reply = nil
		return c
	}()); err != nil {
		t.Fatal(err)
	}
	if _, err := x.Decide(ctx, sam, "company", "qs_d", 2, true); err != nil {
		t.Fatal(err)
	}
	th, err = x.EditSection(ctx, sam, "company", "qs_d", 2, "# Decided\nVans are washed on Fridays at 3.")
	if err != nil || len(th.Editors) != 1 || th.Editors[0] != "sam" || th.Generated == "" || !strings.HasSuffix(th.Text, "at 3.\n") {
		t.Fatalf("edit: %v %+v", err, th)
	}
	if _, err := x.Reply(ctx, sam, "company", "qs_d", th.Text, []string{"docs:refunds@2"}); err != nil {
		t.Fatal(err)
	}
	zero := 0
	res, err := x.Note(ctx, sam, "company", "qs_d", ask.NoteRequest{Title: "Brief", Confirm: &zero})
	if err != nil || len(bus.calls) != 1 {
		t.Fatalf("publish: %+v %v", res, err)
	}
	body := string(bus.calls[0].Payload)
	for _, want := range []string{"generated_by: case-brief@rel_7", "asked_by: sam@acme.example", "edited_by: sam@acme.example", "edited: true"} {
		if !strings.Contains(body, want) {
			t.Errorf("the note lacks %q: %s", want, body)
		}
	}
}

// SB-5204: a cite's mark names who asked for it and who edited it.
func TestCiteMarkNamesWhoMadeIt(t *testing.T) {
	got := ask.CiteMark("case-brief@rel_7", "sam@x", true, "kim@x", "kim@x, lee@x")
	if got != "Written by case-brief, version rel_7, asked for by kim@x, edited by kim@x, lee@x, edited and approved by sam@x" {
		t.Fatalf("%q", got)
	}
}
