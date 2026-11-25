package manifest

import (
	"strings"
	"testing"
)

// SB-5101: a space's suspect is skip or monitor, a team inherits its department's, and
// Format reads it back.
func TestSpaceSuspect(t *testing.T) {
	with := func(line string) string {
		return strings.Replace(treeHead, "    label: confidential\n", "    label: confidential\n"+line, 1)
	}
	if _, ds := Check([]byte(with("    suspect: ignore\n"))); len(ds) == 0 {
		t.Fatal("suspect: ignore was accepted")
	}
	team := "  ap:\n    parent: finance\n    members: [group:ap]\n    scope: { docs: [\"/Finance/AP/**\"] }\n"
	m, ds := Check([]byte(with("    suspect: monitor\n") + team))
	if len(ds) > 0 || m.SuspectOf("finance") != "monitor" || m.SuspectOf("ap") != "monitor" {
		t.Fatalf("monitor: %v %q %q", ds, m.SuspectOf("finance"), m.SuspectOf("ap"))
	}
	again, ds := Check(Format(m))
	if len(ds) > 0 || again.SuspectOf("finance") != "monitor" {
		t.Fatalf("Format dropped suspect: %v\n%s", ds, Format(m))
	}
	if plain, _ := Check([]byte(treeHead)); plain.SuspectOf("finance") != "skip" {
		t.Fatalf("default: %q", plain.SuspectOf("finance"))
	}
}
