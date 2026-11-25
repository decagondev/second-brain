package inventory

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/proposals"
)

// SB-5007: the inventory lists what the manifest and the registry run, counts personal
// routines without naming them, and its signature verifies, and fails once changed.
func TestInventory(t *testing.T) {
	now := time.Date(2027, 3, 1, 9, 0, 0, 0, time.UTC)
	usd, least := 40.0, 0.6
	m := manifest.Tenant{
		AgentCaps:  map[string]float64{"close-checklist": usd},
		AgentGoals: map[string]manifest.AgentGoal{"close-checklist": {Job: "Closes the month", Targets: []manifest.Target{{Metric: "published_as_is", AtLeast: &least}}}},
		Spaces: map[string]manifest.Space{"finance": {
			Stewards: []string{"group:finance-leads"},
			Skills:   map[string]manifest.Skill{"close-note": {}},
			Rhythms:  map[string]manifest.Rhythm{"month-end": {}},
			Brief:    &manifest.Brief{Agent: "ask", Weekly: "monday"},
			On:       []string{"source-changed"},
		}},
	}
	in := Inputs{Tenant: "acme-logistics", Release: "rel_1", Now: now, Manifest: m,
		Agents: []Agent{{Name: "ask", Spaces: []string{"finance"}}, {Name: "close-checklist", Spaces: []string{"finance"}}},
		Traces: []ask.Trace{{Space: "spc_finance", At: now.Add(-time.Hour), Response: ask.Response{Skill: "close-note"}}},
		Routines: []ask.SavedQuestion{
			{ID: "sq_1", Space: "spc_finance", Question: "What closed?", Routine: &ask.Routine{Every: "weekly", Day: "monday"}, Shared: "finance"},
			{ID: "sq_2", Space: "spc_finance", Question: "my private one", Routine: &ask.Routine{Every: "daily"}, From: "sq_1"},
			{ID: "sq_3", Space: "spc_finance", Question: "my secret routine", Routine: &ask.Routine{Every: "daily"}},
			{ID: "sq_4", Question: "company-wide", Routine: &ask.Routine{Every: "daily"}},
		},
		Starters:   []ask.Suggested{{Space: "spc_finance", Questions: []string{"When is the close?"}, At: now.AddDate(0, 0, -2)}},
		Proposals:  []proposals.Proposal{{ID: "prop_1", Space: "finance", State: proposals.Approved, At: now.AddDate(0, 0, -3), Diff: []string{"+ spaces.finance.skills.close-note"}}},
		RhythmRuns: []ask.RhythmRun{{Space: "finance", Rhythm: "month-end", At: now.AddDate(0, 0, -1)}},
	}
	inv := Build(in)
	kinds := map[string]int{}
	var cc, skill, routine *Item
	for i, it := range inv.Items {
		kinds[it.Kind]++
		switch {
		case it.Name == "close-checklist":
			cc = &inv.Items[i]
		case it.Kind == ItemSkill:
			skill = &inv.Items[i]
		case it.Kind == ItemRoutine:
			routine = &inv.Items[i]
		}
	}
	if kinds[ItemAgent] != 2 || kinds[ItemSkill] != 1 || kinds[ItemRhythm] != 1 || kinds[ItemTrigger] != 2 || kinds[ItemStarter] != 1 || kinds[ItemRoutine] != 1 {
		t.Fatalf("kinds: %v", kinds)
	}
	if cc == nil || *cc.MonthlyUSD != 40 || cc.Job != "Closes the month" || len(cc.Targets) != 1 || cc.Owner == "" {
		t.Fatalf("close-checklist: %+v", cc)
	}
	if skill.LastRun == "" || skill.LastChange != now.AddDate(0, 0, -3).Format(time.RFC3339) || skill.Owner != "group:finance-leads" {
		t.Fatalf("skill: %+v", skill)
	}
	if *routine.Members != 1 || !strings.HasPrefix(routine.Name, "sq_1 weekly monday") {
		t.Fatalf("routine: %+v", routine)
	}
	if len(inv.PersonalRoutines) != 2 || inv.PersonalRoutines[0] != (PersonalRoutines{Space: "company", Count: 1}) || inv.PersonalRoutines[1].Count != 1 {
		t.Fatalf("personal: %+v", inv.PersonalRoutines)
	}
	csv := string(inv.CSV())
	for _, secret := range []string{"my secret routine", "my private one", "company-wide", "sq_3"} {
		if strings.Contains(csv, secret) {
			t.Fatalf("the CSV names a personal routine: %s", csv)
		}
	}
	if !strings.Contains(csv, "agent,close-checklist,") || !strings.Contains(csv, "published_as_is>=0.6") || !strings.Contains(csv, "personal-routines,,,finance,,,,,,1") {
		t.Fatalf("csv:\n%s", csv)
	}
	sg, err := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	signed, err := Sign(inv, sg, now)
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := json.Marshal(signed)
	if _, err := Verify(doc, sg.Key()); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(doc, "ed25519:other"); err == nil {
		t.Fatal("another pinned key verified")
	}
	if _, err := Verify(bytes.Replace(doc, []byte("close-checklist"), []byte("close-checklisx"), 1), ""); err == nil {
		t.Fatal("a changed inventory verified")
	}
}
