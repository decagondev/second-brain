// Package celltest is the contract every Placement and Keys adapter must pass (ADR-0018):
// local now, and AWS, GCP and Azure when they are built. Call Contract from the
// adapter's own tests.
package celltest

import (
	"bytes"
	"context"
	"testing"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// Adapter is one account: its placement and its key store.
type Adapter struct {
	Placement ports.Placement
	Keys      ports.Keys
}

// Contract runs the shared tests. open must return a fresh, empty account each time it
// is called with a new name, and the same account again for a name it has seen.
func Contract(t *testing.T, open func(account string) Adapter) {
	ctx := context.Background()
	acme := manifest.Tenant{Tenant: "acme-logistics", Region: "eu"}
	beta := manifest.Tenant{Tenant: "beta-freight", Region: "us"}

	t.Run("idempotent", func(t *testing.T) {
		a := open("acct-a")
		first, err := cell.Stamp(ctx, a.Placement, a.Keys, acme)
		if err != nil {
			t.Fatal(err)
		}
		again, err := cell.Stamp(ctx, open("acct-a").Placement, open("acct-a").Keys, acme)
		if err != nil {
			t.Fatal(err)
		}
		for i := range first {
			if first[i].ID != again[i].ID {
				t.Fatalf("%s: id changed on re-stamp: %s then %s", first[i].Kind, first[i].ID, again[i].ID)
			}
		}
	})

	t.Run("no shared resources, same or different accounts", func(t *testing.T) {
		seen := map[string]string{}
		for _, c := range []struct {
			acct string
			t    manifest.Tenant
		}{{"acct-b", acme}, {"acct-b", beta}, {"acct-c", acme}} {
			a := open(c.acct)
			rs, err := cell.Stamp(ctx, a.Placement, a.Keys, c.t)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range rs {
				owner := c.acct + "/" + c.t.Tenant
				if o, dup := seen[r.ID]; dup {
					t.Fatalf("id %s shared by %s and %s", r.ID, o, owner)
				}
				seen[r.ID] = owner
			}
		}
	})

	t.Run("envelope keys", func(t *testing.T) {
		a := open("acct-d")
		ra, err := cell.Stamp(ctx, a.Placement, a.Keys, acme)
		if err != nil {
			t.Fatal(err)
		}
		rb, err := cell.Stamp(ctx, a.Placement, a.Keys, beta)
		if err != nil {
			t.Fatal(err)
		}
		rootA, rootB := rootKey(t, ra), rootKey(t, rb)
		plain, wrapped, err := cell.NewDataKey(ctx, a.Keys, rootA)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(wrapped, plain) {
			t.Fatal("wrapped data key contains the plaintext")
		}
		got, err := a.Keys.Unwrap(ctx, rootA, wrapped)
		if err != nil || !bytes.Equal(got, plain) {
			t.Fatalf("unwrap: %v", err)
		}
		if _, err := a.Keys.Unwrap(ctx, rootB, wrapped); err == nil {
			t.Fatal("another cell's root key unwrapped this cell's data key")
		}
	})
}

func rootKey(t *testing.T, rs []cell.Stamped) string {
	t.Helper()
	for _, r := range rs {
		if r.Kind == cell.KindRootKey {
			return r.ID
		}
	}
	t.Fatal("no root key stamped")
	return ""
}
