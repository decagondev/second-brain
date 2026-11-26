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
)

// S53.1 through a running cell (SB-5401 to SB-5404). Ann's pay sheet, with a card number, is
// open to the whole domain: it is held back from Sam's answers, cited to Ann and to Kim (a
// steward), listed on Kim's review list and in Ann's inbox without its text. Ann's Shared on
// purpose releases it; a new version holds it again. A plan shared with one person that a
// sync sees opened to the domain tells its owner once. Who can see this counts the people
// the source lets open the menu and names no group Sam can't see.
func TestKnowWhoCanSeeThroughARunningCell(t *testing.T) {
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
	write("Finance/pay.md", "Quokka pay sheet. The quokka float card is 4111 1111 1111 1111.")
	write("Finance/pay.md.acl", "read domain:acme.example\nowner ann@acme.example\n")
	write("Finance/menu.md", "The quokka canteen menu changes on Mondays.")
	write("Finance/plan.md", "Quokka bonus plan. Pay the quokka bonus to card 5500 0000 0000 0004.")
	write("Finance/plan.md.acl", "owner ann@acme.example\n")
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
    scope: { docs: ["/Finance/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	ids := map[string]string{}
	for _, n := range []string{"sam", "ann", "kim"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	for name, who := range map[string][]string{"finance": {"sam", "ann", "kim"}, "fin-leads": {"kim"}} {
		var ms []string
		for _, n := range who {
			ms = append(ms, `{"value":"`+ids[n]+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	sync := func() {
		t.Helper()
		if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
			t.Fatal(fails)
		}
	}
	sync()
	as := func(n string) {
		t.Helper()
		r.idp.User = ids[n]
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	cites := func(q string) (string, int) {
		t.Helper()
		code, out := r.do(t, "GET", "/v1/search?space=finance&q="+strings.ReplaceAll(q, " ", "+"), "")
		if code != 200 {
			t.Fatalf("search: %d %s", code, out)
		}
		var res struct{ Results []struct{ Cite string } }
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatal(err)
		}
		var cs []string
		for _, x := range res.Results {
			cs = append(cs, x.Cite)
		}
		return strings.Join(cs, " "), len(cs)
	}

	// SB-5401, SB-5402: held from Sam, not from its owner or the steward.
	as("sam")
	if got, _ := cites("quokka pay sheet"); strings.Contains(got, "pay.md") || !strings.Contains(got, "menu.md") {
		t.Fatalf("sam sees the pay sheet: %s", got)
	}
	as("ann")
	if got, _ := cites("quokka pay sheet"); !strings.Contains(got, "pay.md") {
		t.Fatalf("ann doesn't see her own pay sheet: %s", got)
	}
	code, out := r.do(t, "GET", "/v1/me/inbox", "")
	if code != 200 || !strings.Contains(out, `"kind":"wide"`) || !strings.Contains(out, "personal_data") || strings.Contains(out, "4111") {
		t.Fatalf("ann's inbox: %d %s", code, out)
	}
	as("kim")
	if got, _ := cites("quokka pay sheet"); !strings.Contains(got, "pay.md") {
		t.Fatalf("the steward doesn't see it: %s", got)
	}
	code, out = r.do(t, "GET", "/v1/spaces/finance/review?section=wide", "")
	if code != 200 || !strings.Contains(out, "Finance/pay.md") || !strings.Contains(out, `"held":true`) || !strings.Contains(out, `"grant":"domain:acme.example"`) ||
		strings.Contains(out, "menu.md") || strings.Contains(out, "4111") {
		t.Fatalf("review wide: %d %s", code, out)
	}
	as("sam")
	if code, _ := r.do(t, "GET", "/v1/spaces/finance/review?section=wide", ""); code != http.StatusForbidden {
		t.Fatalf("a member saw the wide section: %d", code)
	}

	// Shared on purpose: only the owner, audited, and it releases the hold.
	var pay string
	as("kim")
	_, out = r.do(t, "GET", "/v1/spaces/finance/review?section=wide", "")
	var wl struct{ Items []struct{ Cite string } }
	if err := json.Unmarshal([]byte(out), &wl); err != nil || len(wl.Items) != 1 {
		t.Fatalf("wide items: %s", out)
	}
	pay = wl.Items[0].Cite
	if code, _ := r.do(t, "POST", "/v1/documents/sharing/confirm", `{"cite":"`+pay+`","reason":"the whole team runs payroll"}`); code != http.StatusForbidden {
		t.Fatalf("a steward confirmed someone else's document: %d", code)
	}
	as("ann")
	if code, out := r.do(t, "POST", "/v1/documents/sharing/confirm", `{"cite":"`+pay+`","reason":"the whole team runs payroll"}`); code != 200 {
		t.Fatalf("confirm: %d %s", code, out)
	}
	as("sam")
	if got, _ := cites("quokka pay sheet"); !strings.Contains(got, "pay.md") {
		t.Fatalf("confirmed but still held: %s", got)
	}
	time.Sleep(10 * time.Millisecond) // a new modification time is a new version
	write("Finance/pay.md", "Quokka pay sheet, revised. The quokka float card is 4111 1111 1111 1111.")
	sync()
	if got, _ := cites("quokka pay sheet"); strings.Contains(got, "pay.md") {
		t.Fatalf("a new version wasn't held again: %s", got)
	}

	// SB-5403: the plan opened to the domain tells Ann once.
	write("Finance/plan.md.acl", "read domain:acme.example\nowner ann@acme.example\n")
	sync()
	tl := &told{}
	if _, err := r.cell.NotifyJob(tl).Run(ctx); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(tl.take(), "\n")
	if !strings.Contains(got, "ann@acme.example: The sharing of one of your documents (docs:Finance/plan.md) widened") || strings.Contains(got, "5500") {
		t.Fatalf("widened notice: %q", got)
	}
	if _, err := r.cell.NotifyJob(tl).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if again := tl.take(); len(again) != 0 {
		t.Fatalf("told twice: %v", again)
	}

	// SB-5404: who can see the menu: the three of them, and no group named.
	as("sam")
	code, out = r.do(t, "GET", "/v1/audience?doc=docs:Finance/menu.md", "")
	if code != 200 || !strings.Contains(out, `"people":3`) || !strings.Contains(out, `"domains":["acme.example"]`) {
		t.Fatalf("audience: %d %s", code, out)
	}
	if code, _ := r.do(t, "GET", "/v1/audience?doc=docs:Finance/pay.md", ""); code != http.StatusForbidden {
		t.Fatalf("a held document's audience: %d", code)
	}
	if code, _ := r.do(t, "GET", "/v1/audience?doc=docs:Finance/nothing.md", ""); code != http.StatusForbidden {
		t.Fatalf("a missing document's audience: %d", code)
	}
}
