package cell

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// SB-2306: a person makes a read-only API token that asks and searches as them and does
// nothing else; the cell keeps only its hash; every use is audited with its id; revoking
// it, signing out everywhere or a SCIM deactivate ends it at once.
func TestReadOnlyAPITokens(t *testing.T) {
	r := start(t)
	r.signIn(t, "ann")
	ctx := context.Background()
	code, body := r.do(t, "POST", "/v1/tokens", `{"name":"nightly report","days":30}`)
	var made struct{ ID, Token, Name string }
	if code != 200 || json.Unmarshal([]byte(body), &made) != nil || !strings.HasPrefix(made.Token, "sbt_") || !strings.HasPrefix(made.ID, "tok_") {
		t.Fatalf("make: %d %s", code, body)
	}
	// As a script: no cookie, the token only.
	script := func(method, path, b string) int {
		req, _ := http.NewRequest(method, r.srv.URL+path, strings.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+made.Token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := r.idp.Server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if c := script("POST", "/v1/ask", `{"question":"What is our refund limit?"}`); c != 200 {
		t.Fatalf("ask with the token: %d", c)
	}
	if c := script("GET", "/v1/search?q=refund", ""); c != 200 {
		t.Fatalf("search with the token: %d", c)
	}
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/v1/spaces", ""}, {"POST", "/v1/tokens", `{"name":"x"}`}, {"GET", "/v1/tokens", ""},
		{"POST", "/v1/actions/act_1/approve", "{}"}, {"GET", "/v1/admin/kill", ""}, {"POST", "/v1/sessions/revoke", ""},
		{"POST", "/v1/drafts", `{"question":"x"}`}, {"GET", "/v1/ask", ""},
	} {
		if got := script(c.method, c.path, c.body); got != http.StatusForbidden && got != http.StatusMethodNotAllowed {
			t.Errorf("%s %s with a token: %d, want 403", c.method, c.path, got)
		}
	}
	// Only the hash is kept; each use is audited with the token's id, never the token.
	rows, _ := r.cell.Audit.Store.Range(ctx, 1, 1<<20)
	used := 0
	for _, row := range rows {
		b, _ := json.Marshal(row)
		if strings.Contains(string(b), made.Token) {
			t.Fatalf("an audit row holds the token: %s", b)
		}
		if row.Subject == "token:"+made.ID && row.State == "token_used" {
			used++
		}
	}
	if used != 2 {
		t.Fatalf("%d audited uses, want 2", used)
	}
	all, _ := r.cell.Sessions.ListUser(ctx, tenant, r.idp.User)
	for _, x := range all {
		if b, _ := json.Marshal(x); strings.Contains(string(b), made.Token) {
			t.Fatal("the session store holds the token")
		}
	}
	code, body = r.do(t, "GET", "/v1/tokens", "")
	if code != 200 || !strings.Contains(body, made.ID) || strings.Contains(body, made.Token) {
		t.Fatalf("list: %d %s", code, body)
	}
	// Revoked: refused at once.
	if code, body := r.do(t, "DELETE", "/v1/tokens/"+made.ID, ""); code != 200 {
		t.Fatalf("revoke: %d %s", code, body)
	}
	if c := script("POST", "/v1/ask", `{"question":"What is our refund limit?"}`); c != http.StatusUnauthorized {
		t.Fatalf("a revoked token: %d", c)
	}
	// Signing out everywhere ends tokens too.
	_, body = r.do(t, "POST", "/v1/tokens", `{"name":"again"}`)
	_ = json.Unmarshal([]byte(body), &made)
	if c := script("GET", "/v1/search?q=refund", ""); c != 200 {
		t.Fatalf("a new token: %d", c)
	}
	if code, body := r.do(t, "POST", "/v1/sessions/revoke", ""); code != 200 {
		t.Fatalf("sign out everywhere: %d %s", code, body)
	}
	if c := script("GET", "/v1/search?q=refund", ""); c != http.StatusUnauthorized {
		t.Fatalf("a token after signing out everywhere: %d", c)
	}
	// Deprovisioned: refused at once.
	r.signIn(t, "bob")
	_, body = r.do(t, "POST", "/v1/tokens", `{"name":"bob's"}`)
	_ = json.Unmarshal([]byte(body), &made)
	patch := `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"Replace","path":"active","value":false}]}`
	if code, body := r.do(t, "PATCH", "/scim/v2/Users/"+r.idp.User, patch, "Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != 200 {
		t.Fatalf("deactivate: %d %s", code, body)
	}
	if c := script("GET", "/v1/search?q=refund", ""); c != http.StatusUnauthorized {
		t.Fatalf("a deprovisioned person's token: %d", c)
	}
}
