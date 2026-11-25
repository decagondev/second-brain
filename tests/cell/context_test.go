package cell

import (
	"context"
	"encoding/json"
	"io"
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
	"example.com/second-brain/internal/core/kill"
)

// SB-4304 to SB-4307 through a running cell: a collection is shared by id and each reader
// sees only what they can open, an ask about it cites nothing outside it; the runs made as
// a person list what they read; the department agent is an MCP tool until it is killed; and
// a sheet's total comes from the table tool, cited to the sheet.
func TestContextForTheWorkThroughARunningCell(t *testing.T) {
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
	write("Finance/close.md", "The quokka close moves to Tuesday, and the AP review is still open.")
	write("Finance/accruals.md", "The quokka accruals are booked on the last working day.")
	write("Finance/bonus.md", "The quokka bonus pool is set by the board.")
	write("Finance/bonus.md.acl", "read sam@acme.example\n")
	write("Finance/invoices.csv", "Customer,Amount\nacme-logistics,1200\nbeta-freight,300.50\nacme-logistics,800\ngamma-parts,99.5\n")
	write("Finance/.acl", "read domain:acme.example\n")
	m := `tenant: acme-logistics
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
    agents: [ask, close-checklist]
    scope: { docs: ["/Finance/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		cfg.MCP = true
		return cfg
	})
	ctx := context.Background()
	code, body := r.do(t, "POST", "/scim/v2/Users", `{"userName":"lee","emails":[{"value":"lee@acme.example"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
	var lee struct{ ID string }
	if err := json.Unmarshal([]byte(body), &lee); err != nil || code != http.StatusCreated {
		t.Fatalf("SCIM lee: %d %s", code, body)
	}
	r.signIn(t, "sam")
	sam := r.idp.User
	for _, g := range []string{"finance", "finance-leads"} {
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+g+`","members":[{"value":"`+sam+`"},{"value":"`+lee.ID+`"}]}`,
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
			t.Fatalf("sign in as %s: %d", id, code)
		}
	}

	// SB-4304: a collection by id, shared with lee, who sees only what lee can open.
	if code, _ := r.do(t, "POST", "/v1/collections", `{"name":"Q4 close","items":["docs:Finance/close.md"],"shared":["nobody@acme.example"]}`); code != http.StatusBadRequest {
		t.Fatalf("shared with someone unknown: %d", code)
	}
	code, out := r.do(t, "POST", "/v1/collections", `{"name":"Q4 close","items":["docs:Finance/close.md@1","docs:Finance/accruals.md","docs:Finance/bonus.md"],"shared":["lee@acme.example"]}`)
	var col ask.CollectionView
	if err := json.Unmarshal([]byte(out), &col); err != nil || code != 200 || len(col.Items) != 3 || col.Hidden != 0 {
		t.Fatalf("collection: %d %s", code, out)
	}
	stored, found, err := r.cell.Traces.GetCollection(ctx, tenant, col.ID)
	if err != nil || !found || strings.Contains(strings.Join(stored.Items, " "), "@") {
		t.Fatalf("stored: %+v %v", stored, err)
	}
	as(lee.ID)
	code, out = r.do(t, "GET", "/v1/collections", "")
	var list struct{ Items []ask.CollectionView }
	if err := json.Unmarshal([]byte(out), &list); err != nil || code != 200 || len(list.Items) != 1 {
		t.Fatalf("lee's collections: %d %s", code, out)
	}
	if c := list.Items[0]; c.Mine || c.Hidden != 1 || len(c.Items) != 2 || strings.Contains(out, "bonus") || c.Shared != nil {
		t.Fatalf("lee sees: %s", out)
	}
	if code, _ := r.do(t, "DELETE", "/v1/collections/"+col.ID, ""); code != http.StatusForbidden {
		t.Fatalf("lee removed sam's collection: %d", code)
	}
	code, out = r.do(t, "POST", "/v1/ask?space=finance", `{"question":"What happens to the quokka close and accruals?","collection":"`+col.ID+`"}`)
	var a ask.Response
	if err := json.Unmarshal([]byte(out), &a); err != nil || code != 200 {
		t.Fatalf("ask the collection: %d %s", code, out)
	}
	cited := 0
	for _, cl := range a.Answer.Claims {
		for _, ci := range cl.Cites {
			cited++
			if !strings.HasPrefix(ci.Cite, "docs:Finance/close.md@") && !strings.HasPrefix(ci.Cite, "docs:Finance/accruals.md@") {
				t.Fatalf("an ask about the collection cited %s", ci.Cite)
			}
		}
	}
	if cited == 0 {
		t.Fatalf("nothing cited: %s", out)
	}
	if code, _ := r.do(t, "POST", "/v1/ask?space=finance", `{"question":"What is it?","collection":"col_nope"}`); code != http.StatusBadRequest {
		t.Fatalf("an unknown collection: %d", code)
	}
	as(sam)

	// SB-4307: the sheet's total is computed in code and cited to the sheet.
	code, out = r.do(t, "POST", "/v1/ask?space=finance", `{"question":"What is the total Amount on the invoices?"}`)
	if err := json.Unmarshal([]byte(out), &a); err != nil || code != 200 || a.Table == nil || a.Table.Value != 2400 || a.Table.Cite == "" || !strings.HasPrefix(a.Table.Cite, "docs:Finance/invoices.csv@") {
		t.Fatalf("table: %d %s", code, out)
	}

	// SB-4305: a research run made as sam, with what it read.
	code, out = r.do(t, "POST", "/v1/research?space=finance", `{"question":"When does the quokka close move?"}`)
	var rv ask.ResearchView
	if err := json.Unmarshal([]byte(out), &rv); err != nil || code != 200 {
		t.Fatalf("plan: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/research/"+rv.ID+"/run?space=finance", ""); code != 200 {
		t.Fatalf("run: %d %s", code, out)
	}
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	code, out = r.do(t, "GET", "/v1/me/agent-runs", "")
	var runs struct{ Items []ask.AgentRun }
	if err := json.Unmarshal([]byte(out), &runs); err != nil || code != 200 || len(runs.Items) == 0 {
		t.Fatalf("agent runs: %d %s", code, out)
	}
	if r0 := runs.Items[0]; r0.Kind != ask.KindResearch || len(r0.Read) == 0 || r0.Title != "When does the quokka close move?" {
		t.Fatalf("the research run: %+v", r0)
	}

	// SB-4306: the department agent is an MCP tool, as sam, until it is killed.
	_, body = r.do(t, "POST", "/v1/tokens", `{"name":"assistant","days":30}`)
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
	b := mcp(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	for _, want := range []string{`"name":"agent_close_checklist"`, `"enum":["finance"]`, `"name":"research_plan"`, `"name":"research_report"`} {
		if !strings.Contains(b, want) {
			t.Fatalf("tools/list lacks %s: %s", want, b)
		}
	}
	b = mcp(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"agent_close_checklist","arguments":{"space":"finance","question":"What is open in the quokka close?"}}}`)
	var called struct {
		Result struct {
			StructuredContent struct{ Trace, Agent string }
			IsError           bool
		}
	}
	if json.Unmarshal([]byte(b), &called) != nil || called.Result.IsError || called.Result.StructuredContent.Agent != "close-checklist" {
		t.Fatalf("the agent tool: %s", b)
	}
	if tr, found, _ := r.cell.Traces.GetTrace(ctx, tenant, called.Result.StructuredContent.Trace); !found || tr.User != sam || tr.Space != "spc_finance" {
		t.Fatalf("the agent tool's trace: %+v", tr)
	}
	if b := mcp(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"agent_close_checklist","arguments":{"space":"ops","question":"What is open?"}}}`); !strings.Contains(b, `"isError":true`) {
		t.Fatalf("a space that isn't theirs: %s", b)
	}
	if b := mcp(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"research_report","arguments":{"id":"` + rv.ID + `"}}}`); !strings.Contains(b, `"state":"done"`) {
		t.Fatalf("research_report: %s", b)
	}
	if err := (kill.Control{Store: r.cell.Kill.Store, Audit: r.cell.Audit, Now: time.Now}).Kill(ctx, tenant, "agent:close-checklist", "test", "ops"); err != nil {
		t.Fatal(err)
	}
	if err := r.cell.Kill.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if b := mcp(`{"jsonrpc":"2.0","id":5,"method":"tools/list"}`); strings.Contains(b, "agent_close_checklist") {
		t.Fatalf("a killed agent is still a tool: %s", b)
	}
}
