package cell

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-6006 through a running cell: a token names the AI tool it is for; with mcp_clients
// only a listed tool's token can be made and use MCP; My data lists the person's tools with
// their last use and families; a revoked tool's next call is refused; the admin's counts
// say how many tokens, people and asks per tool and never who.
func TestAIToolsThroughARunningCell(t *testing.T) {
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.Fixture(t, idp, base)
		cfg.MCP = true
		cfg.MCPClients = []string{"claude-code", "cursor"}
		return cfg
	})
	r.signIn(t, "ann")
	ann := r.idp.User
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"sb-operators","members":[{"value":"`+ann+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatal("sign-in")
	}
	make := func(body string) (int, string) {
		code, out := r.do(t, "POST", "/v1/tokens", body)
		var m struct{ Token, ID string }
		_ = json.Unmarshal([]byte(out), &m)
		return code, m.Token + " " + m.ID
	}
	if code, _ := make(`{"name":"my editor","days":30,"client":"vim-ai"}`); code == http.StatusOK {
		t.Fatal("a token for an unlisted tool was made")
	}
	if code, _ := make(`{"name":"no tool","days":30}`); code == http.StatusOK {
		t.Fatal("with mcp_clients, a token without a tool was made")
	}
	code, both := make(`{"name":"laptop","days":30,"client":"claude-code"}`)
	if code != http.StatusOK {
		t.Fatalf("claude-code token: %d", code)
	}
	tok, id, _ := strings.Cut(both, " ")
	mcp := func(msg string) string {
		t.Helper()
		req, _ := http.NewRequest("POST", r.srv.URL+"/v1/mcp", strings.NewReader(msg))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		resp, err := r.idp.Server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	if b := mcp(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ask","arguments":{"question":"What is our refund limit?"}}}`); strings.Contains(b, `"error"`) {
		t.Fatalf("ask from claude-code: %s", b)
	}
	code, out := r.do(t, "GET", "/v1/me/ai-tools", "")
	if code != 200 || !strings.Contains(out, `"client":"claude-code"`) || !strings.Contains(out, `"families":["search","ask"`) {
		t.Fatalf("my AI tools: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/admin/ai/clients", "")
	if code != 200 || !strings.Contains(out, `{"client":"claude-code","tokens":1,"people":1,"calls":1}`) || strings.Contains(out, ann) {
		t.Fatalf("admin's counts: %d %s", code, out)
	}
	if code, out := r.do(t, "DELETE", "/v1/tokens/"+id, ""); code != 200 {
		t.Fatalf("revoke: %d %s", code, out)
	}
	if b := mcp(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`); strings.Contains(b, `"tools"`) {
		t.Fatalf("a revoked tool's call worked: %s", b)
	}
}
