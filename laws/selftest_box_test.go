package laws

import (
	"math/rand"
	"testing"
	"time"

	"example.com/second-brain/internal/core/box"
)

// The W17 box law catches a judge that treats a missing fact as healthy (SB-1801).

func TestSelfBoxMissingFactAsPassIsCaught(t *testing.T) {
	broken := func(f box.Facts, now time.Time) []box.Line {
		lines := box.Check(f, now)
		for i := range lines {
			if lines[i].Verdict == box.Unknown {
				lines[i].Verdict = box.Pass
			}
		}
		return lines
	}
	failsSomewhere(t, "unknown judged pass", func(r *rand.Rand) error { return law03Box(r, broken) })
}
