// Package keywrap is a cell's wrap inventory and the hand-off key steps (sb-factory(7), Key
// wraps, exactly; SB-704). Data keys are stored only wrapped by a custodian's root key; a
// hand-off re-wraps them under the new custodian and never re-encrypts data. Nothing here
// removes the last working wrap of a data key (LAW-18).
package keywrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"example.com/second-brain/internal/core/ports"
)

// Purposes of a cell's data keys.
const PurposeAuditSigning = "audit-signing"

// PurposeSpace is the purpose of a space's data key (SB-1003), e.g. space/spc_finance.
func PurposeSpace(spaceID string) string { return spacePrefix + spaceID }

const spacePrefix = "space/"

// ErrNotSpace refuses to shred a purpose that isn't a space's.
var ErrNotSpace = errors.New("keywrap: ELAW only a space's data key can be shredded (LAW-18)")

// Shred removes every custodian's wrap of a space's data key, so nothing sealed under it
// can be opened again (sb-tenancy(7), Per-space data keys). It refuses any other purpose.
func Shred(ctx context.Context, tenant string, st Store, purpose string) error {
	if !strings.HasPrefix(purpose, spacePrefix) || len(purpose) == len(spacePrefix) {
		return fmt.Errorf("%w: %q", ErrNotSpace, purpose)
	}
	ws, err := st.List(ctx, tenant)
	if err != nil {
		return err
	}
	for _, w := range ws {
		if w.Purpose == purpose {
			if err := st.Delete(ctx, tenant, purpose, w.Custodian); err != nil {
				return err
			}
		}
	}
	return nil
}

// DataKey unwraps a purpose's data key with the first custodian whose wrap unwraps now.
// found is false when no custodian holds a wrap of it at all (never made, or shredded).
func DataKey(ctx context.Context, tenant string, st Store, purpose string, cs ...Custodian) (key []byte, found bool, err error) {
	ws, err := st.List(ctx, tenant)
	if err != nil {
		return nil, false, err
	}
	var last error
	for _, w := range ws {
		if w.Purpose != purpose {
			continue
		}
		found = true
		for _, c := range cs {
			if c.Name == w.Custodian {
				k, err := unwrap(ctx, c, w)
				if err == nil {
					return k, true, nil
				}
				last = err
			}
		}
	}
	if found {
		if last == nil {
			last = errors.New("no custodian at hand holds a wrap")
		}
		return nil, true, fmt.Errorf("keywrap: %s doesn't unwrap: %w", purpose, last)
	}
	return nil, false, nil
}

// Wrap is one custodian's wrap of one data key.
type Wrap struct {
	Purpose   string `json:"purpose"`
	Custodian string `json:"custodian"`
	KeyID     string `json:"key_id"`
	Wrapped   []byte `json:"wrapped"`
}

// Store keeps a cell's wraps; Put replaces the wrap of the same purpose and custodian.
type Store interface {
	List(ctx context.Context, tenant string) ([]Wrap, error)
	Put(ctx context.Context, tenant string, w Wrap) error
	Delete(ctx context.Context, tenant, purpose, custodian string) error
}

// Custodian is a holder of a root key: its name, its key store and its root key id.
type Custodian struct {
	Name  string
	Keys  ports.Keys
	KeyID string
}

// Errors.
var (
	ErrOrphan  = errors.New("keywrap: ELAW the wrap can't go: another custodian must unwrap every data key now (LAW-18)")
	ErrMissing = errors.New("keywrap: no wrap to start from")
	ErrMatch   = errors.New("keywrap: the new wrap doesn't give back the same data key")
)

func find(ws []Wrap, purpose, custodian string) (Wrap, bool) {
	for _, w := range ws {
		if w.Purpose == purpose && w.Custodian == custodian {
			return w, true
		}
	}
	return Wrap{}, false
}

func purposes(ws []Wrap) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range ws {
		if !seen[w.Purpose] {
			seen[w.Purpose] = true
			out = append(out, w.Purpose)
		}
	}
	sort.Strings(out)
	return out
}

func unwrap(ctx context.Context, c Custodian, w Wrap) ([]byte, error) {
	if w.KeyID != c.KeyID {
		return nil, fmt.Errorf("keywrap: %s's wrap of %s is under %s, not %s", c.Name, w.Purpose, w.KeyID, c.KeyID)
	}
	return c.Keys.Unwrap(ctx, c.KeyID, w.Wrapped)
}

// Seed makes a data key for purpose wrapped by c, unless c already has a wrap for it.
func Seed(ctx context.Context, tenant string, st Store, c Custodian, purpose string) error {
	ws, err := st.List(ctx, tenant)
	if err != nil {
		return err
	}
	if _, ok := find(ws, purpose, c.Name); ok {
		return nil
	}
	for _, w := range ws {
		if w.Purpose == purpose {
			return fmt.Errorf("keywrap: %s already has a data key under %s; re-wrap it instead", purpose, w.Custodian)
		}
	}
	plain := make([]byte, 32)
	if _, err := rand.Read(plain); err != nil {
		return err
	}
	wrapped, err := c.Keys.Wrap(ctx, c.KeyID, plain)
	if err != nil {
		return err
	}
	return st.Put(ctx, tenant, Wrap{Purpose: purpose, Custodian: c.Name, KeyID: c.KeyID, Wrapped: wrapped})
}

// Rewrap gives `to` a wrap of every data key `from` holds, verified by unwrapping it again.
// It is idempotent: a wrap of `to` that already gives back the same key is kept.
func Rewrap(ctx context.Context, tenant string, st Store, from, to Custodian) (int, error) {
	ws, err := st.List(ctx, tenant)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, p := range purposes(ws) {
		fw, ok := find(ws, p, from.Name)
		if !ok {
			return n, fmt.Errorf("%w: %s has no %s wrap", ErrMissing, p, from.Name)
		}
		plain, err := unwrap(ctx, from, fw)
		if err != nil {
			return n, fmt.Errorf("keywrap: unwrap %s with %s: %w", p, from.Name, err)
		}
		if tw, ok := find(ws, p, to.Name); ok {
			if got, err := unwrap(ctx, to, tw); err == nil && bytes.Equal(got, plain) {
				continue
			}
		}
		wrapped, err := to.Keys.Wrap(ctx, to.KeyID, plain)
		if err != nil {
			return n, fmt.Errorf("keywrap: wrap %s under %s: %w", p, to.Name, err)
		}
		w := Wrap{Purpose: p, Custodian: to.Name, KeyID: to.KeyID, Wrapped: wrapped}
		if got, err := unwrap(ctx, to, w); err != nil || !bytes.Equal(got, plain) {
			return n, fmt.Errorf("%w: %s under %s", ErrMatch, p, to.Name)
		}
		if err := st.Put(ctx, tenant, w); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// Destroy removes `gone`'s wraps, only if `keep` unwraps every data key now to the same key
// as `gone` does. It checks at the moment of destroying, never from an earlier record.
func Destroy(ctx context.Context, tenant string, st Store, gone, keep Custodian) error {
	ws, err := st.List(ctx, tenant)
	if err != nil {
		return err
	}
	ps := purposes(ws)
	if len(ps) == 0 {
		return fmt.Errorf("%w: the cell has no data keys on record", ErrOrphan)
	}
	for _, p := range ps {
		kw, ok := find(ws, p, keep.Name)
		if !ok {
			return fmt.Errorf("%w: %s has no %s wrap", ErrOrphan, p, keep.Name)
		}
		kept, err := unwrap(ctx, keep, kw)
		if err != nil {
			return fmt.Errorf("%w: %s's wrap of %s doesn't unwrap: %v", ErrOrphan, keep.Name, p, err)
		}
		if gw, ok := find(ws, p, gone.Name); ok {
			if old, err := unwrap(ctx, gone, gw); err == nil && !bytes.Equal(old, kept) {
				return fmt.Errorf("%w: %s and %s hold different %s keys", ErrOrphan, gone.Name, keep.Name, p)
			}
		}
	}
	for _, p := range ps {
		if _, ok := find(ws, p, gone.Name); ok {
			if err := st.Delete(ctx, tenant, p, gone.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// Abort removes the new custodian's wraps when a hand-off is aborted. It is Destroy with
// the roles swapped, so it too keeps a purpose's wrap unless the other custodian unwraps it now.
func Abort(ctx context.Context, tenant string, st Store, added, keep Custodian) error {
	return Destroy(ctx, tenant, st, added, keep)
}

// Entry is one purpose in the inventory as checked now.
type Entry struct {
	Purpose string   `json:"purpose"`
	Working []string `json:"working"` // custodians whose wrap unwraps now
	Broken  []string `json:"broken,omitempty"`
}

// Inventory unwraps every wrap now with its custodian's key store (from keys, by custodian
// name) and the key the wrap names. A custodian not in keys, or a wrap that fails, is broken.
func Inventory(ctx context.Context, tenant string, st Store, keys map[string]ports.Keys) ([]Entry, error) {
	ws, err := st.List(ctx, tenant)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, p := range purposes(ws) {
		e := Entry{Purpose: p}
		for _, w := range ws {
			if w.Purpose != p {
				continue
			}
			works := false
			if k, ok := keys[w.Custodian]; ok && k != nil {
				_, err := k.Unwrap(ctx, w.KeyID, w.Wrapped)
				works = err == nil
			}
			if works {
				e.Working = append(e.Working, w.Custodian)
			} else {
				e.Broken = append(e.Broken, w.Custodian)
			}
		}
		sort.Strings(e.Working)
		sort.Strings(e.Broken)
		out = append(out, e)
	}
	return out, nil
}

// Orphaned reports whether any purpose has no working wrap.
func Orphaned(es []Entry) bool {
	for _, e := range es {
		if len(e.Working) == 0 {
			return true
		}
	}
	return false
}

// MemoryStore is an in-process Store (development and tests).
type MemoryStore struct {
	mu   sync.Mutex
	m    map[string][]Wrap
	Fail error
}

// List implements Store.
func (m *MemoryStore) List(_ context.Context, tenant string) ([]Wrap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Fail != nil {
		return nil, m.Fail
	}
	return append([]Wrap(nil), m.m[tenant]...), nil
}

// Put implements Store.
func (m *MemoryStore) Put(_ context.Context, tenant string, w Wrap) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.m == nil {
		m.m = map[string][]Wrap{}
	}
	ws := m.m[tenant][:0:0]
	for _, x := range m.m[tenant] {
		if x.Purpose != w.Purpose || x.Custodian != w.Custodian {
			ws = append(ws, x)
		}
	}
	m.m[tenant] = append(ws, w)
	return nil
}

// Delete implements Store.
func (m *MemoryStore) Delete(_ context.Context, tenant, purpose, custodian string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ws []Wrap
	for _, x := range m.m[tenant] {
		if x.Purpose != purpose || x.Custodian != custodian {
			ws = append(ws, x)
		}
	}
	m.m[tenant] = ws
	return nil
}
