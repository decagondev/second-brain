// Package localfs keeps the local control plane's hand-off states (handoff.Store; SB-1507):
// ROOT/TENANT.json, written atomically, so a hand-off survives between sb invocations and
// resumes from its last finished step.
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

	"example.com/second-brain/internal/core/handoff"
)

// Dir is rooted at Root.
type Dir struct{ Root string }

var (
	_    handoff.Store = Dir{}
	safe               = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

func (d Dir) path(tenant string) (string, error) {
	if !safe.MatchString(tenant) {
		return "", fmt.Errorf("handoff: unsafe tenant %q", tenant)
	}
	return filepath.Join(d.Root, tenant+".json"), nil
}

// Get implements handoff.Store; a state that can't be read is an error, never "none".
func (d Dir) Get(_ context.Context, tenant string) (handoff.State, bool, error) {
	p, err := d.path(tenant)
	if err != nil {
		return handoff.State{}, false, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return handoff.State{}, false, nil
	}
	if err != nil {
		return handoff.State{}, false, err
	}
	var s handoff.State
	if err := json.Unmarshal(b, &s); err != nil {
		return handoff.State{}, false, fmt.Errorf("handoff: %s is unreadable", filepath.Base(p))
	}
	return s, true, nil
}

// Put implements handoff.Store.
func (d Dir) Put(_ context.Context, tenant string, s handoff.State) error {
	p, err := d.path(tenant)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d.Root, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", " ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
