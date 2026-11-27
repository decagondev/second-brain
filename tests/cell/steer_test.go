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

// S54.1 through a running cell (SB-5501 to SB-5504). Sam's answer's context names the
// documents it read and the memory line it was given; Drop leaves a document out of the
// next ask and Keep brings one in; Don't use this for me keeps the gate notes out of Sam's
// search but not Kim's, until Sam undoes it, and the list is on Sam's My data.
func TestSeeAndSteerTheContextThroughARunningCell(t *testing.T) {
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
	write("Ops/gate.md", "The quokka gate closes at nine.")
	write("Ops/canteen.md", "The canteen serves soup on Fridays.")
	write("Ops/.acl", "read domain:acme.example\n")
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  ops:
    members: [group:ops]
    stewards: [group:ops]
    scope: { docs: ["/Ops/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	ids := map[string]string{}
	for _, n := range []string{"sam", "kim"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"ops","members":[{"value":"`+ids["sam"]+`"},{"value":"`+ids["kim"]+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
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
	as("sam")
	if code, out := r.do(t, "POST", "/v1/me/memory", `{"text":"I run the night shift."}`); code != 200 {
		t.Fatalf("remember: %d %s", code, out)
	}
	type answer struct {
		Trace  string   `json:"trace"`
		Memory []string `json:"memory"`
		Answer struct {
			Claims []struct{ Cites []struct{ Cite string } }
		}
	}
	askIt := func(body string) (answer, string) {
		t.Helper()
		code, out := r.do(t, "POST", "/v1/ask?space=ops", body)
		var a answer
		if err := json.Unmarshal([]byte(out), &a); err != nil || code != 200 {
			t.Fatalf("ask: %d %s", code, out)
		}
		var cs []string
		for _, c := range a.Answer.Claims {
			for _, ci := range c.Cites {
				cs = append(cs, ci.Cite)
			}
		}
		return a, strings.Join(cs, " ")
	}

	// SB-5501, SB-5504: the context and the memory line.
	a, cited := askIt(`{"question":"When does the quokka dock open?"}`)
	if !strings.Contains(cited, "dock.md") || len(a.Memory) != 1 {
		t.Fatalf("first answer: %s %+v", cited, a)
	}
	code, out := r.do(t, "GET", "/v1/traces/"+a.Trace+"/context", "")
	if code != 200 || !strings.Contains(out, "Ops/dock.md") || !strings.Contains(out, "night shift") {
		t.Fatalf("context: %d %s", code, out)
	}
	as("kim")
	if code, _ := r.do(t, "GET", "/v1/traces/"+a.Trace+"/context", ""); code != http.StatusForbidden {
		t.Fatalf("someone else's context: %d", code)
	}

	// SB-5502: Drop and Keep, sent by the browser with the ask.
	as("sam")
	if _, cited := askIt(`{"question":"When does the quokka dock open?","drop":["docs:Ops/dock.md"]}`); strings.Contains(cited, "dock.md") {
		t.Fatalf("a dropped document was cited: %s", cited)
	}
	if _, cited := askIt(`{"question":"When does the quokka dock open?","keep":["docs:Ops/canteen.md"]}`); !strings.Contains(cited, "canteen.md") {
		t.Fatalf("a kept document wasn't used: %s", cited)
	}

	// SB-5503: Don't use this for me.
	search := func() string {
		t.Helper()
		_, out := r.do(t, "GET", "/v1/search?space=ops&q=quokka+gate", "")
		return out
	}
	if code, out := r.do(t, "POST", "/v1/me/exclusions", `{"document":"docs:Ops/gate.md"}`); code != 200 || !strings.Contains(out, "Ops/gate.md") {
		t.Fatalf("exclude: %d %s", code, out)
	}
	if strings.Contains(search(), "gate.md") {
		t.Fatal("an excluded document was found for sam")
	}
	as("kim")
	if !strings.Contains(search(), "gate.md") {
		t.Fatal("sam's exclusion hid it from kim")
	}
	as("sam")
	if code, out := r.do(t, "GET", "/v1/me/exclusions", ""); code != 200 || !strings.Contains(out, "gate") {
		t.Fatalf("my data: %d %s", code, out)
	}
	if code, out := r.do(t, "DELETE", "/v1/me/exclusions?document=docs:Ops/gate.md", ""); code != 200 {
		t.Fatalf("undo: %d %s", code, out)
	}
	if !strings.Contains(search(), "gate.md") {
		t.Fatal("undoing didn't bring it back")
	}

	// SB-5504: forget the line; the next answer wasn't given it.
	_, out = r.do(t, "GET", "/v1/me/memory", "")
	var mv struct{ Lines []struct{ ID string } }
	if err := json.Unmarshal([]byte(out), &mv); err != nil || len(mv.Lines) != 1 {
		t.Fatalf("memory: %s", out)
	}
	if code, _ := r.do(t, "DELETE", "/v1/me/memory/"+mv.Lines[0].ID, ""); code != 200 {
		t.Fatalf("forget: %d", code)
	}
	if a, _ := askIt(`{"question":"When does the quokka gate close?"}`); len(a.Memory) != 0 {
		t.Fatalf("a forgotten line was given: %+v", a.Memory)
	}
}
