package laws

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/export"
)

// LAW-11 through detach (SB-708): a detach's export package verifies offline, and any change
// to it is caught: an altered, missing or extra file, an audit export cut short or rebuilt
// with another key, or a bundle that isn't the signed one. The check never needs the factory.

// ExportWorld builds one package and tampers with it.
type ExportWorld interface {
	Build(rows int) (dir string, err error)
	Tamper(dir, how string) error // "", "byte", "drop", "extra", "cut", "rekey", "bundle"
	Verify(dir string) error
}

// NewExportWorld builds the system under test: the real export writer and verifier over a
// real audit log and a signed, gated bundle; packages go under root.
var NewExportWorld = func(root string) ExportWorld { return &exportWorld{root: root, verify: export.Verify} }

type exportWorld struct {
	root   string
	n      int
	verify func(context.Context, string) (export.Report, error)
}

func (w *exportWorld) Build(rows int) (string, error) {
	ctx := context.Background()
	w.n++
	now := time.Date(2027, 3, 2, 3, 0, 0, 0, time.UTC)
	st := auditmem.New()
	alog := &audit.Log{Tenant: "gamma-parts", Release: func() string { return "rel_x" }, Store: st, Now: func() time.Time { return now }}
	for i := 0; i < rows; i++ {
		if _, err := alog.Write(ctx, audit.Event{Actor: "usr_ann", Kind: "custody", Subject: "handoff:gamma-parts", Reason: fmt.Sprintf("step %d", i)}); err != nil {
			return "", err
		}
	}
	signer, _ := audit.NewSigner("gamma-parts", bytes.Repeat([]byte{byte(w.n)}, 32))
	var chain bytes.Buffer
	if _, err := audit.Export(ctx, st, signer, 1, now, &chain); err != nil {
		return "", err
	}
	rw := newReleaseWorld(false)
	id, err := rw.Build("")
	if err != nil {
		return "", err
	}
	if err := rw.Sign(id); err != nil {
		return "", err
	}
	if err := rw.Gate(id, true); err != nil {
		return "", err
	}
	files := map[string][]byte{
		"manifest.yaml":            []byte("tenant: gamma-parts\nregion: eu\n"),
		"policies/tenant.cedar":    []byte("// tenant policies\n"),
		"evals/results.json":       []byte(`{"suite":"citation","passed":true}`),
		"runbooks/sb-factory.7.md": []byte("# sb-factory(7)\n"),
		"audit/audit.ndjson":       chain.Bytes(),
		"audit/anchor-key.txt":     []byte(signer.Key() + "\n"),
		"release/sbom.spdx.json":   []byte(`{"spdxVersion":"SPDX-2.3"}`),
		"release/release-key.txt":  []byte(rw.signer.Public() + "\n"),
	}
	for _, f := range []string{"bundle.json", "bundle.sig", "gate.json", "gate.sig"} {
		b, err := rw.store.Get(ctx, id, f)
		if err != nil {
			return "", err
		}
		files["release/"+f] = b
	}
	dir := filepath.Join(w.root, fmt.Sprintf("export-%d", w.n))
	return dir, export.Write(dir, export.Package{Tenant: "gamma-parts", Files: files})
}

// rewrite replaces a file and updates MANIFEST.json, as a careful forger would.
func rewrite(dir, name string, b []byte) error {
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), b, 0o644); err != nil {
		return err
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "MANIFEST.json"))
	var man map[string]any
	_ = json.Unmarshal(raw, &man)
	s := sha256.Sum256(b)
	man["files"].(map[string]any)[name] = hex.EncodeToString(s[:])
	out, _ := json.Marshal(man)
	return os.WriteFile(filepath.Join(dir, "MANIFEST.json"), out, 0o644)
}

func (w *exportWorld) Tamper(dir, how string) error {
	switch how {
	case "":
		return nil
	case "byte": // edit a file without updating the manifest
		p := filepath.Join(dir, "manifest.yaml")
		b, _ := os.ReadFile(p)
		return os.WriteFile(p, append(b, '#'), 0o644)
	case "drop":
		return os.Remove(filepath.Join(dir, "policies", "tenant.cedar"))
	case "extra":
		return os.WriteFile(filepath.Join(dir, "evals", "planted.json"), []byte("{}"), 0o644)
	case "cut": // drop the audit export's last row and its anchor, manifest updated
		b, _ := os.ReadFile(filepath.Join(dir, "audit", "audit.ndjson"))
		lines := bytes.Split(bytes.TrimSpace(b), []byte("\n"))
		if len(lines) < 3 {
			return errors.New("too short to cut")
		}
		return rewrite(dir, "audit/audit.ndjson", append(bytes.Join(lines[:len(lines)-2], []byte("\n")), '\n'))
	case "rekey": // a different anchor key on file, manifest updated
		other, _ := audit.NewSigner("gamma-parts", bytes.Repeat([]byte{0xEE}, 32))
		return rewrite(dir, "audit/anchor-key.txt", []byte(other.Key()+"\n"))
	case "bundle": // a different bundle.json, manifest updated
		b, _ := os.ReadFile(filepath.Join(dir, "release", "bundle.json"))
		return rewrite(dir, "release/bundle.json", bytes.Replace(b, []byte(`"`), []byte(`" `), 1))
	}
	return fmt.Errorf("unknown tamper %q", how)
}

func (w *exportWorld) Verify(dir string) error {
	_, err := w.verify(context.Background(), dir)
	return err
}

func law11Export(r *rand.Rand, w ExportWorld) error {
	dir, err := w.Build(1 + r.Intn(30))
	if err != nil {
		return err
	}
	if err := w.Verify(dir); err != nil {
		return fmt.Errorf("an untouched package didn't verify: %v", err)
	}
	how := []string{"byte", "drop", "extra", "cut", "rekey", "bundle"}[r.Intn(6)]
	dir, err = w.Build(2 + r.Intn(30))
	if err != nil {
		return err
	}
	if err := w.Tamper(dir, how); err != nil {
		return err
	}
	if err := w.Verify(dir); err == nil {
		return fmt.Errorf("a package with %q tampering verified", how)
	}
	return nil
}
