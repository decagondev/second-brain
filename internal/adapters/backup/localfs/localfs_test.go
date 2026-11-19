package localfs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWriteOnce(t *testing.T) {
	d := Dir{Root: t.TempDir(), Label: "a", Media: "disk"}
	ctx := context.Background()
	now := time.Now()
	if err := d.Put(ctx, "acme-logistics/2027-03-02/audit.ndjson", []byte("x"), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := d.Put(ctx, "acme-logistics/2027-03-02/audit.ndjson", []byte("y"), now.Add(time.Hour)); !errors.Is(err, ErrExists) {
		t.Fatalf("overwrite: %v", err)
	}
	if err := d.Delete(ctx, "acme-logistics/2027-03-02/audit.ndjson", now); err == nil {
		t.Fatal("deleted before retention")
	}
	if err := d.Delete(ctx, "acme-logistics/2027-03-02/audit.ndjson", now.Add(2*time.Hour)); err != nil {
		t.Fatalf("delete after retention: %v", err)
	}
	if err := d.Put(ctx, "../escape", []byte("x"), now); err == nil {
		t.Fatal("escaped the root")
	}
}
