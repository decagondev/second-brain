package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/atlassian/atlassiantest"
	"example.com/second-brain/internal/adapters/msgraph/graphtest"
)

// SB-6203: a read-only account passes; one that can write fails and names the permissions;
// a space or project that doesn't answer fails; every fix is a real runbook section.
func TestCheckAtlassian(t *testing.T) {
	f := atlassiantest.New()
	t.Cleanup(f.Server.Close)
	f.Spaces["FIN"] = false
	f.Projects["OPS"] = nil
	g := graphtest.New()
	t.Cleanup(g.Server.Close)
	cfg, vault, _ := proveFixture(t, g)
	if err := os.WriteFile(filepath.Join(vault, "vault", "tenants", "acme-logistics", "atlassian"), []byte(atlassiantest.Creds), 0o600); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(cfg)
	c := strings.Replace(string(b), `"labels":`, `"atlassian": { "base": "`+f.Base()+`", "spaces": ["FIN"], "projects": ["OPS"] }, "labels":`, 1)
	_ = os.WriteFile(cfg, []byte(c), 0o600)
	code, out, errOut := run("selfhost", "check-atlassian", "--config", cfg)
	if code != 0 || strings.Contains(out, "fail") {
		t.Fatalf("good: %d %q %q", code, out, errOut)
	}
	all := out
	f.Writes = []string{"CREATE_ISSUES", "ADD_COMMENTS"}
	code, out, _ = run("selfhost", "check-atlassian", "--config", cfg)
	if code == 0 || !strings.Contains(out, "ADD_COMMENTS, CREATE_ISSUES") || !strings.Contains(out, "#read-only-account") {
		t.Fatalf("a writing account: %d %q", code, out)
	}
	all += out
	f.Writes = nil
	delete(f.Projects, "OPS")
	if code, out, _ = run("selfhost", "check-atlassian", "--config", cfg); code == 0 || !strings.Contains(out, "OPS doesn't answer") {
		t.Fatalf("a missing project: %d %q", code, out)
	}
	all += out
	md, _ := os.ReadFile("../../docs/runbooks/atlassian.md")
	for _, m := range regexp.MustCompile(`atlassian\.md#([a-z-]+)`).FindAllStringSubmatch(all, -1) {
		if !strings.Contains(string(md), "\n## "+m[1]+"\n") {
			t.Errorf("fix %s isn't a section of the runbook", m[1])
		}
	}
	if strings.Contains(all, "atl-token") {
		t.Fatal("a secret was printed")
	}
}

// SB-6001 for W61: a proof of Confluence, of Atlassian groups and of Jira against the
// recorded site passes and holds none of its names.
func TestProveAtlassian(t *testing.T) {
	f := atlassiantest.New()
	t.Cleanup(f.Server.Close)
	f.User("sam@acme.example")
	f.Groups["finance"] = []string{"sam@acme.example"}
	f.Spaces["FIN"] = false
	f.Pages["1"] = &atlassiantest.Page{ID: "1", Space: "FIN", Title: "Quokka close", Body: "<p>quokka</p>", Version: 1, Groups: []string{"finance"}}
	f.Projects["OPS"] = nil
	f.Issues["OPS-1"] = &atlassiantest.Issue{Key: "OPS-1", Project: "OPS", Summary: "Wombat", Description: "wombat", Status: "Open", Updated: "2026-11-27T10:00:00.000+0000", Created: "2026-11-27T10:00:00.000+0000"}
	g := graphtest.New()
	t.Cleanup(g.Server.Close)
	cfg, vault, state := proveFixture(t, g)
	_ = os.WriteFile(filepath.Join(vault, "vault", "tenants", "acme-logistics", "atlassian"), []byte(atlassiantest.Creds), 0o600)
	b, _ := os.ReadFile(cfg)
	_ = os.WriteFile(cfg, []byte(strings.Replace(string(b), `"labels":`, `"atlassian": { "base": "`+f.Base()+`", "spaces": ["FIN"], "projects": ["OPS"] }, "labels":`, 1)), 0o600)
	for _, ticket := range []string{"SB-6201", "SB-6202", "SB-6204"} {
		if code, out, errOut := run("prove", ticket, "--config", cfg, "--person", "sam@acme.example"); code != 0 || !strings.Contains(out, "passed: true") {
			t.Fatalf("%s: %d %q %q", ticket, code, out, errOut)
		}
	}
	files, _ := filepath.Glob(filepath.Join(state, "proofs", "*.json"))
	for _, p := range files {
		raw, _ := os.ReadFile(p)
		for _, s := range []string{"quokka", "Quokka", "wombat", "Wombat", "finance", "sam@", "OPS-1"} {
			if strings.Contains(string(raw), s) {
				t.Fatalf("%s holds %q", p, s)
			}
		}
	}
}
