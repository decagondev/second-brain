package cell

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

const planningManifest = `tenant: acme-logistics
channel: testing
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  customer:
    template: support
    label: internal
    members: [group:customer]
    stewards: [group:customer-leads]
    glossary: { refunds: "money back to a customer" }
    scope: { docs: ["/Customer/**"] }
  support:
    template: support
    label: internal
    parent: customer
    members: [group:customer]
    stewards: [group:support-leads]
    scope: { docs: ["/Customer/Support/**"] }
  helpdesk:
    template: support
    label: internal
    parent: customer
    members: [group:customer]
    stewards: [group:helpdesk-leads]
    scope: { docs: ["/Customer/Helpdesk/**"] }
`

// SB-4805 to SB-4807 through a running cell: each team's planning draft proposes its
// unfinished goals, cited, and the department's steward sees them rolled up, a goal they
// can't open counted, with the overlap on a glossary term flagged. A manager's 1:1 brief
// for a report holds the report's own goal and is the same text in the report's queue, and
// the report's line reaches both. The reorg dry run shows what is under way, for operators.
func TestPlanningOneOnOneAndTreePlanThroughARunningCell(t *testing.T) {
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
	now := time.Now().UTC()
	quarter := fmt.Sprintf("%d-Q%d", now.Year(), (int(now.Month())-1)/3+1)
	goal := func(owner, status string) string {
		return "---\ntype: goal\nowner: " + owner + "@acme.example\nquarter: " + quarter + "\nstatus: " + status + "\n---\n\nWhat we do this quarter.\n"
	}
	for _, d := range []string{"Customer", "Customer/Support", "Customer/Helpdesk"} {
		write(d+"/.acl", "read domain:acme.example\nowner sam@acme.example\n")
	}
	write("Customer/Support/refunds-backlog.md", goal("lee", "at risk"))
	write("Customer/Support/onboarding-guide.md", goal("sam", "done"))
	write("Customer/Helpdesk/refunds-speed.md", goal("kim", "on track"))
	write("Customer/Helpdesk/vendor-switch.md", goal("kim", "on track"))
	write("Customer/Helpdesk/vendor-switch.md.acl", "read kim@acme.example\n")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, planningManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ids := map[string]string{}
	for _, n := range []string{"kim", "lee", "sam"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	as := func(name string) {
		t.Helper()
		r.idp.User = ids[name]
		if code, body := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in as %s: %d %s", name, code, body)
		}
	}
	scim := func(method, path, body string) {
		t.Helper()
		code, out := r.do(t, method, path, body, "Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
		if code != http.StatusCreated && code != http.StatusOK {
			t.Fatalf("SCIM %s %s: %d %s", method, path, code, out)
		}
	}
	for name, members := range map[string][]string{"customer": {"sam", "lee", "kim"}, "customer-leads": {"sam"}, "support-leads": {"sam"},
		"helpdesk-leads": {"kim"}, "sb-operators": {"sam"}} {
		var ms []string
		for _, m := range members {
			ms = append(ms, `{"value":"`+ids[m]+`"}`)
		}
		scim("POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`)
	}
	scim("PATCH", "/scim/v2/Users/"+ids["lee"], `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"add","path":"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager","value":{"value":"`+ids["sam"]+`"}}]}`)
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	start := func(space, body string) ask.RhythmView {
		t.Helper()
		code, out := r.do(t, "POST", "/v1/spaces/"+space+"/rhythms", body)
		var v ask.RhythmView
		if err := json.Unmarshal([]byte(out), &v); err != nil || code != 200 {
			t.Fatalf("start in %s: %d %s", space, code, out)
		}
		return v
	}
	queue := func(who string) string {
		t.Helper()
		code, out := r.do(t, "GET", "/v1/spaces/"+url.PathEscape(ask.PersonKey(ids[who]))+"/questions", "")
		if code != 200 {
			t.Fatalf("%s's queue: %d %s", who, code, out)
		}
		return out
	}

	// SB-4805: each team's stewards start planning; the drafts wait in their space's queue.
	as("sam")
	start("support", `{"rhythm":"quarterly-planning"}`)
	code, out := r.do(t, "GET", "/v1/spaces/support/questions", "")
	if code != 200 || !strings.Contains(out, "refunds-backlog (at risk, at risk now) [docs:") || strings.Contains(out, "onboarding-guide") {
		t.Fatalf("support's planning draft: %d %s", code, out)
	}
	as("kim")
	start("helpdesk", `{"rhythm":"quarterly-planning"}`)
	as("sam")
	var plan ask.PlanningResponse
	code, out = r.do(t, "GET", "/v1/spaces/customer/planning", "")
	if err := json.Unmarshal([]byte(out), &plan); err != nil || code != 200 || len(plan.Teams) != 2 {
		t.Fatalf("planning roll-up: %d %s", code, out)
	}
	for _, team := range plan.Teams {
		if team.Space == "helpdesk" && (team.Hidden != 1 || len(team.Proposed) != 1 || team.Proposed[0].Title != "refunds-speed") {
			t.Fatalf("helpdesk as the department's steward sees it (LAW-01): %+v", team)
		}
	}
	if len(plan.Overlaps) != 1 || plan.Overlaps[0].Term != "refunds" {
		t.Fatalf("the overlap on refunds isn't flagged: %+v", plan.Overlaps)
	}
	as("lee")
	if code, out := r.do(t, "GET", "/v1/spaces/customer/planning", ""); code == 200 {
		t.Fatalf("a member who isn't a steward read the roll-up: %s", out)
	}

	// SB-4807: kim isn't lee's manager; sam is, and both get the same brief.
	as("kim")
	if code, out := r.do(t, "POST", "/v1/spaces/support/rhythms", `{"rhythm":"one-on-one","subject":"lee@acme.example"}`); code == 200 {
		t.Fatalf("someone who isn't lee's manager started their 1:1 prep: %s", out)
	}
	as("sam")
	oo := start("support", `{"rhythm":"one-on-one","subject":"lee@acme.example"}`)
	brief := func(who string) string {
		t.Helper()
		as(who)
		var qs struct{ Items []ask.StewardItem }
		out := queue(who)
		if err := json.Unmarshal([]byte(out), &qs); err != nil {
			t.Fatal(err)
		}
		for _, q := range qs.Items {
			if q.Rhythm != nil && q.Rhythm.Run == oo.ID && q.Thread != nil {
				return q.Thread.Text
			}
		}
		t.Fatalf("no 1:1 brief in %s's queue: %s", who, out)
		return ""
	}
	mgr := brief("sam")
	if !strings.Contains(mgr, "Goal refunds-backlog: at risk [docs:") || strings.Contains(mgr, "refunds-speed") || strings.Contains(mgr, "onboarding-guide") {
		t.Fatalf("the 1:1 brief: %q", mgr)
	}
	if rep := brief("lee"); rep != mgr {
		t.Fatalf("lee's copy differs from sam's brief (LAW-30):\n%q\n%q", rep, mgr)
	}
	if code, out := r.do(t, "POST", "/v1/rhythms/"+oo.ID+"/line", `{"text":"I want to talk about the refunds tooling"}`); code != 200 {
		t.Fatalf("lee's line: %d %s", code, out)
	}
	if got := brief("sam"); !strings.HasSuffix(got, "- From them: I want to talk about the refunds tooling") || got != brief("lee") {
		t.Fatalf("lee's line isn't in both copies: %q", got)
	}

	// SB-4806: the dry run of making helpdesk a department of its own, with what is under way.
	as("sam")
	var tp struct {
		Plan struct {
			Spaces []struct {
				Space        string `json:"space"`
				ParentBefore string `json:"parent_before"`
			} `json:"spaces"`
		} `json:"plan"`
		Manifest string `json:"manifest"`
		InFlight map[string]struct {
			Rhythms int `json:"rhythms"`
		} `json:"in_flight"`
	}
	code, out = r.do(t, "GET", "/v1/admin/tree/plan?op=move&space=helpdesk&to=", "")
	if err := json.Unmarshal([]byte(out), &tp); err != nil || code != 200 || tp.InFlight["helpdesk"].Rhythms != 1 || !strings.Contains(tp.Manifest, "helpdesk:") {
		t.Fatalf("tree plan: %d %s", code, out)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/tree/plan?op=merge&space=helpdesk&to=nope", ""); code != http.StatusBadRequest {
		t.Fatalf("an unknown space: %d", code)
	}
	as("lee")
	if code, _ := r.do(t, "GET", "/v1/admin/tree/plan?op=move&space=helpdesk&to=", ""); code != http.StatusForbidden {
		t.Fatalf("a non-operator read the tree plan: %d", code)
	}
}
