package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTreePlan(t *testing.T) {
	dir := t.TempDir()
	m := filepath.Join(dir, "acme.yaml")
	src := []byte(`tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
spaces:
  customer:
    template: support
    members: [group:support, group:sales]
    stewards: [group:cx-leads]
    glossary: { SLA: "service level agreement" }
    scope: { docs: ["/Customer/**"] }
  ops:
    template: engineering
    members: [group:support, group:eng]
    stewards: [group:ops-leads]
    glossary: { runbook: "steps for an incident" }
    scope: { docs: ["/Ops/**"] }
  support:
    template: support
    parent: ops
    members: [group:support]
    scope: { docs: ["/Support/**"] }
`)
	if err := os.WriteFile(m, src, 0o644); err != nil {
		t.Fatal(err)
	}
	next := filepath.Join(dir, "next.yaml")
	code, out, _ := run("tree", "move", m, "support", "--to", "customer", "--plan", "--write", next)
	if code != 0 || !strings.Contains(out, "support  parent ops -> customer  escalates support > ops > company -> support > customer > company\n  glossary  gains SLA  loses runbook\n") ||
		!strings.HasSuffix(out, "nothing applied; propose the manifest diff for review\n") {
		t.Fatalf("move: %d\n%s", code, out)
	}
	if got, _ := os.ReadFile(m); string(got) != string(src) {
		t.Fatal("the dry run changed the manifest it read")
	}
	if code, out, _ := run("tenant", "check", next); code != 0 {
		t.Fatalf("the written manifest fails the check: %s", out)
	}
	if code, _, _ := run("tree", "merge", m, "support", "customer", "--plan"); code != 3 {
		t.Fatalf("a merge that widens exits %d, want 3", code)
	}
	if code, _, _ := run("tree", "move", m, "support", "--to", "customer"); code != 2 {
		t.Fatalf("a move without --plan exits %d, want 2", code)
	}
	if code, _, _ := run("tree", "move", m, "nope", "--to", "customer", "--plan"); code != 2 {
		t.Fatalf("an unknown space exits %d, want 2", code)
	}
}
