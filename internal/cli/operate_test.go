package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/smtpmail"
)

// SB-2601: the mail server answers and offers STARTTLS with a certificate --ca trusts.
func TestSelfhostCheckEmail(t *testing.T) {
	f, err := smtpmail.NewFake()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ca := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(ca, f.CertPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := run("selfhost", "check-email", f.Addr, "--ca", ca); code != 0 || !strings.Contains(out, "offers STARTTLS") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
	if code, _, _ := run("selfhost", "check-email", f.Addr); code == 0 {
		t.Fatal("a certificate nothing trusts passed")
	}
	f.NoTLS = true
	if code, _, errOut := run("selfhost", "check-email", f.Addr, "--ca", ca); code == 0 || !strings.Contains(errOut, "doesn't offer STARTTLS") {
		t.Fatalf("no STARTTLS: %d %q", code, errOut)
	}
	if len(f.Messages()) != 0 {
		t.Fatal("the check sent mail")
	}
}

// SB-2603: each missing Slack scope is named with where it is granted; the token never
// appears in the output.
func TestSelfhostCheckChatSlack(t *testing.T) {
	for _, missing := range []string{"", "commands", "chat:write", "im:write", "users:read.email"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/auth.test" || r.Header.Get("Authorization") != "Bearer xoxb-secret-token" {
				_, _ = io.WriteString(w, `{"ok":false,"error":"invalid_auth"}`)
				return
			}
			var have []string
			for _, s := range []string{"commands", "chat:write", "im:write", "users:read.email", "users:read"} {
				if s != missing {
					have = append(have, s)
				}
			}
			w.Header().Set("X-OAuth-Scopes", strings.Join(have, ","))
			_, _ = io.WriteString(w, `{"ok":true,"team":"acme-logistics","user":"brain"}`)
		}))
		tok := filepath.Join(t.TempDir(), "chat")
		_ = os.WriteFile(tok, []byte(`{"bot_token":"xoxb-secret-token","signing_secret":"s"}`), 0o600)
		code, out, errOut := run("selfhost", "check-chat", "--slack", tok, "--api", srv.URL+"/api")
		srv.Close()
		if strings.Contains(out+errOut, "xoxb-secret-token") {
			t.Fatal("the token was printed")
		}
		if missing == "" {
			if code != 0 || strings.Contains(out, "missing") {
				t.Fatalf("all scopes: %d %q %q", code, out, errOut)
			}
			continue
		}
		if code != 1 || !strings.Contains(out, "missing "+missing+" (") || !strings.Contains(out, "api.slack.com/apps") {
			t.Fatalf("without %s: %d %q", missing, code, out)
		}
	}
	tok := filepath.Join(t.TempDir(), "bad")
	_ = os.WriteFile(tok, []byte("xoxb-wrong"), 0o600)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"ok":false,"error":"invalid_auth"}`)
	}))
	defer srv.Close()
	if code, out, errOut := run("selfhost", "check-chat", "--slack", tok, "--api", srv.URL); code == 0 || strings.Contains(out+errOut, "xoxb-wrong") || !strings.Contains(errOut, "invalid_auth") {
		t.Fatalf("a refused token: %d %q %q", code, out, errOut)
	}
}

// SB-2603: a Mattermost bot outside the team, or without people's emails, is told where to
// fix each.
func TestSelfhostCheckChatMattermost(t *testing.T) {
	for _, c := range []struct {
		name, teams, users, want string
	}{
		{"all", `[{"id":"team1"}]`, `[{"id":"bot1"},{"id":"u1","email":"ann@acme.example"}]`, ""},
		{"no team", `[]`, `[{"id":"u1","email":"ann@acme.example"}]`, "missing team membership"},
		{"other team", `[{"id":"team2"}]`, `[{"id":"u1","email":"ann@acme.example"}]`, "missing team membership"},
		{"no emails", `[{"id":"team1"}]`, `[{"id":"bot1","email":"bot@x"},{"id":"u1","email":""}]`, "Show Email Address"},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer mm-secret" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			switch r.URL.Path {
			case "/api/v4/users/me":
				_, _ = io.WriteString(w, `{"id":"bot1","username":"brain","is_bot":true}`)
			case "/api/v4/users/me/teams":
				_, _ = io.WriteString(w, c.teams)
			case "/api/v4/users":
				_, _ = io.WriteString(w, c.users)
			}
		}))
		tok := filepath.Join(t.TempDir(), "mm")
		_ = os.WriteFile(tok, []byte(`{"command_token":"c","bot_token":"mm-secret"}`), 0o600)
		code, out, errOut := run("selfhost", "check-chat", "--mattermost", srv.URL, "--token-file", tok, "--team", "team1")
		srv.Close()
		if strings.Contains(out+errOut, "mm-secret") {
			t.Fatal("the token was printed")
		}
		if c.want == "" && (code != 0 || strings.Contains(out, "missing")) || c.want != "" && (code != 1 || !strings.Contains(out, c.want)) {
			t.Errorf("%s: %d %q %q", c.name, code, out, errOut)
		}
	}
}

const goodConfig = `{
  "tenant": "acme-logistics",
  "manifest": "/var/lib/sb/state/tenant.yaml",
  "state": "/var/lib/sb/state",
  "vault": "VAULT",
  "policies": "/etc/sb/policy",
  "listen": { "api": "0.0.0.0:8080" },
  "base_url": "https://brain.acme-logistics.example",
  "oidc": { "client_id": "sb", "issuer": "https://login.acme-logistics.example" },
  "model": { "provider": "extractive" },
  "database": "vault://tenants/acme-logistics/database"
}
`

// SB-2604: a typo, a missing secret and a world-readable one each fail, naming the fault;
// no secret is read or printed.
func TestCellCheckConfig(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	sec := filepath.Join(vault, "vault", "tenants", "acme-logistics")
	for _, p := range []string{"oidc", "scim", "control-plane", "audit/anchor-seed", "attest-seed", "database"} {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(sec, p)), 0o700)
		if err := os.WriteFile(filepath.Join(sec, p), []byte("SECRETVALUE-"+p), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := filepath.Join(dir, "cell.json")
	write := func(s string) {
		if err := os.WriteFile(cfg, []byte(strings.ReplaceAll(s, "VAULT", vault)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(goodConfig)
	code, out, errOut := run("cell", "check-config", cfg)
	if code != 0 || !strings.Contains(out, "6 secrets") {
		t.Fatalf("good: %d %q %q", code, out, errOut)
	}
	write(strings.Replace(goodConfig, `"base_url"`, `"base_ulr"`, 1))
	if code, out, errOut := run("cell", "check-config", cfg); code == 0 || !strings.Contains(errOut, "line 8") || !strings.Contains(errOut, "base_ulr") {
		t.Fatalf("a typo: %d %q %q", code, out, errOut)
	}
	write(strings.Replace(goodConfig, `"model": { "provider": "extractive" },`, `"model": { "provider": "extractive" }`, 1))
	if code, _, errOut := run("cell", "check-config", cfg); code == 0 || !strings.Contains(errOut, "line 1") && !strings.Contains(errOut, "line 11") {
		t.Fatalf("a missing comma: %d %q", code, errOut)
	}
	write(goodConfig)
	_ = os.Remove(filepath.Join(sec, "scim"))
	if code, _, errOut := run("cell", "check-config", cfg); code == 0 || !strings.Contains(errOut, "the secret scim isn't in the vault") {
		t.Fatalf("a missing secret: %d %q", code, errOut)
	}
	_ = os.WriteFile(filepath.Join(sec, "scim"), []byte("SECRETVALUE-scim"), 0o600)
	_ = os.Chmod(filepath.Join(sec, "database"), 0o644)
	code, out, errOut = run("cell", "check-config", cfg, "--vault", vault)
	if code == 0 || !strings.Contains(errOut, "the secret database is readable by others") {
		t.Fatalf("a world-readable secret: %d %q", code, errOut)
	}
	if strings.Contains(out+errOut, "SECRETVALUE") {
		t.Fatal("a secret was printed")
	}
}
