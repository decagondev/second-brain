package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/ask"
)

// SB-4301 to SB-4303 through a running cell: the inbox lists a steward's draft, a policy to
// read, a research plan and an action item, each checked as them when listed (LAW-26); a
// saved question becomes a routine (a sixth is refused), the notify job asks it as its
// owner and keeps the answer; and a memory keeps only what its person wrote (LAW-25), is
// in their export and goes with Forget.
func TestWorkComesToYouThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	write := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Finance/close.md", "The quokka close moves to Tuesday, and the AP review is still open for the whole quarter.")
	write("Finance/expenses.md", "Expense policy: every quokka trip over 500 needs a receipt.")
	write("Finance/standup.vtt", "WEBVTT\n\n00:01:00.000 --> 00:01:04.000\nWe decided the quokka close moves to Tuesday.\n\n"+
		"00:01:05.000 --> 00:01:09.000\nAction: sam to send the quokka report by 2026-12-01\n\n")
	write("Finance/.acl", "read domain:acme.example\n")
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  finance:
    template: finance
    label: confidential
    members: [group:finance]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	r.signIn(t, "sam")
	sam := r.idp.User
	for name := range map[string]bool{"finance": true, "finance-leads": true} {
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[{"value":"`+sam+`"}]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(context.Background()); err != nil { // the transcript's decisions draft
		t.Fatal(err)
	}
	inbox := func() ask.Inbox {
		t.Helper()
		code, out := r.do(t, "GET", "/v1/me/inbox", "")
		var in ask.Inbox
		if err := json.Unmarshal([]byte(out), &in); err != nil || code != 200 {
			t.Fatalf("inbox: %d %s", code, out)
		}
		if in.Counts.NeedsYou != len(in.NeedsYou) || in.Counts.Failed != len(in.Failed) || in.Counts.Done != len(in.Done) {
			t.Fatalf("counts don't match the lists: %+v", in)
		}
		return in
	}
	has := func(xs []ask.InboxItem, kind, link string) *ask.InboxItem {
		for i := range xs {
			if xs[i].Kind == kind && (link == "" || xs[i].Link == link) {
				return &xs[i]
			}
		}
		return nil
	}

	// One inbox: the decisions draft, a policy to read and a plan to run.
	if code, out := r.do(t, "POST", "/v1/spaces/finance/policies", `{"doc":"docs:Finance/expenses.md"}`); code != 200 {
		t.Fatalf("policy: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/research?space=finance", `{"question":"When does the quokka close move?"}`); code != 200 {
		t.Fatalf("plan: %d %s", code, out)
	}
	in := inbox()
	if has(in.NeedsYou, ask.InboxDraft, "/?steward=finance") == nil || has(in.NeedsYou, ask.InboxPolicy, "/?space=finance") == nil ||
		has(in.NeedsYou, ask.InboxResearch, "/?tab=legwork") == nil {
		t.Fatalf("needs you: %+v", in.NeedsYou)
	}

	// An action item is listed while its transcript can be opened, and goes when it can't.
	code, out := r.do(t, "GET", "/v1/spaces/finance/questions", "")
	var q struct{ Items []ask.StewardItem }
	if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
		t.Fatalf("queue: %d %s", code, out)
	}
	for _, it := range q.Items {
		if it.Thread != nil && it.Thread.Kind == ask.DraftDecisions {
			body, _ := json.Marshal(map[string]any{"draft": it.ID, "items": it.Thread.Actions})
			if code, out := r.do(t, "POST", "/v1/spaces/finance/actions", string(body)); code != 200 {
				t.Fatalf("confirm: %d %s", code, out)
			}
		}
	}
	if has(inbox().NeedsYou, ask.InboxAction, "") == nil {
		t.Fatal("the action item isn't in the inbox")
	}
	write("Finance/standup.vtt.acl", "read lee@acme.example\n")
	if it := has(inbox().NeedsYou, ask.InboxAction, ""); it != nil {
		t.Fatalf("an action item whose transcript they lost is still listed: %+v", it)
	}

	// Routines: at most five; the job asks one due as its owner and keeps the answer.
	var ids []string
	for i := 0; i < 6; i++ {
		code, out := r.do(t, "POST", "/v1/saved?space=finance", `{"question":"When does the quokka close move? `+strings.Repeat("?", i)+`"}`)
		var sq ask.SavedQuestion
		if err := json.Unmarshal([]byte(out), &sq); err != nil || code != 200 {
			t.Fatalf("save: %d %s", code, out)
		}
		ids = append(ids, sq.ID)
	}
	for i, id := range ids {
		code, out := r.do(t, "PUT", "/v1/saved/"+id+"/routine?space=finance", `{"every":"weekly","day":"monday"}`)
		if want := i < ask.MaxRoutines; (code == 200) != want {
			t.Fatalf("routine %d: %d %s", i, code, out)
		}
	}
	if code, out := r.do(t, "PUT", "/v1/saved/"+ids[1]+"/routine", `{"every":"weekly","day":"sunday"}`); code == 200 {
		t.Fatalf("a weekend day was taken: %s", out)
	}
	for _, id := range ids[1:] {
		r.do(t, "PUT", "/v1/saved/"+id+"/routine", `{"every":"off"}`)
	}
	if _, err := r.cell.Traces.UpdateSaved(context.Background(), "acme-logistics", sam, ids[0], func(sq *ask.SavedQuestion) {
		sq.Routine.Last = time.Now().Add(-8 * 24 * time.Hour) // a week ago: due
	}); err != nil {
		t.Fatal(err)
	}
	told := &agentsTold{}
	if _, err := r.cell.NotifyJob(told).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(told.got, "\n"), "routine "+sam+": Your routine answer is ready: ") {
		t.Fatalf("told: %v", told.got)
	}
	it := has(inbox().Done, ask.InboxRoutine, "/?tab=legwork")
	if it == nil {
		t.Fatal("the routine's answer isn't done in the inbox")
	}
	code, out = r.do(t, "GET", "/v1/research/"+it.ID+"?space=finance", "")
	var rv ask.ResearchView
	if err := json.Unmarshal([]byte(out), &rv); err != nil || code != 200 || rv.Kind != ask.KindRoutine || rv.Changed == nil || *rv.Changed == 0 ||
		!strings.Contains(out, `"cite":"docs:Finance/close.md@`) {
		t.Fatalf("routine answer: %d %s", code, out)
	}
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(context.Background()); err != nil { // not due again this week
		t.Fatal(err)
	}
	n := 0
	for _, x := range inbox().Done {
		if x.Kind == ask.InboxRoutine {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the routine ran again before its day: %d", n)
	}

	// Memory: the person's own words, never a document's (LAW-25).
	if code, out := r.do(t, "POST", "/v1/me/memory", `{"text":"I lead the finance team and like short answers."}`); code != 200 {
		t.Fatalf("remember: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/me/memory", `{"text":"the quokka close moves to Tuesday, and the AP review is still open"}`); code != 400 || !strings.Contains(out, "document") {
		t.Fatalf("a pasted passage was kept: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/me/memory", ""); code != 200 || !strings.Contains(out, "finance team") || strings.Contains(out, "Tuesday") {
		t.Fatalf("memory: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/ask?space=finance", `{"question":"When does the quokka close move?"}`); code != 200 {
		t.Fatalf("ask with a memory: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/me/export", ""); code != 200 || !strings.Contains(out, `"memory":[`) || !strings.Contains(out, "finance team") {
		t.Fatalf("export: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/me/forget", `{"confirm":"forget my questions"}`); code != 200 || !strings.Contains(out, `"memory":1`) {
		t.Fatalf("forget: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/me/memory", ""); code != 200 || strings.Contains(out, "finance team") {
		t.Fatalf("memory after forget: %d %s", code, out)
	}
}
