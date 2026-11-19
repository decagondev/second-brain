package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// sb tenant verify's checks on a real local cell (the permission world): every law passes;
// a cell that cites what the asker can't open fails LAW-01 (SB-601).
func TestVerifyLocalCell(t *testing.T) {
	raw, err := os.ReadFile("../../evals/fixtures/permission/world.yaml")
	if err != nil {
		t.Fatal(err)
	}
	w := worldFromYAMLSubset(t, raw)
	in := verifyInput{World: w, Policies: "../../policy/platform", ForeignIDs: []string{"docs:beta-freight-payroll"}, Sample: 12, Seed: 7}
	rep, err := verifyWorld(context.Background(), in, false)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed() {
		t.Fatalf("report:\n%s", rep)
	}
	out := rep.String()
	for _, banned := range []string{"Quarterly bonus", "merger", "Refunds over"} {
		if strings.Contains(out, banned) {
			t.Fatalf("the report printed content %q:\n%s", banned, out)
		}
	}
	bad, _ := verifyWorld(context.Background(), in, true)
	if bad.Passed() || bad.Results[0].Status != "fail" {
		t.Fatalf("a leaky cell passed:\n%s", bad)
	}
	none := in
	none.World.Users = nil
	if r, _ := verifyWorld(context.Background(), none, false); r.Passed() || r.Results[0].Status != "unknown" {
		t.Fatalf("no users must be unknown, which fails:\n%s", r)
	}
}

// worldFromYAMLSubset converts the fixture (YAML) to the world type via JSON, using the
// eval harness's own converter so the two never disagree.
func worldFromYAMLSubset(t *testing.T, raw []byte) world {
	t.Helper()
	b, err := yamlToJSON(raw)
	if err != nil {
		t.Skip("python3 with pyyaml is needed to read the fixture: ", err)
	}
	var w world
	if err := json.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	return w
}

func yamlToJSON(raw []byte) ([]byte, error) {
	cmd := exec.Command("python3", "-c", "import sys,yaml,json; print(json.dumps(yaml.safe_load(sys.stdin.read())))")
	cmd.Stdin = strings.NewReader(string(raw))
	return cmd.Output()
}
