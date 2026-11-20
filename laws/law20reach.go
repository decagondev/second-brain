package laws

import (
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/route"
)

// LAW-20 at check time (SB-1208, PT-0003): no manifest that sb tenant check accepts has a
// space whose scope reaches into a restricted space's scope for people who aren't that
// space's members, so ingest can never make them holders of restricted documents.

// ManifestDiags is sb tenant check's verdict (manifest.Check's diagnostics in the real system).
type ManifestDiags func(src []byte) []manifest.Diag

// RealManifestDiags is the shipped check.
func RealManifestDiags(src []byte) []manifest.Diag {
	_, ds := manifest.Check(src)
	return ds
}

func law20Reach(r *rand.Rand, check ManifestDiags) error {
	pats := []string{"/**", "/People/**", "/People/Cases/**", "/Finance/**", "/Engineering/**", "/Shared/*"}
	groups := []string{"group:people-team", "group:finance", "group:eng"}
	type sp struct{ name, label, member, pat string }
	var spec []sp
	var b strings.Builder
	b.WriteString("tenant: acme-logistics\nchannel: stable\nidp: oidc://login.acme.example\nregion: eu\nconnectors: { docs: sharepoint }\nagents: [ask]\nspaces:\n")
	for i, name := range []string{"people", "finance", "other"} {
		s := sp{name, []string{"restricted", "internal"}[r.Intn(2)], groups[(i+r.Intn(2))%len(groups)], pats[r.Intn(len(pats))]}
		spec = append(spec, s)
		fmt.Fprintf(&b, "  %s:\n    template: custom\n    label: %s\n    members: [%s]\n    stewards: [group:leads]\n    scope: { docs: [%q] }\n",
			s.name, s.label, s.member, s.pat)
	}
	if len(check([]byte(b.String()))) > 0 {
		return nil // refused: nothing to apply
	}
	for _, a := range spec {
		for _, x := range spec {
			if a.name == x.name || x.label != "restricted" || a.member == x.member {
				continue
			}
			if route.Match(a.pat, x.pat) || route.Match(x.pat, a.pat) {
				return fmt.Errorf("an accepted manifest lets space %s (%s, %q) hold restricted space %s's %q", a.name, a.member, a.pat, x.name, x.pat)
			}
		}
	}
	return nil
}
