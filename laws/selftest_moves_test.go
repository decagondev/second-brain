package laws

import (
	"math/rand"
	"testing"
)

// LAW-21 through SCIM catches groups cached across requests, and groups taken from the
// IdP's claims rather than the directory.
func TestSelf_SCIMMoveMutationsAreCaught(t *testing.T) {
	for name, mk := range map[string]func() MoveSystem{
		"cached groups":  func() MoveSystem { m := newSCIMMoves(); m.cacheGroups = true; return m },
		"trusted claims": func() MoveSystem { m := newSCIMMoves(); m.trustClaims = true; return m },
	} {
		if !violates(runs, func(r *rand.Rand) error { return law21(r, mk()) }) {
			t.Errorf("LAW-21: a cell with %s was not caught", name)
		}
	}
}
