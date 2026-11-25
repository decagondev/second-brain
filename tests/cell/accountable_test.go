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

const accountableManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: { ask: {}, goal-check-in: { monthly_usd: 1 } }
spaces:
  support:
    template: support
    label: internal
    members: [group:support]
    stewards: [group:support-leads]
    scope: { docs: ["/Support/**"] }
  it:
    members: [group:it]
    stewards: [group:it]
    scope: { docs: ["/IT/**"] }
`

// SB-4701 to SB-4703 through a running cell: an agent's value counts its drafts as the
// audit log has them; Pause here stops it in one space at the next trigger, tells its owner
// and is audited, and resume brings it back; a cap pauses its runs and tells the owner.
func TestEveryAgentAccountableThroughARunningCell(t *testing.T) {
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
	write("Support/.acl", "read domain:acme.example\n")
	write("Support/queue-runbook.md", "Runbook: when the payments queue stops draining, restart the payments queue consumer.")
	write("Support/inc-1.md", "Labels: sev-1\n\nThe payments queue stopped draining at 09:12.")
	write("Support/inc-2.md", "Labels: sev-1\n\nThe payments queue stopped draining again at 14:02.")
	write("IT/.acl", "read domain:acme.example\n")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, accountableManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	got := &agentsTold{}
	r.cell.Send = got
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, members := range map[string][]string{"support": {sam, lee}, "support-leads": {sam}, "brain-agents": {lee}, "sb-operators": {sam}, "it": {lee}} {
		var ms []string
		for _, m := range members {
			ms = append(ms, `{"value":"`+m+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group %s: %d %s", name, code, body)
		}
	}
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	if _, err := r.cell.NotifyJob(got).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	queue := func() []ask.StewardItem {
		t.Helper()
		code, out := r.do(t, "GET", "/v1/spaces/support/questions", "")
		var q struct{ Items []ask.StewardItem }
		if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
			t.Fatalf("queue: %d %s", code, out)
		}
		return q.Items
	}
	var drafts []ask.StewardItem
	for _, it := range queue() {
		if it.Thread != nil && it.Thread.Kind == ask.DraftIncident {
			drafts = append(drafts, it)
		}
	}
	if len(drafts) != 2 {
		t.Fatalf("incident drafts: %+v", drafts)
	}
	// One published as it is, one discarded.
	if code, out := r.do(t, "POST", "/v1/spaces/support/questions/"+drafts[0].ID+"/reply", `{"text":`+jsonString(drafts[0].Thread.Text)+`,"cites":[]}`); code != 200 {
		t.Fatalf("publish: %d %s", code, out)
	}
	if code, out := r.do(t, "DELETE", "/v1/spaces/support/questions/"+drafts[1].ID, ""); code != 200 && code != 204 {
		t.Fatalf("discard: %d %s", code, out)
	}

	// SB-4701: sam, an operator, sees every agent; lee, its owner, sees incident-brief's.
	var v ask.ValueReport
	code, out := r.do(t, "GET", "/v1/admin/agents/value", "")
	if err := json.Unmarshal([]byte(out), &v); err != nil || code != 200 || v.Weeks != 12 {
		t.Fatalf("admin value: %d %s", code, out)
	}
	row := func(v ask.ValueReport, agent, space string) *ask.ValueRow {
		for i := range v.Rows {
			if v.Rows[i].Agent == agent && v.Rows[i].Space == space {
				return &v.Rows[i]
			}
		}
		return nil
	}
	if x := row(v, ask.AgentIncident, "support"); x == nil || x.Drafts != 2 || x.Published != 1 || x.PublishedEdited != 0 || x.Discarded != 1 || x.MedianHours == nil {
		t.Fatalf("incident-brief's value: %s", out)
	}
	if strings.Contains(out, "payments queue") {
		t.Fatalf("value carries a draft's text: %s", out)
	}
	audit := r.auditRows(t)
	if n := strings.Count(audit, `"state":"drafted"`); n != 2 || strings.Count(audit, `"state":"discarded"`) != 1 {
		t.Fatalf("the audit log has %d drafted rows: %s", n, audit)
	}
	code, out = r.do(t, "GET", "/v1/agents/value", "")
	if err := json.Unmarshal([]byte(out), &v); err != nil || code != 200 || len(v.Rows) != 0 {
		t.Fatalf("sam owns no agent, and sees no value: %d %s", code, out)
	}
	r.idp.User = lee
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatal("lee's sign-in")
	}
	code, out = r.do(t, "GET", "/v1/agents/value", "")
	if err := json.Unmarshal([]byte(out), &v); err != nil || code != 200 || row(v, ask.AgentIncident, "support") == nil {
		t.Fatalf("the owner's value: %d %s", code, out)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/agents/value", ""); code != http.StatusForbidden {
		t.Fatalf("a non-operator read every agent's value: %d", code)
	}

	// SB-4702: lee owns incident-brief and pauses it in support without being a steward there.
	if code, out := r.do(t, "POST", "/v1/spaces/support/agents/incident-brief/pause", `{"reason":"it keeps naming the wrong runbook"}`); code != 200 {
		t.Fatalf("the owner's pause: %d %s", code, out)
	}
	if code, _ := r.do(t, "POST", "/v1/spaces/support/agents/case-brief/pause", `{"reason":"no"}`); code != http.StatusForbidden {
		t.Fatalf("lee paused an agent they neither own nor steward: %d", code)
	}
	if !strings.Contains(strings.Join(got.got, "\n"), "agent "+lee+": incident-brief was paused in support: it keeps naming the wrong runbook") {
		t.Fatalf("the owner wasn't told: %v", got.got)
	}
	write("Support/inc-3.md", "Labels: sev-1\n\nThe payments queue stopped a third time.")
	if fails := svc.SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	if _, err := r.cell.NotifyJob(got).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.idp.User = sam
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatal("sam's sign-in")
	}
	n := 0
	for _, it := range queue() {
		if it.Thread != nil && it.Thread.Kind == ask.DraftIncident {
			n++
		}
	}
	if n != 1 { // the published one is answered and listed; the paused run drafted nothing
		t.Fatalf("after the pause, %d incident drafts", n)
	}
	code, out = r.do(t, "GET", "/v1/spaces/support/agents", "")
	if code != 200 || !strings.Contains(out, `"paused":["incident-brief"]`) {
		t.Fatalf("the space's agents: %d %s", code, out)
	}
	if !strings.Contains(r.auditRows(t), `"subject":"agent:incident-brief@support"`) {
		t.Fatal("the pause isn't audited")
	}
	if code, out := r.do(t, "POST", "/v1/spaces/support/agents/incident-brief/resume", ""); code != 200 {
		t.Fatalf("resume: %d %s", code, out)
	}
	write("Support/inc-4.md", "Labels: sev-1\n\nThe payments queue stopped a fourth time.")
	if fails := svc.SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	if _, err := r.cell.NotifyJob(got).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	n = 0
	for _, it := range queue() {
		if it.Thread != nil && it.Thread.Kind == ask.DraftIncident {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("after resuming, %d incident drafts", n)
	}

	// SB-4703: the cap is in the value's spend.
	code, out = r.do(t, "GET", "/v1/admin/agents/value", "")
	if err := json.Unmarshal([]byte(out), &v); err != nil || code != 200 {
		t.Fatalf("value: %d %s", code, out)
	}
	capped := false
	for _, s := range v.Spend {
		capped = capped || (s.Agent == ask.AgentCheckin && s.MonthlyUSD != nil && *s.MonthlyUSD == 1)
	}
	if !capped {
		t.Fatalf("goal-check-in's cap: %s", out)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// auditRows is the cell's audit log as JSON lines.
func (r running) auditRows(t *testing.T) string {
	t.Helper()
	rows, err := r.cell.Audit.Store.Range(context.Background(), 1, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, row := range rows {
		j, _ := json.Marshal(row)
		b.Write(j)
		b.WriteString("\n")
	}
	return b.String()
}
