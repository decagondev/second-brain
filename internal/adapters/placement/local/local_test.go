package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/cell/celltest"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

func TestContract(t *testing.T) {
	root := t.TempDir()
	celltest.Contract(t, func(name string) celltest.Adapter {
		a := Account{Root: root, Name: name}
		return celltest.Adapter{Placement: a, Keys: a}
	})
}

// An unreadable resource record is unknown, so Ensure errors instead of creating a second one.
func TestUnreadableIsNotAbsent(t *testing.T) {
	a := Account{Root: t.TempDir(), Name: "acct-a"}
	dir := filepath.Join(a.Root, "acct-a", "acme-logistics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "queue.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := ports.CellResource{Kind: "queue", Name: "acme-logistics-jobs", Tenant: "acme-logistics", Region: "eu"}
	if _, err := a.Ensure(context.Background(), r); err == nil {
		t.Fatal("want an error for an unreadable record")
	}
}

// A resource record that belongs to a different identity (e.g. another region) is refused.
func TestIdentityMismatchRefused(t *testing.T) {
	ctx := context.Background()
	a := Account{Root: t.TempDir(), Name: "acct-a"}
	if _, err := cell.Stamp(ctx, a, a, manifest.Tenant{Tenant: "acme-logistics", Region: "eu"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cell.Stamp(ctx, a, a, manifest.Tenant{Tenant: "acme-logistics", Region: "us"}); err == nil {
		t.Fatal("re-stamping a cell into another region must fail")
	}
}

func TestUnsafeNamesRefused(t *testing.T) {
	a := Account{Root: t.TempDir(), Name: "acct-a"}
	r := ports.CellResource{Kind: "queue", Name: "x", Tenant: "../escape", Region: "eu"}
	if _, err := a.Ensure(context.Background(), r); err == nil {
		t.Fatal("want an error for an unsafe tenant name")
	}
}
