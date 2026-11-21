// Package localfs is the local control plane's access-path source (SB-706): DIR/TENANT.json
// lists the factory's paths to a cell in local accounts, which have no IAM of their own. A
// missing file means no paths; an unreadable one is an error, never "none".
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

	"example.com/second-brain/internal/core/access"
)

// Dir holds one file per tenant.
type Dir struct{ Root string }

var (
	_    access.Source = Dir{}
	safe               = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

// Name implements access.Source.
func (Dir) Name() string { return "local-iam" }

// Paths implements access.Source.
func (d Dir) Paths(_ context.Context, tenant string) ([]access.Path, error) {
	if !safe.MatchString(tenant) {
		return nil, fmt.Errorf("access: unsafe tenant %q", tenant)
	}
	b, err := os.ReadFile(filepath.Join(d.Root, tenant+".json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ps []access.Path
	if err := json.Unmarshal(b, &ps); err != nil {
		return nil, fmt.Errorf("access: %s is unreadable", tenant+".json")
	}
	for i := range ps {
		ps[i].Source = "local-iam"
	}
	return ps, nil
}

// Revoke removes one path from the tenant's file (handoff.Access). Removing a path that
// isn't there is not an error: the revoke is done.
func (d Dir) Revoke(ctx context.Context, tenant, id string) error {
	ps, err := d.Paths(ctx, tenant)
	if err != nil {
		return err
	}
	keep := []access.Path{}
	for _, p := range ps {
		if p.ID != id {
			p.Source = ""
			keep = append(keep, p)
		}
	}
	b, _ := json.MarshalIndent(keep, "", " ")
	if err := os.MkdirAll(d.Root, 0o700); err != nil {
		return err
	}
	tmp := filepath.Join(d.Root, "."+tenant+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(d.Root, tenant+".json"))
}
