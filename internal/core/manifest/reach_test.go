package manifest

import (
	"strings"
	"testing"
)

// SB-1208 (PT-0003): a scope that reaches into a restricted space's scope is refused
// unless its members are all the restricted space's members.
func TestScopeReach(t *testing.T) {
	base := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
spaces:
  people:
    template: people
    members: [group:people-team]
    stewards: [group:people-leads]
    scope: { docs: ["/People/**"] }
  everything:
    template: custom
    members: [MEMBERS]
    scope: { docs: ["PATTERN"] }
`
	for _, c := range []struct {
		members, pattern string
		refused          bool
	}{
		{"group:eng", "/**", true},
		{"group:eng", "/People/Cases/**", true}, // narrower, inside the restricted scope
		{"group:eng", "/Engineering/**", false},
		{"group:people-team", "/**", false}, // the same people: nothing new to read
	} {
		src := strings.NewReplacer("MEMBERS", c.members, "PATTERN", c.pattern).Replace(base)
		_, ds := Check([]byte(src))
		got := false
		for _, d := range ds {
			got = got || strings.Contains(d.Msg, "reaches restricted space people")
		}
		if got != c.refused {
			t.Errorf("%s %s: refused %v, want %v (%v)", c.members, c.pattern, got, c.refused, ds)
		}
	}
}
