package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sbRun(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestBackupSealOpen(t *testing.T) {
	dir := t.TempDir()
	pub, priv := filepath.Join(dir, "recovery.pub"), filepath.Join(dir, "recovery.key")
	if code, _, e := sbRun("backup", "keygen", "--public", pub, "--private", priv); code != 0 {
		t.Fatalf("keygen: %s", e)
	}
	if code, _, _ := sbRun("backup", "keygen", "--public", pub, "--private", priv); code == 0 {
		t.Fatal("a second keygen overwrote the key")
	}
	if st, _ := os.Stat(priv); st.Mode().Perm() != 0o600 {
		t.Fatalf("private key mode %v", st.Mode())
	}
	dump := filepath.Join(dir, "sb.dump")
	write(t, dump, strings.Repeat("acme-logistics row\n", 10000))
	if code, _, e := sbRun("backup", "seal", "--to", pub, dump); code != 0 {
		t.Fatalf("seal: %s", e)
	}
	sealed, _ := os.ReadFile(dump + ".sealed")
	if bytes.Contains(sealed, []byte("acme-logistics row")) {
		t.Fatal("the sealed copy is readable")
	}
	out := filepath.Join(dir, "back.dump")
	if code, _, e := sbRun("backup", "open", "--key", priv, "--out", out, dump+".sealed"); code != 0 {
		t.Fatalf("open: %s", e)
	}
	a, _ := os.ReadFile(dump)
	b, _ := os.ReadFile(out)
	if !bytes.Equal(a, b) {
		t.Fatal("opened copy differs")
	}
	sealed[len(sealed)/2] ^= 1
	write(t, dump+".sealed", string(sealed))
	_ = os.Remove(out)
	if code, _, e := sbRun("backup", "open", "--key", priv, "--out", out, dump+".sealed"); code != 1 || !strings.Contains(e, "nothing was written") {
		t.Fatalf("changed byte: %d %s", code, e)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("a changed copy left a file behind")
	}
}

func TestRecoverySplitJoin(t *testing.T) {
	dir := t.TempDir()
	arch := filepath.Join(dir, "keys.tar")
	write(t, arch, "the box's keys\n")
	kitDir := filepath.Join(dir, "kit")
	code, o, e := sbRun("--json", "recovery", "split", "--tenant", "acme-logistics", "--out", kitDir, "--custodian", "1=Olga (IT)", "--custodian", "4=the safe", arch)
	if code != 0 {
		t.Fatalf("split: %s %s", o, e)
	}
	m, _ := filepath.Glob(filepath.Join(kitDir, "kit_*.sealed"))
	pages, _ := filepath.Glob(filepath.Join(kitDir, "kit_*-share-*.txt"))
	if len(m) != 1 || len(pages) != 5 {
		t.Fatalf("kit files: %v %v", m, pages)
	}
	rec, _ := os.ReadFile(strings.TrimSuffix(m[0], ".sealed") + ".json")
	if !strings.Contains(string(rec), `"4": "the safe"`) || strings.Contains(string(rec), "checksum") {
		t.Fatalf("record: %s", rec)
	}
	out := filepath.Join(dir, "back.tar")
	if code, o, e := sbRun("recovery", "join", "--kit", m[0], "--out", out, pages[4], pages[0], pages[2]); code != 0 || !strings.Contains(o, "rebuilt from shares 1, 3, 5") {
		t.Fatalf("join: %d %s %s", code, o, e)
	}
	if b, _ := os.ReadFile(out); string(b) != "the box's keys\n" {
		t.Fatalf("rebuilt %q", b)
	}
	_ = os.Remove(out)
	if code, _, e := sbRun("recovery", "join", "--kit", m[0], "--out", out, pages[0], pages[1], pages[1]); code != 2 || !strings.Contains(e, "needs 3") {
		t.Fatalf("two shares: %d %s", code, e)
	}
	bad, _ := os.ReadFile(pages[3])
	typo := strings.Replace(string(bad), "\nchecksum: ", "\nchecksum: 0", 1)
	write(t, pages[3], typo)
	if code, _, e := sbRun("recovery", "join", "--kit", m[0], "--out", out, pages[0], pages[1], pages[3]); code != 2 || !strings.Contains(e, "checksum") {
		t.Fatalf("typo: %d %s", code, e)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("a failed join wrote a file")
	}
}
