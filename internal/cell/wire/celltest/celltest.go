// Package celltest builds one local cell's files for tests (SB-1502): a manifest, the
// platform and tenant policies, a vault and empty state, and the configuration over them.
package celltest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
)

const tenant = "acme-logistics"

func here() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Dir(f)
}

// Manifest is the fixture tenant: docs, chat and tickets, one approval-gated workflow.
const Manifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: gdrive, chat: slack, tickets: linear }
agents: [ask, draft-reply]
workflows: [ticket-update]
approval: { ticket-update: group:it-leads }
retention_days: 365
budget_usd_month: 1500
`

func root(t *testing.T) string {
	t.Helper()
	d, err := filepath.Abs(filepath.Join(here(), "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Write writes one file, making its directory.
func Write(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	b, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	Write(t, to, string(b), 0o600)
}

// Fixture is one local cell's files: a manifest, policies, a vault and empty state.
func Fixture(t *testing.T, idp *oidctest.IdP, base string) wire.Config {
	t.Helper()
	return FixtureFor(t, idp, base, Manifest, nil)
}

// FixtureFor is Fixture with another manifest and more vault secrets (path -> value, under
// vault://tenants/acme-logistics/).
func FixtureFor(t *testing.T, idp *oidctest.IdP, base, manifest string, more map[string]string) wire.Config {
	t.Helper()
	dir := t.TempDir()
	Write(t, filepath.Join(dir, "tenant.yaml"), manifest, 0o600)
	for _, f := range []string{"base.cedar", "spaces.cedar"} {
		copyFile(t, filepath.Join(root(t), "policy", "platform", f), filepath.Join(dir, "policy", "platform", f))
	}
	copyFile(t, filepath.Join(root(t), "examples", "policies", "acme-logistics.cedar"), filepath.Join(dir, "policy", "acme-logistics.cedar"))
	secrets := filepath.Join(dir, "vault", "vault", "tenants", tenant)
	Write(t, filepath.Join(secrets, "oidc"), idp.Secret, 0o600)
	Write(t, filepath.Join(secrets, "scim"), "scim-token", 0o600)
	Write(t, filepath.Join(secrets, "connectors", "chat"), `{"bot_token":"xoxb-test","signing_secret":"sign"}`, 0o600)
	for p, v := range more {
		Write(t, filepath.Join(secrets, filepath.FromSlash(p)), v, 0o600)
	}
	if err := os.MkdirAll(filepath.Join(dir, "state"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg, err := wire.Parse([]byte(`{"tenant": "acme-logistics", "manifest": "` + filepath.ToSlash(filepath.Join(dir, "tenant.yaml")) + `",
		"state": "` + filepath.ToSlash(filepath.Join(dir, "state")) + `", "vault": "` + filepath.ToSlash(filepath.Join(dir, "vault")) + `",
		"policies": "` + filepath.ToSlash(filepath.Join(dir, "policy")) + `", "listen": {"api": "127.0.0.1:0"}, "base_url": "` + base + `",
		"oidc": {"client_id": "` + idp.ClientID + `", "issuer": "` + idp.Issuer() + `"}, "model": {"provider": "extractive"},
		"chat": {"team_id": "T1"}}`))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
