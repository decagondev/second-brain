// Package local is the Placement and Keys adapter for development and tests (ADR-0018).
// An "account" is a directory: ROOT/ACCOUNT/TENANT/KIND.json for resources and
// ROOT/ACCOUNT/TENANT/root-key.key for the cell root key. Cloud adapters (AWS, GCP,
// Azure) implement the same ports and pass the same contract tests.
package local

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// Account is one mock cloud account rooted in a directory.
type Account struct {
	Root, Name string
}

var (
	_ ports.Placement = Account{}
	_ ports.Keys      = Account{}

	safe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

// Account returns the placement name, as a custody record would carry it.
func (a Account) Account() string { return "local://" + a.Name }

type record struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Tenant string `json:"tenant"`
	Region string `json:"region"`
}

func (a Account) dir(r ports.CellResource) (string, error) {
	for _, s := range []string{a.Name, r.Tenant, r.Kind} {
		if !safe.MatchString(s) {
			return "", fmt.Errorf("local: unsafe name %q", s)
		}
	}
	return filepath.Join(a.Root, a.Name, r.Tenant), nil
}

func newID(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(b), nil
}

// Ensure finds or creates one resource and returns its id.
func (a Account) Ensure(_ context.Context, r ports.CellResource) (string, error) {
	dir, err := a.dir(r)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, r.Kind+".json")
	if rec, found, err := read(path); err != nil {
		return "", err
	} else if found {
		if rec.Tenant != r.Tenant || rec.Region != r.Region || rec.Name != r.Name || rec.Kind != r.Kind {
			return "", fmt.Errorf("local: %s exists with different identity (%s %s in %s)", path, rec.Kind, rec.Name, rec.Region)
		}
		return rec.ID, nil
	}
	id, err := newID(fmt.Sprintf("local:%s:%s:%s", a.Name, r.Region, r.Kind))
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(record{ID: id, Kind: r.Kind, Name: r.Name, Tenant: r.Tenant, Region: r.Region})
	if err != nil {
		return "", err
	}
	if err := writeNew(dir, path, b, 0o644); err != nil {
		return "", err
	}
	return id, nil
}

func read(path string) (record, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return record{}, false, nil
	}
	if err != nil {
		return record{}, false, err
	}
	var rec record
	if err := json.Unmarshal(b, &rec); err != nil || rec.ID == "" {
		return record{}, false, fmt.Errorf("local: %s is unreadable", path) // unknown, not absent
	}
	return rec, true, nil
}

// writeNew creates path exclusively, so two concurrent stamps can't both create it.
func writeNew(dir, path string, b []byte, mode os.FileMode) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// EnsureRootKey finds or creates the cell root key and returns its id. The key id names
// the account and cell; the key bytes never leave the adapter.
func (a Account) EnsureRootKey(ctx context.Context, r ports.CellResource) (string, error) {
	if r.Kind != "root-key" {
		return "", fmt.Errorf("local: %s is not a root key", r.Kind)
	}
	id, err := a.Ensure(ctx, r)
	if err != nil {
		return "", err
	}
	dir, _ := a.dir(r)
	keyPath := filepath.Join(dir, "root-key.key")
	if _, err := os.Stat(keyPath); errors.Is(err, fs.ErrNotExist) {
		k := make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return "", err
		}
		if err := writeNew(dir, keyPath, k, 0o600); err != nil && !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	return id, nil
}

func (a Account) aead(keyID string) (cipher.AEAD, error) {
	// keyID is local:ACCOUNT:REGION:root-key-HEX; find its cell by scanning this account.
	if !strings.HasPrefix(keyID, "local:"+a.Name+":") {
		return nil, fmt.Errorf("local: key %s is not in account %s", keyID, a.Name)
	}
	cells, err := os.ReadDir(filepath.Join(a.Root, a.Name))
	if err != nil {
		return nil, err
	}
	for _, c := range cells {
		rec, found, err := read(filepath.Join(a.Root, a.Name, c.Name(), "root-key.json"))
		if err != nil {
			return nil, err
		}
		if !found || rec.ID != keyID {
			continue
		}
		k, err := os.ReadFile(filepath.Join(a.Root, a.Name, c.Name(), "root-key.key"))
		if err != nil {
			return nil, err
		}
		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	}
	return nil, fmt.Errorf("local: no root key %s", keyID)
}

// Wrap encrypts a data key under the root key (AES-256-GCM, random nonce, key id as AAD).
func (a Account) Wrap(_ context.Context, keyID string, dataKey []byte) ([]byte, error) {
	g, err := a.aead(keyID)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return g.Seal(nonce, nonce, dataKey, []byte(keyID)), nil
}

// Unwrap reverses Wrap. A wrapped key from another root key fails to open.
func (a Account) Unwrap(_ context.Context, keyID string, wrapped []byte) ([]byte, error) {
	g, err := a.aead(keyID)
	if err != nil {
		return nil, err
	}
	if len(wrapped) < g.NonceSize() {
		return nil, errors.New("local: wrapped key too short")
	}
	return g.Open(nil, wrapped[:g.NonceSize()], wrapped[g.NonceSize():], []byte(keyID))
}
