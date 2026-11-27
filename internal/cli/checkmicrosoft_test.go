package cli

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/msgraph/graphtest"
)

// SB-6003: a write permission fails and is named; mail without a refused probe fails;
// unmapped labels are listed; a good setup passes; every fix names a real runbook section.
func TestCheckMicrosoft(t *testing.T) {
	f := graphtest.Seeded()
	t.Cleanup(f.Server.Close)
	f.Granted = []string{"Files.Read.All", "Sites.Read.All", "Mail.Read", "InformationProtectionPolicy.Read.All"}
	f.Labels = []string{"lbl-conf"}
	cfg, _, _ := proveFixture(t, f)
	good := []string{"selfhost", "check-microsoft", "--config", cfg, "--mailbox", "support@acme.example", "--probe", "ann@acme.example"}
	code, out, errOut := run(good...)
	if code != 0 || strings.Contains(out, "fail") {
		t.Fatalf("good: %d %q %q", code, out, errOut)
	}
	all := out
	f.Granted = append(f.Granted, "Files.ReadWrite.All", "Mail.Send")
	code, out, _ = run(good...)
	if code == 0 || !strings.Contains(out, "Files.ReadWrite.All, Mail.Send") || !strings.Contains(out, "#write-permission") {
		t.Fatalf("a write permission: %d %q", code, out)
	}
	all += out
	f.Granted = f.Granted[:4]
	code, out, _ = run("selfhost", "check-microsoft", "--config", cfg, "--mailbox", "support@acme.example")
	if code == 0 || !strings.Contains(out, "#mail-access-policy") {
		t.Fatalf("no probe: %d %q", code, out)
	}
	all += out
	f.Boxes["ann@acme.example"] = nil
	if code, out, _ = run(good...); code == 0 || !strings.Contains(out, "a mailbox not named can be read") {
		t.Fatalf("a readable probe: %d %q", code, out)
	}
	all += out
	delete(f.Boxes, "ann@acme.example")
	f.Labels = append(f.Labels, "lbl-new", "lbl-other")
	if code, out, _ = run(good...); code == 0 || !strings.Contains(out, "2 of 3 labels aren't mapped") || !strings.Contains(out, "lbl-new") {
		t.Fatalf("unmapped labels: %d %q", code, out)
	}
	all += out
	md, err := os.ReadFile("../../docs/runbooks/microsoft.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`microsoft\.md#([a-z-]+)`).FindAllStringSubmatch(all, -1) {
		if !strings.Contains(string(md), "\n## "+m[1]+"\n") {
			t.Errorf("fix %s isn't a section of the runbook", m[1])
		}
	}
	if strings.Contains(all, "s3cret") {
		t.Fatal("a secret was printed")
	}
}
