package route

import (
	"reflect"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

func TestMatch(t *testing.T) {
	for _, tc := range []struct {
		p, s string
		want bool
	}{
		{"/Finance/**", "/Finance/Q3/Forecast", true},
		{"/Finance/**", "/Finance", false},
		{"/Finance/**", "/FinanceOld/x", false},
		{"/Finance/*", "/Finance/Q3/Forecast", false},
		{"/Finance/*", "/Finance/Plan", true},
		{"#eng-*", "#eng-platform", true},
		{"#eng-*", "#engineering", false},
		{"ENG", "ENG", true},
		{"ENG", "ENGX", false},
		{"**", "anything/at/all", true},
		{"/Specs/**/draft", "/Specs/a/b/draft", true},
	} {
		if got := Match(tc.p, tc.s); got != tc.want {
			t.Errorf("Match(%q, %q) = %v", tc.p, tc.s, got)
		}
	}
}

func TestRoute(t *testing.T) {
	sp := []ports.Space{
		{ID: "spc_eng", Label: "internal", Scope: map[string][]string{"docs": {"/Engineering/**", "/Specs/**"}, "chat": {"#eng-*"}}},
		{ID: "spc_product", Label: "internal", Scope: map[string][]string{"docs": {"/Specs/**"}}},
		{ID: "spc_finance", Label: "restricted", Scope: map[string][]string{"docs": {"/Finance/**", "/Specs/budget/**"}}},
	}
	for _, tc := range []struct {
		kind, loc, src string
		holders        []string
		label          string
	}{
		{"docs", "/Engineering/arch", "", []string{"spc_eng"}, "internal"},
		{"docs", "/Specs/plan", "", []string{"spc_eng", "spc_product"}, "internal"},
		{"docs", "/Specs/budget/2027", "", []string{"spc_eng", "spc_finance", "spc_product"}, "restricted"}, // the stricter default
		{"docs", "/Finance/q3", "public", []string{"spc_finance"}, "public"},                                // the source wins
		{"docs", "/Finance/q3", "top-secret", []string{"spc_finance"}, "restricted"},                        // unknown never widens
		{"docs", "/Other/x", "", []string{}, "internal"},
		{"chat", "#eng-platform", "confidential", []string{"spc_eng"}, "confidential"},
		{"tickets", "ENG", "", []string{}, "internal"},
	} {
		h, l := Route(sp, tc.kind, tc.loc, tc.src)
		if !reflect.DeepEqual(h, tc.holders) || l != tc.label {
			t.Errorf("%s %s %q: %v %q, want %v %q", tc.kind, tc.loc, tc.src, h, l, tc.holders, tc.label)
		}
	}
	if Fingerprint(nil) != "" || Fingerprint(sp) == "" {
		t.Fatal("fingerprint of none must be empty, of some not")
	}
	moved := append([]ports.Space(nil), sp...)
	moved[2] = ports.Space{ID: "spc_finance", Label: "restricted", Scope: map[string][]string{"docs": {"/Finance/**"}}}
	if Fingerprint(sp) == Fingerprint(moved) {
		t.Fatal("a scope change must change the fingerprint")
	}
	rev := []ports.Space{sp[2], sp[0], sp[1]}
	if Fingerprint(sp) != Fingerprint(rev) {
		t.Fatal("order must not change the fingerprint")
	}
}
