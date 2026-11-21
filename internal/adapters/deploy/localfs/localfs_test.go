package localfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

// SB-1604: what runs is what the host step recorded starting, not what was asked for.
func TestRunningIsWhatTheHostStarted(t *testing.T) {
	d := Dir{Root: t.TempDir()}
	ctx := context.Background()
	svc := runtime.SelfHosted[0]
	asked := "local/api@sha256:" + strings.Repeat("b", 64)
	_ = d.Attest(ctx, asked)
	if err := d.Deploy(ctx, svc, asked, "rel_2027-03-02_bbbbbb"); err != nil {
		t.Fatal(err)
	}
	if img, _, _ := d.Running(ctx, svc); img != asked {
		t.Fatalf("before any host step, running is what was asked for: %q", img)
	}
	started := "local/api@sha256:" + strings.Repeat("c", 64)
	if err := os.MkdirAll(filepath.Join(d.Root, "running"), 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(d.Root, "running", "api.json"), []byte(`{"image":"`+started+`","release":"rel_2027-03-01_cccccc"}`), 0o600)
	if img, rel, _ := d.Running(ctx, svc); img != started || rel != "rel_2027-03-01_cccccc" {
		t.Fatalf("running %q %q, want what the host started", img, rel)
	}
}
