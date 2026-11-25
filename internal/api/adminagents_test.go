package api

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
)

// SB-4004: every enabled agent and handover, with its department, the spaces that list it,
// its runs and unhelpful ratings, and its kill and eval state; a space sees its own only.
func TestAgentRegistry(t *testing.T) {
	m := manifest.Tenant{Agents: []string{"ask", "incident-summary", "close-checklist"}, Spaces: map[string]manifest.Space{
		"eng":     {Template: "engineering", Agents: []string{"incident-summary"}},
		"finance": {Template: "finance", Agents: []string{"close-checklist"}},
		"hr":      {Template: "people"},
	}}
	traces := []ask.Trace{
		{ID: "t1", Space: "spc_eng", Response: ask.Response{Agent: "incident-summary"}},
		{ID: "t2", Space: "spc_eng", Response: ask.Response{Agent: "incident-summary"}},
		{ID: "t3", Space: "spc_finance"},
		{ID: "t4", Space: "spc_eng"},
		{ID: "t5", Space: "spc_eng", Response: ask.Response{Agent: ask.AgentSearch}},
		{ID: "t6", Space: "company", Response: ask.Response{Agent: ask.AgentHandover}},
	}
	fb := []ask.Feedback{{Trace: "t1", Rating: "down"}, {Trace: "t2", Rating: "up"}, {Trace: "t5", Rating: "down"}, {Trace: "t4", Rating: "down"}}
	killed := func(n string) bool { return n == "close-checklist" }
	got := AgentRegistry(m, "", traces, fb, killed, false)
	names := []string{}
	for _, a := range got {
		names = append(names, a.Name)
	}
	if len(got) != 4 || names[0] != "ask" || names[1] != "close-checklist" || names[2] != "handover" || names[3] != "incident-summary" {
		t.Fatalf("agents %v", names)
	}
	ask0, cc, ho, is := got[0], got[1], got[2], got[3]
	if ask0.Runs != 2 || ask0.Unhelpful != 1 || len(ask0.Spaces) != 3 || ask0.Department != "" || ask0.Suite != "gate" {
		t.Fatalf("ask %+v", ask0)
	}
	if is.Runs != 2 || is.Rated != 2 || is.Unhelpful != 1 || is.Department != "engineering" || len(is.Spaces) != 1 || is.Spaces[0] != "eng" || is.Suite != "agent-incident-summary" || is.Eval != "passed" {
		t.Fatalf("incident-summary %+v", is)
	}
	if !cc.Killed || cc.Runs != 0 || ho.Runs != 1 || ho.Suite != "agent-handover" {
		t.Fatalf("close-checklist %+v handover %+v", cc, ho)
	}
	// The steward of eng sees eng's runs only, and no other department's agent.
	got = AgentRegistry(m, "eng", traces, fb, nil, true)
	if len(got) != 3 || got[0].Name != "ask" || got[0].Runs != 1 || got[2].Name != "incident-summary" || got[2].Eval != "held" {
		t.Fatalf("eng %+v", got)
	}
	for _, a := range got {
		if a.Name == "close-checklist" || len(a.Spaces) != 1 || a.Spaces[0] != "eng" {
			t.Fatalf("eng sees %+v", a)
		}
	}
}

// runsOnly is a report source with traces only.
type runsOnly []ask.Trace

func (r runsOnly) Traces(context.Context, string, time.Time, time.Time) ([]ask.Trace, error) {
	return r, nil
}
func (runsOnly) Drafts(context.Context, string, time.Time, time.Time) (int, int, error) {
	return 0, 0, nil
}
func (runsOnly) Actions(context.Context, string, time.Time, time.Time) ([]approvals.Action, error) {
	return nil, nil
}
func (runsOnly) Seats(context.Context, string) (int, error) { return 0, nil }
func (runsOnly) Tokens(context.Context, string, time.Time, time.Time) (int, int, error) {
	return 0, 0, nil
}

// SB-4402: each agent shows its owner from the bundle, flagged when no one active answers
// for it; a department agent no one ran in a space for 60 days is proposed off, and only
// when traces reach back that far.
func TestAgentOwnersAndUnused(t *testing.T) {
	rows := withOwners(context.Background(), []AdminAgent{{Name: "ask"}, {Name: "incident-summary"}},
		func(_ context.Context, owner string) bool { return owner == "group:brain-agents" })
	if rows[0].Owner != "group:brain-agents" || rows[0].OwnerInactive || rows[1].Owner == "" || !rows[1].OwnerInactive {
		t.Fatalf("owners: %+v", rows)
	}
	if rows := withOwners(context.Background(), []AdminAgent{{Name: "ask"}}, nil); rows[0].OwnerInactive {
		t.Fatal("no directory flagged an owner")
	}
	m := manifest.Tenant{RetentionDays: 90, Agents: []string{"ask", "incident-summary", "spec-digest"}, Spaces: map[string]manifest.Space{
		"eng": {Template: "engineering", Agents: []string{"incident-summary", "spec-digest"}},
	}}
	src := runsOnly{{ID: "t1", Space: "spc_eng", Response: ask.Response{Agent: "spec-digest"}}}
	rows = []AdminAgent{{Name: "incident-summary"}, {Name: "spec-digest"}}
	keep, err := unused(context.Background(), src, m, "acme-logistics", "eng", rows, time.Now())
	if err != nil || len(keep) != 1 || keep[0] != "spec-digest" || !rows[0].Unused || rows[1].Unused {
		t.Fatalf("unused: %v %v %+v", keep, err, rows)
	}
	m.RetentionDays = 30
	if keep, err := unused(context.Background(), src, m, "acme-logistics", "eng", rows, time.Now()); err != nil || keep != nil {
		t.Fatalf("traces kept 30 days proposed something: %v %v", keep, err)
	}
}
