package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

// SB-1101: sb space init writes a valid space from a template, keeps the rest of the file,
// and changes nothing when the result wouldn't check.
func TestSpaceInit(t *testing.T) {
	src, err := os.ReadFile("../../examples/tenants/pilot.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "acme.yaml")
	if err := os.WriteFile(f, src, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run("space", "init", f, "--template", "people", "--members", "group:people-team", "--stewards", "group:people-leads")
	if code != 0 || !strings.Contains(out, "space people added") || !strings.Contains(out, "label restricted") {
		t.Fatalf("init people: %d %s", code, out)
	}
	code, out, _ = run("space", "init", f, "--template", "engineering", "--name", "eng", "--members", "group:eng")
	if code != 0 {
		t.Fatalf("init eng: %d %s", code, out)
	}
	b, _ := os.ReadFile(f)
	if !strings.HasPrefix(string(b), string(src)) || strings.Count(string(b), "spaces:") != 1 {
		t.Fatalf("the rest of the file changed, or spaces: appears twice:\n%s", b)
	}
	m, ds := manifest.Check(b)
	if len(ds) > 0 {
		t.Fatalf("the result doesn't check: %v\n%s", ds, b)
	}
	if p := m.Spaces["people"]; p.Label != "restricted" || p.RetentionDays != 365 || p.Scope["docs"][0] != "/People/**" {
		t.Fatalf("people = %+v", p)
	}
	if e := m.Spaces["eng"]; e.Scope["docs"][0] != "/Eng/**" || e.Scope["chat"][0] != "#eng-*" || len(e.Scope) != 2 {
		t.Fatalf("eng = %+v", e) // code and tickets have no connector here, so they're left out
	}
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"--template", "people", "--members", "group:x", "--stewards", "group:y"}, 1, "already has a space"},
		{[]string{"--template", "legal", "--members", "group:x"}, 2, "needs --stewards"},
		{[]string{"--template", "astrology", "--members", "group:x"}, 2, "no template"},
		{[]string{"--template", "sales", "--name", "Sales!", "--members", "group:x"}, 1, "doesn't check"},
		{[]string{"--template", "design", "--name", "d2", "--members", "user-not-group"}, 1, "doesn't check"},
	} {
		before, _ := os.ReadFile(f)
		code, out, errOut := run(append([]string{"space", "init", f}, tc.args...)...)
		after, _ := os.ReadFile(f)
		if code != tc.code || !strings.Contains(out+errOut, tc.want) || string(before) != string(after) {
			t.Errorf("%v: exit %d %s%s (file changed: %v)", tc.args, code, out, errOut, string(before) != string(after))
		}
	}
	g := filepath.Join(t.TempDir(), "t.yaml")
	_ = os.WriteFile(g, []byte("tenant: acme-logistics\nchannel: stable\nidp: oidc://login.acme.example\nregion: eu\nconnectors: { tickets: jira }\nagents: [ask]\n"), 0o644)
	if code, out, errOut := run("space", "init", g, "--template", "design", "--members", "group:design"); code != 1 || !strings.Contains(out+errOut, "no connector") {
		t.Fatalf("no connector for the template: %d %s%s", code, out, errOut)
	}
	if code, out, _ := run("space", "templates"); code != 0 || strings.Count(out, "\n") != 13 || !strings.Contains(out, "people       restricted") {
		t.Fatalf("templates: %d\n%s", code, out)
	}
}

// SB-1206: sb space diff previews gains, losses and flags, offline, and writes nothing.
func TestSpaceDiff(t *testing.T) {
	dir := t.TempDir()
	m := filepath.Join(dir, "acme.yaml")
	src := []byte(`tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
spaces:
  people:
    template: people
    members: [group:people-team]
    stewards: [group:people-leads]
    scope: { docs: ["/People/**"] }
  engineering:
    template: engineering
    members: [group:eng]
    scope: { docs: ["/Engineering/**"] }
barriers:
  - [group:deal-team, group:eng]
`)
	snap := filepath.Join(dir, "reorg.json")
	sj := []byte(`{"version": 1, "people": [
  {"user": "ana@acme.example", "before": ["group:finance"], "after": ["group:people-team"]},
  {"user": "bo@acme.example", "before": ["group:eng"], "after": []},
  {"user": "cy@acme.example", "before": ["group:eng"], "after": ["group:eng", "group:deal-team"]},
  {"user": "di@acme.example", "before": ["group:eng"], "after": ["group:eng"]}
]}`)
	for f, b := range map[string][]byte{m: src, snap: sj} {
		if err := os.WriteFile(f, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, out, _ := run("space", "diff", m, "--idp-snapshot", snap)
	want := "ana@acme.example  gains people  loses -  [gains-restricted]\n" +
		"bo@acme.example  gains -  loses engineering  [loses-all]\n" +
		"cy@acme.example  gains -  loses -  [barrier]\n" +
		"1 unchanged\n"
	if code != 0 || out != want {
		t.Fatalf("diff: %d\n%s\nwant\n%s", code, out, want)
	}
	for f, b := range map[string][]byte{m: src, snap: sj} {
		if got, _ := os.ReadFile(f); string(got) != string(b) {
			t.Fatalf("%s changed", f)
		}
	}
	if err := os.WriteFile(snap, []byte(`{"version": 2, "people": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := run("space", "diff", m, "--idp-snapshot", snap); code != 2 {
		t.Fatalf("a bad snapshot exits %d, want 2", code)
	}
}
