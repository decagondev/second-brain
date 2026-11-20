package laws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"example.com/second-brain/internal/adapters/placement/local"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/seal"
	"example.com/second-brain/internal/core/spaces"
)

// LAW-18 with per-space data keys (SB-1003): over random runs of adding and dropping
// spaces, sealing records, re-wrapping to the tenant and destroying the factory's wraps,
// the audit signing key and every live space's key always have a working wrap, every live
// space's records open to what was sealed, a dropped space's records never open again,
// and only a space's key can ever be shredded.

// SpaceSync applies a manifest's spaces with keys (spaces.Sync in the real system).
type SpaceSync func(ctx context.Context, st ports.SpaceStore, keys *spaces.Keys, t manifest.Tenant) error

// Shredder shreds one purpose (keywrap.Shred in the real system).
type Shredder func(ctx context.Context, tenant string, st keywrap.Store, purpose string) error

func law18Spaces(r *rand.Rand, sync SpaceSync, shred Shredder) error {
	ctx := context.Background()
	root, err := os.MkdirTemp("", "sb-law18s-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(root) }()
	const tenant = "gamma-parts"
	key := func(name string) keywrap.Custodian {
		a := local.Account{Root: filepath.Join(root, "accounts"), Name: name + "-acct"}
		rs, err := cell.Stamp(ctx, a, a, manifest.Tenant{Tenant: tenant, Region: "eu"})
		if err != nil {
			panic(err)
		}
		for _, x := range rs {
			if x.Kind == cell.KindRootKey {
				return keywrap.Custodian{Name: name, Keys: a, KeyID: x.ID}
			}
		}
		panic("no root key")
	}
	factory, tenantC := key("factory"), key("tenant")
	wraps := &keywrap.MemoryStore{}
	if err := keywrap.Seed(ctx, tenant, wraps, factory, keywrap.PurposeAuditSigning); err != nil {
		return err
	}
	reg := spacemem.New()
	sealer := seal.Sealer{Tenant: tenant, Wraps: wraps, Custodians: []keywrap.Custodian{factory, tenantC}}
	custodian := factory
	live := map[string]manifest.Space{}
	sealed := map[string][][2][]byte{} // space id -> (plain, sealed)
	dropped := map[string][][]byte{}
	names := []string{"finance", "people", "eng", "legal"}
	apply := func() error {
		return sync(ctx, reg, &spaces.Keys{Wraps: wraps, Custodian: custodian}, manifest.Tenant{Tenant: tenant, Spaces: live})
	}
	for i := 0; i < 14; i++ {
		switch r.Intn(6) {
		case 0, 1: // add a space
			n := names[r.Intn(len(names))]
			if _, ok := live[n]; ok {
				continue
			}
			live[n] = manifest.Space{Template: "custom", Members: []string{"group:" + n}, Label: "internal"}
			if err := apply(); err != nil {
				return fmt.Errorf("adding %s: %w", n, err)
			}
		case 2: // drop a space
			for n := range live {
				delete(live, n)
				if err := apply(); err != nil {
					return fmt.Errorf("dropping %s: %w", n, err)
				}
				for _, p := range sealed[spaces.ID(n)] {
					dropped[spaces.ID(n)] = append(dropped[spaces.ID(n)], p[1])
				}
				delete(sealed, spaces.ID(n))
				break
			}
		case 3: // seal a record
			for n := range live {
				plain := []byte(fmt.Sprintf("record %d of %s", i, n))
				out, err := sealer.Seal(ctx, spaces.ID(n), plain)
				if err != nil {
					return fmt.Errorf("sealing in %s: %w", n, err)
				}
				sealed[spaces.ID(n)] = append(sealed[spaces.ID(n)], [2][]byte{plain, out})
				break
			}
		case 4: // the hand-off, once: re-wrap every key to the tenant, then destroy the factory's wraps
			if custodian.Name == tenantC.Name {
				continue
			}
			if _, err := keywrap.Rewrap(ctx, tenant, wraps, factory, tenantC); err != nil {
				return fmt.Errorf("rewrap: %w", err)
			}
			if err := keywrap.Destroy(ctx, tenant, wraps, factory, tenantC); err != nil {
				return fmt.Errorf("destroy after rewrap: %w", err)
			}
			custodian = tenantC
		default: // try to shred something that isn't a space
			p := []string{keywrap.PurposeAuditSigning, "space/", "backup"}[r.Intn(3)]
			if err := shred(ctx, tenant, wraps, p); err == nil {
				if inv, _ := keywrap.Inventory(ctx, tenant, wraps, map[string]ports.Keys{"factory": factory.Keys, "tenant": tenantC.Keys}); !has(inv, keywrap.PurposeAuditSigning) {
					return fmt.Errorf("shredding %q removed the audit signing key", p)
				}
				return fmt.Errorf("shredding %q, which isn't a space's key, was allowed", p)
			}
		}
		inv, err := keywrap.Inventory(ctx, tenant, wraps, map[string]ports.Keys{"factory": factory.Keys, "tenant": tenantC.Keys})
		if err != nil {
			return err
		}
		if keywrap.Orphaned(inv) || !has(inv, keywrap.PurposeAuditSigning) {
			return fmt.Errorf("a data key has no working wrap: %+v", inv)
		}
		for n := range live {
			if !has(inv, keywrap.PurposeSpace(spaces.ID(n))) {
				return fmt.Errorf("live space %s has no data key: %+v", n, inv)
			}
		}
		for id, ps := range sealed {
			for _, p := range ps {
				got, err := sealer.Open(ctx, id, p[1])
				if err != nil || !bytes.Equal(got, p[0]) {
					return fmt.Errorf("a record of live space %s doesn't open: %v", id, err)
				}
			}
		}
		for id, ps := range dropped {
			_, readded := live[id[len("spc_"):]] // re-added: a new key, and the old records still never open
			for _, p := range ps {
				if _, err := sealer.Open(ctx, id, p); err == nil || !readded && !errors.Is(err, seal.ErrShredded) {
					return fmt.Errorf("a record of dropped space %s opened or failed wrongly: %v", id, err)
				}
			}
		}
	}
	return nil
}

func has(inv []keywrap.Entry, purpose string) bool {
	for _, e := range inv {
		if e.Purpose == purpose && len(e.Working) > 0 {
			return true
		}
	}
	return false
}
