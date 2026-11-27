package cell

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// S54.2 through a running cell (SB-5505 to SB-5507). Why not says a document ranked below
// the cut, was dropped, or was excluded, and answers a document Sam can't open exactly as one
// that doesn't exist. Kim, the steward, replays an answer Sam rated unhelpful, not one he
// didn't. An MCP client reads Sam's memory and proposes a line, which counts only once Sam
// confirms it in the web app; the token can't confirm it.
func TestAskWhyThroughARunningCell(t *testing.T) {
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
	for i := 0; i < 14; i++ { // more quokka dock passages than an answer reads, so the canteen ranks below the cut
		write(fmt.Sprintf("Ops/dock-%02d.md", i), fmt.Sprintf("The quokka dock opens at seven, says note %d about the quokka dock.", i))
	}
	write("Ops/board.md", "The quokka board meets in secret.")
	write("Ops/board.md.acl", "read kim@acme.example\n")
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
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		cfg.MCP = true
		return cfg
	})
	ctx := context.Background()
	ids := map[string]string{}
	for _, n := range []string{"sam", "kim"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	for name, who := range map[string][]string{"ops": {"sam", "kim"}, "ops-leads": {"kim"}} {
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
	askIt := func(body string) string {
		t.Helper()
		code, out := r.do(t, "POST", "/v1/ask?space=ops", body)
		var a struct{ Trace string }
		if err := json.Unmarshal([]byte(out), &a); err != nil || code != 200 {
			t.Fatalf("ask: %d %s", code, out)
		}
		return a.Trace
	}
	why := func(trace, doc string) string {
		t.Helper()
		code, out := r.do(t, "POST", "/v1/traces/"+trace+"/why-not", `{"document":"`+doc+`"}`)
		if code != 200 {
			t.Fatalf("why not %s: %d %s", doc, code, out)
		}
		return out
	}

	// SB-5506.
	as("sam")
	tr := askIt(`{"question":"When does the quokka dock open?"}`)
	if out := why(tr, "docs:Ops/canteen.md"); !strings.Contains(out, `"reason":"ranked_below"`) {
		t.Fatalf("ranked below: %s", out)
	}
	about := askIt(`{"question":"When does the quokka dock open?","documents":["docs:Ops/gate.md"]}`)
	if out := why(about, "docs:Ops/dock.md"); !strings.Contains(out, `"reason":"outside_scope"`) {
		t.Fatalf("outside the documents asked about: %s", out)
	}
	missing, secret := why(tr, "docs:Ops/nothing.md"), why(tr, "docs:Ops/board.md")
	if !strings.Contains(missing, `"reason":"no_such_document"`) || missing != secret || strings.Contains(secret, "board") {
		t.Fatalf("can't open vs missing: %s / %s", missing, secret)
	}
	dropped := askIt(`{"question":"When does the quokka gate close?","drop":["docs:Ops/gate.md"]}`)
	if out := why(dropped, "docs:Ops/gate.md"); !strings.Contains(out, `"reason":"dropped"`) {
		t.Fatalf("dropped: %s", out)
	}
	if code, _ := r.do(t, "POST", "/v1/me/exclusions", `{"document":"docs:Ops/canteen.md"}`); code != 200 {
		t.Fatal("exclude")
	}
	if out := why(tr, "docs:Ops/canteen.md"); !strings.Contains(out, `"reason":"excluded"`) {
		t.Fatalf("excluded: %s", out)
	}

	// SB-5507: only a flagged answer, only for the steward.
	as("kim")
	if code, _ := r.do(t, "GET", "/v1/spaces/ops/review/retrieval?trace="+tr, ""); code != http.StatusBadRequest {
		t.Fatalf("an answer nobody flagged: %d", code)
	}
	as("sam")
	if code, out := r.do(t, "POST", "/v1/feedback", `{"trace":"`+tr+`","rating":"down"}`); code != 200 {
		t.Fatalf("feedback: %d %s", code, out)
	}
	if code, _ := r.do(t, "GET", "/v1/spaces/ops/review/retrieval?trace="+tr, ""); code != http.StatusForbidden {
		t.Fatalf("a member saw the retrieval view: %d", code)
	}
	as("kim")
	code, out := r.do(t, "GET", "/v1/spaces/ops/review/retrieval?trace="+tr, "")
	if code != 200 || !strings.Contains(out, `"why":"kept"`) || !strings.Contains(out, "dock.md") || strings.Contains(out, "night") {
		t.Fatalf("retrieval: %d %s", code, out)
	}

	// SB-5505: memory over MCP.
	as("sam")
	if code, _ := r.do(t, "POST", "/v1/me/memory", `{"text":"I run the night shift."}`); code != 200 {
		t.Fatal("remember")
	}
	_, body := r.do(t, "POST", "/v1/tokens", `{"name":"assistant","days":30}`)
	var made struct{ Token string }
	_ = json.Unmarshal([]byte(body), &made)
	mcp := func(msg string) string {
		t.Helper()
		req, _ := http.NewRequest("POST", r.srv.URL+"/v1/mcp", strings.NewReader(msg))
		req.Header.Set("Authorization", "Bearer "+made.Token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := r.idp.Server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	if out := mcp(`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"memory://me"}}`); !strings.Contains(out, "night shift") {
		t.Fatalf("resources/read: %s", out)
	}
	out = mcp(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"memory_propose","arguments":{"text":"I prefer bullet points."}}}`)
	var call struct {
		Result struct {
			StructuredContent struct{ ID string } `json:"structuredContent"`
		}
	}
	if err := json.Unmarshal([]byte(out), &call); err != nil || call.Result.StructuredContent.ID == "" {
		t.Fatalf("memory_propose: %s", out)
	}
	pid := call.Result.StructuredContent.ID
	if code, out := r.do(t, "GET", "/v1/me/inbox", ""); code != 200 || !strings.Contains(out, `"kind":"memory"`) {
		t.Fatalf("inbox: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/ask?space=ops", `{"question":"When does the quokka dock open?"}`); code != 200 || strings.Count(out, "mem_") != 1 {
		t.Fatalf("an unconfirmed line was used: %d %s", code, out)
	}
	req, _ := http.NewRequest("POST", r.srv.URL+"/v1/me/memory/proposals/"+pid+"/confirm", nil)
	req.Header.Set("Authorization", "Bearer "+made.Token)
	resp, err := r.idp.Server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a token confirmed a proposal: %d", resp.StatusCode)
	}
	if code, out := r.do(t, "POST", "/v1/me/memory/proposals/"+pid+"/confirm", ""); code != 200 || !strings.Contains(out, "bullet points") {
		t.Fatalf("confirm: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/ask?space=ops", `{"question":"When does the quokka dock open?"}`); code != 200 || strings.Count(out, "mem_") != 2 {
		t.Fatalf("the confirmed line wasn't used: %d %s", code, out)
	}
}
