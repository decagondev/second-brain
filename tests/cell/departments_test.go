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

const departmentsManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask, sop-steps]
spaces:
  ops:
    template: operations
    members: [group:ops]
    stewards: [group:ops-leads]
    scope: { docs: ["/Ops/Dept/**"] }
  returns:
    parent: ops
    members: [group:returns]
    stewards: [group:ops-leads]
    agents: [ask, sop-steps]
    glossary: { RMA: return merchandise authorisation }
    scope: { docs: ["/Ops/Returns/**"] }
  fleet:
    parent: ops
    members: [group:fleet]
    stewards: [group:fleet]
    agents: [ask, sop-steps]
    glossary: { MOT: vehicle inspection }
    scope: { docs: ["/Ops/Fleet/**"] }
  it:
    members: [group:ops, group:fleet]
    stewards: [group:fleet]
    requests: { categories: [Access, Laptop], due_days: 2 }
    scope: { docs: ["/IT/**"] }
`

// SB-4501 to SB-4503 through a running cell: a consulting branch ask consults only the teams
// the asker is in and counts the rest; Hand off drafts a request the asker sends with the
// cites they chose, and the receiving steward sees only what they can open; the leader's
// week waits for the department's steward, cited lines then counts.
func TestDepartmentsWorkTogetherThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	for path, text := range map[string]string{
		"Ops/Dept/week.md":    "This week the operations department decided to open the second dock on Saturdays.",
		"Ops/Returns/rma.md":  "An RMA is approved by the returns desk within two working days.",
		"Ops/Fleet/mot.md":    "Every van has its MOT in March.",
		"IT/access/how-to.md": "Access to tools is granted by IT.",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for dir, acl := range map[string]string{"Ops/Dept": "read domain:acme.example\n", "Ops/Returns": "read sam@acme.example\n",
		"Ops/Fleet": "read lee@acme.example\n", "IT/access": "read domain:acme.example\n"} {
		if err := os.WriteFile(filepath.Join(share, dir, ".acl"), []byte(acl), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, departmentsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	signInAs := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, ids := range map[string][]string{"ops": {sam, lee}, "ops-leads": {sam}, "returns": {sam}, "fleet": {lee}} {
		var ms []string
		for _, id := range ids {
			ms = append(ms, `{"value":"`+id+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	ctx := context.Background()
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	signInAs(sam)

	// SB-4501: sam is in returns, not fleet: returns is consulted, fleet only counted.
	code, out := r.do(t, "POST", "/v1/ask?space=ops&branch=true", `{"question":"How is an RMA approved, and when is the MOT?","consult":true}`)
	var resp ask.Response
	if err := json.Unmarshal([]byte(out), &resp); err != nil || code != 200 {
		t.Fatalf("consult: %d %s", code, out)
	}
	if len(resp.Departments) != 1 || resp.Departments[0].Space != "returns" || resp.Departments[0].Agent != "sop-steps" ||
		len(resp.Departments[0].Claims) == 0 || !strings.Contains(resp.Departments[0].Claims[0].Cites[0].Cite, "Returns/rma") {
		t.Fatalf("sections: %s", out)
	}
	if len(resp.NotConsulted) != 1 || resp.NotConsulted[0].Why != ask.WhyNotMember || resp.NotConsulted[0].Count != 1 ||
		strings.Contains(out, "fleet") || strings.Contains(out, "Fleet") {
		t.Fatalf("not consulted: %s", out)
	}
	if code, out := r.do(t, "POST", "/v1/ask?space=ops", `{"question":"How is an RMA approved?","consult":true}`); code != http.StatusBadRequest {
		t.Fatalf("consult without a branch: %d %s", code, out)
	}

	// SB-4502: a question naming IT's Access category offers Hand off; the draft carries the
	// cites sam can open, and nothing is sent until sam sends it.
	code, out = r.do(t, "POST", "/v1/ask?space=returns", `{"question":"I need access to the RMA approval tool"}`)
	if err := json.Unmarshal([]byte(out), &resp); err != nil || code != 200 || resp.Handoff == nil || resp.Handoff.Space != "it" || resp.Handoff.Category != "Access" {
		t.Fatalf("hand off offer: %d %s", code, out)
	}
	code, out = r.do(t, "POST", "/v1/spaces/it/handoff", `{"trace":"`+resp.Trace+`","question":"I need access to the RMA approval tool"}`)
	var d ask.HandoffDraft
	if err := json.Unmarshal([]byte(out), &d); err != nil || code != 200 || d.Category != "Access" || len(d.Cites) != 1 || !strings.Contains(d.Cites[0].Cite, "Returns/rma") {
		t.Fatalf("draft: %d %s", code, out)
	}
	signInAs(lee)
	if code, out := r.do(t, "GET", "/v1/spaces/it/questions", ""); code != 200 || strings.Contains(out, "RMA") {
		t.Fatalf("a draft reached the stewards: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/spaces/it/handoff", `{"trace":"`+resp.Trace+`","question":"x"}`); code == 200 {
		t.Fatalf("lee drafted from sam's answer: %s", out)
	}
	fleet := searchCite(t, r, "/v1/search?q=MOT&space=fleet")
	signInAs(sam)
	if code, out := r.do(t, "POST", "/v1/spaces/it/requests", `{"category":"Access","text":"x","cites":["`+fleet+`"]}`); code != http.StatusBadRequest {
		t.Fatalf("sam shared a cite they can't open: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/spaces/it/requests", `{"category":"Access","text":"`+d.Text+`","cites":["`+d.Cites[0].Cite+`"]}`); code != 200 {
		t.Fatalf("send: %d %s", code, out)
	}
	signInAs(lee)
	code, out = r.do(t, "GET", "/v1/spaces/it/questions", "")
	var q struct{ Items []ask.StewardItem }
	if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 || len(q.Items) != 1 || q.Items[0].SharedHidden != 1 || len(q.Items[0].Shared) != 0 ||
		strings.Contains(out, "Returns/rma") || strings.Contains(out, "rma.md") {
		t.Fatalf("lee's queue: %d %s", code, out)
	}

	// SB-4503: the leader's week waits in ops for sam, cited lines then counts per team.
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	signInAs(sam)
	code, out = r.do(t, "GET", "/v1/spaces/ops/questions", "")
	if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
		t.Fatalf("ops queue: %d %s", code, out)
	}
	var week *ask.ThreadView
	for _, it := range q.Items {
		if it.Thread != nil && it.Thread.Kind == ask.DraftLeaderWeek {
			week = it.Thread
		}
	}
	if week == nil || week.Agent != ask.AgentLeaderWeek || !strings.Contains(week.Text, "By team (counts only):\n- fleet: 0 open requests (0 overdue), 0 open questions, 0 waiting on another space, fewer than 5 sent in 12 weeks\n- returns:") ||
		strings.Contains(week.Text, "MOT") {
		t.Fatalf("leader's week: %s", out)
	}
}

// searchCite is the first cite a search finds.
func searchCite(t *testing.T, r running, path string) string {
	t.Helper()
	code, out := r.do(t, "GET", path, "")
	var res ask.SearchResponse
	if err := json.Unmarshal([]byte(out), &res); err != nil || code != 200 || len(res.Results) == 0 {
		t.Fatalf("%s: %d %s", path, code, out)
	}
	return res.Results[0].Cite
}
