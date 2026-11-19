// Package localfs keeps a cell's wrap inventory (keywrap.Store) as DIR/TENANT.json, for the
// local control plane (sb(1), --endpoint file:///DIR; SB-704). Wrapped keys only: a data
// key in the clear never touches the file.
package localfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"example.com/second-brain/internal/core/keywrap"
)

// Dir is one directory of inventories; a package lock serialises writes within a process.
type Dir struct{ Root string }

var (
	_    keywrap.Store = Dir{}
	safe               = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	lock               = &sync.Mutex{}
)

func (d Dir) path(tenant string) (string, error) {
	if !safe.MatchString(tenant) {
		return "", fmt.Errorf("keywraps: unsafe tenant %q", tenant)
	}
	return filepath.Join(d.Root, tenant+".json"), nil
}

func (d Dir) read(tenant string) ([]keywrap.Wrap, error) {
	p, err := d.path(tenant)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ws []keywrap.Wrap
	if err := json.Unmarshal(b, &ws); err != nil {
		return nil, fmt.Errorf("keywraps: %s is unreadable", p) // unknown, not empty
	}
	return ws, nil
}

func (d Dir) write(tenant string, ws []keywrap.Wrap) error {
	p, err := d.path(tenant)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d.Root, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(ws, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// List implements keywrap.Store.
func (d Dir) List(_ context.Context, tenant string) ([]keywrap.Wrap, error) {
	lock.Lock()
	defer lock.Unlock()
	return d.read(tenant)
}

// Put implements keywrap.Store.
func (d Dir) Put(_ context.Context, tenant string, w keywrap.Wrap) error {
	lock.Lock()
	defer lock.Unlock()
	ws, err := d.read(tenant)
	if err != nil {
		return err
	}
	var out []keywrap.Wrap
	for _, x := range ws {
		if x.Purpose != w.Purpose || x.Custodian != w.Custodian {
			out = append(out, x)
		}
	}
	return d.write(tenant, append(out, w))
}

// Delete implements keywrap.Store.
func (d Dir) Delete(_ context.Context, tenant, purpose, custodian string) error {
	lock.Lock()
	defer lock.Unlock()
	ws, err := d.read(tenant)
	if err != nil {
		return err
	}
	var out []keywrap.Wrap
	for _, x := range ws {
		if x.Purpose != purpose || x.Custodian != custodian {
			out = append(out, x)
		}
	}
	return d.write(tenant, out)
}
