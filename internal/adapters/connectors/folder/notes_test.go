package folder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

func notePay(text string) []byte {
	return []byte(`{"markdown":"` + text + `","readers":["ann@acme.example"],"owner":"sam@acme.example","label":"internal"}`)
}

// SB-1705: act writes only a note and its ACL file under notes/, never over another note,
// and only with notes enabled.
func TestNotes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "policy.md"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	a := ports.Action{Target: "notes/support/qs_1.md", Payload: notePay("# Refunds")}
	if _, err := c.Act(ctx, a, "k"); err == nil || !strings.Contains(err.Error(), "ENOTSUP") {
		t.Fatalf("act without notes: %v", err)
	}
	if err := c.EnableNotes(dir); err != nil {
		t.Fatal(err)
	}
	rc, err := c.Act(ctx, a, "k")
	if err != nil || !strings.HasPrefix(rc.TargetID, "notes/support/qs_1.md@sha256:") {
		t.Fatalf("%+v %v", rc, err)
	}
	acl, _ := os.ReadFile(filepath.Join(dir, "notes", "support", "qs_1.md.acl"))
	if string(acl) != "owner sam@acme.example\nread ann@acme.example\nlabel internal\n" {
		t.Fatalf("acl %q", acl)
	}
	if again, err := c.Act(ctx, a, "k"); err != nil || again != rc {
		t.Fatalf("the same note again: %+v %v", again, err)
	}
	if _, err := c.Act(ctx, ports.Action{Target: a.Target, Payload: notePay("# Other")}, "k2"); err == nil {
		t.Fatal("a note was overwritten")
	}
	for _, bad := range []string{"policy.md", "notes/support/qs_1.txt", "notes/../policy.md", "notes/support/x/qs_1.md", "notes/Support/qs_1.md", "/etc/qs_1.md"} {
		if _, err := c.Act(ctx, ports.Action{Target: bad, Payload: notePay("# x")}, "k3"); err == nil {
			t.Errorf("act wrote %q", bad)
		}
	}
	for _, p := range [][]byte{[]byte(`{"markdown":"x","readers":["not an email"],"owner":"sam@acme.example"}`),
		[]byte(`{"markdown":"x","readers":[],"owner":"sam@acme.example"}`), []byte(`{"markdown":"x","extra":1}`)} {
		if _, err := c.Act(ctx, ports.Action{Target: "notes/support/qs_9.md", Payload: p}, "k4"); err == nil {
			t.Errorf("payload %s was written", p)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "policy.md")); string(b) != "keep" {
		t.Fatal("the rest of the share changed")
	}
	// The note is a document like any other, readable by its readers only.
	vs, err := c.ACL(ctx, ports.UserRef{Email: "ann@acme.example"}, []ports.SourceRef{{Kind: "docs", ID: "notes/support/qs_1.md"}})
	if err != nil || vs[0] != ports.VerdictAllow {
		t.Fatalf("ann on her note: %v %v", vs, err)
	}
	vs, _ = c.ACL(ctx, ports.UserRef{Email: "bob@acme.example"}, []ports.SourceRef{{Kind: "docs", ID: "notes/support/qs_1.md"}})
	if vs[0] != ports.VerdictDeny {
		t.Fatalf("bob on ann's note: %v", vs)
	}
}
