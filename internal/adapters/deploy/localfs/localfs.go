// Package localfs is a local cell's deployer (runtime.Deployer; SB-1505): it records which
// image, by digest, and which release each service should run, under ROOT/services, and
// which images this cell has attested, under ROOT/attested. Like Binary Authorization on
// GCP, it refuses to deploy an image the cell hasn't attested. The local launcher (`make
// cell`, SB-1508) starts each service at the release recorded here.
package localfs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"example.com/second-brain/internal/core/runtime"
)

// Dir is rooted at Root.
type Dir struct{ Root string }

var _ runtime.Deployer = Dir{}

// ErrNotAttested is a deploy of an image the cell never attested.
var ErrNotAttested = errors.New("deploy: the image isn't attested by this cell; refused")

type running struct {
	Image   string `json:"image"`
	Release string `json:"release"`
}

func key(image string) string { s := sha256.Sum256([]byte(image)); return hex.EncodeToString(s[:]) }

func digestOnly(image string) error {
	at := strings.LastIndex(image, "@sha256:")
	if at < 0 || len(image)-at-len("@sha256:") != 64 {
		return fmt.Errorf("deploy: %q isn't pinned by digest; tags are never deployed", image)
	}
	return nil
}

// Attest implements runtime.Deployer.
func (d Dir) Attest(_ context.Context, image string) error {
	if err := digestOnly(image); err != nil {
		return err
	}
	dir := filepath.Join(d.Root, "attested")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, key(image)), []byte(image+"\n"), 0o600)
}

// Deploy implements runtime.Deployer.
func (d Dir) Deploy(_ context.Context, svc runtime.Service, image, releaseID string) error {
	if err := digestOnly(image); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(d.Root, "attested", key(image))); err != nil {
		return fmt.Errorf("%w: %s", ErrNotAttested, image)
	}
	dir := filepath.Join(d.Root, "services")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(running{Image: image, Release: releaseID})
	tmp := filepath.Join(dir, "."+svc.Name+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, svc.Name+".json"))
}

// Running implements runtime.Deployer.
func (d Dir) Running(_ context.Context, svc runtime.Service) (string, string, error) {
	b, err := os.ReadFile(filepath.Join(d.Root, "services", svc.Name+".json"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	var r running
	if err := json.Unmarshal(b, &r); err != nil {
		return "", "", fmt.Errorf("deploy: %s's record can't be read: %w", svc.Name, err)
	}
	return r.Image, r.Release, nil
}
