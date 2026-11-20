// Package spaces tests the space registry's sync and resolve over the memory adapter
// (SB-1002).
package spaces

import (
	"context"
	"errors"
	"os"
	"testing"

	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/spaces"
)

func TestSyncMatchesTheManifest(t *testing.T) {
	ctx := context.Background()
	src, err := os.ReadFile("../../examples/tenants/spaces.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m, ds := manifest.Check(src)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	st := spacemem.New()
	if err := spaces.Sync(ctx, st, nil, m); err != nil {
		t.Fatal(err)
	}
	got, _ := st.Spaces(ctx, "acme-logistics")
	if len(got) != 3 || got[0].ID != "spc_engineering" || got[1].ID != "spc_finance" || got[1].Label != "restricted" || got[1].RetentionDays != 365 {
		t.Fatalf("registry: %+v", got)
	}
	delete(m.Spaces, "support")
	if err := spaces.Sync(ctx, st, nil, m); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := st.Space(ctx, "acme-logistics", "spc_support"); found {
		t.Fatal("a space the manifest dropped is still in the registry")
	}
	st.Fail = errors.New("down")
	if err := spaces.Sync(ctx, st, nil, m); err == nil {
		t.Fatal("a failing registry must fail the sync")
	}
}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	st := spacemem.New()
	m, _ := manifest.Check([]byte("tenant: acme-logistics\nchannel: stable\nidp: oidc://login.acme.example\nregion: eu\nconnectors: { docs: sharepoint }\nagents: [ask]\n" +
		"spaces:\n  finance:\n    members: [group:finance]\n    stewards: [group:finance-leads]\n    label: restricted\n    scope: { docs: [\"/Finance/**\"] }\n"))
	if err := spaces.Sync(ctx, st, nil, m); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		groups    []string
		requested string
		want      string
		err       error
	}{
		{nil, "", "company", nil},
		{nil, "company", "company", nil},
		{[]string{"finance"}, "finance", "spc_finance", nil},
		{[]string{"finance-leads"}, "finance", "", spaces.ErrSpace}, // stewards aren't members
		{[]string{"finance"}, "legal", "", spaces.ErrSpace},
	} {
		got, err := spaces.Resolve(ctx, st, "acme-logistics", tc.groups, tc.requested)
		if got != tc.want || !errors.Is(err, tc.err) && (err != nil || tc.err != nil) {
			t.Errorf("%v in %q: %q %v", tc.groups, tc.requested, got, err)
		}
	}
	if _, err := spaces.Resolve(ctx, st, "beta-freight", []string{"finance"}, "finance"); !errors.Is(err, spaces.ErrSpace) {
		t.Fatalf("another tenant resolved acme's space: %v", err)
	}
	if _, err := spaces.Resolve(ctx, nil, "acme-logistics", []string{"finance"}, "finance"); !errors.Is(err, spaces.ErrSpace) {
		t.Fatalf("no registry: %v", err)
	}
	st.Fail = errors.New("down")
	if _, err := spaces.Resolve(ctx, st, "acme-logistics", []string{"finance"}, "finance"); !errors.Is(err, spaces.ErrUnavailable) {
		t.Fatalf("broken registry: %v", err)
	}
}
