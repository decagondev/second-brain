package localfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFinishAndRead(t *testing.T) {
	d := Dir{Root: filepath.Join(t.TempDir(), "runs")}
	ctx := context.Background()
	if _, found, err := d.LastFinish(ctx, "anchor"); found || err != nil {
		t.Fatalf("empty: %v %v", found, err)
	}
	at := time.Date(2027, 3, 10, 12, 5, 0, 0, time.UTC)
	if err := d.Finished(ctx, "anchor", at); err != nil {
		t.Fatal(err)
	}
	got, found, err := d.LastFinish(ctx, "anchor")
	if err != nil || !found || !got.Equal(at) {
		t.Fatalf("%v %v %v", got, found, err)
	}
	if err := os.WriteFile(filepath.Join(d.Root, "backup"), []byte("yesterday"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.LastFinish(ctx, "backup"); err == nil {
		t.Fatal("an unreadable record must be an error")
	}
	if err := d.Finished(ctx, "../x", at); err == nil {
		t.Fatal("unsafe name accepted")
	}
}
