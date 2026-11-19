package laws

import (
	"context"
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/adapters/placement/local"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
)

// LAW-02 at the cell level (SB-103): isolation is a property of the template. Cells
// stamped for different tenants, in one account or in several, share no resource, and
// every resource carries its own tenant and region. Stamping again changes nothing.
// The retrieval-level LAW-02 check (law02) stays red until SB-301.

// Stamped is one resource of a stamped cell, with the id its placement gave it.
type Stamped struct{ Kind, Name, ID, Tenant, Region string }

// CellStamper stamps t's cell into account and returns its resources.
type CellStamper func(account string, t manifest.Tenant) ([]Stamped, error)

// NewCellStamper builds the system under test for cell isolation: the real template
// (SB-103) stamped through the local adapter, with mock accounts under root.
var NewCellStamper = func(root string) CellStamper {
	return func(account string, t manifest.Tenant) ([]Stamped, error) {
		a := local.Account{Root: root, Name: account}
		rs, err := cell.Stamp(context.Background(), a, a, t)
		if err != nil {
			return nil, err
		}
		out := make([]Stamped, len(rs))
		for i, r := range rs {
			out[i] = Stamped{Kind: r.Kind, Name: r.Name, ID: r.ID, Tenant: r.Tenant, Region: r.Region}
		}
		return out, nil
	}
}

// CellKinds is what every cell must contain (sb-tenancy(7) Cell anatomy).
var CellKinds = []string{"database", "index-schema", "vault-namespace", "root-key", "queue"}

func law02Cells(r *rand.Rand, stamp CellStamper) error {
	names := []string{"acme-logistics", "beta-freight", "gamma-parts"}
	r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	owner := map[string]string{} // resource id -> tenant
	for _, name := range names[:2+r.Intn(2)] {
		t := genTenant(r)
		t.Tenant = name
		account := []string{"acct-a", "acct-b"}[r.Intn(2)]
		first, err := stamp(account, t)
		if err != nil {
			return err
		}
		again, err := stamp(account, t)
		if err != nil {
			return err
		}
		if fmt.Sprint(first) != fmt.Sprint(again) {
			return fmt.Errorf("stamping %s twice gave different resources:\n%v\n%v", name, first, again)
		}
		kinds := map[string]bool{}
		for _, res := range first {
			kinds[res.Kind] = true
			if res.Tenant != t.Tenant || res.Region != t.Region {
				return fmt.Errorf("%s %s of %s carries tenant %q region %q", res.Kind, res.ID, name, res.Tenant, res.Region)
			}
			if res.ID == "" {
				return fmt.Errorf("%s of %s has no id", res.Kind, name)
			}
			if o, shared := owner[res.ID]; shared {
				return fmt.Errorf("resource %s is shared by %s and %s", res.ID, o, name)
			}
			owner[res.ID] = name
		}
		for _, k := range CellKinds {
			if !kinds[k] {
				return fmt.Errorf("cell %s has no %s", name, k)
			}
		}
	}
	return nil
}
