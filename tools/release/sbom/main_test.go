package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The SBOM of a real build lists the vendored modules with their go.sum hashes and the
// web lockfile's packages, with unique SPDX ids.
func TestSBOMOfARealBuild(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "sb")
	cmd := exec.Command("go", "build", "-trimpath", "-o", bin, "./tools/evalcell") // links cedar-go
	cmd.Dir = "../../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	d, err := buildWith(bin, "../../../web/package-lock.json", "sb", "2027-03-02T00:00:00Z", goSum("../../../go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if d.SPDXVersion != "SPDX-2.3" || d.DataLicense != "CC0-1.0" || len(d.Packages) < 10 {
		t.Fatalf("doc %+v", d)
	}
	ids := map[string]bool{}
	var names []string
	for _, p := range d.Packages {
		if ids[p.SPDXID] {
			t.Fatalf("duplicate SPDX id %s", p.SPDXID)
		}
		ids[p.SPDXID] = true
		names = append(names, p.Name)
	}
	all := strings.Join(names, " ")
	for _, want := range []string{"github.com/cedar-policy/cedar-go", "react", "vite"} {
		if !strings.Contains(all, want) {
			t.Errorf("SBOM lacks %s", want)
		}
	}
	for _, p := range d.Packages {
		if p.Name == "github.com/cedar-policy/cedar-go" && (len(p.Checksums) != 1 || len(p.Checksums[0].Value) != 64) {
			t.Errorf("cedar-go has no sha256 from go.sum: %+v", p.Checksums)
		}
	}
	again, _ := build(bin, "../../../web/package-lock.json", "sb", "2027-03-02T00:00:00Z")
	if len(again.Packages) != len(d.Packages) {
		t.Fatal("the SBOM isn't reproducible")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatal(err)
	}
}
