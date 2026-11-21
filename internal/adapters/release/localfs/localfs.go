// Package localfs is a release store on a filesystem (development and tests):
// DIR/<release>/bundle.json, bundle.sig, gate.json, gate.sig and artefacts/<name>.
// A cell reads it; nothing here can reach into a cell.
package localfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Dir is the store's root.
type Dir struct{ Root string }

var (
	idRe   = regexp.MustCompile(`^rel_\d{4}-\d{2}-\d{2}_[0-9a-f]{6}$`)
	nameRe = regexp.MustCompile(`^(bundle\.json|bundle\.sig|gate\.json|gate\.sig|artefacts/[a-z][a-z0-9._-]{0,40})$`)
)

func (d Dir) path(release, name string) (string, error) {
	if !idRe.MatchString(release) || !nameRe.MatchString(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("release store: bad path %s/%s", release, name)
	}
	return filepath.Join(d.Root, release, filepath.FromSlash(name)), nil
}

// Get implements release.Store.
func (d Dir) Get(_ context.Context, release, name string) ([]byte, error) {
	p, err := d.path(release, name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

// Put writes one file (the release pipeline's side).
func (d Dir) Put(release, name string, data []byte) error {
	p, err := d.path(release, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

var channelRe = regexp.MustCompile(`^(stable|testing)$`)

// Channel returns the release a channel should run (DIR/channels/CHANNEL), as the release
// controller last set it.
func (d Dir) Channel(_ context.Context, channel string) (string, error) {
	if !channelRe.MatchString(channel) {
		return "", fmt.Errorf("release store: bad channel %q", channel)
	}
	b, err := os.ReadFile(filepath.Join(d.Root, "channels", channel))
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(b))
	if !idRe.MatchString(id) {
		return "", fmt.Errorf("release store: channel %s names %q, not a release", channel, id)
	}
	return id, nil
}

// SetChannel points a channel at a release (the release controller's side).
func (d Dir) SetChannel(channel, id string) error {
	if !channelRe.MatchString(channel) || !idRe.MatchString(id) {
		return fmt.Errorf("release store: bad channel %q or release %q", channel, id)
	}
	if err := os.MkdirAll(filepath.Join(d.Root, "channels"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d.Root, "channels", channel), []byte(id+"\n"), 0o644)
}
