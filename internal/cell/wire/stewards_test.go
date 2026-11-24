package wire

import (
	"context"
	"reflect"
	"testing"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	"example.com/second-brain/internal/core/health"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// SB-3102: a space whose steward groups have no active member warns in health; a group the
// directory can't answer for is unknown, never "has a steward"; a space with no stewards
// set isn't listed.
func TestUnstewardedHealth(t *testing.T) {
	ctx := context.Background()
	d := dirmem.New()
	for _, u := range []ports.User{
		{ID: "u1", Tenant: "acme-logistics", UserName: "kim", Active: true},
		{ID: "u2", Tenant: "acme-logistics", UserName: "lee", Active: false},
	} {
		if err := d.PutUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	for _, g := range []ports.Group{
		{ID: "g1", Tenant: "acme-logistics", Name: "ops-leads", Members: []string{"u1"}},
		{ID: "g2", Tenant: "acme-logistics", Name: "hr-leads", Members: []string{"u2"}},
	} {
		if err := d.PutGroup(ctx, g); err != nil {
			t.Fatal(err)
		}
	}
	m := manifest.Tenant{Spaces: map[string]manifest.Space{
		"ops":     {Stewards: []string{"group:ops-leads"}},
		"hr":      {Stewards: []string{"group:hr-leads"}},
		"finance": {Stewards: []string{"group:no-such-group"}},
		"docs":    {},
	}}
	got := unstewardedOf(ctx, d, "acme-logistics", m)
	want := &health.Unstewarded{Spaces: []string{"hr"}, Unknown: []string{"finance"}, Warning: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
	if none, known := stewardless(ctx, d, "acme-logistics", m, "spc_ops"); none || !known {
		t.Fatalf("ops: %v %v", none, known)
	}
	if none, known := stewardless(ctx, nil, "acme-logistics", m, "spc_hr"); none || known {
		t.Fatalf("no directory: %v %v", none, known)
	}
	if unstewardedOf(ctx, d, "acme-logistics", manifest.Tenant{}) != nil {
		t.Fatal("a cell with no spaces warned")
	}
}
