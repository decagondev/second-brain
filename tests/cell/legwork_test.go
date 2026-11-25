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
}
