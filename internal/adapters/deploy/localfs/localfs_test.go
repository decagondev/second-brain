package localfs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/runtime"
)

func TestAttestThenDeploy(t *testing.T) {
	d := Dir{Root: t.TempDir()}
	ctx := context.Background()
	svc := runtime.Services[0]
	img := "registry.example/api@sha256:" + strings.Repeat("a", 64)
	if cur, rel, err := d.Running(ctx, svc); cur != "" || rel != "" || err != nil {
		t.Fatalf("nothing deployed yet: %q %q %v", cur, rel, err)
	}
	if err := d.Deploy(ctx, svc, img, "rel_2027-03-02_abcdef"); !errors.Is(err, ErrNotAttested) {
		t.Fatalf("deploy before attest: %v", err)
	}
	if err := d.Attest(ctx, "registry.example/api:latest"); err == nil {
		t.Fatal("a tag was attested")
	}
	if err := d.Attest(ctx, img); err != nil {
		t.Fatal(err)
	}
	if err := d.Deploy(ctx, svc, img, "rel_2027-03-02_abcdef"); err != nil {
		t.Fatal(err)
	}
	if cur, rel, err := d.Running(ctx, svc); cur != img || rel != "rel_2027-03-02_abcdef" || err != nil {
		t.Fatalf("running %q %q %v", cur, rel, err)
	}
}
