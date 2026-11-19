package localfs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExpireAndErase(t *testing.T) {
	dir := t.TempDir()
	day := time.Date(2027, 3, 1, 12, 0, 0, 0, time.UTC)
	f := &Files{Dir: dir, Now: func() time.Time { return day }}
	for _, l := range []string{`{"msg":"ask","user":"usr_ann"}`, `{"msg":"ask","user":"usr_anna"}`, `{"msg":"x","attrs":{"by":["usr_ann"]}}`, `plain usr_ann line`} {
		if _, err := f.Write([]byte(l + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	day = day.Add(24 * time.Hour)
	_, _ = f.Write([]byte(`{"msg":"ask","user":"usr_bob"}` + "\n"))

	ctx := context.Background()
	n, err := f.Erase(ctx, "acme-logistics", "usr_ann")
	if err != nil || n != 3 {
		t.Fatalf("erase: %d %v", n, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "2027-03-01.log"))
	if strings.Contains(string(b), `"usr_ann"`) || !strings.Contains(string(b), "usr_anna") {
		t.Fatalf("after erase: %q", b)
	}

	// The cutoff inside 2027-03-02 keeps that day's file; 2027-03-01 is wholly before it.
	n, err = f.Expire(ctx, "acme-logistics", time.Date(2027, 3, 2, 6, 0, 0, 0, time.UTC))
	if err != nil || n != 1 {
		t.Fatalf("expire: %d %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2027-03-01.log")); !os.IsNotExist(err) {
		t.Fatal("an expired day was kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "2027-03-02.log")); err != nil {
		t.Fatal("a day holding lines after the cutoff was deleted")
	}
}

func TestEmptyDir(t *testing.T) {
	f := &Files{Dir: filepath.Join(t.TempDir(), "none"), Now: time.Now}
	if n, err := f.Expire(context.Background(), "t", time.Now()); n != 0 || err != nil {
		t.Fatal(n, err)
	}
	if n, err := f.Erase(context.Background(), "t", "usr_a"); n != 0 || err != nil {
		t.Fatal(n, err)
	}
}
