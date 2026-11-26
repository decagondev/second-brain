package laws

import (
	"math/rand"
	"testing"
)

// SB-5303: a gate that shows a lent draft to anyone, or to a reader who can open any one of
// its cites, is caught.
func TestSelfLentGateIsCaught(t *testing.T) {
	failsSomewhere(t, "a lent draft for anyone", func(r *rand.Rand) error {
		return law31(r, func([]string, func(string) bool) bool { return true })
	})
	failsSomewhere(t, "a lent draft for whoever opens one cite", func(r *rand.Rand) error {
		return law31(r, func(cs []string, open func(string) bool) bool {
			for _, c := range cs {
				if open(c) {
					return true
				}
			}
			return len(cs) == 0
		})
	})
}
