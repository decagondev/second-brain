package laws

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"

	"example.com/second-brain/internal/adapters/placement/local"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// LAW-18 in the wrap inventory (SB-704): random runs of seed, re-wrap, destroy, drop (abort)
// and a corrupted wrap, across the factory's key store and the tenant's, never leave a data
// key without a working wrap; destroying the factory's wrap succeeds only when the tenant's
// unwraps the same key at that moment.

// WrapWorld is one cell's inventory with two custodians.
type WrapWorld interface {
	Seed(purpose string) error
	Rewrap() error                       // factory -> tenant
	Destroy(gone string) error           // "factory" or "tenant"
	Drop(custodian string) error         // abort's removal
	Corrupt(purpose, custodian string)   // the custodian's wrap stops unwrapping
	Inventory() ([]keywrap.Entry, error) // checked now
}

// NewWrapWorld builds the system under test: keywrap over two local key stores under root.
var NewWrapWorld = func(root string) WrapWorld {
	ctx := context.Background()
	fa := local.Account{Root: filepath.Join(root, "accounts"), Name: "factory-acct"}
	ta := local.Account{Root: filepath.Join(root, "accounts"), Name: "tenant-acct"}
	t := manifest.Tenant{Tenant: "gamma-parts", Region: "eu"}
	key := func(a local.Account) string {
		rs, err := cell.Stamp(ctx, a, a, t)
		if err != nil {
			panic(err)
		}
		for _, r := range rs {
			if r.Kind == cell.KindRootKey {
				return r.ID
			}
		}
		panic("no root key")
	}
	return &wrapWorld{st: &keywrap.MemoryStore{},
		cs: map[string]keywrap.Custodian{"factory": {Name: "factory", Keys: fa, KeyID: key(fa)}, "tenant": {Name: "tenant", Keys: ta, KeyID: key(ta)}}}
}

type wrapWorld struct {
	st *keywrap.MemoryStore
	cs map[string]keywrap.Custodian
}

const wrapTenant = "gamma-parts"

func (w *wrapWorld) Seed(p string) error {
	return keywrap.Seed(context.Background(), wrapTenant, w.st, w.cs["factory"], p)
}
func (w *wrapWorld) Rewrap() error {
	_, err := keywrap.Rewrap(context.Background(), wrapTenant, w.st, w.cs["factory"], w.cs["tenant"])
	return err
}
func (w *wrapWorld) Destroy(gone string) error {
	keep := map[string]string{"factory": "tenant", "tenant": "factory"}[gone]
	return keywrap.Destroy(context.Background(), wrapTenant, w.st, w.cs[gone], w.cs[keep])
}
func (w *wrapWorld) Drop(c string) error {
	keep := map[string]string{"factory": "tenant", "tenant": "factory"}[c]
	return keywrap.Abort(context.Background(), wrapTenant, w.st, w.cs[c], w.cs[keep])
}
func (w *wrapWorld) Corrupt(p, c string) {
	ws, _ := w.st.List(context.Background(), wrapTenant)
	for _, x := range ws {
		if x.Purpose == p && x.Custodian == c && len(x.Wrapped) > 0 {
			x.Wrapped = append([]byte(nil), x.Wrapped...)
			x.Wrapped[len(x.Wrapped)-1] ^= 1
			_ = w.st.Put(context.Background(), wrapTenant, x)
		}
	}
}
func (w *wrapWorld) Inventory() ([]keywrap.Entry, error) {
	return keywrap.Inventory(context.Background(), wrapTenant, w.st, map[string]ports.Keys{"factory": w.cs["factory"].Keys, "tenant": w.cs["tenant"].Keys})
}

func law18Wraps(r *rand.Rand, w WrapWorld) error {
	purposes := []string{keywrap.PurposeAuditSigning, "backup", "secrets"}[:1+r.Intn(3)]
	for _, p := range purposes {
		if err := w.Seed(p); err != nil {
			return err
		}
	}
	corrupted := false // the law allows one human-made failure to be detected, never caused
	for i := 0; i < 12; i++ {
		before, err := w.Inventory()
		if err != nil {
			return err
		}
		var op string
		var opErr error
		switch r.Intn(5) {
		case 0, 1:
			op, opErr = "rewrap", w.Rewrap()
		case 2:
			gone := []string{"factory", "tenant"}[r.Intn(2)]
			op, opErr = "destroy "+gone, w.Destroy(gone)
			if opErr == nil {
				// It may only have succeeded if the other custodian unwrapped every key.
				keep := map[string]string{"factory": "tenant", "tenant": "factory"}[gone]
				for _, e := range before {
					if !contains(e.Working, keep) {
						return fmt.Errorf("destroying %s's wraps succeeded though %s couldn't unwrap %s (%+v)", gone, keep, e.Purpose, e)
					}
				}
			}
		case 3:
			c := []string{"factory", "tenant"}[r.Intn(2)]
			op, opErr = "drop "+c, w.Drop(c)
		default:
			if !corrupted {
				corrupted = true
				p := purposes[r.Intn(len(purposes))]
				c := []string{"factory", "tenant"}[r.Intn(2)]
				w.Corrupt(p, c)
				op = "corrupt " + c + "'s " + p
			}
		}
		_ = opErr // refusals are fine; the inventory is what the law judges
		if strings.HasPrefix(op, "corrupt") {
			continue // the test's own fault injection, not an operation under test
		}
		after, err := w.Inventory()
		if err != nil {
			return err
		}
		if len(after) != len(purposes) {
			return fmt.Errorf("after %s the inventory has %d purposes, want %d: %+v", op, len(after), len(purposes), after)
		}
		for _, e := range after {
			if len(e.Working) == 0 && !orphanedBy(before, e.Purpose) {
				return fmt.Errorf("after %s, %s has no working wrap (before: %+v, after: %+v)", op, e.Purpose, before, after)
			}
		}
	}
	return nil
}

// orphanedBy reports whether a purpose already had no working wrap (a corruption the law test
// itself injected into the only wrap); an operation can't be blamed for that.
func orphanedBy(inv []keywrap.Entry, purpose string) bool {
	for _, e := range inv {
		if e.Purpose == purpose {
			return len(e.Working) == 0
		}
	}
	return false
}
