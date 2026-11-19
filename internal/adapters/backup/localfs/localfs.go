// Package localfs is a write-once backup target on a local or mounted filesystem
// (SB-412). Objects are created exclusively and made read-only; a second write to the
// same name, or a delete, is refused. Each object's retention date sits beside it.
package localfs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/backup"
)

// Dir is one copy.
type Dir struct {
	Root  string
	Label string // name in drill reports
	Media string // e.g. "disk", "nas", "object-store"
	Away  bool   // off-site
}

var _ backup.Target = Dir{}

// ErrExists is a write to a name that's already there (write-once).
var ErrExists = errors.New("backup: object exists; copies are write-once")

func (d Dir) Name() string   { return d.Label }
func (d Dir) Medium() string { return d.Media }
func (d Dir) Offsite() bool  { return d.Away }

func (d Dir) path(name string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(name))
	if strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || clean == "." || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("backup: bad object name %q", name)
	}
	return filepath.Join(d.Root, filepath.FromSlash(clean)), nil
}

// Put implements backup.Target.
func (d Dir) Put(_ context.Context, name string, data []byte, until time.Time) error {
	p, err := d.path(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return ErrExists
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.WriteFile(p+".retain", []byte(until.UTC().Format(time.RFC3339)+"\n"), 0o400); err != nil {
		return err
	}
	return os.Chmod(p, 0o400)
}

// Get implements backup.Target.
func (d Dir) Get(_ context.Context, name string) ([]byte, error) {
	p, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

// List implements backup.Target: every object under prefix, sorted.
func (d Dir) List(_ context.Context, prefix string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(d.Root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if e.IsDir() || strings.HasSuffix(p, ".retain") {
			return nil
		}
		rel, _ := filepath.Rel(d.Root, p)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, prefix) {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// Delete is refused before the retention date, which for audit copies is 7 years.
func (d Dir) Delete(_ context.Context, name string, now time.Time) error {
	p, err := d.path(name)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(p + ".retain")
	if err != nil {
		return fmt.Errorf("backup: no retention record; refusing to delete")
	}
	until, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	if err != nil || now.Before(until) {
		return fmt.Errorf("backup: %s is retained until %s", name, strings.TrimSpace(string(b)))
	}
	return os.Remove(p)
}
