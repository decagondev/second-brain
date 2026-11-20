package manifest

import (
	"encoding/json"
	"strings"
	"testing"
)

const govBase = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
spaces:
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    scope: { docs: ["/Finance/**"] }
  people:
    template: people
    members: [group:people-team]
    stewards: [group:people-leads]
    label: internal
    scope: { docs: ["/People/**"] }
`

// SB-1401: a label hidden behind a YAML comment can't lower a space's label, and no
// value may read back differently from how it was proposed.
func TestStewardChangeRefusesValuesThatReadBackDifferently(t *testing.T) {
	m, ds := Check([]byte(govBase))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	for name, sp := range map[string]map[string]any{
		"label comment":  {"label": "public #", "members": []string{"group:finance"}, "stewards": []string{"group:finance-leads"}, "scope": map[string][]string{"docs": {"/Finance/**"}}},
		"unknown label":  {"label": "top-secret", "members": []string{"group:finance"}, "stewards": []string{"group:finance-leads"}, "scope": map[string][]string{"docs": {"/Finance/**"}}},
		"member comment": {"members": []string{"group:finance #, group:eng"}, "stewards": []string{"group:finance-leads"}, "scope": map[string][]string{"docs": {"/Finance/**"}}},
	} {
		raw, _ := json.Marshal(sp)
		next, ds, err := StewardChange(m, "finance", raw)
		if err == nil && len(ds) == 0 && next.Spaces["finance"].Label != "restricted" {
			t.Errorf("%s: accepted with label %q", name, next.Spaces["finance"].Label)
		}
		if err == nil && len(ds) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
}

// SB-1401: the diff names the space a line belongs to, so a change can't hide behind the
// same text in another space.
func TestDiffNamesTheSpace(t *testing.T) {
	m, _ := Check([]byte(govBase))
	next := m
	next.Spaces = map[string]Space{}
	for k, v := range m.Spaces {
		next.Spaces[k] = v
	}
	f := next.Spaces["finance"]
	f.Label, f.Members = "internal", []string{"group:people-team"}
	next.Spaces["finance"] = f
	d := strings.Join(Diff(m, next), "\n")
	for _, want := range []string{"- spaces.finance.label: restricted", "+ spaces.finance.label: internal", "+ spaces.finance.members: [group:people-team]"} {
		if !strings.Contains(d, want) {
			t.Errorf("diff misses %q:\n%s", want, d)
		}
	}
}
