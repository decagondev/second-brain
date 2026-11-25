package manifest

import (
	"reflect"
	"strings"
	"testing"
)

const treeHead = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask, draft-reply, close-checklist]
retention_days: 730
spaces:
  finance:
    template: finance
    members: [group:finance]
    stewards: [group:finance-leads]
    label: confidential
    retention_days: 365
    agents: [ask, close-checklist]
    review_after_days: 90
    glossary: { "PO": "purchase order", "AP": "accounts payable" }
    start_here: ["docs:/Finance/Start/"]
    scope: { docs: ["/Finance/**"] }
`

// SB-3901, SB-3902: a team under a department takes its template, label, retention and
// agents, and the department's glossary, start-here list and review interval flow down.
func TestTreeFlowsDown(t *testing.T) {
	src := treeHead + `  ap:
    parent: finance
    members: [group:ap]
    glossary: { "ap": "the AP team" }
    start_here: ["docs:/Finance/AP/Start/"]
    scope: { docs: ["/Finance/AP/**"] }
`
	m, ds := Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	ap := m.Spaces["ap"]
	if ap.Parent != "finance" || ap.Template != "finance" || ap.Label != "confidential" || ap.RetentionDays != 365 {
		t.Fatalf("ap = %+v", ap)
	}
	if !reflect.DeepEqual(ap.Agents, []string{"ask", "close-checklist"}) {
		t.Fatalf("agents = %v", ap.Agents)
	}
	e := m.Effective("ap")
	if e.ReviewAfterDays != 90 || e.Glossary["PO"] != "purchase order" || e.Glossary["ap"] != "the AP team" || e.Glossary["AP"] != "" {
		t.Fatalf("effective = %+v", e)
	}
	if !reflect.DeepEqual(e.StartHere, []string{"docs:/Finance/Start/", "docs:/Finance/AP/Start/"}) {
		t.Fatalf("start_here = %v", e.StartHere)
	}
	if m.Depth("ap") != 3 || m.Depth("finance") != 2 || m.Depth("") != 1 {
		t.Fatal("depth")
	}
	if !reflect.DeepEqual(m.Branch("finance"), []string{"finance", "ap"}) || !reflect.DeepEqual(m.Up("ap"), []string{"ap", "finance"}) {
		t.Fatal("branch or up")
	}
	if !reflect.DeepEqual(m.Effective("finance"), m.Spaces["finance"]) {
		t.Fatal("a department is its own effective space")
	}
	again, ds := Check(Format(m))
	if len(ds) > 0 || again.Spaces["ap"].Parent != "finance" {
		t.Fatalf("format round trip: %v", ds)
	}
	if _, ok := SpaceAttrs(ap)["parent"]; !ok {
		t.Fatal("plan doesn't show the parent")
	}
}

// SB-3901: a fourth level, a missing parent, a cycle and a looser child are refused.
func TestTreeRefused(t *testing.T) {
	cases := map[string]string{
		"is itself under": `  ap:
    parent: finance
    members: [group:ap]
    scope: { docs: ["/Finance/AP/**"] }
  ap-eu:
    parent: ap
    members: [group:ap-eu]
    scope: { docs: ["/Finance/AP/EU/**"] }
`,
		"not a space": `  ap:
    parent: treasury
    members: [group:ap]
    scope: { docs: ["/Finance/AP/**"] }
`,
		"its own parent": `  ap:
    parent: ap
    members: [group:ap]
    scope: { docs: ["/Finance/AP/**"] }
`,
		"looser than its parent": `  ap:
    parent: finance
    label: internal
    members: [group:ap]
    scope: { docs: ["/Finance/AP/**"] }
`,
		"keep less, never more": `  ap:
    parent: finance
    retention_days: 400
    members: [group:ap]
    scope: { docs: ["/Finance/AP/**"] }
`,
		"cycle": `  aa:
    parent: bb
    members: [group:a]
    scope: { docs: ["/A/**"] }
  bb:
    parent: aa
    members: [group:b]
    scope: { docs: ["/B/**"] }
`,
	}
	for want, extra := range cases {
		_, ds := Check([]byte(treeHead + extra))
		got := format("m.yaml", ds)
		if want == "cycle" {
			want = "the tree is at most"
		}
		if !strings.Contains(got, want) {
			t.Errorf("%s: got\n%s", want, got)
		}
		for _, d := range ds {
			if d.Line == 0 {
				t.Errorf("%s: an error without a line: %+v", want, d)
			}
		}
	}
}

// A team may be stricter than its department.
func TestTreeStricterChildIsFine(t *testing.T) {
	_, ds := Check([]byte(treeHead + `  payroll:
    parent: finance
    label: restricted
    stewards: [group:payroll-leads]
    retention_days: 180
    members: [group:finance]
    scope: { docs: ["/Finance/Payroll/**"] }
`))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
}

// SB-4503: the leader's week runs in a space with teams and stewards, unless it is off.
func TestLeaderWeekNeedsTeamsAndStewards(t *testing.T) {
	off := false
	m := Tenant{Spaces: map[string]Space{
		"finance": {Stewards: []string{"group:finance-leads"}},
		"ap":      {Parent: "finance", Stewards: []string{"group:ap-leads"}},
		"ops":     {Stewards: []string{"group:ops-leads"}, LeaderWeek: &off},
		"returns": {Parent: "ops"},
	}}
	for name, want := range map[string]bool{"finance": true, "ap": false, "ops": false, "returns": false, "gone": false} {
		if got := m.LeaderWeek(name); got != want {
			t.Errorf("LeaderWeek(%s) = %v, want %v", name, got, want)
		}
	}
	team := "  ap:\n    parent: finance\n    members: [group:ap]\n    scope: { docs: [\"/Finance/AP/**\"] }\n"
	if _, ds := Check([]byte(strings.Replace(treeHead, "    label: confidential\n", "    label: confidential\n    leader_week: maybe\n", 1) + team)); len(ds) == 0 {
		t.Error("leader_week: maybe was accepted")
	}
	got, ds := Check([]byte(strings.Replace(treeHead, "    label: confidential\n", "    label: confidential\n    leader_week: false\n", 1) + team))
	if len(ds) > 0 || got.LeaderWeek("finance") {
		t.Fatalf("leader_week: false: %v, on %v", ds, got.LeaderWeek("finance"))
	}
	if !strings.Contains(string(Format(got)), "leader_week: false") {
		t.Error("Format dropped leader_week")
	}
}
