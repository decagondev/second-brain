// Package localvault is a development vault (sb-tenant(5), Secrets): one file per secret
// under DIR/vault/tenants/…, readable only by its owner. Production cells use a real
// vault behind the same Secrets port.
package localvault

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/secrets"
)

// Dir is a vault rooted at Root (the control plane's file:///DIR).
type Dir struct{ Root string }

var _ ports.Secrets = Dir{}

func (d Dir) path(raw string) (string, error) {
	ref, err := secrets.ParseRef(raw)
	if err != nil {
		return "", err
	}
	return filepath.Join(d.Root, "vault", "tenants", ref.Tenant, filepath.FromSlash(ref.Path)), nil
}

// ErrExposed is a secret file that someone other than its owner can read.
var ErrExposed = errors.New("localvault: secret file is readable by others; chmod 600 it")

// Get reads one secret. Errors name the reference, never the value.
func (d Dir) Get(_ context.Context, raw string) (string, error) {
	p, err := d.path(raw)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("localvault: %s: %w", raw, err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		return "", ErrExposed
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("localvault: %s: %w", raw, err)
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

// Put stores one secret with owner-only permissions (operators and tests).
func (d Dir) Put(raw, value string) error {
	p, err := d.path(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(value), 0o600)
}
