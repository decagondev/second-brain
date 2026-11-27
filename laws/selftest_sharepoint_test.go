package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/adapters/msgraph"
	"example.com/second-brain/internal/core/ports"
)

func TestLAW03_SharePointUnknownIsNeverAllow(t *testing.T) {
	w := NewSharePointWorld()
	Check(t, "LAW-03", 20, func(r *rand.Rand) error { return law03Connector(r, w) })
}

func TestLAW01_SharePointAccessIsLive(t *testing.T) {
	Check(t, "LAW-01", 20, func(r *rand.Rand) error { return law01SharePoint(r, liveSharePoint) })
}

// SB-5901: a check that keeps its first answer instead of asking Graph each time is caught.
func TestSelf_SharePointMutationsAreCaught(t *testing.T) {
	cached := map[string]ports.Verdict{}
	stale := func(sp *msgraph.SharePoint, ref ports.SourceRef) ports.Verdict {
		if v, ok := cached[ref.ID]; ok {
			return v
		}
		v := liveSharePoint(sp, ref)
		cached[ref.ID] = v
		return v
	}
	if !violates(20, func(r *rand.Rand) error {
		clear(cached)
		return law01SharePoint(r, stale)
	}) {
		t.Error("LAW-01 (sharepoint): a check that keeps its first answer was not caught")
	}
}
