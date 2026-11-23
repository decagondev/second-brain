package s3

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/backup/s3/s3test"
)

// An off-box copy (SB-2303): a bucket with no lock, under a prefix, its store named by the
// key file.
func TestPlainBucketUnderAPrefix(t *testing.T) {
	f := &s3test.Fake{Bucket: "acme-backups", Access: "ak", PageAt: 1}
	srv := httptest.NewServer(f)
	defer srv.Close()
	keys := func(access string) func(context.Context) (string, error) {
		cr, _ := json.Marshal(map[string]string{"access_key": access, "secret_key": "sk", "endpoint": srv.URL, "region": "eu-central-1"})
		return func(context.Context) (string, error) { return string(cr), nil }
	}
	b := Bucket{BucketName: "acme-backups", Prefix: "second-brain/box1", Plain: true, Creds: keys("ak"), HTTP: srv.Client(), Away: true}
	ctx := context.Background()
	if err := b.Check(ctx); err != nil {
		t.Fatalf("a plain bucket: %v", err)
	}
	for _, n := range []string{"db/sb-2026-11-22T020000Z.dump.sealed", "db/sb-2026-11-23T020000Z.dump.sealed", "kit/kit_1a2b.sealed"} {
		if err := b.Put(ctx, n, []byte("sealed "+n), time.Time{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.Put(ctx, "kit/kit_1a2b.sealed", []byte("other"), time.Time{}); !errors.Is(err, ErrExists) {
		t.Fatalf("an overwrite: %v", err)
	}
	got, err := b.List(ctx, "db/")
	if err != nil || !reflect.DeepEqual(got, []string{"db/sb-2026-11-22T020000Z.dump.sealed", "db/sb-2026-11-23T020000Z.dump.sealed"}) {
		t.Fatalf("list: %v %v", got, err)
	}
	if d, err := b.Get(ctx, "kit/kit_1a2b.sealed"); err != nil || string(d) != "sealed kit/kit_1a2b.sealed" {
		t.Fatalf("get: %q %v", d, err)
	}
	// A plain bucket isn't the write-once store: the locked check still refuses it.
	if err := (Bucket{BucketName: "acme-backups", Creds: keys("ak"), HTTP: srv.Client()}).Check(ctx); !errors.Is(err, ErrNotLocked) {
		t.Fatalf("a plain bucket as the write-once store: %v", err)
	}
	for name, bad := range map[string]Bucket{
		"the wrong key": {BucketName: "acme-backups", Plain: true, Creds: keys("other"), HTTP: srv.Client()},
		"no bucket":     {BucketName: "acme-nothing", Plain: true, Creds: keys("ak"), HTTP: srv.Client()},
	} {
		if err := bad.Check(ctx); !errors.Is(err, ErrRefused) {
			t.Errorf("%s: %v, want ErrRefused", name, err)
		}
	}
}

func TestURL(t *testing.T) {
	for raw, want := range map[string][2]string{
		"s3://acme-backups":               {"acme-backups", ""},
		"s3://acme-backups/second-brain/": {"acme-backups", "second-brain"},
		"s3://acme.backups/sb/Box_1":      {"acme.backups", "sb/Box_1"},
		"s3://Acme/sb":                    {},
		"https://acme-backups/sb":         {},
		"s3://acme-backups/../sb":         {},
		"s3://acme-backups/sb//x":         {},
		"/mnt/backup-nas/second-brain":    {},
	} {
		b, p, err := URL(raw)
		if (err == nil) != (want[0] != "") || b != want[0] || p != want[1] {
			t.Errorf("URL(%q) = %q %q %v, want %q", raw, b, p, err, want)
		}
	}
}
