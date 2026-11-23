package cli

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/backup/s3/s3test"
)

// SB-2303: the shell scripts' way to the off-box bucket. Only sealed files and records go;
// each is written once; the doctor's check tells a refused key from an unreachable store.
func TestBackupPushPull(t *testing.T) {
	f := &s3test.Fake{Bucket: "acme-backups", Access: "ak"}
	srv := httptest.NewServer(f)
	defer srv.Close()
	dir := t.TempDir()
	keys := filepath.Join(dir, "offsite")
	write(t, keys, `{"access_key":"ak","secret_key":"sk","endpoint":"`+srv.URL+`"}`)
	to := "s3://acme-backups/second-brain"
	pub, priv := filepath.Join(dir, "recovery.pub"), filepath.Join(dir, "recovery.key")
	if code, _, e := sbRun("backup", "keygen", "--public", pub, "--private", priv); code != 0 {
		t.Fatal(e)
	}
	dump := filepath.Join(dir, "sb-2026-11-23T020000Z.dump")
	write(t, dump, "acme-logistics rows")
	write(t, dump+".json", `{"tenant":"acme-logistics","keys":"0123456789abcdef"}`)
	if code, _, e := sbRun("backup", "seal", "--to", pub, dump); code != 0 {
		t.Fatal(e)
	}
	// Plain content is refused, sealed files and records go.
	if code, _, e := sbRun("backup", "push", "--to", to, "--keys", keys, "--as", "db/x.dump", dump); code == 0 || !strings.Contains(e, "nothing unsealed leaves the box") {
		t.Fatalf("a plain dump was pushed: %d %s", code, e)
	}
	big := filepath.Join(dir, "big.json")
	write(t, big, `{"x":"`+strings.Repeat("a", MaxRecord)+`"}`)
	if code, _, _ := sbRun("backup", "push", "--to", to, "--keys", keys, big); code == 0 {
		t.Fatal("a record bigger than MaxRecord was pushed unsealed")
	}
	for _, p := range []string{dump + ".sealed", dump + ".json"} {
		if code, _, e := sbRun("backup", "push", "--to", to, "--keys", keys, "--as", "db/"+filepath.Base(p), p); code != 0 {
			t.Fatalf("push %s: %s", p, e)
		}
	}
	if code, _, _ := sbRun("backup", "push", "--to", to, "--keys", keys, "--as", "db/"+filepath.Base(dump)+".sealed", dump+".sealed"); code == 0 {
		t.Fatal("an object was written twice")
	}
	if f.Objects() != 2 {
		t.Fatalf("%d objects, want 2", f.Objects())
	}
	code, out, e := sbRun("backup", "list", "--from", to, "--keys", keys, "db/")
	if code != 0 || out != "db/sb-2026-11-23T020000Z.dump.json\ndb/sb-2026-11-23T020000Z.dump.sealed\n" {
		t.Fatalf("list: %d %q %s", code, out, e)
	}
	back := filepath.Join(dir, "back")
	if code, _, e := sbRun("backup", "pull", "--from", to, "--keys", keys, "--out", back, "db/sb-2026-11-23T020000Z.dump.sealed"); code != 0 {
		t.Fatal(e)
	}
	if code, _, e := sbRun("backup", "open", "--key", priv, filepath.Join(back, "sb-2026-11-23T020000Z.dump.sealed")); code != 0 {
		t.Fatal(e)
	}
	if b, _ := os.ReadFile(filepath.Join(back, "sb-2026-11-23T020000Z.dump")); string(b) != "acme-logistics rows" {
		t.Fatalf("pulled and opened: %q", b)
	}
	// The doctor's check.
	if code, out, _ := sbRun("backup", "check", "--to", to, "--keys", keys); code != 0 || !strings.HasSuffix(out, ": ok\n") {
		t.Fatalf("check: %d %q", code, out)
	}
	write(t, keys, `{"access_key":"other","secret_key":"sk","endpoint":"`+srv.URL+`"}`)
	if code, out, _ := sbRun("--json", "backup", "check", "--to", to, "--keys", keys); code == 0 || !strings.Contains(out, `"answer":"refused"`) {
		t.Fatalf("a refused key: %d %q", code, out)
	}
	srv.Close()
	write(t, keys, `{"access_key":"ak","secret_key":"sk","endpoint":"`+srv.URL+`"}`)
	if code, out, _ := sbRun("--json", "backup", "check", "--to", to, "--keys", keys); code == 0 || !strings.Contains(out, `"answer":"unreachable"`) {
		t.Fatalf("an unreachable store: %d %q", code, out)
	}
	if code, _, _ := sbRun("backup", "list", "--from", "/mnt/nas", "--keys", keys); code == 0 {
		t.Fatal("a directory was taken for a bucket")
	}
}
