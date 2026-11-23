package route

import (
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// SB-2903: a typo matches none; /** is too wide; a fitting pattern has no problem.
func TestScopeReport(t *testing.T) {
	spaces := []ports.Space{
		{Name: "ops", Scope: map[string][]string{"docs": {"/Ops/**", "/Opps/**"}}},
		{Name: "all", Scope: map[string][]string{"docs": {"/**"}, "chat": {"#ops-*"}}},
	}
	locs := map[string][]string{"docs": {"/Ops/a", "/Ops/b", "/HR/c", "/Finance/d", "/Finance/e"}, "chat": {"#ops-1", "#ops-2", "#hr"}}
	got := Report(spaces, locs)
	want := []PatternCount{
		{Space: "all", Kind: "chat", Pattern: "#ops-*", Matches: 2, Of: 3, Problem: ProblemTooWide},
		{Space: "all", Kind: "docs", Pattern: "/**", Matches: 5, Of: 5, Problem: ProblemTooWide},
		{Space: "ops", Kind: "docs", Pattern: "/Ops/**", Matches: 2, Of: 5},
		{Space: "ops", Kind: "docs", Pattern: "/Opps/**", Of: 5, Problem: ProblemNone},
	}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}
