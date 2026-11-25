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

const acrossManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  ops:
    members: [group:ops]
    stewards: [group:ops-leads]
    scope: { docs: ["/Ops/Dept/**"] }
  returns:
    parent: ops
    members: [group:returns]
    stewards: [group:returns]
    scope: { docs: ["/Ops/Returns/**"] }
  fleet:
    parent: ops
    members: [group:fleet]
    stewards: [group:fleet]
    scope: { docs: ["/Ops/Fleet/**"] }
  it:
    members: [group:ops]
    stewards: [group:it]
    requests: { categories: [Access], due_days: 2 }
    scope: { docs: ["/IT/**"] }
`

// SB-4504 to SB-4507 through a running cell: a request waits on another department, each
// home names it only to who can see it and counts it for the rest, either side clears it,
// and the roll-up counts a team's waiting goal; a decision's trail and replacements show
// only what the reader can open; catch-up reads only what changed that the person can
// open; service levels are for stewards, masked under 5, and a pause's end offers catch-up
// once.
func TestWorkAcrossDepartmentsThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	today := time.Now().UTC().Format("2006-01-02")
	soon := time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02")
	for path, text := range map[string]string{
		"Ops/Dept/dock.md":           "---\ntype: decision\n---\nDecision: the dock opens at seven.",
		"Ops/Dept/dock-dec.md":       "---\ntype: decision\nsupersedes: docs:Ops/Dept/dock.md\n---\nDecision: from December the dock opens at six.",
		"Ops/Dept/private/dock-2.md": "---\ntype: decision\nsupersedes: docs:Ops/Dept/dock-dec.md\n---\nDecision: from January the dock opens at five.",
		"Ops/Returns/goal.md":        "---\ntype: goal\nowner: sam@acme.example\nquarter: 2026-Q4\nstatus: at risk\nwaiting_on: fleet\nwaiting_reason: vans for collections\nwaiting_until: " + soon + "\n---\nReturns collected the same day.",
		"Ops/Fleet/mot.md":           "Every van has its MOT in March.",
		"IT/access.md":               "Access to tools is granted by IT.",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for dir, acl := range map[string]string{"Ops/Dept": "read domain:acme.example\n", "Ops/Dept/private": "read sam@acme.example\n",
		"Ops/Returns": "read sam@acme.example\n", "Ops/Fleet": "read lee@acme.example\n", "IT": "read domain:acme.example\n"} {
		if err := os.WriteFile(filepath.Join(share, dir, ".acl"), []byte(acl), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, acrossManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	as := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	ids := map[string]string{}
	for _, n := range []string{"sam", "lee", "kim", "ana"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	for name, who := range map[string][]string{"ops": {"sam", "lee", "kim", "ana"}, "ops-leads": {"kim"}, "returns": {"sam"}, "fleet": {"lee"}, "it": {"kim"}} {
		var ms []string
		for _, n := range who {
			ms = append(ms, `{"value":"`+ids[n]+`"}`)
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

	// SB-4504: sam's request to IT waits on fleet.
	as(ids["sam"])
	code, out := r.do(t, "POST", "/v1/spaces/it/requests", `{"category":"Access","text":"Access to the van tracker for collections"}`)
	var sent struct{ ID string }
	if err := json.Unmarshal([]byte(out), &sent); err != nil || code != 200 {
		t.Fatalf("request: %d %s", code, out)
	}
	wait := `{"kind":"request","id":"` + sent.ID + `","on":"fleet","reason":"fleet must name the vans","until":"` + soon + `"}`
	if code, out := r.do(t, "PUT", "/v1/spaces/it/waits", strings.Replace(wait, "fleet\",\"reason", "it\",\"reason", 1)); code != http.StatusBadRequest {
		t.Fatalf("a wait on its own space: %d %s", code, out)
	}
	if code, out := r.do(t, "PUT", "/v1/spaces/it/waits", wait); code != 200 || !strings.Contains(out, `"on":"fleet"`) {
		t.Fatalf("mark: %d %s", code, out)
	}
	as(ids["ana"])
	if code, out := r.do(t, "PUT", "/v1/spaces/it/waits", wait); code != http.StatusForbidden {
		t.Fatalf("ana marked someone else's request: %d %s", code, out)
	}
	var ws ask.WaitsResponse
	code, out = r.do(t, "GET", "/v1/spaces/it/waits", "")
	if err := json.Unmarshal([]byte(out), &ws); err != nil || code != 200 || len(ws.Waiting) != 0 || ws.Hidden != 1 || strings.Contains(out, "vans") {
		t.Fatalf("ana sees the wait: %d %s", code, out)
	}
	as(ids["kim"])
	code, out = r.do(t, "GET", "/v1/spaces/it/waits", "")
	if err := json.Unmarshal([]byte(out), &ws); err != nil || code != 200 || len(ws.Waiting) != 1 || ws.Waiting[0].Title != "Access to the van tracker for collections" || !ws.Waiting[0].CanClear {
		t.Fatalf("IT's home: %d %s", code, out)
	}
	as(ids["lee"])
	code, out = r.do(t, "GET", "/v1/spaces/fleet/waits", "")
	if err := json.Unmarshal([]byte(out), &ws); err != nil || code != 200 || len(ws.WaitedOn) != 1 || ws.WaitedOn[0].Title != "Request: Access" ||
		ws.WaitedOn[0].Reason != "fleet must name the vans" || strings.Contains(out, "van tracker") || ws.Hidden != 1 || strings.Contains(out, "collections") {
		t.Fatalf("fleet's home: %d %s", code, out) // the returns goal lee can't open is counted
	}
	if code, out := r.do(t, "DELETE", "/v1/spaces/it/waits?kind=request&id="+sent.ID, ""); code != 200 {
		t.Fatalf("clear from fleet: %d %s", code, out)
	}
	as(ids["kim"])
	if code, out := r.do(t, "GET", "/v1/spaces/it/waits", ""); code != 200 || !strings.Contains(out, `"waiting":[]`) {
		t.Fatalf("cleared: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/spaces/ops/review", "")
	var rev ask.ReviewResponse
	if err := json.Unmarshal([]byte(out), &rev); err != nil || code != 200 {
		t.Fatalf("ops review: %d %s", code, out)
	}
	for _, tm := range rev.Teams {
		if want := map[string]int{"returns": 1, "fleet": 0}[tm.Space]; tm.Waiting != want {
			t.Fatalf("roll-up %s waiting %d, want %d: %s", tm.Space, tm.Waiting, want, out)
		}
	}

	// SB-4505: the trail, and replacements only as far as the reader can open.
	as(ids["ana"])
	var tr ask.TrailResponse
	code, out = r.do(t, "GET", "/v1/spaces/ops/decisions?topic=dock", "")
	if err := json.Unmarshal([]byte(out), &tr); err != nil || code != 200 || len(tr.Decisions) != 2 || tr.Hidden != 1 {
		t.Fatalf("ana's trail: %d %s", code, out)
	}
	if !strings.Contains(tr.Decisions[0].Cite, "Dept/dock.md") || tr.Decisions[0].ReplacedBy == nil || !strings.Contains(tr.Decisions[0].ReplacedBy.Cite, "dock-dec.md") ||
		tr.Decisions[1].Supersedes == nil || tr.Decisions[1].ReplacedBy != nil || strings.Contains(out, "five") || strings.Contains(out, "dock-2") {
		t.Fatalf("ana's links: %s", out)
	}
	as(ids["sam"])
	code, out = r.do(t, "GET", "/v1/spaces/ops/decisions?topic=dock", "")
	if err := json.Unmarshal([]byte(out), &tr); err != nil || code != 200 || len(tr.Decisions) != 3 || tr.Hidden != 0 ||
		tr.Decisions[1].ReplacedBy == nil || !strings.Contains(tr.Decisions[1].ReplacedBy.Cite, "dock-2.md") {
		t.Fatalf("sam's trail: %d %s", code, out)
	}

	// SB-4506: catch-up as sam across ops and returns, never what sam can't open.
	code, out = r.do(t, "POST", "/v1/catch-up?space=ops&branch=true", `{"since":"`+today+`"}`)
	var cu ask.CatchUpResponse
	if err := json.Unmarshal([]byte(out), &cu); err != nil || code != 200 || cu.CatchUp.Documents != 4 || cu.Agent != ask.AgentCatchUp || strings.Contains(out, "MOT") {
		t.Fatalf("catch-up: %d %s", code, out)
	}
	for _, c := range cu.Answer.Claims {
		if len(c.Cites) == 0 {
			t.Fatalf("an uncited line: %s", out)
		}
	}
	if code, out := r.do(t, "POST", "/v1/catch-up?space=ops", `{"since":"`+time.Now().UTC().AddDate(0, 0, -100).Format("2006-01-02")+`"}`); code != http.StatusBadRequest {
		t.Fatalf("100 days back: %d %s", code, out)
	}
	as(ids["ana"])
	code, out = r.do(t, "POST", "/v1/catch-up?space=ops", `{}`)
	if err := json.Unmarshal([]byte(out), &cu); err != nil || code != 200 || cu.CatchUp.Documents != 2 || strings.Contains(out, "five") {
		t.Fatalf("ana's catch-up: %d %s", code, out)
	}
	// A pause's end offers it once.
	for _, b := range []string{`{"pause_days":3}`, `{"pause_days":0}`} {
		if code, out := r.do(t, "PUT", "/v1/me/notifications", b); code != 200 {
			t.Fatalf("pause: %d %s", code, out)
		}
	}
	if code, out := r.do(t, "GET", "/v1/me/inbox", ""); code != 200 || !strings.Contains(out, `"catch_up":{"since":"`+today+`"}`) {
		t.Fatalf("offer: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/me/inbox", ""); code != 200 || strings.Contains(out, "catch_up") {
		t.Fatalf("offered twice: %d %s", code, out)
	}

	// SB-4507: IT's service levels, for its stewards, a row under 5 masked.
	as(ids["sam"])
	for i := 0; i < 4; i++ {
		if code, out := r.do(t, "POST", "/v1/spaces/it/requests", `{"category":"Access","text":"Another tool"}`); code != 200 {
			t.Fatalf("request: %d %s", code, out)
		}
	}
	as(ids["lee"])
	if code, out := r.do(t, "GET", "/v1/spaces/it/service", ""); code != http.StatusForbidden {
		t.Fatalf("a member saw service levels: %d %s", code, out)
	}
	as(ids["kim"])
	if code, out := r.do(t, "POST", "/v1/spaces/it/questions/"+sent.ID+"/reply", `{"text":"Done.","cites":[]}`); code != 200 {
		t.Fatalf("reply: %d %s", code, out)
	}
	var sv ask.ServiceResponse
	code, out = r.do(t, "GET", "/v1/spaces/it/service", "")
	if err := json.Unmarshal([]byte(out), &sv); err != nil || code != 200 || len(sv.Weeks) != ask.ServiceWeeks || len(sv.Rows) != 1 ||
		sv.Rows[0].Count != 5 || sv.Rows[0].Close == nil || sv.Rows[0].LateShare == nil || *sv.Rows[0].LateShare != 0 || strings.Contains(out, "tool") {
		t.Fatalf("service: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/spaces/ops/service", "")
	if err := json.Unmarshal([]byte(out), &sv); err != nil || code != 200 || len(sv.Teams) != 2 || !sv.Teams[0].FewerThan5 {
		t.Fatalf("ops service: %d %s", code, out)
	}
}
