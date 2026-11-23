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

func TestSelfSupportWholeValuesOnlyIsCaught(t *testing.T) {
	wholeOnly := func(stored []string, text string) map[int]bool {
		out := map[int]bool{}
		for i, line := range strings.Split(text, "\n") {
			for _, v := range stored {
				if strings.Contains(line, v) {
					out[i+1] = true
				}
			}
		}
		return out
	}
	failsSomewhere(t, "whole-value scan", func(r *rand.Rand) error { return law07Support(r, wholeOnly) })
}

func TestSelfProxyCredentialsAcceptedIsCaught(t *testing.T) {
	failsSomewhere(t, "credentials accepted", func(r *rand.Rand) error { return law07Proxy(r, box.CheckProxyFile) })
}

func TestSelfDSNRequireAcceptedIsCaught(t *testing.T) {
	lax := func(dsn string) error {
		if strings.Contains(dsn, "require") {
			return nil
		}
		return box.CheckDSN(dsn)
	}
	failsSomewhere(t, "sslmode=require accepted", func(r *rand.Rand) error { return law02DSN(r, lax) })
}
