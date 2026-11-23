package laws

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"math/rand"
	"strings"
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

// The offline-bundle law catches a check that trusts index.json and skips the blobs.
func TestSelfBundleIndexOnlyIsCaught(t *testing.T) {
	indexOnly := func(r *bytes.Reader, id string) error {
		tr := tar.NewReader(r)
		for {
			h, err := tr.Next()
			if err != nil {
				return errors.New("no index")
			}
			if h.Name == "index.json" {
				b, _ := io.ReadAll(tr)
				if strings.Contains(string(b), id) {
					return nil
				}
				return errors.New("other image")
			}
		}
	}
	failsSomewhere(t, "index-only check", func(r *rand.Rand) error { return law06Bundle(r, indexOnly) })
}
