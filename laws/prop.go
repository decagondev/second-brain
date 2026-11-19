// Package laws holds the platform's law tests (sb-laws(7)). Each law is a check
// function run over many generated worlds against the system under test.
//
// The system under test is wired in by later tickets. Until then it returns
// ErrPending, and the law fails "for the right reason" (SB-010). tools/check/laws.py
// enforces that ratchet against laws/expected_red.txt.
package laws

import (
	"errors"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

// ErrPending means the system under test for this law isn't built yet.
var ErrPending = errors.New("not implemented")

// Check runs f over n deterministic seeds (or only SB_SEED if set) and fails the
// test with the seed on the first violation, so any failure is reproducible.
func Check(t *testing.T, law string, n int, f func(r *rand.Rand) error) {
	t.Helper()
	seeds := make([]int64, 0, n)
	if s := os.Getenv("SB_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("SB_SEED: %v", err)
		}
		seeds = append(seeds, v)
	} else {
		for i := 1; i <= n; i++ {
			seeds = append(seeds, int64(i))
		}
	}
	for _, seed := range seeds {
		err := f(rand.New(rand.NewSource(seed)))
		if errors.Is(err, ErrPending) {
			t.Fatalf("%s: %v", law, err)
		}
		if err != nil {
			t.Fatalf("%s: violated (reproduce with SB_SEED=%d): %v", law, seed, err)
		}
	}
}

// violates reports whether f finds a violation for any of n seeds (used by self-tests).
func violates(n int, f func(r *rand.Rand) error) bool {
	for seed := int64(1); seed <= int64(n); seed++ {
		if err := f(rand.New(rand.NewSource(seed))); err != nil && !errors.Is(err, ErrPending) {
			return true
		}
	}
	return false
}
