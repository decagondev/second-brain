// Package smoke holds black-box smoke tests: build the real binary and run it.
// After every deploy the same idea runs against a live cell (docs/04-testing-and-assurance.md).
package smoke

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var sb string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "sb-smoke")
	if err != nil {
		panic(err)
	}
	sb = filepath.Join(dir, "sb")
	if out, err := exec.Command("go", "build", "-o", sb, "../../cmd/sb").CombinedOutput(); err != nil {
		panic("build sb: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func runSB(args ...string) (int, string) {
	out, err := exec.Command(sb, args...).CombinedOutput()
	if x, ok := err.(*exec.ExitError); ok {
		return x.ExitCode(), string(out)
	}
	return 0, string(out)
}

// Exit statuses are a contract (sb(1) EXIT STATUS); scripts and agents branch on them.
func TestExitStatusContract(t *testing.T) {
	cases := []struct {
		args []string
		code int
		has  string
	}{
		{[]string{"version"}, 0, "sb "},
		{[]string{"--help"}, 0, "usage: sb"},
		{[]string{}, 2, "usage: sb"},
		{[]string{"nosuch"}, 2, "[EUSAGE]"},
		{[]string{"tenant", "verify", "x.yaml"}, 2, "[ENOTIMPL]"},
		{[]string{"tenant", "check", "nosuch.yaml"}, 2, "[EUSAGE]"},
		{[]string{"tenant", "check", "../../examples/tenants/pilot.yaml"}, 0, "pilot.yaml: ok"},
		{[]string{"tenant", "check", "../../examples/tenants/invalid.yaml"}, 1, "[EMANIFEST]"},
		{[]string{"--json", "tenant", "check", "../../examples/tenants/invalid.yaml"}, 1, `"field":"custody"`},
		{[]string{"--json", "kill", "agent", "ask"}, 2, `"code":"EUSAGE"`}, // built in SB-403: --reason is required
		{[]string{"--json", "kill", "agent", "ask", "--all-tenants", "--reason", "x"}, 2, `"code":"ENOTIMPL"`},
	}
	for _, c := range cases {
		code, out := runSB(c.args...)
		if code != c.code || !strings.Contains(out, c.has) {
			t.Errorf("sb %v: exit %d, output %q; want %d containing %q", c.args, code, out, c.code, c.has)
		}
	}
}

// sb tenant apply --plan against a local registry (SB-102): a new cell, then a change.
func TestTenantPlan(t *testing.T) {
	reg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(reg, "tenants"), 0o755); err != nil {
		t.Fatal(err)
	}
	ep := "file://" + reg
	code, out := runSB("--endpoint", ep, "tenant", "apply", "../../examples/tenants/pilot.yaml", "--plan")
	if code != 0 || !strings.HasSuffix(out, "plan: 10 to add, 0 to change, 0 to remove\n") {
		t.Fatalf("new cell: exit %d\n%s", code, out)
	}
	// Record the pilot as applied, then plan the full manifest against it.
	code, js := runSB("--json", "--endpoint", ep, "tenant", "apply", "../../examples/tenants/pilot.yaml", "--plan")
	if code != 0 {
		t.Fatalf("json plan: exit %d %s", code, js)
	}
	var p struct {
		Changes []struct {
			Kind, Name string
			Attrs      map[string]string
		}
	}
	if err := json.Unmarshal([]byte(js), &p); err != nil {
		t.Fatal(err)
	}
	var rs []map[string]any
	for _, c := range p.Changes {
		rs = append(rs, map[string]any{"kind": c.Kind, "name": c.Name, "attrs": c.Attrs})
	}
	rec, _ := json.Marshal(map[string]any{"tenant": "acme-logistics", "resources": rs})
	if err := os.WriteFile(filepath.Join(reg, "tenants", "acme-logistics.json"), rec, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out = runSB("--endpoint", ep, "tenant", "apply", "../../examples/tenants/full.yaml", "--plan")
	want := `~ setting budget_usd_month = 1500 (was 1000)
- setting pilot_group = group:sb-pilot
+ connector tickets (jira, read-write)
~ exclude chat (#hr-*, #legal-*, #board; was #hr-*, #legal-*)
+ agent draft-reply
+ workflow ticket-update (approval group:it-leads)
plan: 3 to add, 2 to change, 1 to remove
`
	if code != 0 || out != want {
		t.Fatalf("pilot -> full: exit %d\n%s\nwant:\n%s", code, out, want)
	}
	if code, out := runSB("--endpoint", ep, "tenant", "apply", "../../examples/tenants/full.yaml", "--plan"); code != 0 || out != want {
		t.Fatal("plan is not repeatable")
	}
	if code, out := runSB("tenant", "apply", "../../examples/tenants/pilot.yaml", "--plan"); code != 7 || !strings.Contains(out, "[EUNAVAIL]") {
		t.Fatalf("no endpoint must be EUNAVAIL, never a new-cell plan: exit %d %s", code, out)
	}
}

// sb tenant apply against a local control plane (SB-104): create, no changes, change.
func TestTenantApply(t *testing.T) {
	cp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cp, "tenants"), 0o755); err != nil {
		t.Fatal(err)
	}
	ep := "file://" + cp
	apply := func(f string) (int, string) {
		return runSB("--endpoint", ep, "tenant", "apply", "../../examples/tenants/"+f)
	}
	if code, out := apply("pilot.yaml"); code != 0 || !strings.HasPrefix(out, "+ cell acme-logistics (region eu)\n") ||
		!strings.HasSuffix(out, "apply: 10 added, 0 changed, 0 removed\n") {
		t.Fatalf("first apply: exit %d\n%s", code, out)
	}
	cells, _ := filepath.Glob(filepath.Join(cp, "accounts", "factory-account", "acme-logistics", "*.json"))
	if len(cells) != 5 {
		t.Fatalf("want 5 cell resources, got %v", cells)
	}
	journal, _ := os.ReadFile(filepath.Join(cp, "journal", "acme-logistics.ndjson"))
	if n := strings.Count(string(journal), "\n"); n != 20 {
		t.Fatalf("want 20 journal lines (start and done for 10 steps), got %d", n)
	}
	if code, out := apply("pilot.yaml"); code != 0 || out != "apply: no changes\n" {
		t.Fatalf("second apply: exit %d %q", code, out)
	}
	if again, _ := os.ReadFile(filepath.Join(cp, "journal", "acme-logistics.ndjson")); len(again) != len(journal) {
		t.Fatal("a no-change apply wrote to the journal")
	}
	// The full manifest enables ticket-update, which waits for a passing eval record (SB-410).
	if code, out := apply("full.yaml"); code != 1 || !strings.Contains(out, "[EEVAL]") || !strings.Contains(out, "ticket-update") {
		t.Fatalf("full without an eval record: exit %d\n%s", code, out)
	}
	if again, _ := os.ReadFile(filepath.Join(cp, "journal", "acme-logistics.ndjson")); len(again) != len(journal) {
		t.Fatal("a refused apply wrote to the journal")
	}
	rec := filepath.Join(cp, "evals", "acme-logistics", "ticket-update.json")
	if err := os.MkdirAll(filepath.Dir(rec), 0o755); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UTC().Format(time.RFC3339)
	if err := os.WriteFile(rec, []byte(`{"passed":true,"at":"`+stamp+`","suites":["tenant/acme-logistics"],"model":"extractive"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := apply("full.yaml"); code != 0 || !strings.HasSuffix(out, "apply: 3 added, 2 changed, 1 removed\n") {
		t.Fatalf("pilot -> full: exit %d\n%s", code, out)
	}
	if code, out := runSB("--endpoint", ep, "tenant", "apply", "../../examples/tenants/full.yaml", "--plan"); code != 0 || out != "plan: no changes\n" {
		t.Fatalf("after apply, plan: exit %d %q", code, out)
	}
	want := "custody        factory\nplacement      factory-account\nkey_custodian  factory\nkey_wraps      audit-signing: factory\nchannel        stable\nhandoff        none\n"
	if code, out := runSB("--endpoint", ep, "factory", "status", "acme-logistics"); code != 0 || out != want {
		t.Fatalf("factory status: exit %d %q", code, out)
	}
	if code, out := runSB("--endpoint", ep, "factory", "status", "nobody-here"); code != 2 || !strings.Contains(out, "[EUSAGE]") {
		t.Fatalf("factory status for no cell: exit %d %q", code, out)
	}
}

// sb kill and unkill against a local control plane (SB-403): audited, listed, cleared.
func TestKillLocal(t *testing.T) {
	ep := "file://" + t.TempDir()
	code, out := runSB("--endpoint", ep, "--tenant", "acme-logistics", "kill", "agent", "ask", "--reason", "smoke drill")
	if code != 0 || !strings.Contains(out, "agent:ask killed for acme-logistics") {
		t.Fatalf("kill: %d %s", code, out)
	}
	if code, out := runSB("--endpoint", ep, "--tenant", "acme-logistics", "kill", "list"); code != 0 || !strings.Contains(out, "agent:ask") || !strings.Contains(out, "smoke drill") {
		t.Fatalf("list: %d %s", code, out)
	}
	if code, out := runSB("--endpoint", ep, "--tenant", "acme-logistics", "unkill", "agent", "ask", "--reason", "done"); code != 0 {
		t.Fatalf("unkill: %d %s", code, out)
	}
	if code, out := runSB("--endpoint", ep, "--tenant", "acme-logistics", "kill", "list"); code != 0 || !strings.Contains(out, "no kill switches") {
		t.Fatalf("list after unkill: %d %s", code, out)
	}
	audit, err := os.ReadFile(filepath.Join(strings.TrimPrefix(ep, "file://"), "audit", "acme-logistics.ndjson"))
	if err != nil || strings.Count(string(audit), `"kind":"kill"`) != 2 || !strings.Contains(string(audit), `"state":"cleared"`) {
		t.Fatalf("audit rows: %v %s", err, audit)
	}
	if code, out := runSB("--endpoint", ep, "--tenant", "acme-logistics", "kill", "connector", "email", "--reason", "x"); code != 2 || !strings.Contains(out, "[EUSAGE]") {
		t.Fatalf("bad scope: %d %s", code, out)
	}
}

// sb release status, promote and rollback against a local control plane (SB-507, SB-508).
func TestReleaseCommands(t *testing.T) {
	root := t.TempDir()
	ep := "file://" + root
	if err := os.MkdirAll(filepath.Join(root, "releases"), 0o755); err != nil {
		t.Fatal(err)
	}
	soaked := time.Now().Add(-72 * time.Hour).UTC().Format(time.RFC3339)
	state := `{"stable":"rel_2027-03-02_aaaaaa","prev_stable":"rel_2027-02-23_cccccc",
 "releases":{"rel_2027-02-23_cccccc":{"id":"rel_2027-02-23_cccccc","order":0,"gated":true},
  "rel_2027-03-02_aaaaaa":{"id":"rel_2027-03-02_aaaaaa","order":1,"gated":true},
  "rel_2027-03-09_bbbbbb":{"id":"rel_2027-03-09_bbbbbb","order":2,"gated":true,"ring0_at":"` + soaked + `","ring1_at":"` + soaked + `"}},
 "tenants":{"internal":{"name":"internal","channel":"testing","ring":0,"running":"rel_2027-03-09_bbbbbb"},
  "acme-logistics":{"name":"acme-logistics","channel":"stable","ring":2,"running":"rel_2027-03-02_aaaaaa"},
  "gamma-parts":{"name":"gamma-parts","channel":"stable","ring":2,"running":"rel_2027-02-23_cccccc"}}}`
	if err := os.WriteFile(filepath.Join(root, "releases", "state.json"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runSB("--endpoint", ep, "release", "status"); code != 0 || !strings.Contains(out, "gamma-parts") || !strings.Contains(out, "testing  rel_2027-03-09_bbbbbb") {
		t.Fatalf("status: %d\n%s", code, out)
	}
	code, out := runSB("--endpoint", ep, "release", "promote", "rel_2027-03-09_bbbbbb", "--to", "stable")
	if code != 1 || !strings.Contains(out, "promote guard: gamma-parts runs rel_2027-02-23_cccccc") {
		t.Fatalf("guard: %d\n%s", code, out)
	}
	if code, out := runSB("--endpoint", ep, "release", "rollback", "--reason", "smoke drill"); code != 0 || !strings.Contains(out, "rel_2027-03-02_aaaaaa revoked") || !strings.Contains(out, "acme-logistics") {
		t.Fatalf("rollback: %d\n%s", code, out)
	}
	if code, out := runSB("--endpoint", ep, "release", "rollback"); code != 1 || !strings.Contains(out, "no previous stable") {
		t.Fatalf("second rollback: %d\n%s", code, out)
	}
	audit, _ := os.ReadFile(filepath.Join(root, "audit", "platform.ndjson"))
	if !strings.Contains(string(audit), `"kind":"release"`) || !strings.Contains(string(audit), `"state":"revoked"`) {
		t.Fatalf("audit: %s", audit)
	}
}
