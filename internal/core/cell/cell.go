// Package cell is the cell template (sb-tenancy(7), ADR-0001, ADR-0018): what every
// tenant's cell contains, declared once and cloud-neutral, and stamped through the
// Placement and Keys ports. Only tenant and region come from the manifest.
package cell

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// Resource kinds, in stamp order.
const (
	KindDatabase       = "database"
	KindIndexSchema    = "index-schema"
	KindVaultNamespace = "vault-namespace"
	KindRootKey        = "root-key"
	KindQueue          = "queue"
)

// Template lists the resources of t's cell. It reads only t.Tenant and t.Region.
func Template(t manifest.Tenant) []ports.CellResource {
	r := func(kind, name string) ports.CellResource {
		return ports.CellResource{Kind: kind, Name: name, Tenant: t.Tenant, Region: t.Region}
	}
	return []ports.CellResource{
		r(KindDatabase, "sb_"+strings.ReplaceAll(t.Tenant, "-", "_")),
		r(KindIndexSchema, "idx"),
		r(KindVaultNamespace, "tenants/"+t.Tenant),
		r(KindRootKey, "cell-"+t.Tenant+"-root"),
		r(KindQueue, t.Tenant+"-jobs"),
	}
}

// Stamped is a template resource with the id its placement or key store gave it.
type Stamped struct {
	ports.CellResource
	ID string
}

// Stamp creates whatever of t's cell is missing and returns every resource with its id.
// It is idempotent: stamping a complete cell creates nothing and returns the same ids.
// The root key goes to the key store; everything else goes to the placement.
func Stamp(ctx context.Context, p ports.Placement, k ports.Keys, t manifest.Tenant) ([]Stamped, error) {
	var out []Stamped
	for _, r := range Template(t) {
		var id string
		var err error
		if r.Kind == KindRootKey {
			id, err = k.EnsureRootKey(ctx, r)
		} else {
			id, err = p.Ensure(ctx, r)
		}
		if err != nil {
			return nil, fmt.Errorf("stamp %s %s in %s: %w", r.Kind, r.Name, p.Account(), err)
		}
		if id == "" {
			return nil, fmt.Errorf("stamp %s %s in %s: placement returned no id", r.Kind, r.Name, p.Account())
		}
		out = append(out, Stamped{CellResource: r, ID: id})
	}
	return out, nil
}

// NewDataKey makes a fresh 256-bit data key and wraps it under the cell root key.
// Callers keep only the wrapped form at rest (envelope encryption).
func NewDataKey(ctx context.Context, k ports.Keys, rootKeyID string) (plain, wrapped []byte, err error) {
	plain = make([]byte, 32)
	if _, err := rand.Read(plain); err != nil {
		return nil, nil, err
	}
	wrapped, err = k.Wrap(ctx, rootKeyID, plain)
	if err != nil {
		return nil, nil, err
	}
	return plain, wrapped, nil
}
