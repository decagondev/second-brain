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
)

// SB-4904 to SB-4906 through a running cell. A steward runs scenarios against a proposal
// that lists a department agent, and keeps one as an eval case. A new version of a document
// the weekly brief cited redrafts it once that day, with the earlier text kept for the diff,
// and a source its approver can no longer open drops the next redraft. Asks two people found
// helpful become starter-question candidates with nobody named. A published starter is
// listed over MCP, and two people with different access get different answers to it.
func TestScenariosSourcesAndStartersThroughARunningCell(t *testing.T) {
	day := strings.ToLower(time.Now().Weekday().String())
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
	write("Finance/.acl", "read domain:acme.example\n")
	write("Finance/close.md", "This week in this space the quokka close moves to Tuesday.\n")
	write("Finance/bonus.md", "The quokka close bonus is held until March.\n")
	write("Finance/bonus.md.acl", "read sam@acme.example\n")
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
    agents: []
    brief: { agent: ask, weekly: ` + day + ` }
    on: [source-changed]
    scope: { docs: ["/Finance/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		cfg.MCP = true
		return cfg
	})
	ctx := context.Background()
	ids := map[string]string{}
	for _, n := range []string{"lee", "kim", "sam"} {
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
	for name, members := range map[string][]string{"finance": {"sam", "lee", "kim"}, "finance-leads": {"sam"}, "sb-operators": {"kim"}} {
		var ms []string
		for _, x := range members {
			ms = append(ms, `{"value":"`+ids[x]+`"}`)
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

	// SB-4904: sam proposes the close-checklist agent and runs two scenarios against it.
	as("sam")
	code, out := r.do(t, "POST", "/v1/spaces/finance/proposals/agents", `{"agents":["close-checklist"]}`)
	var prop struct{ ID string }
	if err := json.Unmarshal([]byte(out), &prop); err != nil || code != 200 || prop.ID == "" {
		t.Fatalf("the proposal: %d %s", code, out)
	}
	scen := `{"scenarios":[{"question":"When is the quokka close?","agent":"close-checklist","cites":"docs:Finance/close.md"},` +
		`{"question":"When is the quokka close?","says":"Tuesday","never":"Wednesday"}],"keep":[1]}`
	code, out = r.do(t, "POST", "/v1/spaces/finance/proposals/"+prop.ID+"/scenarios", scen)
	var run struct {
		Results    []ask.ScenarioResult
		Passed, Of int
	}
	if err := json.Unmarshal([]byte(out), &run); err != nil || code != 200 || len(run.Results) != 2 {
		t.Fatalf("scenarios: %d %s", code, out)
	}
	first, second := run.Results[0], run.Results[1]
	if first.Current.Pass || first.Current.Why == "" || !first.Proposed.Pass || !strings.HasPrefix(first.Kept, "ev_") {
		t.Fatalf("the agent's scenario, side by side: %s", out)
	}
	if !second.Current.Pass || !second.Proposed.Pass || run.Passed != 2 || run.Of != 2 {
		t.Fatalf("the plain scenario: %s", out)
	}
	if code, out := r.do(t, "GET", "/v1/spaces/finance/evals", ""); code != 200 || !strings.Contains(out, first.Kept) {
		t.Fatalf("the kept scenario isn't an eval case: %d %s", code, out)
	}
	as("lee")
	if code, out := r.do(t, "POST", "/v1/spaces/finance/proposals/"+prop.ID+"/scenarios", scen); code != http.StatusForbidden {
		t.Fatalf("a member who isn't a steward ran scenarios: %d %s", code, out)
	}
	as("kim")
	if code, out := r.do(t, "GET", "/v1/admin/proposals", ""); code != 200 || !strings.Contains(out, `"scenarios":[{"question":"When is the quokka close?","current":false,"proposed":true}`) || strings.Contains(out, "Tuesday") {
		t.Fatalf("the operator sees each scenario's result and no answer: %d %s", code, out)
	}

	// SB-4905: the brief cites close.md; a new version redrafts it once today.
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	as("sam")
	drafts := func() []ask.StewardItem {
		t.Helper()
		code, out := r.do(t, "GET", "/v1/spaces/finance/questions", "")
		var q struct{ Items []ask.StewardItem }
		if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
			t.Fatalf("queue: %d %s", code, out)
		}
		var ds []ask.StewardItem
		for _, it := range q.Items {
			if it.Thread != nil && it.Thread.Agent != "" {
				ds = append(ds, it)
			}
		}
		return ds
	}
	if ds := drafts(); len(ds) != 1 || !strings.Contains(ds[0].Thread.Text, "Tuesday") {
		t.Fatalf("the brief: %+v", ds)
	}
	time.Sleep(10 * time.Millisecond)
	write("Finance/close.md", "This week in this space the quokka close moves to Thursday.\n")
	sync()
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	ds := drafts()
	var redraft *ask.StewardItem
	for i := range ds {
		if strings.HasPrefix(ds[i].Question, "Source changed: ") {
			redraft = &ds[i]
		}
	}
	if len(ds) != 2 || redraft == nil || !strings.Contains(redraft.Thread.Text, "Thursday") || len(redraft.Thread.Versions) != 1 ||
		!strings.Contains(redraft.Thread.Versions[0], "Tuesday") || len(redraft.Thread.Changed) != 1 || redraft.Thread.Changed[0] != "docs:Finance/close.md" ||
		!strings.Contains(strings.Join(redraft.Thread.Diff, "\n"), "+ ") {
		t.Fatalf("the redraft: %+v", ds)
	}

	// A second change the same day waits; then sam loses the document and the redraft drops.
	time.Sleep(10 * time.Millisecond)
	write("Finance/close.md", "This week in this space the quokka close moves to Friday.\n")
	sync()
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := len(drafts()); got != 2 {
		t.Fatalf("a second change the same day redrafted again: %d drafts", got)
	}
	latest := func() ask.StewardQuestion {
		t.Helper()
		q, ok, err := r.cell.Parts().Stewards.Store.GetStewardQuestion(ctx, "acme-logistics", redraft.ID)
		if err != nil || !ok {
			t.Fatal(ok, err)
		}
		return q
	}
	if q := latest(); len(q.Thread.Pending) != 1 {
		t.Fatalf("the change isn't waiting for tomorrow: %+v", q.Thread)
	}
	time.Sleep(10 * time.Millisecond)
	write("Finance/close.md", "This week in this space the quokka close moves to Monday.\n")
	write("Finance/close.md.acl", "read lee@acme.example\n")
	sync()
	if _, err := r.cell.NotifyJob(&agentsTold{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if q := latest(); len(q.Thread.Pending) != 0 || len(drafts()) != 2 {
		t.Fatalf("a source sam can't open didn't drop the redraft: %+v", q.Thread)
	}
	if rows := r.auditRows(t); !strings.Contains(rows, `"state":"source_redrafted"`) || !strings.Contains(rows, `"state":"source_dropped"`) {
		t.Fatal("the redraft and the drop aren't audited")
	}

	// SB-4906: lee and kim both found the same ask helpful; sam sees it as a candidate.
	for _, who := range []string{"lee", "kim"} {
		as(who)
		code, body := r.do(t, "POST", "/v1/ask?space=finance", `{"question":"Is the quokka close bonus held?"}`)
		var a struct{ Trace string }
		if err := json.Unmarshal([]byte(body), &a); err != nil || code != 200 || a.Trace == "" {
			t.Fatalf("ask as %s: %d %s", who, code, body)
		}
		if code, body := r.do(t, "POST", "/v1/feedback", `{"trace":"`+a.Trace+`","rating":"up"}`); code != 200 {
			t.Fatalf("feedback: %d %s", code, body)
		}
	}
	as("sam")
	code, out = r.do(t, "GET", "/v1/spaces/finance/suggested/candidates", "")
	if code != 200 || !strings.Contains(out, `{"question":"Is the quokka close bonus held?","from":"helpful","people":2}`) {
		t.Fatalf("candidates: %d %s", code, out)
	}
	for _, id := range []string{ids["lee"], ids["kim"], "lee@", "kim@"} {
		if strings.Contains(out, id) {
			t.Fatalf("a candidate names its asker: %s", out)
		}
	}
	twelve := `{"questions":["Is the quokka close bonus held?","2","3","4","5","6","7","8","9","10","11","12"]}`
	if code, out := r.do(t, "PUT", "/v1/spaces/finance/suggested", strings.Replace(twelve, `"12"]`, `"12","13"]`, 1)); code != http.StatusBadRequest {
		t.Fatalf("13 starter questions: %d %s", code, out)
	}
	if code, out := r.do(t, "PUT", "/v1/spaces/finance/suggested", twelve); code != 200 {
		t.Fatalf("12 starter questions: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/spaces/finance/suggested/candidates", ""); code != 200 || strings.Contains(out, "bonus") {
		t.Fatalf("a published starter is still a candidate: %d %s", code, out)
	}
	as("lee")
	if code, out := r.do(t, "PUT", "/v1/spaces/finance/suggested", `{"questions":["x"]}`); code != http.StatusForbidden {
		t.Fatalf("a member who isn't a steward published starters: %d %s", code, out)
	}
	if code, out := r.do(t, "GET", "/v1/spaces/finance/suggested/candidates", ""); code != http.StatusForbidden {
		t.Fatalf("a member who isn't a steward read the candidates: %d %s", code, out)
	}
	_, body := r.do(t, "POST", "/v1/tokens", `{"name":"assistant","days":30}`)
	var made struct{ Token string }
	_ = json.Unmarshal([]byte(body), &made)
	req, _ := http.NewRequest("POST", r.srv.URL+"/v1/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"prompts/list"}`))
	req.Header.Set("Authorization", "Bearer "+made.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.idp.Server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(b), `"name":"finance/start-1"`) || !strings.Contains(string(b), "Is the quokka close bonus held?") {
		t.Fatalf("MCP prompts/list: %s", b)
	}

	// Each answer is composed fresh, as the person: sam can open bonus.md and lee can't.
	answer := func(who string) string {
		t.Helper()
		as(who)
		code, body := r.do(t, "POST", "/v1/ask?space=finance", `{"question":"Is the quokka close bonus held?"}`)
		if code != 200 {
			t.Fatalf("ask as %s: %d %s", who, code, body)
		}
		return body
	}
	if sam, lee := answer("sam"), answer("lee"); !strings.Contains(sam, "bonus.md") || strings.Contains(lee, "bonus.md") {
		t.Fatalf("the starter's answers don't follow access:\nsam %s\nlee %s", sam, lee)
	}
}
