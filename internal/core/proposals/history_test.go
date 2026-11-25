package proposals

import (
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/manifest"
)

const histHead = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask, close-checklist]
spaces:
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    label: confidential
    agents: [ASPACE]
    scope: { docs: ["/Finance/**"] }
`

func histManifest(t *testing.T, spaceAgents string) (manifest.Tenant, string) {
	t.Helper()
	src := strings.Replace(histHead, "ASPACE", spaceAgents, 1)
	m, ds := manifest.Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("%v\n%s", ds, src)
	}
	return m, string(manifest.Format(m))
}

// SB-5104: an agent's versions come from the approved proposals that changed its settings,
// with who proposed and approved each; the applied manifest is last; a restore puts the old
// settings back and still checks.
func TestAgentHistoryAndRestore(t *testing.T) {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	_, v1 := histManifest(t, "ask")
	_, v2 := histManifest(t, "ask, close-checklist")
	applied, _ := histManifest(t, "ask, close-checklist")
	ps := []Proposal{
		{ID: "prop_2", State: Approved, At: at.Add(2 * time.Hour), Requester: "usr_ann", DecidedBy: "usr_sam", Manifest: v2},
		{ID: "prop_1", State: Approved, At: at, Requester: "usr_ann", DecidedBy: "usr_sam", Manifest: v1},
		{ID: "prop_x", State: Rejected, At: at.Add(time.Hour), Manifest: v2},
		{ID: "prop_same", State: Approved, At: at.Add(3 * time.Hour), Manifest: v2},
	}
	vs := History(ps, applied, func(m manifest.Tenant) []string { return AgentLines(m, "close-checklist") })
	if len(vs) != 2 || vs[0].Proposal != "prop_1" || vs[1].Proposal != "prop_2" || !vs[1].Current || vs[0].Current || vs[1].By != "usr_ann" || vs[1].Approved != "usr_sam" {
		t.Fatalf("versions %+v", vs)
	}
	ch := Changes(vs[0], vs[1])
	if len(ch) != 1 || ch[0] != "+ spaces: finance" {
		t.Fatalf("changes %v", ch)
	}
	back := RestoreAgent(applied, vs[0].Manifest, "close-checklist")
	again, ds := manifest.Check(manifest.Format(back))
	if len(ds) > 0 || strings.Join(AgentLines(again, "close-checklist"), "\n") != strings.Join(vs[0].Lines, "\n") {
		t.Fatalf("restore: %v %v", ds, AgentLines(again, "close-checklist"))
	}
	if strings.Join(AgentLines(applied, "close-checklist"), "\n") == strings.Join(vs[0].Lines, "\n") {
		t.Fatal("restore changed the applied manifest")
	}
}
