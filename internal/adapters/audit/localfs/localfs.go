// Package localfs is the local control plane's audit store (file:///DIR endpoints): one
// append-only NDJSON file per tenant under DIR/audit. Rows are only ever appended; the
// last row is read back to chain the next one.
package localfs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"example.com/second-brain/internal/core/audit"
)

// Dir is the control plane's root directory, for one tenant.
type Dir struct {
	Root   string
	Tenant string
}

var (
	_          audit.Store = Dir{}
	tenantName             = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)
	mu         sync.Mutex
)

func (d Dir) path() (string, error) {
	if !tenantName.MatchString(d.Tenant) {
		return "", fmt.Errorf("audit: bad tenant %q", d.Tenant)
	}
	return filepath.Join(d.Root, "audit", d.Tenant+".ndjson"), nil
}

func (d Dir) all() ([]audit.Row, error) {
	p, err := d.path()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []audit.Row
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		var r audit.Row
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("audit: %s is corrupt: %w", p, err)
		}
		out = append(out, r)
	}
	return out, sc.Err()
}

// Append implements audit.Store.
func (d Dir) Append(_ context.Context, build func(int64, string) (audit.Row, error)) (audit.Row, error) {
	mu.Lock()
	defer mu.Unlock()
	rows, err := d.all()
	if err != nil {
		return audit.Row{}, err
	}
	seq, hash := int64(0), audit.Zero
	if n := len(rows); n > 0 {
		seq, hash = rows[n-1].Seq, rows[n-1].Hash
	}
	r, err := build(seq, hash)
	if err != nil {
		return audit.Row{}, err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return audit.Row{}, err
	}
	p, _ := d.path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return audit.Row{}, err
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return audit.Row{}, err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		return audit.Row{}, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return audit.Row{}, err
	}
	return r, f.Close()
}

// Range implements audit.Store.
func (d Dir) Range(_ context.Context, from int64, limit int) ([]audit.Row, error) {
	mu.Lock()
	defer mu.Unlock()
	rows, err := d.all()
	if err != nil {
		return nil, err
	}
	var out []audit.Row
	for _, r := range rows {
		if r.Seq >= from && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}
