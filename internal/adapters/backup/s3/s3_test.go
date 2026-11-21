package s3

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/backup/s3/s3test"
)

const locked = s3test.Locked

// The GET Object example of AWS's Signature Version 4 documentation for S3. The example
// key pair is built from parts, so secret scanners don't flag the documentation's own.
func TestSignMatchesTheAWSExample(t *testing.T) {
	access := "AKIA" + "IOSFODNN7EXAMPLE"
	secret := "wJalrXUtnFEMI/K7MDENG/" + "bPxRfiCYEXAMPLEKEY"
	req, _ := http.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	req.Header.Set("Range", "bytes=0-9")
	Sign(req, nil, access, secret, "us-east-1", time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC))
	want := "AWS4-HMAC-SHA256 Credential=" + access + "/20130524/us-east-1/s3/aws4_request,SignedHeaders=host;range;x-amz-content-sha256;x-amz-date," +
		"Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization\n got %s\nwant %s", got, want)
	}
}

func bucket(t *testing.T, f *s3test.Fake) Bucket {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	cr, _ := json.Marshal(map[string]string{"access_key": "ak", "secret_key": "sk"})
	return Bucket{Endpoint: srv.URL, BucketName: "sb-worm", HTTP: srv.Client(), Label: "worm", Away: true,
		Creds: func(context.Context) (string, error) { return string(cr), nil }}
}

func TestWriteOnceAndLocked(t *testing.T) {
	f := s3test.New("sb-worm", "ak")
	f.PageAt = 2
	b := bucket(t, f)
	ctx := context.Background()
	if err := b.Check(ctx); err != nil {
		t.Fatal(err)
	}
	until := time.Date(2034, 3, 10, 0, 0, 0, 0, time.UTC)
	for _, n := range []string{"anchors/000000000001.json", "anchors/000000000002.json", "anchors/000000000003.json", "backup/2027-03-10/audit.ndjson"} {
		if err := b.Put(ctx, n, []byte("x "+n), until); err != nil {
			t.Fatal(err)
		}
	}
	if f.Objects() != 4 {
		t.Fatalf("%d objects, want 4", f.Objects())
	}
	if err := b.Put(ctx, "anchors/000000000001.json", []byte("other"), until); !errors.Is(err, ErrExists) {
		t.Fatalf("an overwrite: %v", err)
	}
	if got, _ := b.Get(ctx, "anchors/000000000001.json"); string(got) != "x anchors/000000000001.json" {
		t.Fatalf("get %q", got)
	}
	ks, err := b.List(ctx, "anchors/")
	if err != nil || len(ks) != 3 {
		t.Fatalf("list over pages: %v %v", ks, err)
	}
	if err := b.Put(ctx, "../escape", nil, until); err == nil {
		t.Fatal("a bad name was written")
	}
	if b.Medium() != "object-store" || !b.Offsite() {
		t.Fatal("medium or off-site wrong")
	}
}

// A bucket without object lock, or with governance mode, or with less than seven years,
// is refused: a copy someone can delete isn't WORM (LAW-11).
func TestCheckRefusesWeakLocks(t *testing.T) {
	for name, cfg := range map[string]string{
		"none":       "",
		"governance": strings.Replace(locked, "COMPLIANCE", "GOVERNANCE", 1),
		"one year":   strings.Replace(locked, "<Years>7</Years>", "<Years>1</Years>", 1),
		"disabled":   strings.Replace(locked, "<ObjectLockEnabled>Enabled", "<ObjectLockEnabled>", 1),
	} {
		f := s3test.New("sb-worm", "ak")
		f.Lock = cfg
		if err := bucket(t, f).Check(context.Background()); !errors.Is(err, ErrNotLocked) {
			t.Errorf("%s: %v, want ErrNotLocked", name, err)
		}
	}
	f := s3test.New("sb-worm", "ak")
	f.Lock = strings.Replace(locked, "<Years>7</Years>", "<Days>2555</Days>", 1)
	if err := bucket(t, f).Check(context.Background()); err != nil {
		t.Fatalf("2,555 days is seven years: %v", err)
	}
}

func TestBadCredentials(t *testing.T) {
	b := bucket(t, s3test.New("sb-worm", "ak"))
	b.Creds = func(context.Context) (string, error) { return "not json", nil }
	if err := b.Check(context.Background()); err == nil {
		t.Fatal("bad credentials were used")
	}
}

// Enforced: a store whose lock refuses the delete passes; one that only claims the lock,
// and lets the delete through, is refused (LAW-11).
func TestEnforcedNeedsARefusedDelete(t *testing.T) {
	if err := bucket(t, s3test.New("sb-worm", "ak")).Enforced(context.Background()); err != nil {
		t.Fatalf("an enforcing store: %v", err)
	}
	f := s3test.New("sb-worm", "ak")
	f.Unenforced = true
	if err := bucket(t, f).Enforced(context.Background()); !errors.Is(err, ErrNotLocked) {
		t.Fatalf("a store that lets the delete through: %v, want ErrNotLocked", err)
	}
}
