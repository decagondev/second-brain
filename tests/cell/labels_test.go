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
)

// S53.2 through a running cell (SB-5405 to SB-5407, LAW-32). A plan whose front matter says
// confidential labels an answer that cites it; the canteen menu's answer has no such label.
// The ask agent's reach is what its stewards can open, with the plan counted as labelled.
// The sensitive map counts the space's documents by kind, for its steward and operators
// only, and names none.
func TestLabelsReachAndTheSensitiveMapThroughARunningCell(t *testing.T) {
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
	write("Finance/plan.md", "---\nsensitivity: confidential\n---\nQuokka acquisition plan. The quokka deal closes in March.")
	write("Finance/menu.md", "The quokka canteen menu changes on Mondays.")
	write("Finance/pay.md", "Quokka pay sheet. The quokka float card is 4111 1111 1111 1111.")
	write("Finance/.acl", "read domain:acme.example\n")
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  finance:
    members: [group:finance]
    stewards: [group:fin-leads]
    agents: [ask]
    scope: { docs: ["/Finance/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	ids := map[string]string{}
	for _, n := range []string{"sam", "kim", "ops"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	for name, who := range map[string][]string{"finance": {"sam", "kim"}, "fin-leads": {"kim"}, "sb-operators": {"ops"}} {
		var ms []string
		for _, n := range who {
			ms = append(ms, `{"value":"`+ids[n]+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	as := func(n string) {
		t.Helper()
		r.idp.User = ids[n]
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}

	// SB-5405: the label is read from front matter and carried by the search result. Shared
	// with the whole domain, the confidential plan is held from Sam (SB-5402), not from Kim.
	as("sam")
	if code, out := r.do(t, "GET", "/v1/search?space=finance&q=quokka+acquisition+plan", ""); code != 200 || strings.Contains(out, "plan.md") {
		t.Fatalf("the plan wasn't held from sam: %d %s", code, out)
	}
	as("kim")
	code, out := r.do(t, "GET", "/v1/search?space=finance&q=quokka+acquisition+plan", "")
	if code != 200 || !strings.Contains(out, `"label":"confidential"`) {
		t.Fatalf("the plan's label: %d %s", code, out)
	}
	as("sam")

	// SB-5407: counts for the steward, refused for a member; the operators see every space.
	if code, _ := r.do(t, "GET", "/v1/spaces/finance/review?section=sensitive", ""); code != http.StatusForbidden {
		t.Fatalf("a member saw the sensitive map: %d", code)
	}
	as("kim")
	code, out = r.do(t, "GET", "/v1/spaces/finance/review?section=sensitive", "")
	var sc struct {
		Personal int            `json:"personal_data"`
		Labels   map[string]int `json:"labels"`
		Wide     int            `json:"wide"`
		Held     int            `json:"held"`
	}
	if err := json.Unmarshal([]byte(out), &sc); err != nil || code != 200 {
		t.Fatalf("sensitive: %d %s", code, out)
	}
	if sc.Personal != 1 || sc.Labels["confidential"] != 1 || sc.Wide != 2 || sc.Held != 2 || strings.Contains(out, "plan.md") || strings.Contains(out, "quokka") {
		t.Fatalf("sensitive counts: %s", out)
	}
	as("ops")
	if code, out := r.do(t, "GET", "/v1/admin/sensitive", ""); code != 200 || !strings.Contains(out, `"space":"finance"`) {
		t.Fatalf("admin sensitive: %d %s", code, out)
	}

	// SB-5406: the ask agent reaches the three documents through its one steward.
	code, out = r.do(t, "GET", "/v1/admin/agents/ask/reach", "")
	var rc struct {
		Documents, Labelled int
		Spaces              []struct {
			Space  string
			People int
		}
	}
	if err := json.Unmarshal([]byte(out), &rc); err != nil || code != 200 {
		t.Fatalf("reach: %d %s", code, out)
	}
	if rc.Documents != 3 || rc.Labelled != 1 || len(rc.Spaces) != 1 || rc.Spaces[0].People != 1 || strings.Contains(out, "quokka") {
		t.Fatalf("reach: %s", out)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/agents/no-such-agent/reach", ""); code != http.StatusBadRequest {
		t.Fatalf("an unknown agent's reach: %d", code)
	}
}
