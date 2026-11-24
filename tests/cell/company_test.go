package cell

import (
	"context"
	"encoding/json"
	"fmt"
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

const companyManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
notes: true
spaces:
  finance:
    template: finance
    label: confidential
    members: [group:finance]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/Dept/**", "/notes/finance/**"] }
  ap:
    parent: finance
    members: [group:ap]
    stewards: [group:finance-leads]
    updates: true
    scope: { docs: ["/Finance/AP/**", "/notes/ap/**"] }
`

// SB-4101 to SB-4103 through a running cell: a branch's goals nested and checked as the
// asker, a team's weekly update drafted, published under notes/ap/updates/ and listed on
// its department, and a policy everyone is asked to read once per version.
func TestRunTheCompanyThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	files := map[string]string{
		"Finance/Dept/goals/faster.md":   "---\ntype: goal\nowner: sam@acme.example\nquarter: 2027-Q1\nstatus: on track\n---\n# Faster finance\n\nThe zebra close takes three days.",
		"Finance/AP/goals/close.md":      "---\ntype: goal\nowner: sam@acme.example\nquarter: 2027-Q1\nstatus: at risk\nparent_goal: faster\n---\n# Close faster\n\nInvoices close in two days.",
		"Finance/Dept/secret/goal.md":    "---\ntype: goal\nquarter: 2027-Q1\nstatus: done\n---\n# Secret goal\n",
		"Finance/Dept/policies/leave.md": "# Leave policy\n\nBook leave two weeks ahead.",
		"Finance/AP/week.md":             "Last week the AP team decided invoices close in two days. Changed: the close checklist.",
	}
	for path, text := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		acl := "read domain:acme.example\n"
		if strings.Contains(path, "secret") {
			acl = "read nobody@acme.example\n"
		}
		_ = os.WriteFile(filepath.Join(share, filepath.Dir(path), ".acl"), []byte(acl), 0o644)
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, companyManifest, nil)
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
	for name, ids := range map[string][]string{"finance": {sam, lee}, "ap": {sam}, "finance-leads": {sam}} {
		var ms []string
		for _, id := range ids {
			ms = append(ms, `{"value":"`+id+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	sync := func() {
		t.Helper()
		if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
			t.Fatal(fails)
		}
	}
	sync()
	signInAs(sam)

	// SB-4101: the department's goal with its team's under it; the one sam can't open counted.
	code, out := r.do(t, "GET", "/v1/spaces/finance/goals", "")
	var g ask.GoalsResponse
	if err := json.Unmarshal([]byte(out), &g); err != nil || code != 200 || g.Hidden != 1 || len(g.Quarters) != 1 || g.Quarters[0].Quarter != "2027-Q1" {
		t.Fatalf("sam's goals: %d %s", code, out)
	}
	if top := g.Quarters[0].Goals; len(top) != 1 || top[0].Status != "on track" || len(top[0].Goals) != 1 || top[0].Goals[0].Space != "ap" || top[0].Goals[0].Status != "at risk" {
		t.Fatalf("nesting: %s", out)
	}
	if strings.Contains(out, "Secret") {
		t.Fatalf("a goal sam can't open is named: %s", out)
	}

	// SB-4103: sam marks the leave policy; both members are told once.
	if code, out := r.do(t, "POST", "/v1/spaces/finance/policies", `{"doc":"docs:Finance/Dept/policies/leave.md"}`); code != 200 {
		t.Fatalf("mark: %d %s", code, out)
	}
	s := &agentsTold{}
	if _, err := r.cell.NotifyJob(s).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	policyTold := 0
	for _, m := range s.got {
		if strings.HasPrefix(m, "policy ") {
			policyTold++
		}
	}
	if policyTold != 2 {
		t.Fatalf("told: %v", s.got)
	}

	// SB-4102: ap's weekly update waits for sam; sam replies and publishes it as a note.
	code, out = r.do(t, "GET", "/v1/spaces/ap/questions", "")
	var q struct{ Items []ask.StewardItem }
	_ = json.Unmarshal([]byte(out), &q)
	id := ""
	for _, it := range q.Items {
		if it.Thread != nil && it.Thread.Kind == "update" && it.Thread.Agent == "update" && it.Thread.Text != "" {
			id = it.ID
		}
	}
	if code != 200 || id == "" {
		t.Fatalf("no update draft: %d %s", code, out)
	}
	base := "/v1/spaces/ap/questions/" + id
	if code, out := r.do(t, "POST", base+"/reply", `{"text":"AP closes invoices in two days now.","cites":[]}`); code != 200 {
		t.Fatalf("reply: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", base+"/note", `{"title":"AP update","confirm":0}`); code != 200 || !strings.Contains(out, `"state":"Ran"`) {
		t.Fatalf("publish: %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(share, "notes", "ap", "updates", id+".md")); err != nil {
		t.Fatalf("the update isn't under notes/ap/updates/: %v", err)
	}
	sync()
	code, out = r.do(t, "GET", "/v1/spaces/finance/updates", "")
	if code != 200 || !strings.Contains(out, `"space":"ap"`) || !strings.Contains(out, "updates/"+id+".md") {
		t.Fatalf("sam's team updates: %d %s", code, out)
	}

	// SB-4103: lee reads it; the steward's list names only who hasn't.
	signInAs(lee)
	code, out = r.do(t, "GET", "/v1/spaces/finance/goals", "")
	if code != 200 || strings.Contains(out, "AP/goals/close") || !strings.Contains(out, `"title":"faster"`) {
		t.Fatalf("lee's goals: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/spaces/finance/updates", "")
	if code != 200 || !strings.Contains(out, `"updates":[]`) || !strings.Contains(out, `"hidden":1`) {
		t.Fatalf("lee sees ap's update: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/spaces/finance/policies", "")
	var ps struct{ Policies []ask.PolicyItem }
	if err := json.Unmarshal([]byte(out), &ps); err != nil || code != 200 || len(ps.Policies) != 1 || ps.Policies[0].Read {
		t.Fatalf("lee's policies: %d %s", code, out)
	}
	v := ps.Policies[0].Version
	if code, out := r.do(t, "POST", "/v1/spaces/finance/policies/read", fmt.Sprintf(`{"doc":"docs:Finance/Dept/policies/leave.md","version":%d}`, v+1)); code != http.StatusBadRequest {
		t.Fatalf("an old version was read: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/spaces/finance/policies/read", fmt.Sprintf(`{"doc":"docs:Finance/Dept/policies/leave.md","version":%d}`, v)); code != 200 {
		t.Fatalf("read: %d %s", code, out)
	}
	if code, _ := r.do(t, "GET", "/v1/spaces/finance/policies/status", ""); code != http.StatusForbidden {
		t.Fatalf("a member saw who has read: %d", code)
	}
	signInAs(sam)
	code, out = r.do(t, "GET", "/v1/spaces/finance/policies/status", "")
	var st struct{ Policies []ask.PolicyStatus }
	if err := json.Unmarshal([]byte(out), &st); err != nil || code != 200 || len(st.Policies) != 1 || st.Policies[0].ReadCount != 1 || len(st.Policies[0].NotRead) != 1 || !strings.HasPrefix(st.Policies[0].NotRead[0], "sam") {
		t.Fatalf("status: %d %s", code, out)
	}
	// Told once: the next run tells nobody again.
	s.got = nil
	if _, err := r.cell.NotifyJob(s).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, m := range s.got {
		if strings.HasPrefix(m, "policy ") {
			t.Fatalf("told twice: %v", s.got)
		}
	}
}
