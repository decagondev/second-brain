// Package filestore keeps sync cursors in files (development and tests):
// ROOT/cursors/TENANT/CONNECTOR, written atomically. Cells keep them in their database.
package filestore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"example.com/second-brain/internal/core/ports"
)

// Store is rooted at a directory.
type Store struct{ Root string }

var _ ports.CursorStore = Store{}

var safe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func (s Store) path(tenant, connector string) (string, error) {
	if !safe.MatchString(tenant) || !safe.MatchString(connector) {
		return "", fmt.Errorf("filestore: unsafe name %q/%q", tenant, connector)
	}
	return filepath.Join(s.Root, "cursors", tenant, connector), nil
}

func (s Store) Cursor(_ context.Context, tenant, connector string) (ports.Cursor, bool, error) {
	p, err := s.path(tenant, connector)
	if err != nil {
		return "", false, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return ports.Cursor(b), true, nil
}

func (s Store) SaveCursor(_ context.Context, tenant, connector string, c ports.Cursor) error {
	p, err := s.path(tenant, connector)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(c), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
