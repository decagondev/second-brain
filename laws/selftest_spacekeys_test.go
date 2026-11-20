package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
)

func TestLAW18_SpaceKeysShredAloneAndSurviveHandoff(t *testing.T) {
	Check(t, "LAW-18", 40, func(r *rand.Rand) error { return law18Spaces(r, spaces.Sync, keywrap.Shred) })
}

// keptWraps ignores deletes, so a dropped space's key survives.
type keptWraps struct{ keywrap.Store }

func (keptWraps) Delete(context.Context, string, string, string) error { return nil }

// LAW-18's space-key test catches a sync that never shreds, and a shred that removes any key.
func TestSelf_LAW18SpaceKeyMutationsAreCaught(t *testing.T) {
	noShred := func(ctx context.Context, st ports.SpaceStore, keys *spaces.Keys, tn manifest.Tenant) error {
		k := *keys
		k.Wraps = keptWraps{keys.Wraps}
		return spaces.Sync(ctx, st, &k, tn)
	}
	if !violates(40, func(r *rand.Rand) error { return law18Spaces(r, noShred, keywrap.Shred) }) {
		t.Error("LAW-18 (space keys): a sync that never shreds a dropped space's key was not caught")
	}
	anyShred := func(ctx context.Context, tenant string, st keywrap.Store, purpose string) error {
		ws, err := st.List(ctx, tenant)
		if err != nil {
			return err
		}
		for _, w := range ws {
			if w.Purpose == purpose {
				_ = st.Delete(ctx, tenant, purpose, w.Custodian)
			}
		}
		return nil
	}
	if !violates(40, func(r *rand.Rand) error { return law18Spaces(r, spaces.Sync, anyShred) }) {
		t.Error("LAW-18 (space keys): a shred that removes any purpose was not caught")
	}
}

func TestLAW10_SpaceKillStopsTheSpaceOnly(t *testing.T) {
	Check(t, "LAW-10", 40, func(r *rand.Rand) error {
		return law10Spaces(r, func(c *kill.Checker) func(string) bool { return c.SpaceKilled })
	})
}

func TestLAW11_SpaceIsInTheChain(t *testing.T) {
	Check(t, "LAW-11", runs, func(r *rand.Rand) error { return law11Spaces(r, audit.HashOf) })
}

// LAW-10's space test catches retrieval that ignores space kills; LAW-11's catches a hash
// that leaves the space out.
func TestSelf_SpaceKillAndChainMutationsAreCaught(t *testing.T) {
	if !violates(40, func(r *rand.Rand) error { return law10Spaces(r, func(*kill.Checker) func(string) bool { return nil }) }) {
		t.Error("LAW-10 (space): retrieval that ignores a killed space was not caught")
	}
	blind := func(row audit.Row) (string, error) { row.Space = ""; return audit.HashOf(row) }
	if !violates(runs, func(r *rand.Rand) error { return law11Spaces(r, blind) }) {
		t.Error("LAW-11 (space): a hash without the space was not caught")
	}
}
