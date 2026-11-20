package laws

import (
	"math/rand"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/manifest"
)

func TestLAW20_NoScopeReachesARestrictedSpace(t *testing.T) {
	Check(t, "LAW-20", runs, func(r *rand.Rand) error { return law20Reach(r, RealManifestDiags) })
}

// The reach law test catches a check without rule 13.
func TestSelf_ReachMutationIsCaught(t *testing.T) {
	without := func(src []byte) []manifest.Diag {
		var keep []manifest.Diag
		for _, d := range RealManifestDiags(src) {
			if !strings.Contains(d.Msg, "reaches restricted space") {
				keep = append(keep, d)
			}
		}
		return keep
	}
	if !violates(runs, func(r *rand.Rand) error { return law20Reach(r, without) }) {
		t.Error("LAW-20 (reach): a check without rule 13 was not caught")
	}
}
