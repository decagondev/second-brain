package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

type rows struct{ rs []audit.Row }

func (s *rows) Append(_ context.Context, b func(int64, string) (audit.Row, error)) (audit.Row, error) {
	seq, h := int64(0), audit.Zero
	if n := len(s.rs); n > 0 {
		seq, h = s.rs[n-1].Seq, s.rs[n-1].Hash
	}
	r, err := b(seq, h)
	if err == nil {
		s.rs = append(s.rs, r)
	}
	return r, err
}

func (s *rows) Range(_ context.Context, from int64, limit int) ([]audit.Row, error) {
	var out []audit.Row
	for _, r := range s.rs {
		if r.Seq >= from && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}

func TestAuditVerifyExport(t *testing.T) {
	st := &rows{}
	l := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: st, Now: time.Now}
	for i := 0; i < 4; i++ {
		if _, err := l.Write(context.Background(), audit.Event{Actor: "usr_1", Kind: "kill", Subject: "agent:ask", Reason: "drill"}); err != nil {
			t.Fatal(err)
		}
	}
	s, _ := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{9}, 32))
	var b bytes.Buffer
	if _, err := audit.Export(context.Background(), st, s, 1, time.Now(), &b); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	good := filepath.Join(dir, "good.ndjson")
	_ = os.WriteFile(good, b.Bytes(), 0o600)
	if c, out, errOut := run("audit", "verify", "--export", good, "--key", s.Key()); c != 0 || !strings.Contains(out, "4 rows") || errOut != "" {
		t.Fatalf("good: %d %q %q", c, out, errOut)
	}
	if c, _, errOut := run("audit", "verify", "--export", good); c != 0 || !strings.Contains(errOut, "key not pinned") {
		t.Fatalf("unpinned: %d %q", c, errOut)
	}
	bad := filepath.Join(dir, "bad.ndjson")
	_ = os.WriteFile(bad, []byte(strings.Replace(b.String(), `"reason":"drill"`, `"reason":"dril1"`, 1)), 0o600)
	if c, _, errOut := run("audit", "verify", "--export", bad, "--key", s.Key()); c != 1 || !strings.Contains(errOut, "[ECHAIN]") || !strings.Contains(errOut, "seq 1") {
		t.Fatalf("tampered: %d %q", c, errOut)
	}
	if c, _, errOut := run("audit", "verify"); c != 2 || !strings.Contains(errOut, "[EUSAGE]") {
		t.Fatalf("no cell and no file: %d %q", c, errOut)
	}
}

// BUG-0027: a new cell's log is one start anchor and no rows. With no --tenant, the tenant
// can't be read from it, and verify used to report a forged anchor; it says so instead,
// and with the tenant the empty log verifies.
func TestAuditVerifyEmptyLog(t *testing.T) {
	s, _ := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{9}, 32))
	var b bytes.Buffer
	if _, err := audit.Export(context.Background(), &rows{}, s, 1, time.Now(), &b); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "empty.ndjson")
	_ = os.WriteFile(f, b.Bytes(), 0o600)
	if c, _, errOut := run("audit", "verify", "--export", f, "--key", s.Key()); c != 2 || !strings.Contains(errOut, "pass --tenant") || strings.Contains(errOut, "ECHAIN") {
		t.Fatalf("empty log, no tenant: %d %q", c, errOut)
	}
	if c, out, errOut := run("--tenant", "acme-logistics", "audit", "verify", "--export", f, "--key", s.Key()); c != 0 || !strings.Contains(out, "0 rows") {
		t.Fatalf("empty log with its tenant: %d %q %q", c, out, errOut)
	}
}
