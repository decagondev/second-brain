package localfs

import (
	"context"
	"testing"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/ports"
)

// SB-708: custody changes are recorded in history; a detached cell is tombstoned and stops
// routing, and only its history stays.
func TestCustodyHistoryAndTombstone(t *testing.T) {
	ctx := context.Background()
	d := Dir{Root: t.TempDir()}
	if err := d.PutCell(ctx, ports.CellRecord{Tenant: "gamma-parts", Region: "eu", Hosts: []string{"gamma-parts.sb.localhost"},
		Custody: cell.CustodyFactory, Placement: cell.PlacementFactoryAccount, KeyCustodian: cell.KeyCustodianFactory}); err != nil {
		t.Fatal(err)
	}
	if err := d.Tombstone(ctx, "gamma-parts"); err == nil {
		t.Fatal("a factory-custody cell was tombstoned")
	}
	to := cell.TenantAccountPrefix + "gcp/gamma-prod-7731"
	if err := d.SetCustody(ctx, "gamma-parts", cell.CustodyTenant, to, cell.KeyCustodianTenant); err != nil {
		t.Fatal(err)
	}
	if err := d.SetCustody(ctx, "gamma-parts", cell.CustodyDetached, to, cell.KeyCustodianTenant); err != nil {
		t.Fatal(err)
	}
	if err := d.Tombstone(ctx, "gamma-parts"); err != nil {
		t.Fatal(err)
	}
	if _, found, err := d.CellByHost(ctx, "gamma-parts.sb.localhost"); found || err != nil {
		t.Fatalf("a tombstoned cell still routes: %v %v", found, err)
	}
	h, err := d.History(ctx, "gamma-parts")
	if err != nil || len(h) != 2 || h[0].Custody != cell.CustodyTenant || h[1].Custody != cell.CustodyDetached {
		t.Fatalf("history: %+v %v", h, err)
	}
}
