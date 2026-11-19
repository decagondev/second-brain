// Package localfs is the local control plane's kill store (file:///DIR endpoints): one
// JSON file per tenant under DIR/kill, replaced atomically on every change.
package localfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"

	"example.com/second-brain/internal/core/kill"
)

// Dir is the control plane's root directory.
type Dir struct{ Root string }

var (
	_          kill.Store = Dir{}
	tenantName            = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)
	mu         sync.Mutex
)

func (d Dir) path(tenant string) (string, error) {
	if !tenantName.MatchString(tenant) {
		return "", fmt.Errorf("kill: bad tenant %q", tenant)
	}
	return filepath.Join(d.Root, "kill", tenant+".json"), nil
}

func (d Dir) read(tenant string) (map[string]kill.Switch, string, error) {
	p, err := d.path(tenant)
	if err != nil {
		return nil, "", err
	}
	m := map[string]kill.Switch{}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return m, p, nil
	}
	if err != nil {
		return nil, "", err
	}
	var list []kill.Switch
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, "", fmt.Errorf("kill: %s is corrupt: %w", p, err)
	}
	for _, s := range list {
		m[s.Scope] = s
	}
	return m, p, nil
}

func write(p string, m map[string]kill.Switch) error {
	list := make([]kill.Switch, 0, len(m))
	for _, s := range m {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Scope < list[j].Scope })
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Set implements kill.Store.
func (d Dir) Set(_ context.Context, tenant string, s kill.Switch) error {
	mu.Lock()
	defer mu.Unlock()
	m, p, err := d.read(tenant)
	if err != nil {
		return err
	}
	m[s.Scope] = s
	return write(p, m)
}

// Clear implements kill.Store.
func (d Dir) Clear(_ context.Context, tenant, scope string) error {
	mu.Lock()
	defer mu.Unlock()
	m, p, err := d.read(tenant)
	if err != nil {
		return err
	}
	delete(m, scope)
	return write(p, m)
}

// List implements kill.Store.
func (d Dir) List(_ context.Context, tenant string) ([]kill.Switch, error) {
	mu.Lock()
	defer mu.Unlock()
	m, _, err := d.read(tenant)
	if err != nil {
		return nil, err
	}
	out := make([]kill.Switch, 0, len(m))
	for _, s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scope < out[j].Scope })
	return out, nil
}
