package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/keywrap"
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
