package localfs

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/registrytest"
)

var ctx = context.Background()

func TestRoundTrip(t *testing.T) {
	d := Dir{Root: t.TempDir()}
	rs := []ports.Resource{{Kind: "cell", Name: "acme-logistics", Attrs: map[string]string{"region": "eu"}}, {Kind: "agent", Name: "ask"}}
	if err := d.Put(ctx, "acme-logistics", rs); err != nil {
		t.Fatal(err)
	}
	got, found, err := d.Applied(ctx, "acme-logistics")
	if err != nil || !found || !reflect.DeepEqual(got, rs) {
		t.Fatalf("got %v %v %v", got, found, err)
	}
}

func TestNoCellOnlyWhenTheRegistrySaysSo(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tenants"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, found, err := (Dir{Root: root}).Applied(ctx, "beta-freight"); err != nil || found {
		t.Fatalf("empty registry: found=%v err=%v; want no cell, no error", found, err)
	}
}

// An unknown answer is never "no cell" (sb-tenant(1)): that would plan a second cell.
func TestUnknownIsAnError(t *testing.T) {
	root := t.TempDir()
	tenants := filepath.Join(root, "tenants")
	if err := os.MkdirAll(tenants, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(tenants, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("bad-json.json", "{")
	write("extra-field.json", `{"tenant":"extra-field","resources":[],"surprise":1}`)
	write("wrong-tenant.json", `{"tenant":"gamma-parts","resources":[]}`)
	cases := map[string]Dir{
		"missing root": {Root: filepath.Join(root, "nope")},
		"bad-json":     {Root: root},
		"extra-field":  {Root: root},
		"wrong-tenant": {Root: root},
		"../escape":    {Root: root},
	}
	for name, d := range cases {
		tenant := name
		if name == "missing root" {
			tenant = "acme-logistics"
		}
		if _, found, err := d.Applied(ctx, tenant); err == nil || found {
			t.Errorf("%s: found=%v err=%v; want an error", name, found, err)
		}
	}
}

func TestCells(t *testing.T) {
	d := Dir{Root: t.TempDir()}
	a := ports.CellRecord{Tenant: "acme-logistics", Region: "eu", Hosts: []string{"acme-logistics.sb.example"}}
	if err := d.PutCell(ctx, a); err != nil {
		t.Fatal(err)
	}
	if c, found, err := d.CellByHost(ctx, "acme-logistics.sb.example"); err != nil || !found || c.Tenant != a.Tenant {
		t.Fatalf("by host: %v %v %v", c, found, err)
	}
	if _, found, err := d.CellByHost(ctx, "nope.sb.example"); err != nil || found {
		t.Fatalf("unknown host: %v %v", found, err)
	}
	b := ports.CellRecord{Tenant: "beta-freight", Hosts: []string{"acme-logistics.sb.example"}}
	if err := d.PutCell(ctx, b); err == nil {
		t.Fatal("a host must belong to only one cell")
	}
	if err := os.WriteFile(filepath.Join(d.Root, "cells", "gamma-parts.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.CellByHost(ctx, "nope.sb.example"); err == nil {
		t.Fatal("an unreadable cell record must make lookups fail, not miss")
	}
}

func TestRegistryContract(t *testing.T) {
	root := t.TempDir() // an empty local registry is a directory with tenants/ in it
	if err := os.MkdirAll(filepath.Join(root, "tenants"), 0o755); err != nil {
		t.Fatal(err)
	}
	registrytest.Contract(t, Dir{Root: root})
}
