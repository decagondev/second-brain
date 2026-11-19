package laws

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"

	"example.com/second-brain/internal/adapters/placement/local"
	"example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
)

// LAW-18 in W1 (SB-114): every cell the registry knows has a custody record naming a
// key custodian, and its root key exists and can unwrap the data keys wrapped under it.
// (Hand-off, where the custodian changes, is modelled in Handoff.tla and built in W7.)

// CustodyCheck applies manifests on a fresh local control plane at root, then reports,
// for every tenant, the key custodian on record and whether its root key round-trips.
type CustodyCheck func(root string, ts []manifest.Tenant) (map[string]CustodyState, error)

// CustodyState is what the law inspects for one cell.
type CustodyState struct {
	KeyCustodian string
	KeyWorks     bool
}

// NewCustodyCheck builds the system under test.
var NewCustodyCheck = func() CustodyCheck {
	return func(root string, ts []manifest.Tenant) (map[string]CustodyState, error) {
		apply := NewApplier()
		reg := localfs.Dir{Root: root}
		acct := local.Account{Root: filepath.Join(root, "accounts"), Name: cell.PlacementFactoryAccount}
		out := map[string]CustodyState{}
		ctx := context.Background()
		for _, t := range ts {
			if _, err := apply(root, t); err != nil {
				return nil, err
			}
		}
		for _, t := range ts {
			c, found, err := reg.CellByTenant(ctx, t.Tenant)
			if err != nil || !found {
				return nil, fmt.Errorf("no cell record for %s (found %v, err %v)", t.Tenant, found, err)
			}
			st := CustodyState{KeyCustodian: c.KeyCustodian}
			rs, err := cell.Stamp(ctx, acct, acct, t) // idempotent: finds the existing root key
			if err != nil {
				return nil, err
			}
			for _, r := range rs {
				if r.Kind != cell.KindRootKey {
					continue
				}
				plain, wrapped, err := cell.NewDataKey(ctx, acct, r.ID)
				if err == nil {
					got, err := acct.Unwrap(ctx, r.ID, wrapped)
					st.KeyWorks = err == nil && string(got) == string(plain)
				}
			}
			out[t.Tenant] = st
		}
		return out, nil
	}
}

func law18(r *rand.Rand, root string, check CustodyCheck) error {
	names := []string{"acme-logistics", "beta-freight", "gamma-parts"}
	var ts []manifest.Tenant
	for _, n := range names[:1+r.Intn(3)] {
		t := genTenant(r)
		t.Tenant = n
		ts = append(ts, t)
	}
	got, err := check(root, ts)
	if err != nil {
		return err
	}
	for _, t := range ts {
		st := got[t.Tenant]
		switch {
		case st.KeyCustodian != cell.KeyCustodianFactory && st.KeyCustodian != cell.KeyCustodianTenant:
			return fmt.Errorf("cell %s has key custodian %q", t.Tenant, st.KeyCustodian)
		case !st.KeyWorks:
			return fmt.Errorf("cell %s's root key does not unwrap its data keys", t.Tenant)
		}
	}
	return nil
}
