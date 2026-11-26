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

// SB-5305 through a running cell: with mcp_groups, a member of a listed group sees and calls
// only the tool families it grants, someone in no listed group is refused on every method,
// and the AI command centre counts the MCP ask under its own client.
func TestManagedMCPAccessThroughARunningCell(t *testing.T) {
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.Fixture(t, idp, base)
		cfg.MCP = true
		cfg.MCPGroups = map[string][]string{"analysts": {"ask"}}
		return cfg
	})
	r.signIn(t, "bea")
	bea := r.idp.User
	r.signIn(t, "ann")
	ann := r.idp.User
	for name, who := range map[string][]string{"analysts": {ann}, "sb-operators": {ann}} {
		var ms []string
		for _, x := range who {
			ms = append(ms, `{"value":"`+x+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	token := func(user string) string {
		t.Helper()
		r.idp.User = user
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
		_, body := r.do(t, "POST", "/v1/tokens", `{"name":"assistant","days":30}`)
		var made struct{ Token string }
		if err := json.Unmarshal([]byte(body), &made); err != nil || made.Token == "" {
			t.Fatalf("token: %s", body)
		}
		return made.Token
	}
	mcp := func(tok, msg string) string {
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
	beaTok, annTok := token(bea), token(ann)
	if b := mcp(beaTok, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`); !strings.Contains(b, "EPERM") {
		t.Fatalf("someone in no listed group used MCP: %s", b)
	}
	b := mcp(annTok, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if !strings.Contains(b, `"name":"ask"`) || !strings.Contains(b, `"name":"passage"`) || strings.Contains(b, `"name":"search"`) {
		t.Fatalf("tools/list: %s", b)
	}
	if b := mcp(annTok, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":"refund"}}}`); !strings.Contains(b, `"code":-32602`) {
		t.Fatalf("a tool not granted was called: %s", b)
	}
	if b := mcp(annTok, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"ask","arguments":{"question":"What is our refund limit?"}}}`); strings.Contains(b, `"error"`) {
		t.Fatalf("ask: %s", b)
	}
	if b := mcp(annTok, `{"jsonrpc":"2.0","id":5,"method":"prompts/list"}`); !strings.Contains(b, `"prompts":[]`) {
		t.Fatalf("prompts without skills: %s", b)
	}
	r.idp.User = ann
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	code, out := r.do(t, "GET", "/v1/admin/ai/usage", "")
	if code != 200 || !strings.Contains(out, `{"client":"mcp","uses":1`) {
		t.Fatalf("usage by client: %d %s", code, out)
	}
}
