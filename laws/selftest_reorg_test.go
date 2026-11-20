package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/reorg"
)

func TestLAW16_ReorgPreviewIsExactAndChangesNothing(t *testing.T) {
	Check(t, "LAW-16", runs, func(r *rand.Rand) error { return law16Reorg(r, reorg.Diff) })
}

// The reorg law test catches a preview that drops the loses-all flag, one that flags only
// the space's template label, and one that rewrites the snapshot.
func TestSelf_ReorgMutationsAreCaught(t *testing.T) {
	noLosesAll := func(tn manifest.Tenant, s reorg.Snapshot) reorg.Report {
		rep := reorg.Diff(tn, s)
		for i := range rep.People {
			var keep []string
			for _, f := range rep.People[i].Flags {
				if f != reorg.LosesAll {
					keep = append(keep, f)
				}
			}
			rep.People[i].Flags = keep
		}
		return rep
	}
	firstGroup := func(tn manifest.Tenant, s reorg.Snapshot) reorg.Report {
		c := s
		c.People = nil
		for _, p := range s.People {
			if len(p.After) > 1 {
				p.After = p.After[:1] // only the first group after the move
			}
			c.People = append(c.People, p)
		}
		return reorg.Diff(tn, c)
	}
	writes := func(tn manifest.Tenant, s reorg.Snapshot) reorg.Report {
		if len(s.People) > 0 {
			s.People[0].Before = nil // "applies" the move to the input
		}
		return reorg.Diff(tn, s)
	}
	for name, d := range map[string]ReorgDiff{"no loses-all": noLosesAll, "first group only": firstGroup, "writes its input": writes} {
		if !violates(runs, func(r *rand.Rand) error { return law16Reorg(r, d) }) {
			t.Errorf("LAW-16 (reorg): a preview with %s was not caught", name)
		}
	}
}
