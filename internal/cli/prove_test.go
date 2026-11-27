package cli

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/msgraph/graphtest"
)

// proveFixture is a cell configuration on a recorded Graph and a vault with its secrets.
func proveFixture(t *testing.T, f *graphtest.Fake) (cfg, vault, state string) {
	t.Helper()
	dir := t.TempDir()
	vault = filepath.Join(dir, "vault")
	state = filepath.Join(dir, "state")
	sec := filepath.Join(vault, "vault", "tenants", "acme-logistics")
	if err := os.MkdirAll(filepath.Join(sec, "audit"), 0o700); err != nil {
		t.Fatal(err)
	}
	seed := base64.StdEncoding.EncodeToString(make([]byte, 32))
	for p, v := range map[string]string{"msgraph": graphtest.Creds, "audit/anchor-seed": seed} {
		if err := os.WriteFile(filepath.Join(sec, p), []byte(v), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c := strings.NewReplacer("VAULT", vault, "STATE", state, "BASE", f.Base(), "LOGIN", f.Login()).Replace(`{
  "tenant": "acme-logistics", "manifest": "STATE/tenant.yaml", "state": "STATE", "vault": "VAULT",
  "policies": "/etc/sb/policy", "listen": { "api": "127.0.0.1:0" }, "base_url": "https://brain.acme-logistics.example",
  "oidc": { "client_id": "sb", "issuer": "https://login.acme-logistics.example" }, "model": { "provider": "extractive" },
  "microsoft": { "base": "BASE", "login": "LOGIN", "domain": "acme.example", "drives": ["d1"] },
  "labels": { "microsoft": { "lbl-conf": "confidential" } }
}`)
	cfg = filepath.Join(dir, "cell.json")
	if err := os.WriteFile(cfg, []byte(c), 0o600); err != nil {
		t.Fatal(err)
	}
	return cfg, vault, state
}

// SB-6001: a proof against the recorded Graph passes and verifies; it holds no text, name or
// unhashed id; a tampered proof doesn't verify; a criterion that fails says which.
func TestProveSharePoint(t *testing.T) {
	f := graphtest.New()
	t.Cleanup(f.Server.Close)
	f.Mails["g-fin"] = "finance@acme.example"
	f.Groups["g-fin"] = []string{"sam@acme.example"}
	f.Put("d1", graphtest.Item{ID: "close-quokka", Name: "close.md", Path: "/Finance", Owner: "ann@acme.example", Label: "lbl-conf",
		Content: []byte("The quokka close moves to Tuesday."), Groups: []string{"g-fin"}, Links: []string{"organization"}})
	f.Put("d1", graphtest.Item{ID: "draft-wombat", Name: "draft.md", Path: "/Finance", Label: "lbl-unknown", Content: []byte("A wombat draft."), Users: []string{"sam@acme.example"}})
	cfg, _, state := proveFixture(t, f)

	code, out, errOut := run("prove", "SB-5901", "--config", cfg, "--person", "sam@acme.example")
	if code != 0 || !strings.Contains(out, "passed: true") {
		t.Fatalf("prove: %d %q %q", code, out, errOut)
	}
	files, _ := filepath.Glob(filepath.Join(state, "proofs", "SB-5901-*.json"))
	if len(files) != 1 {
		t.Fatalf("proof files: %v", files)
	}
	b, _ := os.ReadFile(files[0])
	for _, s := range []string{"quokka", "wombat", "close", "draft", "ann@", "sam@", "finance", "Finance", "d1\"", "lbl-"} {
		if strings.Contains(string(b), s) {
			t.Fatalf("the proof holds %q:\n%s", s, b)
		}
	}
	if code, out, errOut := run("prove", "--verify", files[0]); code != 0 || !strings.Contains(out, "verifies") {
		t.Fatalf("verify: %d %q %q", code, out, errOut)
	}
	var doc map[string]any
	_ = json.Unmarshal(b, &doc)
	doc["proof"].(map[string]any)["passed"] = false
	bad, _ := json.Marshal(doc)
	tampered := filepath.Join(t.TempDir(), "p.json")
	_ = os.WriteFile(tampered, bad, 0o600)
	if code, _, _ := run("prove", "--verify", tampered); code == 0 {
		t.Fatal("a tampered proof verified")
	}

	// Without a person the live check can't run: that criterion fails, by name.
	code, out, _ = run("prove", "SB-5901", "--config", cfg)
	if code == 0 || !strings.Contains(out, "fail  the live check answers for a person") || !strings.Contains(out, "passed: false") {
		t.Fatalf("no person: %d %q", code, out)
	}
	if code, _, _ := run("prove", "SB-9999", "--config", cfg); code == 0 {
		t.Fatal("an uncovered ticket ran")
	}
}

// SB-6001 and SB-5907: a named shared mailbox reads, and a mailbox the access policy should
// refuse is refused; a probe that can be read fails the proof.
func TestProveMailbox(t *testing.T) {
	f := graphtest.New()
	t.Cleanup(f.Server.Close)
	f.Boxes["support@acme.example"] = []graphtest.Message{{ID: "m1", From: "a@client.example", Subject: "invoice quokka", Body: "please"}}
	cfg, _, _ := proveFixture(t, f)
	code, out, errOut := run("prove", "SB-5907", "--config", cfg, "--mailbox", "support@acme.example", "--probe", "ann@acme.example")
	if code != 0 || strings.Contains(out, "quokka") {
		t.Fatalf("mailbox: %d %q %q", code, out, errOut)
	}
	f.Boxes["ann@acme.example"] = nil
	if code, out, _ := run("prove", "SB-5907", "--config", cfg, "--mailbox", "support@acme.example", "--probe", "ann@acme.example"); code == 0 || !strings.Contains(out, "fail  a mailbox not named is refused") {
		t.Fatalf("a readable probe passed: %d %q", code, out)
	}
	if code, out, _ := run("prove", "--list"); code != 0 || !strings.Contains(out, "SB-5901") || !strings.Contains(out, "SB-2603") {
		t.Fatalf("list: %d %q", code, out)
	}
}
