package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/proposals"
)

const rolloutManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask, close-checklist]
spaces:
  finance:
    template: finance
    label: confidential
    members: [group:finance]
    stewards: [group:finance-leads]
    agents: [ask]
    scope: { docs: ["/Finance/**"] }
  ap:
    parent: finance
    members: [group:ap]
    stewards: [group:ap-leads]
    agents: [ask, close-checklist]
    skills: { month-end: { instruction: "Write it as a checklist.", cases: { "When is AP close?": "docs:Finance/AP/close.md", "Who signs invoices?": "docs:Finance/AP/sign.md", "When are vendors paid?": "docs:Finance/AP/pay.md" } } }
    scope: { docs: ["/Finance/AP/**"] }
`

// SB-5104 to SB-5106 through a running cell. An agent's history lists the approved proposals
// that changed it with who proposed and approved each, and Restore is an open proposal, not a
// change. A department steward in the pilot group pauses a team's agent and sees the branch;
// one outside it can't yet. An operator holds and releases the pilot feature, audited.
func TestRollOutWithCareThroughARunningCell(t *testing.T) {
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, rolloutManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + t.TempDir()}}
		return cfg
	})
	ctx := context.Background()
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, members := range map[string][]string{"finance": {sam, lee}, "finance-leads": {sam, lee}, "pilot": {sam}, "sb-operators": {sam}} {
		var ms []string
		for _, x := range members {
			ms = append(ms, `{"value":"`+x+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	as := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}

	// SB-5104: a version from an approved proposal before close-checklist was listed in ap.
	cur := r.cell.Manifest()
	old := cur
	old.Spaces = map[string]manifest.Space{}
	for k, v := range cur.Spaces {
		old.Spaces[k] = v
	}
	ap := old.Spaces["ap"]
	ap.Agents = []string{"ask"}
	old.Spaces["ap"] = ap
	if err := r.cell.Proposals.Put(ctx, proposals.Proposal{ID: "prop_old", Space: "ap", Requester: lee, DecidedBy: sam, State: proposals.Approved,
		At: time.Now().Add(-48 * time.Hour), Manifest: string(manifest.Format(old))}); err != nil {
		t.Fatal(err)
	}
	as(sam)
	code, out := r.do(t, "GET", "/v1/admin/agents/close-checklist/history?from=1&to=2", "")
	var h struct {
		Versions []struct {
			N                      int
			Proposal, By, Approved string
			Current                bool
		}
		Changes []string
	}
	if err := json.Unmarshal([]byte(out), &h); err != nil || code != 200 {
		t.Fatalf("history: %d %s", code, out)
	}
	if len(h.Versions) != 2 || h.Versions[0].Proposal != "prop_old" || h.Versions[0].By != lee || h.Versions[0].Approved != sam || !h.Versions[1].Current ||
		len(h.Changes) != 1 || h.Changes[0] != "+ spaces: ap" {
		t.Fatalf("history: %s", out)
	}
	code, out = r.do(t, "POST", "/v1/admin/agents/close-checklist/history/1/restore", "")
	var p struct {
		ID   string
		Diff []string
	}
	if err := json.Unmarshal([]byte(out), &p); err != nil || code != 200 || p.ID == "" || len(p.Diff) == 0 {
		t.Fatalf("restore: %d %s", code, out)
	}
	if !contains(r.cell.Manifest().Spaces["ap"].Agents, "close-checklist") {
		t.Fatal("restore changed the applied manifest")
	}
	if code, _ := r.do(t, "POST", "/v1/admin/agents/close-checklist/history/2/restore", ""); code != http.StatusBadRequest {
		t.Fatalf("restoring the current version: %d", code)
	}
	if code, out := r.do(t, "GET", "/v1/spaces/ap/skills/month-end/history", ""); code != http.StatusForbidden {
		t.Fatalf("a non-steward read a skill's history: %d %s", code, out)
	}

	// SB-5105, SB-5106: sam, in the pilot group, moderates the branch; lee can't yet.
	code, out = r.do(t, "GET", "/v1/spaces/finance/branch", "")
	if code != 200 || !strings.Contains(out, `"space":"ap"`) || !strings.Contains(out, `"skills":["month-end"]`) || strings.Contains(out, "checklist.\"") {
		t.Fatalf("branch: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/spaces/ap/agents/close-checklist/pause", `{"reason":"wrong totals"}`); code != 200 {
		t.Fatalf("branch pause: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/spaces/finance/branch", ""); code != 200 || !strings.Contains(out, `{"name":"close-checklist","paused":true}`) {
		t.Fatalf("branch after the pause: %d %s", code, out)
	}
	as(lee)
	if code, _ := r.do(t, "POST", "/v1/spaces/ap/agents/ask/pause", `{"reason":"too slow"}`); code != http.StatusForbidden {
		t.Fatalf("lee paused before the feature is on for lee: %d", code)
	}
	if code, _ := r.do(t, "GET", "/v1/spaces/finance/branch", ""); code != http.StatusForbidden {
		t.Fatalf("lee saw the branch: %d", code)
	}

	// SB-5106: the operator holds and releases it, audited.
	as(sam)
	code, out = r.do(t, "GET", "/v1/admin/rollouts", "")
	if code != 200 || !strings.Contains(out, `"name":"branch-moderators"`) || !strings.Contains(out, `"state":"pilot"`) {
		t.Fatalf("rollouts: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/admin/rollouts/branch-moderators/hold", ""); code != 200 || !strings.Contains(out, `"state":"held"`) {
		t.Fatalf("hold: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/admin/rollouts/branch-moderators/release", ""); code != 200 || strings.Contains(out, `"state":"held"`) {
		t.Fatalf("release: %d %s", code, out)
	}
	if code, _ := r.do(t, "POST", "/v1/admin/rollouts/no-such/hold", ""); code != http.StatusBadRequest {
		t.Fatalf("an unknown feature: %d", code)
	}
	rows, err := r.cell.Audit.Store.Range(ctx, 1, 10000)
	if err != nil {
		t.Fatal(err)
	}
	var held, branch int
	for _, x := range rows {
		if x.Subject == "rollout:branch-moderators" {
			held++
		}
		if x.Kind == "kill" && strings.Contains(x.Reason, "(branch moderator of finance)") {
			branch++
		}
	}
	if held != 2 || branch != 1 {
		t.Fatalf("audit: %d rollout rows, %d branch pauses", held, branch)
	}
}
