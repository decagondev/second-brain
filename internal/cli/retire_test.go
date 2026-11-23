package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/core/audit"
)

// SB-1907: check the export first, then sign a statement that verifies under the printed key.
func TestRetireStatement(t *testing.T) {
	dir := t.TempDir()
	seed := bytes.Repeat([]byte{3}, 32)
	seedFile := filepath.Join(dir, "anchor-seed")
	_ = os.WriteFile(seedFile, []byte(base64.StdEncoding.EncodeToString(seed)+"\n"), 0o600)
	st := auditmem.New()
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: st, Now: time.Now}
	for i := 0; i < 3; i++ {
		_, _ = log.Write(context.Background(), audit.Event{Actor: "sys_jobs", Kind: "config", Subject: "x", Reason: "r"})
	}
	s, _ := audit.NewSigner("acme-logistics", seed)
	var b bytes.Buffer
	if _, err := audit.Export(context.Background(), st, s, 1, time.Now(), &b); err != nil {
		t.Fatal(err)
	}
	exp := filepath.Join(dir, "audit.ndjson")
	_ = os.WriteFile(exp, b.Bytes(), 0o600)
	code, out, _ := run("selfhost", "retire-statement", "--check", "--tenant", "acme-logistics", "--export", exp, "--anchor-seed", seedFile)
	if code != 0 || strings.TrimSpace(out) != s.Key() {
		t.Fatalf("check: %d %q", code, out)
	}
	stmt := filepath.Join(dir, "statement.json")
	code, out, _ = run("selfhost", "retire-statement", "--tenant", "acme-logistics", "--export", exp, "--cell-key", s.Key(), "--host", "cell.acme.example",
		"--destroyed", "volumes: pgdata", "--destroyed", "12 files", "--kept", "sealed copies", "--out", stmt)
	if code != 0 || !strings.Contains(out, "last anchor: seq 3") {
		t.Fatalf("sign: %d %q", code, out)
	}
	key := ""
	for _, l := range strings.Split(out, "\n") {
		if k, ok := strings.CutPrefix(l, "key: "); ok {
			key = k
		}
	}
	if code, out, _ := run("selfhost", "verify-retirement", stmt, "--key", key); code != 0 || !strings.Contains(out, "destroyed: 12 files") {
		t.Fatalf("verify: %d %q", code, out)
	}
	if code, _, _ := run("selfhost", "verify-retirement", stmt, "--key", s.Key()); code == 0 {
		t.Fatal("verified under the cell's key")
	}
	// Another cell's seed: the export doesn't verify, so nothing would be destroyed.
	_ = os.WriteFile(seedFile, []byte(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{4}, 32))), 0o600)
	if code, _, _ := run("selfhost", "retire-statement", "--check", "--tenant", "acme-logistics", "--export", exp, "--anchor-seed", seedFile); code == 0 {
		t.Fatal("an export checked under another key passed")
	}
	if code, _, _ := run("selfhost", "retire-statement", "--tenant", "acme-logistics", "--export", exp, "--cell-key", s.Key(), "--out", stmt); code != 2 {
		t.Fatalf("nothing destroyed: exit %d", code)
	}
}
