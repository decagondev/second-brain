package manifest

import (
	"fmt"
	"strings"
	"testing"
)

func rhythmSrc(steps string) string {
	return treeHead + `    onboarding: false
    rhythms:
      welcome: { subject: person, plan: steward, steps: [ ` + steps + ` ] }
  ap:
    parent: finance
    members: [group:ap]
    scope: { docs: ["/Finance/AP/**"] }
`
}

const twoSteps = `{ day: 0, kind: draft, ask: "Where to start", approver: subject, agent: start-here }, { day: 30, kind: request, ask: "What would you add to start here?", approver: steward }`

// SB-4801: a rhythm is kept, written back the same and flows down the tree with
// onboarding: false; the built-in rhythms are always there unless turned off.
func TestRhythmsKeptAndInherited(t *testing.T) {
	m, ds := Check([]byte(rhythmSrc(twoSteps)))
	if len(ds) > 0 {
		t.Fatalf("%v", ds)
	}
	r := m.Spaces["finance"].Rhythms["welcome"]
	if r.Subject != "person" || r.Plan != "steward" || len(r.Steps) != 2 || r.Steps[0].Agent != "start-here" || r.Steps[1].Day != 30 {
		t.Fatalf("rhythm = %+v", r)
	}
	team := m.Rhythms("ap")
	if _, ok := team["welcome"]; !ok {
		t.Fatal("the team didn't inherit the rhythm")
	}
	if _, ok := team["onboarding"]; ok {
		t.Fatal("onboarding: false must flow down to the team")
	}
	if _, ok := team["kickoff"]; !ok {
		t.Fatal("the built-in kickoff rhythm is missing")
	}
	again, ds := Check(Format(m))
	if len(ds) > 0 || len(again.Spaces["finance"].Rhythms["welcome"].Steps) != 2 || again.Spaces["finance"].Onboarding == nil {
		t.Fatalf("format round trip: %v %+v", ds, again.Spaces["finance"])
	}
	for name, b := range BuiltinRhythms {
		if p := RhythmProblem(b); p != "" {
			t.Errorf("built-in %s: %s", name, p)
		}
	}
}

// SB-4801: the lint refuses a thirteenth step, a step past day 120, a step without an
// approver, steps out of order and an ask that acts or widens.
func TestRhythmsRefused(t *testing.T) {
	var thirteen []string
	for i := 0; i < 13; i++ {
		thirteen = append(thirteen, fmt.Sprintf(`{ day: %d, kind: draft, ask: "Step", approver: subject }`, i))
	}
	for name, c := range map[string]struct{ src, want string }{
		"thirteen": {rhythmSrc(strings.Join(thirteen, ", ")), "at most 12 steps"},
		"late":     {rhythmSrc(`{ day: 121, kind: draft, ask: "Late", approver: subject }`), "121"},
		"approver": {rhythmSrc(`{ day: 1, kind: draft, ask: "Who?", approver: nobody }`), "named approver"},
		"none":     {rhythmSrc(`{ day: 1, kind: draft, ask: "Who?" }`), "named approver"},
		"order":    {rhythmSrc(`{ day: 9, kind: draft, ask: "B", approver: subject }, { day: 2, kind: draft, ask: "A", approver: subject }`), "day order"},
		"kind":     {rhythmSrc(`{ day: 1, kind: send, ask: "Hi", approver: subject }`), "kind must be"},
		"action":   {rhythmSrc(`{ day: 1, kind: draft, ask: "Send messages to every new joiner", approver: subject }`), "asks for an action"},
		"widen":    {rhythmSrc(`{ day: 1, kind: draft, ask: "Search all spaces for it", approver: subject }`), "widens"},
		"agent":    {rhythmSrc(`{ day: 1, kind: routine, ask: "Weekly", approver: subject, agent: catch-up }`), "only a draft step"},
		"field":    {rhythmSrc(`{ day: 1, kind: draft, ask: "Hi", approver: subject, then: publish }`), "unknown field in a rhythm step"},
	} {
		_, ds := Check([]byte(c.src))
		if !strings.Contains(fmt.Sprint(ds), c.want) {
			t.Errorf("%s: want %q, got %v", name, c.want, ds)
		}
	}
}
