package localfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"example.com/second-brain/internal/core/handoff"
)

func TestPutGet(t *testing.T) {
	d := Dir{Root: filepath.Join(t.TempDir(), "handoffs")}
	ctx := context.Background()
	if _, found, err := d.Get(ctx, "gamma-parts"); found || err != nil {
		t.Fatalf("%v %v", found, err)
	}
	s := handoff.State{Phase: handoff.Planned, Hash: "ph_x", Serving: "factoryCell",
		Approvals: map[string]handoff.Approval{"factory": {Who: "fo_bob", Hash: "ph_x"}}}
	if err := d.Put(ctx, "gamma-parts", s); err != nil {
		t.Fatal(err)
	}
	got, found, err := d.Get(ctx, "gamma-parts")
	if err != nil || !found || got.Phase != handoff.Planned || got.Approvals["factory"].Who != "fo_bob" {
		t.Fatalf("%+v %v %v", got, found, err)
	}
	_ = os.WriteFile(filepath.Join(d.Root, "beta-freight.json"), []byte("{"), 0o600)
	if _, _, err := d.Get(ctx, "beta-freight"); err == nil {
		t.Fatal("an unreadable state was read as none")
	}
	if _, _, err := d.Get(ctx, "../x"); err == nil {
		t.Fatal("unsafe name")
	}
}
