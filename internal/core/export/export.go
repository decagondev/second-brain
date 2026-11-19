// Package export builds and verifies a detach's export package (sb-factory(7), Detach,
// exactly; SB-708). Verification is offline: file digests from MANIFEST.json, the audit
// chain with the cell's pinned anchor key (LAW-11), and the release bundle through the same
// admission checks a cell runs (LAW-06).
package export

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/release"
)

// Package is what goes in an export, by path inside the package.
type Package struct {
	Tenant string
	Files  map[string][]byte
}

// Required are the paths every package has; policies/, evals/ and runbooks/ need at least
// one file each.
var Required = []string{"manifest.yaml", "audit/audit.ndjson", "audit/anchor-key.txt", "release/bundle.json",
	"release/bundle.sig", "release/gate.json", "release/gate.sig", "release/sbom.spdx.json", "release/release-key.txt"}

// Dirs need at least one file each.
var Dirs = []string{"policies/", "evals/", "runbooks/"}

// Errors, mapped to sb-errors(7) codes by the CLI.
var (
	ErrFiles  = errors.New("export: ELAW the package's files don't match MANIFEST.json")
	ErrChain  = errors.New("export: ECHAIN the audit export doesn't verify")
	ErrBundle = errors.New("export: EUNSIGNED the release doesn't verify")
)

var safePath = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*(/[A-Za-z0-9][A-Za-z0-9._-]*)*$`)

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func missing(files map[string]bool) []string {
	var out []string
	for _, r := range Required {
		if !files[r] {
			out = append(out, r)
		}
	}
	for _, d := range Dirs {
		found := false
		for f := range files {
			found = found || strings.HasPrefix(f, d)
		}
		if !found {
			out = append(out, d)
		}
	}
	return out
}

// Write writes the package into an empty or new directory, MANIFEST.json last.
func Write(dir string, p Package) error {
	have := map[string]bool{}
	for name := range p.Files {
		if !safePath.MatchString(name) || strings.Contains(name, "..") || name == "MANIFEST.json" {
			return fmt.Errorf("export: bad path %q", name)
		}
		have[name] = true
	}
	if m := missing(have); len(m) > 0 {
		return fmt.Errorf("%w: missing %s", ErrFiles, strings.Join(m, ", "))
	}
	if ents, err := os.ReadDir(dir); err == nil && len(ents) > 0 {
		return fmt.Errorf("export: %s isn't empty", dir)
	}
	manifest := map[string]string{}
	names := make([]string, 0, len(p.Files))
	for name := range p.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, p.Files[name], 0o644); err != nil {
			return err
		}
		manifest[name] = sum(p.Files[name])
	}
	b, _ := json.MarshalIndent(map[string]any{"tenant": p.Tenant, "files": manifest}, "", "  ")
	return os.WriteFile(filepath.Join(dir, "MANIFEST.json"), append(b, '\n'), 0o644)
}

// Report is what Verify checked.
type Report struct {
	Tenant  string `json:"tenant"`
	Files   int    `json:"files"`
	Rows    int    `json:"audit_rows"`
	LastSeq int64  `json:"audit_last_seq"`
	Release string `json:"release"`
}

// dirStore serves the package's release/ files to release.Admit.
type dirStore struct{ dir string }

func (d dirStore) Get(_ context.Context, _, name string) ([]byte, error) {
	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("export: %s isn't in the package", name)
	}
	return os.ReadFile(filepath.Join(d.dir, "release", name))
}

// Verify checks a package offline.
func Verify(ctx context.Context, dir string) (Report, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "MANIFEST.json"))
	if err != nil {
		return Report{}, fmt.Errorf("%w: no MANIFEST.json", ErrFiles)
	}
	var man struct {
		Tenant string            `json:"tenant"`
		Files  map[string]string `json:"files"`
	}
	if err := json.Unmarshal(raw, &man); err != nil || man.Tenant == "" {
		return Report{}, fmt.Errorf("%w: MANIFEST.json is unreadable", ErrFiles)
	}
	rep := Report{Tenant: man.Tenant, Files: len(man.Files)}
	listed := map[string]bool{}
	for name, want := range man.Files {
		if !safePath.MatchString(name) || strings.Contains(name, "..") {
			return rep, fmt.Errorf("%w: bad path %q", ErrFiles, name)
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil || sum(b) != want {
			return rep, fmt.Errorf("%w: %s is missing or altered", ErrFiles, name)
		}
		listed[name] = true
	}
	if m := missing(listed); len(m) > 0 {
		return rep, fmt.Errorf("%w: missing %s", ErrFiles, strings.Join(m, ", "))
	}
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if rel != "MANIFEST.json" && !listed[rel] {
			return fmt.Errorf("%w: %s isn't in MANIFEST.json", ErrFiles, rel)
		}
		return nil
	})
	if err != nil {
		return rep, err
	}
	key, _ := os.ReadFile(filepath.Join(dir, "audit", "anchor-key.txt"))
	chain, _ := os.ReadFile(filepath.Join(dir, "audit", "audit.ndjson"))
	ar, err := audit.Verify(bytes.NewReader(chain), man.Tenant, strings.TrimSpace(string(key)))
	if err != nil || ar.FirstSeq > 1 || !ar.KeyPinned {
		return rep, fmt.Errorf("%w: %v (first seq %d)", ErrChain, err, ar.FirstSeq)
	}
	rep.Rows, rep.LastSeq = ar.Rows, ar.LastSeq
	var m release.Manifest
	braw, _ := os.ReadFile(filepath.Join(dir, "release", "bundle.json"))
	if err := json.Unmarshal(braw, &m); err != nil {
		return rep, fmt.Errorf("%w: bundle.json is unreadable", ErrBundle)
	}
	rk, _ := os.ReadFile(filepath.Join(dir, "release", "release-key.txt"))
	if _, _, err := release.Admit(ctx, dirStore{dir}, m.ID, release.Ed25519Verifier{Trusted: strings.TrimSpace(string(rk))}, nil); err != nil {
		return rep, fmt.Errorf("%w: %v", ErrBundle, err)
	}
	rep.Release = m.ID
	return rep, nil
}
