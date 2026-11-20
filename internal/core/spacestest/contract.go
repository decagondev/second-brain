// Package spacestest is the contract every ports.SpaceStore passes (SB-1002).
package spacestest

import (
	"context"
	"reflect"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// Contract checks put, get, list, replace and delete, and that one tenant never sees
// another's spaces.
func Contract(t *testing.T, st ports.SpaceStore) {
	t.Helper()
	ctx := context.Background()
	fin := ports.Space{ID: "spc_finance", Tenant: "acme-logistics", Name: "finance", Template: "finance", Label: "restricted",
		Members: []string{"group:finance"}, Stewards: []string{"group:finance-leads"}, Agents: []string{"ask"},
		Scope: map[string][]string{"docs": {"/Finance/**"}}, RetentionDays: 365}
	eng := ports.Space{ID: "spc_eng", Tenant: "acme-logistics", Name: "eng", Template: "custom", Label: "internal",
		Members: []string{"group:eng"}, Stewards: []string{}, Agents: []string{"ask"}, Scope: map[string][]string{"docs": {"/Eng/**"}}, RetentionDays: 365}
	other := fin
	other.Tenant = "beta-freight"
	for _, s := range []ports.Space{fin, eng, other} {
		if err := st.PutSpace(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	got, found, err := st.Space(ctx, "acme-logistics", "spc_finance")
	if err != nil || !found || !reflect.DeepEqual(got, fin) {
		t.Fatalf("Space: %+v %v %v", got, found, err)
	}
	list, err := st.Spaces(ctx, "acme-logistics")
	if err != nil || len(list) != 2 || list[0].ID != "spc_eng" || list[1].ID != "spc_finance" {
		t.Fatalf("Spaces: %+v %v", list, err)
	}
	if _, found, err := st.Space(ctx, "gamma-parts", "spc_finance"); err != nil || found {
		t.Fatalf("another tenant's space was found: %v %v", found, err)
	}
	fin.Members = []string{"group:finance", "group:execs"}
	if err := st.PutSpace(ctx, fin); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := st.Space(ctx, "acme-logistics", "spc_finance"); len(got.Members) != 2 {
		t.Fatalf("replace: %+v", got)
	}
	if err := st.DeleteSpace(ctx, "acme-logistics", "spc_finance"); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := st.Space(ctx, "acme-logistics", "spc_finance"); found {
		t.Fatal("a deleted space was found")
	}
	if _, found, _ := st.Space(ctx, "beta-freight", "spc_finance"); !found {
		t.Fatal("deleting one tenant's space deleted another's")
	}
	if err := st.DeleteSpace(ctx, "acme-logistics", "spc_finance"); err != nil {
		t.Fatalf("deleting twice: %v", err)
	}
}
