package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/ask"
)

// SB-4201 to SB-4203 through a running cell: a research report is planned, run by the
// notify job and checked again as its owner when read (LAW-26); a pasted agenda becomes a
// brief; and a transcript's action items reach their owner only once the steward confirms.
func TestLegworkThroughARunningCell(t *testing.T) {
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
	write("Finance/close.md", "The quokka close moves to Tuesday, and the AP review is still open.")
	write("Finance/standup.vtt", "WEBVTT\n\n00:01:00.000 --> 00:01:04.000\nWe decided the quokka close moves to Tuesday.\n\n"+
		"00:01:05.000 --> 00:01:09.000\nAction: sam to send the quokka report by 2026-12-01\n\n")
	write("Finance/.acl", "read domain:acme.example\n")
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
asks_per_minute: 60
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
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, ids := range map[string][]string{"finance": {sam, lee}, "finance-leads": {sam}} {
		var ms []string
		for _, id := range ids {
			ms = append(ms, `{"value":"`+id+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}

	// Research: planned, then run only when asked to.
	code, out := r.do(t, "POST", "/v1/research?space=finance", `{"question":"When does the quokka close move?"}`)
	var rv ask.ResearchView
	if err := json.Unmarshal([]byte(out), &rv); err != nil || code != 200 || rv.State != ask.StatePlanned || len(rv.Steps) == 0 {
		t.Fatalf("plan: %d %s", code, out)
	}
	if code, out = r.do(t, "POST", "/v1/research/"+rv.ID+"/run?space=finance", ""); code != 200 || !strings.Contains(out, `"state":"queued"`) {
		t.Fatalf("run: %d %s", code, out)
	}
	told := &agentsTold{}
	if _, err := r.cell.NotifyJob(told).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(told.got, "\n"), "research "+sam+": Your research report is ready") {
		t.Fatalf("told: %v", told.got)
	}
	code, out = r.do(t, "GET", "/v1/research/"+rv.ID+"?space=finance", "")
	if err := json.Unmarshal([]byte(out), &rv); err != nil || code != 200 || rv.State != ask.StateDone || !strings.Contains(out, `"cite":"docs:`) {
		t.Fatalf("report: %d %s", code, out)
	}

	// LAW-26: the owner loses the documents; the stored report no longer shows them.
	write("Finance/.acl", "read lee@acme.example\n")
	code, out = r.do(t, "GET", "/v1/research/"+rv.ID+"?space=finance", "")
	if code != 200 || strings.Contains(out, `"cite":"docs:`) {
		t.Fatalf("report after the grant went: %d %s", code, out)
	}
	write("Finance/.acl", "read domain:acme.example\n")

	// Meeting prep from a pasted agenda.
	code, out = r.do(t, "POST", "/v1/prepare?space=finance", `{"agenda":"- quokka close\n- AP review","title":"Finance sync"}`)
	if err := json.Unmarshal([]byte(out), &rv); err != nil || code != 200 || rv.Kind != ask.KindPrep || len(rv.Sections) != 2 {
		t.Fatalf("prepare: %d %s", code, out)
	}

	// Action items: in the decisions draft, for the steward it ran as only.
	code, out = r.do(t, "GET", "/v1/spaces/finance/questions", "")
	var q struct{ Items []ask.StewardItem }
	if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
		t.Fatalf("queue: %d %s", code, out)
	}
	draft := ""
	var items []ask.DraftAction
	for _, it := range q.Items {
		if it.Thread != nil && it.Thread.Kind == ask.DraftDecisions {
			draft, items = it.ID, it.Thread.Actions
		}
	}
	if draft == "" || len(items) != 1 || items[0].Owner != "sam" || items[0].Due != "2026-12-01" {
		t.Fatalf("decisions draft: %s", out)
	}
	if code, out = r.do(t, "GET", "/v1/me/actions", ""); code != 200 || strings.Contains(out, "quokka report") {
		t.Fatalf("told before confirming: %d %s", code, out)
	}
	body, _ := json.Marshal(map[string]any{"draft": draft, "items": items})
	if code, out = r.do(t, "POST", "/v1/spaces/finance/actions", string(body)); code != 200 || !strings.Contains(out, `"user":"`+sam+`"`) {
		t.Fatalf("confirm: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/me/actions", "")
	var mine struct{ Actions []ask.ActionView }
	if err := json.Unmarshal([]byte(out), &mine); err != nil || code != 200 || len(mine.Actions) != 1 {
		t.Fatalf("mine: %d %s", code, out)
	}
	if code, out = r.do(t, "POST", "/v1/me/actions/"+mine.Actions[0].ID+"/done", ""); code != 200 || !strings.Contains(out, "done_at") {
		t.Fatalf("done: %d %s", code, out)
	}
	// SB-4903: a long report shows its plan and runs nothing until Run, then in stages with a
	// checkpoint after each, steered, re-checked as its owner, and stopped at its ceiling.
	code, out = r.do(t, "POST", "/v1/research?space=finance", `{"question":"When does the quokka close move?","long":true}`)
	if err := json.Unmarshal([]byte(out), &rv); err != nil || code != 200 || rv.State != ask.StatePlanned || rv.Long == nil || rv.Long.LimitUSD != ask.DefaultLongUSD || rv.Long.EstimateUSD <= 0 {
		t.Fatalf("long plan: %d %s", code, out)
	}
	ctx := context.Background()
	stored, _, err := r.cell.Traces.GetResearch(ctx, "acme-logistics", rv.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.Steps = []string{"When does the quokka close move?", "Is the AP review open?", "What did standup decide?", "Who sends the quokka report?"}
	stored.Long.PerStage = 1
	if err := r.cell.Traces.PutResearch(ctx, stored); err != nil {
		t.Fatal(err)
	}
	told = &agentsTold{}
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if _, out = r.do(t, "GET", "/v1/research/"+rv.ID+"?space=finance", ""); !strings.Contains(out, `"state":"planned"`) || strings.Contains(out, `"cite":"docs:`) {
		t.Fatalf("a long plan ran before Run: %s", out)
	}
	if code, out = r.do(t, "POST", "/v1/research/"+rv.ID+"/run?space=finance", ""); code != 200 {
		t.Fatalf("run: %d %s", code, out)
	}
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	code, out = r.do(t, "GET", "/v1/research/"+rv.ID+"?space=finance", "")
	if err := json.Unmarshal([]byte(out), &rv); err != nil || rv.Long == nil || rv.Long.Stage != 1 || rv.Long.Checkpoint == nil || rv.Long.GoesOnAt == nil || len(rv.Sections) != 1 {
		t.Fatalf("after stage 1: %d %s", code, out)
	}
	if !strings.Contains(strings.Join(told.got, "\n"), "finished a stage") {
		t.Fatalf("not told of the checkpoint: %v", told.got)
	}
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if _, out = r.do(t, "GET", "/v1/research/"+rv.ID+"?space=finance", ""); !strings.Contains(out, `"stage":1`) {
		t.Fatalf("it went on before its owner answered or 30 minutes passed: %s", out)
	}
	if code, out = r.do(t, "POST", "/v1/research/"+rv.ID+"/steer?space=finance", `{"drop":[0]}`); code != http.StatusBadRequest {
		t.Fatalf("dropped a sub-question already asked: %d %s", code, out)
	}
	if code, out = r.do(t, "POST", "/v1/research/"+rv.ID+"/steer?space=finance", `{"drop":[3],"add":["When is the AP review due?"]}`); code != 200 || !strings.Contains(out, "When is the AP review due?") || strings.Contains(out, "Who sends the quokka report?") {
		t.Fatalf("steer: %d %s", code, out)
	}
	if code, out = r.do(t, "POST", "/v1/research/"+rv.ID+"/go-on?space=finance", ""); code != 200 {
		t.Fatalf("go on: %d %s", code, out)
	}
	// LAW-26: the owner loses the documents before stage 2; its check drops stage 1's section.
	write("Finance/.acl", "read lee@acme.example\n")
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	stored, _, err = r.cell.Traces.GetResearch(ctx, "acme-logistics", rv.ID)
	if err != nil || stored.Long.Stage != 2 || len(stored.Sections) != 0 || stored.Long.Dropped == 0 {
		t.Fatalf("stage 2 kept what its owner lost: %v %+v", err, stored.Long)
	}
	write("Finance/.acl", "read domain:acme.example\n")
	if code, out = r.do(t, "POST", "/v1/research/"+rv.ID+"/go-on?space=finance", ""); code != 200 {
		t.Fatalf("go on: %d %s", code, out)
	}
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	// The ceiling: nothing more is asked once the run has spent it.
	stored, _, _ = r.cell.Traces.GetResearch(ctx, "acme-logistics", rv.ID)
	stored.Long.LimitUSD, stored.Long.Checkpoint = 0, nil
	if err := r.cell.Traces.PutResearch(ctx, stored); err != nil {
		t.Fatal(err)
	}
	if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
		t.Fatal(err)
	}
	code, out = r.do(t, "GET", "/v1/research/"+rv.ID+"?space=finance", "")
	if err := json.Unmarshal([]byte(out), &rv); err != nil || rv.State != ask.StateDone || rv.Stopped != "cap" || len(rv.Sections) != 1 {
		t.Fatalf("stopped at its ceiling with what it has: %d %s", code, out)
	}
	if code, out = r.do(t, "POST", "/v1/research/"+rv.ID+"/stop?space=finance", ""); code != http.StatusBadRequest {
		t.Fatalf("stopped a finished report: %d %s", code, out)
	}
}
