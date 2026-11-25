package reviews

import (
	"testing"
	"time"

	"example.com/second-brain/internal/core/manifest"
)

func TestOpenAgentsProposesTheUnusedOff(t *testing.T) {
	now := time.Date(2027, 1, 4, 9, 0, 0, 0, time.UTC)
	ten := manifest.Tenant{Spaces: map[string]manifest.Space{
		"ops":  {Label: "internal", Stewards: []string{"group:leads"}, Agents: []string{"case-brief", "ask"}, Updates: true, Brief: &manifest.Brief{Agent: "ops-brief", Weekly: "monday"}},
		"eng":  {Label: "internal", Agents: []string{"case-brief"}},
		"none": {Label: "internal", Stewards: []string{"group:leads"}},
	}}
	f := AgentFacts{
		Department: func(a string) bool { return a != "ask" },
		Tools:      func(a string) []string { return []string{"search", "cite"} },
		LastUse: func(space, item string) (time.Time, bool) {
			if item == ItemUpdates {
				return now.Add(-24 * time.Hour), true
			}
			if item == "case-brief" {
				return now.Add(-120 * 24 * time.Hour), true
			}
			return time.Time{}, false
		},
	}
	n := 0
	rs := OpenAgents(ten, now, f, nil, func(p string) string { n++; return p + string(rune('0'+n)) })
	if len(rs) != 1 || rs[0].Space != "ops" || rs[0].Kind != KindAgents {
		t.Fatalf("one agent review, for ops: %+v", rs)
	}
	r := rs[0]
	unused := map[string]bool{}
	for _, it := range r.Items {
		if it.User == "ask" {
			t.Fatal("ask isn't a department agent")
		}
		if len(it.Tools) != 2 || it.Tools[0] != "cite" {
			t.Fatalf("tools shown, sorted: %+v", it)
		}
		unused[it.User] = it.Unused
	}
	if !unused["case-brief"] || !unused["brief:ops-brief"] || unused[ItemUpdates] {
		t.Fatalf("unused: %v", unused)
	}
	if len(r.ProposeOff) == 0 {
		t.Fatal("the unused should be proposed off")
	}
	if again := OpenAgents(ten, now, f, func(string, string) bool { return true }, func(p string) string { return p }); len(again) != 0 {
		t.Fatal("one review a quarter")
	}

	next, changed := TakeOff(ten, "ops", []string{"case-brief", "brief:ops-brief"})
	if !changed || len(next.Spaces["ops"].Agents) != 1 || next.Spaces["ops"].Brief != nil || !next.Spaces["ops"].Updates {
		t.Fatalf("take off: %+v", next.Spaces["ops"])
	}
	if len(ten.Spaces["ops"].Agents) != 2 || ten.Spaces["ops"].Brief == nil {
		t.Fatal("TakeOff must not change the tenant it was given")
	}
}
