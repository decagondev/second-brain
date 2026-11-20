// Package seal encrypts content held for a space under that space's own data key
// (sb-tenancy(7), Per-space data keys; SB-1003). AES-256-GCM, a fresh random nonce per
// record, and TENANT/spc_NAME as additional data, so a record moved to another space or
// cell doesn't open. Once a space's key is shredded, opening its records is ErrShredded.
package seal

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"example.com/second-brain/internal/core/keywrap"
)

const version = 1

// ErrShredded is a space whose data key no custodian holds any more.
var ErrShredded = errors.New("seal: the space's data key is gone; its content can't be opened")

// ErrOpen is a record that doesn't open: tampered, or sealed for another space or cell.
var ErrOpen = errors.New("seal: the record doesn't open under this space's key")

// Sealer seals and opens one cell's per-space records. Custodians are tried in order, so
// during a hand-off either custodian's wrap opens a key.
type Sealer struct {
	Tenant     string
	Wraps      keywrap.Store
	Custodians []keywrap.Custodian
}

func (s Sealer) aead(ctx context.Context, space string) (cipher.AEAD, error) {
	k, found, err := keywrap.DataKey(ctx, s.Tenant, s.Wraps, keywrap.PurposeSpace(space), s.Custodians...)
	switch {
	case err != nil:
		return nil, fmt.Errorf("seal: %w", err)
	case !found:
		return nil, ErrShredded
	}
	b, err := aes.NewCipher(k)
	if err != nil {
		return nil, fmt.Errorf("seal: %w", err)
	}
	return cipher.NewGCM(b)
}

func (s Sealer) aad(space string) []byte { return []byte(s.Tenant + "/" + space) }

// Seal encrypts plain for space.
func (s Sealer) Seal(ctx context.Context, space string, plain []byte) ([]byte, error) {
	g, err := s.aead(ctx, space)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 1+g.NonceSize(), 1+g.NonceSize()+len(plain)+g.Overhead())
	out[0] = version
	if _, err := rand.Read(out[1:]); err != nil {
		return nil, err
	}
	return g.Seal(out, out[1:], plain, s.aad(space)), nil
}

// Open decrypts a record sealed for space.
func (s Sealer) Open(ctx context.Context, space string, sealed []byte) ([]byte, error) {
	g, err := s.aead(ctx, space)
	if err != nil {
		return nil, err
	}
	n := g.NonceSize()
	if len(sealed) < 1+n+g.Overhead() || sealed[0] != version {
		return nil, ErrOpen
	}
	plain, err := g.Open(nil, sealed[1:1+n], sealed[1+n:], s.aad(space))
	if err != nil {
		return nil, ErrOpen
	}
	return plain, nil
}
