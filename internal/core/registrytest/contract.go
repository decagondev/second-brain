// Package registrytest is the contract every tenant-registry adapter must pass
// (local directory now, Postgres for the control plane). Call Contract from the
// adapter's tests with a fresh, empty registry.
package registrytest

import (
	"context"
	"reflect"
	"testing"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/ports"
)

// Registry is everything a control-plane registry implements.
type Registry interface {
	ports.Registry
	ports.RegistryWriter
	ports.Cells
	ports.CellsWriter
	ports.Journal
}

// Contract runs the shared tests against r, which must start empty.
func Contract(t *testing.T, r Registry) {
	ctx := context.Background()
	if _, found, err := r.Applied(ctx, "acme-logistics"); err != nil || found {
		t.Fatalf("empty registry: found=%v err=%v", found, err)
	}
	rs := []ports.Resource{
		{Kind: "cell", Name: "acme-logistics", Attrs: map[string]string{"region": "eu"}},
		{Kind: "agent", Name: "ask"},
	}
	if err := r.Put(ctx, "acme-logistics", rs); err != nil {
		t.Fatal(err)
	}
	got, found, err := r.Applied(ctx, "acme-logistics")
	if err != nil || !found || !reflect.DeepEqual(got, rs) {
		t.Fatalf("applied: %v %v %v", got, found, err)
	}
	if err := r.Put(ctx, "acme-logistics", rs[:1]); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := r.Applied(ctx, "acme-logistics"); !reflect.DeepEqual(got, rs[:1]) {
		t.Fatalf("Put must replace the record, got %v", got)
	}

	acme := ports.CellRecord{Tenant: "acme-logistics", Region: "eu", Endpoint: "local://a/cells/acme-logistics",
		Hosts: []string{"acme-logistics.sb.example", "brain.acme.example"}, Custody: cell.CustodyFactory, Placement: cell.PlacementFactoryAccount, KeyCustodian: cell.KeyCustodianFactory}
	if err := r.PutCell(ctx, acme); err != nil {
		t.Fatal(err)
	}
	for _, h := range acme.Hosts {
		if c, found, err := r.CellByHost(ctx, h); err != nil || !found || !reflect.DeepEqual(c, acme) {
			t.Fatalf("by host %s: %+v %v %v", h, c, found, err)
		}
	}
	if c, found, err := r.CellByTenant(ctx, "acme-logistics"); err != nil || !found || !reflect.DeepEqual(c, acme) {
		t.Fatalf("by tenant: %+v %v %v", c, found, err)
	}
	if _, found, err := r.CellByHost(ctx, "evil.example"); err != nil || found {
		t.Fatalf("unknown host: %v %v", found, err)
	}
	beta := ports.CellRecord{Tenant: "beta-freight", Region: "us", Endpoint: "local://a/cells/beta-freight",
		Hosts: []string{"brain.acme.example"}, Custody: cell.CustodyFactory, Placement: cell.PlacementFactoryAccount, KeyCustodian: cell.KeyCustodianFactory}
	if err := r.PutCell(ctx, beta); err == nil {
		t.Fatal("a host must belong to only one cell")
	}
	if _, found, _ := r.CellByTenant(ctx, "beta-freight"); found {
		t.Fatal("a refused PutCell must leave nothing behind")
	}
	if err := r.Record(ctx, ports.JournalEntry{Tenant: "acme-logistics", ApplyID: "apply-1", Step: 1, Op: "add", Kind: "cell", Name: "acme-logistics", Phase: "start"}); err != nil {
		t.Fatal(err)
	}
}
