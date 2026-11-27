package cell

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-3801 to SB-3803 through a running cell: another AI tool searches and asks as the
// token's person over MCP; a file is asked about once and kept nowhere; an admin lists
// every token and revokes one, and MCP refuses it at once.
func TestWorkWherePeopleAreThroughARunningCell(t *testing.T) {
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.Fixture(t, idp, base)
		cfg.MCP = true
		return cfg
	})
	ctx := context.Background()
	r.signIn(t, "ann")
	ann := r.idp.User

	// SB-3802: a file, cited as the person's own, never in the index or the trace.
	secret := "The zebra surcharge for gamma-parts is 7 percent."
	file := base64.StdEncoding.EncodeToString([]byte(secret))
	code, body := r.do(t, "POST", "/v1/ask", `{"question":"What is the zebra surcharge?","file":{"name":"quote.txt","data":"`+file+`"}}`)
	var a struct {
		Trace  string
		Answer struct {
			Claims []struct {
				Cites []struct{ Cite, Title string }
			}
		}
	}
	if err := json.Unmarshal([]byte(body), &a); err != nil || code != 200 {
		t.Fatalf("ask about a file: %d %s", code, body)
	}
	cited := false
	for _, c := range a.Answer.Claims {
		for _, ci := range c.Cites {
			cited = cited || ci.Cite == "file:quote.txt" && ci.Title == "Your file: quote.txt"
		}
	}
	if !cited {
		t.Fatalf("the file isn't cited: %s", body)
	}
	tr, found, err := r.cell.Traces.GetTrace(ctx, tenant, a.Trace)
	if err != nil || !found || tr.Files != 1 {
		t.Fatalf("trace: %v %v %+v", err, found, tr.Files)
	}
	if code, body := r.do(t, "GET", "/v1/traces/"+a.Trace+"/passage?cite=file:quote.txt", ""); code != 200 || !strings.Contains(body, `"gone":true`) || strings.Contains(body, secret) {
		t.Fatalf("the file's passage later: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/search?q=zebra+surcharge", ""); code != 200 || strings.Contains(body, "zebra surcharge") {
		t.Fatalf("the file was indexed: %d %s", code, body)
	}
	big := base64.StdEncoding.EncodeToString(make([]byte, 3<<20))
	if code, _ := r.do(t, "POST", "/v1/ask", `{"question":"What is it?","file":{"name":"big.txt","data":"`+big+`"}}`); code != http.StatusBadRequest {
		t.Fatalf("a 3 MB file: %d", code)
	}
	if code, _ := r.do(t, "POST", "/v1/ask", `{"question":"What is it?","file":{"name":"run.exe","data":"`+file+`"}}`); code != http.StatusBadRequest {
		t.Fatalf("a file of another type: %d", code)
	}

	// SB-3801: MCP with the person's own token.
	_, body = r.do(t, "POST", "/v1/tokens", `{"name":"assistant","days":30}`)
	var made struct{ ID, Token string }
	_ = json.Unmarshal([]byte(body), &made)
	mcp := func(token, msg string) (int, string) {
		t.Helper()
		req, _ := http.NewRequest("POST", r.srv.URL+"/v1/mcp", strings.NewReader(msg))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := r.idp.Server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if c, b := mcp(made.Token, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`); c != 200 || !strings.Contains(b, `"protocolVersion":"2025-06-18"`) {
		t.Fatalf("initialize: %d %s", c, b)
	}
	if c, _ := mcp(made.Token, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); c != http.StatusAccepted {
		t.Fatalf("a notification: %d", c)
	}
	c, b := mcp(made.Token, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	for _, name := range []string{`"name":"search"`, `"name":"ask"`, `"name":"passage"`, `"readOnlyHint":true`} {
		if c != 200 || !strings.Contains(b, name) {
			t.Fatalf("tools/list lacks %s: %d %s", name, c, b)
		}
	}
	if c, b := mcp(made.Token, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":"refund"}}}`); c != 200 || !strings.Contains(b, `"structuredContent"`) || strings.Contains(b, `"isError"`) {
		t.Fatalf("search: %d %s", c, b)
	}
	c, b = mcp(made.Token, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"ask","arguments":{"question":"What is our refund limit?"}}}`)
	var asked struct {
		Result struct {
			StructuredContent struct{ Trace string }
		}
	}
	if c != 200 || json.Unmarshal([]byte(b), &asked) != nil || asked.Result.StructuredContent.Trace == "" {
		t.Fatalf("ask: %d %s", c, b)
	}
	if t2, found, _ := r.cell.Traces.GetTrace(ctx, tenant, asked.Result.StructuredContent.Trace); !found || t2.User != ann {
		t.Fatalf("the MCP ask wasn't the token's person's: %+v", t2.User)
	}
	if c, b := mcp(made.Token, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"passage","arguments":{"trace":"tr_nope","cite":"docs:x@1"}}}`); c != 200 || !strings.Contains(b, `"isError":true`) {
		t.Fatalf("a passage of someone else's trace: %d %s", c, b)
	}
	if c, b := mcp(made.Token, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"delete","arguments":{}}}`); c != 200 || !strings.Contains(b, `"code":-32602`) {
		t.Fatalf("a tool that doesn't exist: %d %s", c, b)
	}
	if c, b := mcp(made.Token, `{"jsonrpc":"2.0","id":7,"method":"completion/complete"}`); c != 200 || !strings.Contains(b, `"code":-32601`) { // resources/list is answered since SB-5505
		t.Fatalf("another method: %d %s", c, b)
	}
	rows, _ := r.cell.Audit.Store.Range(ctx, 1, 1<<20)
	used := 0
	for _, row := range rows {
		if row.Subject == "token:"+made.ID && row.State == "token_used" {
			used++
		}
		if raw, _ := json.Marshal(row); strings.Contains(string(raw), "zebra surcharge") {
			t.Fatalf("the file's text in an audit row: %s", raw)
		}
	}
	if used < 7 {
		t.Fatalf("%d audited MCP uses", used)
	}

	// SB-3803: the admin's list, and a revoke MCP feels at once.
	if code, _ := r.do(t, "GET", "/v1/admin/tokens", ""); code != http.StatusForbidden {
		t.Fatalf("a non-admin listed tokens: %d", code)
	}
	r.operator(t)
	code, body = r.do(t, "GET", "/v1/admin/tokens", "")
	if code != 200 || !strings.Contains(body, `"id":"`+made.ID+`"`) || !strings.Contains(body, `"owner":"`+ann+`"`) || strings.Contains(body, made.Token) || strings.Contains(body, "tokh_") {
		t.Fatalf("admin list: %d %s", code, body)
	}
	if code, _ := r.do(t, "DELETE", "/v1/admin/tokens/"+made.ID, `{"reason":""}`); code != http.StatusBadRequest {
		t.Fatalf("a revoke with no reason: %d", code)
	}
	if code, body := r.do(t, "DELETE", "/v1/admin/tokens/"+made.ID, `{"reason":"left the team"}`); code != 200 {
		t.Fatalf("admin revoke: %d %s", code, body)
	}
	if c, _ := mcp(made.Token, `{"jsonrpc":"2.0","id":8,"method":"tools/list"}`); c != http.StatusUnauthorized {
		t.Fatalf("a revoked token on MCP: %d", c)
	}
	if code, _ := r.do(t, "DELETE", "/v1/admin/tokens/"+made.ID, `{"reason":"again"}`); code != http.StatusBadRequest {
		t.Fatalf("revoking a gone token: %d", code)
	}
	rows, _ = r.cell.Audit.Store.Range(ctx, 1, 1<<20)
	revoked := false
	for _, row := range rows {
		revoked = revoked || row.Subject == "token:"+made.ID && row.State == "token_revoked" && row.Reason == "left the team"
	}
	if !revoked {
		t.Fatal("no token_revoked row")
	}
}

// Without mcp: true the endpoint isn't served.
func TestMCPOffByDefault(t *testing.T) {
	r := start(t)
	r.signIn(t, "ann")
	if code, _ := r.do(t, "POST", "/v1/mcp", `{"jsonrpc":"2.0","id":1,"method":"ping"}`); code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		t.Fatalf("MCP without mcp: true: %d", code)
	}
}

// SB-3805: the daily brief is a setting a person chooses and can take back.
func TestDailyBriefSetting(t *testing.T) {
	r := start(t)
	r.signIn(t, "ann")
	if code, body := r.do(t, "PUT", "/v1/me/notifications", `{"off":[],"digest":"daily"}`); code != 200 || !strings.Contains(body, `"digest":"daily"`) {
		t.Fatalf("daily: %d %s", code, body)
	}
	if code, body := r.do(t, "PUT", "/v1/me/notifications", `{"off":["watch"]}`); code != 200 || !strings.Contains(body, `"digest":"daily"`) {
		t.Fatalf("kept without digest: %d %s", code, body)
	}
	if code, _ := r.do(t, "PUT", "/v1/me/notifications", `{"off":[],"digest":"hourly"}`); code != http.StatusBadRequest {
		t.Fatalf("hourly: %d", code)
	}
	if code, body := r.do(t, "PUT", "/v1/me/notifications", `{"off":[],"digest":"weekly"}`); code != 200 || strings.Contains(body, `"digest"`) {
		t.Fatalf("weekly: %d %s", code, body)
	}
}
