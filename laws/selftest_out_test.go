package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/ingest"
)

// SB-5102: a filter that lets text through, or masks numbers but not credentials, is caught.
func TestSelfOutFilterIsCaught(t *testing.T) {
	through := OutFilter{JSON: func(b []byte) []byte { return b }, Notice: func(s string) string { return s }}
	failsSomewhere(t, "a way out with no filter", func(r *rand.Rand) error { return law07Out(r, through) })
	numbersOnly := OutFilter{JSON: func(b []byte) []byte { return []byte(ingest.MaskPII(string(b), false)) }, Notice: func(s string) string { return ingest.MaskPII(s, false) }}
	failsSomewhere(t, "a filter that misses credentials", func(r *rand.Rand) error { return law07Out(r, numbersOnly) })
}
