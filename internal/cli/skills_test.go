package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SB-4601: sb agents lint --manifest refuses a skill that names a tool, with its line, and
// passes a format-only one.
func TestAgentsLintManifestSkills(t *testing.T) {
	head := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  ops:
    members: [group:ops]
    stewards: [group:ops-leads]
    scope: { docs: ["/Ops/**"] }
    skills:
      weekly-update: { instruction: "%s", cases: { "a?": "docs:a.md", "b?": "docs:b.md", "c?": "docs:c.md" } }
`
	dir := t.TempDir()
	write := func(instruction string) string {
		p := filepath.Join(dir, "tenant.yaml")
		if err := os.WriteFile(p, []byte(strings.Replace(head, "%s", instruction, 1)), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	p := write("Use the note-publish tool to post it.")
	c, out, _ := run("agents", "lint", "--root", "../..", "--manifest", p)
	if c != 1 || !strings.Contains(out, p+":13: skill: spaces.ops.skills.weekly-update.instruction: the instruction names a tool") {
		t.Fatalf("lint: %d %q", c, out)
	}
	if c, out, _ := run("agents", "lint", "--root", "../..", "--manifest", write("Three headings: Shipped, Decided, At risk.")); c != 0 {
		t.Fatalf("a format-only skill: %d %q", c, out)
	}
}
