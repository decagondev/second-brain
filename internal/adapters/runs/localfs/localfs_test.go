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

// SB-1607: a connector's sync record counts failures since its last success.
func TestSyncRecords(t *testing.T) {
	s := Syncs{Root: filepath.Join(t.TempDir(), "runs")}
	ctx := context.Background()
	if _, found, err := s.Sync(ctx, "docs"); found || err != nil {
		t.Fatalf("never synced: %v %v", found, err)
	}
	at := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	_ = s.Succeeded(ctx, "docs", at)
	_ = s.Failed(ctx, "docs", at.Add(time.Minute))
	_ = s.Failed(ctx, "docs", at.Add(2*time.Minute))
	x, found, err := s.Sync(ctx, "docs")
	if err != nil || !found || x.Failures != 2 || !x.LastSuccess.Equal(at) || !x.LastFailure.Equal(at.Add(2*time.Minute)) {
		t.Fatalf("record %+v %v %v", x, found, err)
	}
	_ = s.Succeeded(ctx, "docs", at.Add(3*time.Minute))
	if x, _, _ := s.Sync(ctx, "docs"); x.Failures != 0 {
		t.Fatalf("a success clears the failures: %+v", x)
	}
	if _, _, err := s.Sync(ctx, "../etc"); err == nil {
		t.Fatal("an unsafe kind was read")
	}
	_ = os.WriteFile(filepath.Join(s.Root, "sync-chat.json"), []byte("{"), 0o600)
	if _, _, err := s.Sync(ctx, "chat"); err == nil {
		t.Fatal("an unreadable record read as fine")
	}
}
