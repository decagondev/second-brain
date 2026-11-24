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

const requestsManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
notes: true
spaces:
  it:
    members: [group:it]
    stewards: [group:it-leads]
    requests: { categories: [Access, Laptop], due_days: 2 }
    budget_usd_month: 10
    scope: { docs: ["/IT/Dept/**"] }
  helpdesk:
    parent: it
    members: [group:helpdesk]
    stewards: [group:it-leads]
    updates: true
    budget_usd_month: 5
    scope: { docs: ["/IT/Helpdesk/**", "/notes/helpdesk/**"] }
`

// SB-4104, SB-4105 through a running cell: a member's request reaches the stewards with its
// category and due date; a team that spent its budget pauses its weekly update and still
// answers, its stewards are told, and the admin console shows the spend rolled up.
func TestRequestsAndBudgetsThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	for path, text := range map[string]string{
		"IT/Dept/laptops.md":     "Laptops are replaced every four years.",
		"IT/Helpdesk/week.md":    "Last week the helpdesk decided to reset passwords by chat. Changed: the zebra runbook.",
		"IT/Helpdesk/runbook.md": "The zebra runbook resets passwords.",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(share, filepath.Dir(path), ".acl"), []byte("read domain:acme.example\n"), 0o644)
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, requestsManifest, nil)
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
	for name, ids := range map[string][]string{"it": {sam, lee}, "helpdesk": {sam, lee}, "it-leads": {sam}} {
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

	// SB-4104: lee asks IT for a laptop, in a category IT takes.
	signInAs(lee)
	if code, out := r.do(t, "GET", "/v1/spaces/it/requests", ""); code != 200 || !strings.Contains(out, `"categories":["Access","Laptop"]`) || !strings.Contains(out, `"due_days":2`) {
		t.Fatalf("form: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/spaces/it/requests", `{"category":"Desk","text":"A desk"}`); code != http.StatusBadRequest {
		t.Fatalf("a category IT doesn't take: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/spaces/helpdesk/requests", `{"category":"Laptop","text":"A laptop"}`); code != http.StatusBadRequest {
		t.Fatalf("a space without requests: %d %s", code, out)
	}
	code, out := r.do(t, "POST", "/v1/spaces/it/requests", `{"category":"laptop","text":"My laptop is broken"}`)
	if code != 200 || !strings.Contains(out, `"category":"Laptop"`) || !strings.Contains(out, `"due":"`) {
		t.Fatalf("request: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/questions", ""); code != 200 || !strings.Contains(out, `"request":{"category":"Laptop","due_days":2`) {
		t.Fatalf("lee's requests: %d %s", code, out)
	}
	signInAs(sam)
	code, out = r.do(t, "GET", "/v1/spaces/it/questions", "")
	var q struct{ Items []ask.StewardItem }
	_ = json.Unmarshal([]byte(out), &q)
	if code != 200 || len(q.Items) != 1 || q.Items[0].Request == nil || q.Items[0].Request.Category != "Laptop" || q.Items[0].Overdue || strings.Contains(out, lee) {
		t.Fatalf("sam's queue: %d %s", code, out)
	}

	// SB-4105: the helpdesk spent past its $5; IT's branch is at $6 of $10.
	month := time.Now().UTC().Format("2006-01")
	if err := r.cell.Usage.Spend().AddSpace(ctx, "acme-logistics", "spc_helpdesk", month, 6); err != nil {
		t.Fatal(err)
	}
	s := &agentsTold{}
	if _, err := r.cell.NotifyJob(s).Run(ctx); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(s.got, "\n")
	if !strings.Contains(all, "questions "+sam+": This week's update didn't run: the space's monthly budget is spent") ||
		!strings.Contains(all, "questions "+sam+": The helpdesk space has spent its monthly budget") || strings.Contains(all, "The it space") || strings.Contains(all, lee+": The") {
		t.Fatalf("told: %v", s.got)
	}
	code, out = r.do(t, "GET", "/v1/spaces/helpdesk/questions", "")
	if code != 200 || strings.Contains(out, `"kind":"update"`) {
		t.Fatalf("an update ran over budget: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/ask?space=helpdesk", `{"question":"What does the zebra runbook do?"}`); code != 200 || !strings.Contains(out, "runbook") {
		t.Fatalf("asks carry on over a space's budget: %d %s", code, out)
	}
	s.got = nil
	if _, err := r.cell.NotifyJob(s).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(s.got, "\n"), "budget") {
		t.Fatalf("told twice: %v", s.got)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/costs/spaces?month="+month, ""); code != http.StatusForbidden {
		t.Fatalf("a steward read the spend: %d", code)
	}
	r.operator(t)
	code, out = r.do(t, "GET", "/v1/admin/costs/spaces?month="+month, "")
	var sp struct {
		Spaces []struct {
			Space          string
			OwnUSD         float64 `json:"own_usd"`
			SpentUSD       float64 `json:"spent_usd"`
			BudgetUSDMonth int     `json:"budget_usd_month"`
		}
	}
	if err := json.Unmarshal([]byte(out), &sp); err != nil || code != 200 || len(sp.Spaces) != 2 {
		t.Fatalf("spend: %d %s", code, out)
	}
	for _, x := range sp.Spaces {
		if x.Space == "it" && (x.OwnUSD != 0 || x.SpentUSD != 6 || x.BudgetUSDMonth != 10) || x.Space == "helpdesk" && (x.OwnUSD != 6 || x.SpentUSD != 6) {
			t.Fatalf("spend: %s", out)
		}
	}
}
