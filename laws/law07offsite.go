package laws

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"example.com/second-brain/internal/adapters/backup/s3"
	"example.com/second-brain/internal/adapters/backup/s3/s3test"
	"example.com/second-brain/internal/cli"
	"example.com/second-brain/internal/core/offsite"
)

// LAW-07 and LAW-11 for off-box copies to S3 (SB-2303): whatever the box's scripts hand
// to the company's bucket, only sealed files and small records reach it, so no content is
// ever readable there, and what was sealed opens with the recovery key. The real sb backup
// push over a fake bucket.

// PushSystem sends one file to the bucket.
type PushSystem interface {
	Push(file, to, keys string) error
}

// RealPush is sb backup push.
type RealPush struct{}

// Push implements PushSystem.
func (RealPush) Push(file, to, keys string) error {
	var out bytes.Buffer
	if code := cli.Run([]string{"-q", "backup", "push", "--to", to, "--keys", keys, "--as", "db/" + filepath.Base(file), file}, &out, &out); code != 0 {
		return fmt.Errorf("push: %s", out.String())
	}
	return nil
}

func law07Offsite(r *mrand.Rand, sys PushSystem, dir string) error {
	f := &s3test.Fake{Bucket: "acme-backups", Access: "ak"}
	srv := httptest.NewServer(f)
	defer srv.Close()
	keys := filepath.Join(dir, fmt.Sprintf("keys-%d", r.Int63()))
	kb, _ := json.Marshal(map[string]string{"access_key": "ak", "secret_key": "sk", "endpoint": srv.URL})
	if err := os.WriteFile(keys, kb, 0o600); err != nil {
		return err
	}
	key, err := offsite.Generate(rand.Reader)
	if err != nil {
		return err
	}
	canary := fmt.Sprintf("OFFCANARY%x", r.Int63())
	sealedWant := map[string][]byte{}
	for i, n := 0, 2+r.Intn(6); i < n; i++ {
		content := []byte(strings.Repeat("acme-logistics row "+canary+"\n", 1+r.Intn(400)))
		name := filepath.Join(dir, fmt.Sprintf("f%d-%d", r.Int63(), i))
		var data []byte
		switch r.Intn(4) {
		case 0: // sealed
			var b bytes.Buffer
			if err := offsite.Seal(&b, bytes.NewReader(content), key.Public(), rand.Reader); err != nil {
				return err
			}
			data = b.Bytes()
			sealedWant["db/"+filepath.Base(name)] = content
		case 1: // a record
			data = []byte(`{"tenant":"acme-logistics","keys":"0123456789abcdef"}`)
		case 2: // plain content
			data = content
		default: // content dressed as a record, too big to be one
			data, _ = json.Marshal(map[string]string{"rows": strings.Repeat(canary, cli.MaxRecord/len(canary)+1)})
		}
		if err := os.WriteFile(name, data, 0o600); err != nil {
			return err
		}
		_ = sys.Push(name, "s3://acme-backups/second-brain", keys) // a refusal is the point
	}
	b := s3.Bucket{Endpoint: srv.URL, BucketName: "acme-backups", Prefix: "second-brain", Plain: true, HTTP: srv.Client(),
		Creds: func(context.Context) (string, error) { return string(kb), nil }}
	ctx := context.Background()
	names, err := b.List(ctx, "")
	if err != nil {
		return err
	}
	for _, n := range names {
		data, err := b.Get(ctx, n)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte(canary)) {
			return fmt.Errorf("LAW-07: %s reached the bucket readable", n)
		}
		if want, ok := sealedWant[n]; ok {
			var got bytes.Buffer
			if err := offsite.Open(&got, bytes.NewReader(data), key); err != nil || !bytes.Equal(got.Bytes(), want) {
				return fmt.Errorf("LAW-11: %s doesn't open to what was sealed: %v", n, err)
			}
		}
	}
	for n := range sealedWant {
		if _, err := b.Get(ctx, n); err != nil {
			return fmt.Errorf("LAW-11: the sealed copy %s never reached the bucket: %v", n, err)
		}
	}
	return nil
}
