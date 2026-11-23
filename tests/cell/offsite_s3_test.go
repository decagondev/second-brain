package cell

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/backup/s3"
	"example.com/second-brain/internal/adapters/backup/s3/s3test"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/offsite"
)

// SB-2303: the off-box copy goes to the company's bucket, with no object lock, under its
// prefix, every object sealed; replicated to a second store, it restores from there with
// the recovery key, by pointing the key file at that store.
func TestBackupToAnS3Bucket(t *testing.T) {
	r, svc, _, _ := jobsCell(t, true)
	ctx := context.Background()
	a := &s3test.Fake{Bucket: "acme-backups", Access: "ak"}
	srvA := httptest.NewServer(a)
	defer srvA.Close()
	keys := func(endpoint string) {
		b, _ := json.Marshal(map[string]string{"access_key": "ak", "secret_key": "sk", "endpoint": endpoint})
		p := filepath.Join(r.cell.Config.Vault, "vault", "tenants", tenant, "offsite")
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	keys(srvA.URL)
	key, err := offsite.Generate(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	copyS3 := wire.Backup{S3: "s3://acme-backups/second-brain", Offsite: true, Seal: key.Public().String()}
	r.cell.Config.Backups = []wire.Backup{{Dir: filepath.Join(base, "a"), Media: "disk"}, {Dir: filepath.Join(base, "b"), Media: "nas"}, copyS3}
	if err := svc.Run(ctx, "backup"); err != nil {
		t.Fatal(err)
	}
	// Second store: a replica, and the key file pointed at it.
	b := &s3test.Fake{Bucket: "acme-backups", Access: "ak"}
	srvB := httptest.NewServer(b)
	defer srvB.Close()
	a.ReplicateTo(b)
	srvA.Close()
	keys(srvB.URL)
	bk, err := r.cell.OffsiteBucket(ctx, copyS3)
	if err != nil {
		t.Fatal(err)
	}
	names, err := bk.List(ctx, "")
	if err != nil || len(names) != 4 {
		t.Fatalf("objects in the bucket: %v %v", names, err)
	}
	var export string
	for _, n := range names {
		data, err := bk.Get(ctx, n)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(data, []byte(offsite.Magic)) || !strings.HasSuffix(n, ".sealed") {
			t.Fatalf("%s reached the bucket unsealed", n)
		}
		if strings.HasSuffix(n, "/audit.ndjson.sealed") {
			export = n
		}
	}
	plain, _ := filepath.Glob(filepath.Join(base, "a", "*", "*", "audit.ndjson"))
	if export == "" || len(plain) != 1 {
		t.Fatalf("no export: %v %v", names, plain)
	}
	want, _ := os.ReadFile(plain[0])
	sealed, _ := bk.Get(ctx, export)
	var got bytes.Buffer
	if err := offsite.Open(&got, bytes.NewReader(sealed), key); err != nil || !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("the copy from the second store doesn't open to the box's copy: %v", err)
	}
	// The wrong key pair is refused, and the backup fails rather than skip the copy.
	p := filepath.Join(r.cell.Config.Vault, "vault", "tenants", tenant, "offsite")
	if err := os.WriteFile(p, []byte(`{"access_key":"other","secret_key":"sk","endpoint":"`+srvB.URL+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(ctx, "backup"); !errors.Is(err, s3.ErrRefused) {
		t.Fatalf("a refused key pair: %v", err)
	}
}
