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

const treeManifest = `tenant: acme-logistics
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
    stewards: [group:finance]
    glossary: { ZC: zebra cutoff }
    scope: { docs: ["/Finance/Dept/**"] }
  ap:
    parent: finance
    members: [group:ap]
    stewards: [group:ap]
    scope: { docs: ["/Finance/AP/**"] }
  fpa:
    parent: finance
    members: [group:fpa]
    stewards: [group:fpa]
    scope: { docs: ["/Finance/FPA/**"] }
`

// SB-3901 to SB-3903 through a running cell: a branch ask in a department joins the
// teams the asker is in and no other, a non-member of every team gets the department
// alone, and a team uses its department's glossary.
func TestTreeThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	for dir, text := range map[string]string{"Dept": "The zebra cutoff for the department is noon.", "AP": "The zebra cutoff for invoices is 3pm.", "FPA": "The zebra cutoff for forecasts is Friday."} {
		if err := os.MkdirAll(filepath.Join(share, "Finance", dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, "Finance", dir, "cutoff.md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, "Finance", dir, ".acl"), []byte("read domain:acme.example\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, treeManifest, nil)
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
	for name, ids := range map[string][]string{"finance": {sam, lee}, "ap": {sam}} {
		var ms []string
		for _, id := range ids {
			ms = append(ms, `{"value":"`+id+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	signInAs(sam)
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	found := func(path string) string {
		t.Helper()
		code, out := r.do(t, "GET", path, "")
		var res ask.SearchResponse
		if err := json.Unmarshal([]byte(out), &res); err != nil || code != 200 {
			t.Fatalf("%s: %d %s", path, code, out)
		}
		var got []string
		for _, x := range res.Results {
			switch {
			case strings.Contains(x.Cite, "Finance/Dept/"):
				got = append(got, "dept")
			case strings.Contains(x.Cite, "Finance/AP/"):
				got = append(got, "ap")
			case strings.Contains(x.Cite, "Finance/FPA/"):
				got = append(got, "fpa")
			}
		}
		return strings.Join(sorted(got), ",")
	}
	if got := found("/v1/search?q=zebra+cutoff&space=finance"); got != "dept" {
		t.Fatalf("the department alone: %s", got)
	}
	if got := found("/v1/search?q=zebra+cutoff&space=finance&branch=true"); got != "ap,dept" {
		t.Fatalf("sam's branch: %s", got)
	}
	if got := found("/v1/search?q=ZC&space=ap"); got != "ap" {
		t.Fatalf("the team didn't use its department's glossary: %s", got)
	}
	signInAs(lee)
	if got := found("/v1/search?q=zebra+cutoff&space=finance&branch=true"); got != "dept" {
		t.Fatalf("lee's branch: %s", got)
	}
	if code, out := r.do(t, "GET", "/v1/search?q=zebra&space=ap", ""); code == 200 {
		t.Fatalf("a non-member asked in a team: %s", out)
	}
	code, out := r.do(t, "POST", "/v1/ask?space=finance&branch=true", `{"question":"When is the zebra cutoff?"}`)
	if code != 200 || strings.Contains(out, "Finance/AP/") || strings.Contains(out, "Finance/FPA/") || strings.Contains(out, `"ap"`) || strings.Contains(out, `"fpa"`) {
		t.Fatalf("lee's branch ask: %d %s", code, out)
	}
}

func sorted(xs []string) []string {
	for i := range xs {
		for j := i + 1; j < len(xs); j++ {
			if xs[j] < xs[i] {
				xs[i], xs[j] = xs[j], xs[i]
			}
		}
	}
	return xs
}
