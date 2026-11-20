package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/ports"
)

func TestLAW02_IdentityPlatformTenantsStayApart(t *testing.T) {
	w := NewIPWorld()
	Check(t, "LAW-02", runs, func(r *rand.Rand) error { return law02IdentityPlatform(r, w, verifyIP) })
}

func TestLAW12_ManifestGroupRemovalRevokesFast(t *testing.T) {
	Check(t, "LAW-12", runs, func(r *rand.Rand) error { return law12Groups(r, applyGroups) })
}

// LAW-02's Identity Platform check catches a cell that trusts the whole project, so any
// company's token from it signs in.
func TestSelf_LAW02IdentityPlatformMutationIsCaught(t *testing.T) {
	w := NewIPWorld()
	projectOnly := func(w *IPWorld, cell, token string) string {
		for _, p := range w.Cells {
			if _, err := p.Verify(context.Background(), token, "n"); err == nil {
				return cell
			}
		}
		return ""
	}
	if !violates(runs, func(r *rand.Rand) error { return law02IdentityPlatform(r, w, projectOnly) }) {
		t.Error("LAW-02 (Identity Platform): a cell that ignores firebase.tenant was not caught")
	}
}

// LAW-12's groups check catches an apply that adds people but never removes them.
func TestSelf_LAW12GroupsMutationIsCaught(t *testing.T) {
	addOnly := func(ctx context.Context, dir ports.DirectoryStore, sess ports.Sessions, tenant string, _, next map[string][]string) error {
		_, err := groupsync.Apply(ctx, dir, sess, tenant, nil, next)
		return err
	}
	if !violates(runs, func(r *rand.Rand) error { return law12Groups(r, addOnly) }) {
		t.Error("LAW-12 (groups): an apply that never deactivates removed people was not caught")
	}
}
