package manifest

import (
	"strings"
	"testing"
)

// SB-5002: a space's routing is one of three policies, a team inherits its department's and
// can't go above it, and Format reads it back.
func TestSpaceRouting(t *testing.T) {
	with := func(line string) string {
		return strings.Replace(treeHead, "    label: confidential\n", "    label: confidential\n"+line, 1)
	}
	if _, ds := Check([]byte(with("    routing: cheapest\n"))); len(ds) == 0 {
		t.Fatal("routing: cheapest was accepted")
	}
	team := "  ap:\n    parent: finance\n    members: [group:ap]\n    scope: { docs: [\"/Finance/AP/**\"] }\n"
	m, ds := Check([]byte(with("    routing: economy\n") + team))
	if len(ds) > 0 || m.RoutingOf("finance") != "economy" || m.RoutingOf("ap") != "economy" {
		t.Fatalf("economy: %v %q %q", ds, m.RoutingOf("finance"), m.RoutingOf("ap"))
	}
	if _, ds := Check([]byte(with("    routing: economy\n") + team + "    routing: frontier\n")); len(ds) == 0 || !strings.Contains(ds[0].Msg, "above its department") {
		t.Fatalf("a team above its department: %v", ds)
	}
	again, ds := Check(Format(m))
	if len(ds) > 0 || again.RoutingOf("finance") != "economy" {
		t.Fatalf("Format dropped routing: %v\n%s", ds, Format(m))
	}
	if plain, _ := Check([]byte(treeHead)); plain.RoutingOf("finance") != DefaultRouting {
		t.Fatalf("default: %q", plain.RoutingOf("finance"))
	}
}

// SB-5003, SB-5004: a personal limit is a number in range; an agent's job is one short
// sentence and its targets are up to three known metrics, each with one bound.
func TestLimitsAndAgentGoals(t *testing.T) {
	agents := func(goal string) string {
		return strings.Replace(treeHead, "agents: [ask, draft-reply, close-checklist]\n",
			"agents:\n  ask: {}\n  draft-reply: {}\n  close-checklist: "+goal+"\n", 1)
	}
	for _, bad := range []string{
		"limits: { monthly_usd_per_person: 0 }\n",
		"limits: { monthly_usd_per_person: \"20\" }\n",
		"limits: { per_person: 20 }\n",
	} {
		if _, ds := Check([]byte(treeHead + bad)); len(ds) == 0 {
			t.Errorf("%q was accepted", bad)
		}
	}
	for _, bad := range []string{
		`{ job: "` + strings.Repeat("a", MaxJob+1) + `" }`,
		`{ targets: [{ metric: speed, at_least: 1 }] }`,
		`{ targets: [{ metric: discard_rate }] }`,
		`{ targets: [{ metric: discard_rate, at_most: 0.1, at_least: 0 }] }`,
		`{ targets: [{ metric: discard_rate, at_most: 0.1 }, { metric: discard_rate, at_most: 0.2 }] }`,
	} {
		if _, ds := Check([]byte(agents(bad))); len(ds) == 0 {
			t.Errorf("%s was accepted", bad)
		}
	}
	src := agents(`{ monthly_usd: 40, job: "Gets the month closed on time", targets: [{ metric: published_as_is, at_least: 0.6 }, { metric: hours_to_publish, at_most: 24 }] }`) +
		"limits: { monthly_usd_per_person: 20 }\n"
	m, ds := Check([]byte(src))
	if len(ds) > 0 {
		t.Fatalf("rejected:\n%s", format("m.yaml", ds))
	}
	g := m.AgentGoals["close-checklist"]
	if m.Limits == nil || m.Limits.MonthlyUSDPerPerson != 20 || g.Job != "Gets the month closed on time" || len(g.Targets) != 2 ||
		*g.Targets[0].AtLeast != 0.6 || *g.Targets[1].AtMost != 24 || m.AgentCaps["close-checklist"] != 40 {
		t.Fatalf("read: %+v %+v", m.Limits, g)
	}
	again, ds := Check(Format(m))
	if len(ds) > 0 || again.Limits.MonthlyUSDPerPerson != 20 || len(again.AgentGoals["close-checklist"].Targets) != 2 {
		t.Fatalf("Format dropped them: %v\n%s", ds, Format(m))
	}
}
