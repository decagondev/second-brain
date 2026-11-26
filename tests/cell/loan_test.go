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

// SB-5302, SB-5303 through a running cell (LAW-31). Sam, a steward of ops who can't read
// finance, asks Lee to lend finance for one run; Lee approves; the lent draft lands in ops'
// queue naming Lee, hidden from Sam (who can't open its cites) and shown to Kim (a steward of
// ops who can). The loan can't be answered twice. Try it on a trigger sends nothing.
func TestBorrowAccessThroughARunningCell(t *testing.T) {
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
	write("Ops/dock.md", "The quokka dock opens at seven.")
	write("Ops/.acl", "read domain:acme.example\n")
	write("Finance/close.md", "The quokka month-end close is on the third working day.")
	write("Finance/.acl", "read lee@acme.example\nread kim@acme.example\n")
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  ops:
    members: [group:ops]
    stewards: [group:ops-leads]
    scope: { docs: ["/Ops/**"] }
    on: [ticket-filed]
  finance:
    members: [group:finance]
    stewards: [group:finance]
    scope: { docs: ["/Finance/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	ids := map[string]string{}
	for _, n := range []string{"sam", "lee", "kim"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	for name, who := range map[string][]string{"ops": {"sam", "kim"}, "ops-leads": {"sam", "kim"}, "finance": {"lee", "kim"}} {
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
	as := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}

	// SB-5302: Try it replays, as the steward, and sends nothing.
	as(ids["sam"])
	if code, out := r.do(t, "POST", "/v1/spaces/ops/triggers/ticket-filed/try", ""); code != 200 || !strings.Contains(out, `"events":0`) {
		t.Fatalf("try: %d %s", code, out)
	}
	if code, _ := r.do(t, "POST", "/v1/spaces/ops/triggers/meeting-booked/try", ""); code != http.StatusBadRequest {
		t.Fatalf("an unknown trigger: %d", code)
	}

	// SB-5303: the loan.
	code, out := r.do(t, "POST", "/v1/spaces/ops/loans", `{"lender":"lee@acme.example","space":"finance","purpose":"When is the quokka month-end close?"}`)
	var loan struct{ ID string }
	if err := json.Unmarshal([]byte(out), &loan); err != nil || code != 200 || loan.ID == "" {
		t.Fatalf("request a loan: %d %s", code, out)
	}
	as(ids["kim"])
	if code, _ := r.do(t, "POST", "/v1/me/loans/"+loan.ID+"/approve", ""); code != http.StatusBadRequest {
		t.Fatalf("someone other than the lender approved: %d", code)
	}
	as(ids["lee"])
	if code, out := r.do(t, "GET", "/v1/spaces/me/questions", ""); code != 200 || !strings.Contains(out, "Lend finance for one run") {
		t.Fatalf("lee's queue: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/me/loans/"+loan.ID+"/approve", ""); code != 200 {
		t.Fatalf("approve: %d %s", code, out)
	}
	if code, _ := r.do(t, "POST", "/v1/me/loans/"+loan.ID+"/approve", ""); code != http.StatusBadRequest {
		t.Fatalf("a loan ran twice: %d", code)
	}
	as(ids["sam"])
	code, out = r.do(t, "GET", "/v1/spaces/ops/questions", "")
	if code != 200 || !strings.Contains(out, `"lender":"lee@acme.example"`) || strings.Contains(out, "third working day") || !strings.Contains(out, `"hidden":true`) {
		t.Fatalf("sam sees the lent draft: %d %s", code, out)
	}
	as(ids["kim"])
	if code, out := r.do(t, "GET", "/v1/spaces/ops/questions", ""); code != 200 || !strings.Contains(out, "third working day") {
		t.Fatalf("kim, who can open every cite, doesn't see it: %d %s", code, out)
	}
}
