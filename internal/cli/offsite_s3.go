package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/backup/s3"
	"example.com/second-brain/internal/core/offsite"
	"example.com/second-brain/internal/errs"
)

// MaxRecord is the largest unsealed file sb backup push sends: a record (a dump's tenant
// and key fingerprint, a kit's custodians, a retirement statement), never content.
const MaxRecord = 64 << 10

// offBucket is the bucket an off-box copy goes to (SB-2303): s3://BUCKET/PREFIX, its key
// pair file (the jobs service's vault entry, vault://tenants/TENANT/offsite).
func offBucket(raw, keys string) (s3.Bucket, error) {
	name, prefix, err := s3.URL(raw)
	if err != nil {
		return s3.Bucket{}, errs.New(errs.EUSAGE, "%v", err)
	}
	if keys == "" {
		return s3.Bucket{}, errs.New(errs.EUSAGE, "--keys names the bucket's key pair file")
	}
	return s3.Bucket{BucketName: name, Prefix: prefix, Plain: true, Away: true, Label: raw, HTTP: &http.Client{Timeout: 10 * time.Minute},
		Creds: func(context.Context) (string, error) {
			b, err := os.ReadFile(keys)
			if err != nil {
				return "", errs.New(errs.EINVAL, "can't read the key pair file %s", keys)
			}
			return strings.TrimSpace(string(b)), nil
		}}, nil
}

// s3Err is an S3 failure as an exit code: a refused key or missing bucket is EAUTH, an
// object that is there already EINVAL, anything else (unreachable) EUNAVAIL.
func s3Err(err error) error {
	var x *errs.Error
	switch {
	case errors.As(err, &x):
		return err
	case errors.Is(err, s3.ErrRefused):
		return errs.New(errs.EAUTH, "%v", err)
	case errors.Is(err, s3.ErrExists):
		return errs.New(errs.EINVAL, "%v", err)
	}
	return errs.New(errs.EUNAVAIL, "%v", err)
}

// sealedOrRecord is whether data may leave the box: sealed (to the recovery key or a kit
// key), or a record, a JSON object of at most MaxRecord bytes.
func sealedOrRecord(data []byte) bool {
	if offsite.IsSealed(data) {
		return true
	}
	var m map[string]any
	return len(data) <= MaxRecord && json.Unmarshal(data, &m) == nil
}

// backupPush sends one file to the off-box bucket, written once: `sb backup push --to
// s3://BUCKET/PREFIX --keys FILE [--as NAME] FILE`. A file that isn't sealed, or a record,
// is refused: nothing unsealed leaves the box.
func (e *env) backupPush(args []string) int {
	const cmd = "sb backup push"
	opts, pos, err := flags(cmd, args, "--to", "--keys", "--as")
	if err != nil {
		return e.fail(cmd, err)
	}
	if len(pos) != 1 {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb backup push --to s3://BUCKET/PREFIX --keys FILE [--as NAME] FILE"))
	}
	b, err := offBucket(one(opts, "--to"), one(opts, "--keys"))
	if err != nil {
		return e.fail(cmd, err)
	}
	data, err := os.ReadFile(pos[0])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", pos[0]))
	}
	if !sealedOrRecord(data) {
		return e.fail(cmd, errs.New(errs.EPERM, "%s isn't sealed (sb backup seal) or a record; nothing unsealed leaves the box", pos[0]))
	}
	name := one(opts, "--as")
	if name == "" {
		name = filepath.Base(pos[0])
	}
	if err := b.Put(context.Background(), name, data, time.Time{}); err != nil {
		return e.fail(cmd, s3Err(err))
	}
	if !e.g.Quiet {
		fmt.Fprintf(e.out, "pushed %s to %s/%s\n", pos[0], one(opts, "--to"), name)
	}
	return 0
}

// backupPull fetches objects from the off-box bucket into a directory: `sb backup pull
// --from s3://BUCKET/PREFIX --keys FILE --out DIR NAME...`, each as DIR/BASENAME.
func (e *env) backupPull(args []string) int {
	const cmd = "sb backup pull"
	opts, pos, err := flags(cmd, args, "--from", "--keys", "--out")
	if err != nil {
		return e.fail(cmd, err)
	}
	dir := one(opts, "--out")
	if len(pos) == 0 || dir == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb backup pull --from s3://BUCKET/PREFIX --keys FILE --out DIR NAME..."))
	}
	b, err := offBucket(one(opts, "--from"), one(opts, "--keys"))
	if err != nil {
		return e.fail(cmd, err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't make %s", dir))
	}
	for _, n := range pos {
		data, err := b.Get(context.Background(), n)
		if err != nil {
			return e.fail(cmd, s3Err(err))
		}
		out := filepath.Join(dir, path.Base(n))
		if err := writeAtomic(out, func(w io.Writer) error { _, err := w.Write(data); return err }); err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't write %s", out))
		}
		if !e.g.Quiet {
			fmt.Fprintf(e.out, "pulled %s\n", out)
		}
	}
	return 0
}

// backupList names the objects under PREFIX/[SUB] in the off-box bucket, sorted (so a
// dump's names, which carry their time, come oldest first): `sb backup list --from
// s3://BUCKET/PREFIX --keys FILE [SUB]`.
func (e *env) backupList(args []string) int {
	const cmd = "sb backup list"
	opts, pos, err := flags(cmd, args, "--from", "--keys")
	if err != nil {
		return e.fail(cmd, err)
	}
	if len(pos) > 1 {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb backup list --from s3://BUCKET/PREFIX --keys FILE [SUB]"))
	}
	b, err := offBucket(one(opts, "--from"), one(opts, "--keys"))
	if err != nil {
		return e.fail(cmd, err)
	}
	sub := ""
	if len(pos) == 1 {
		sub = pos[0]
	}
	names, err := b.List(context.Background(), sub)
	if err != nil {
		return e.fail(cmd, s3Err(err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"objects": names})
	}
	var w bytes.Buffer
	for _, n := range names {
		w.WriteString(n + "\n")
	}
	_, _ = e.out.Write(w.Bytes())
	return 0
}

// backupCheck asks whether the off-box bucket answers with the key pair: `sb backup check
// --to s3://BUCKET/PREFIX --keys FILE`. It prints ok, refused (the key pair or the
// bucket) or unreachable, and exits 0 only for ok; the doctor reads it (SB-2303).
func (e *env) backupCheck(args []string) int {
	const cmd = "sb backup check"
	opts, pos, err := flags(cmd, args, "--to", "--keys")
	if err != nil {
		return e.fail(cmd, err)
	}
	if len(pos) > 0 {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb backup check --to s3://BUCKET/PREFIX --keys FILE"))
	}
	b, err := offBucket(one(opts, "--to"), one(opts, "--keys"))
	if err != nil {
		return e.fail(cmd, err)
	}
	answer, code := "ok", 0
	if err := b.Check(context.Background()); err != nil {
		answer, code = "unreachable", 1
		if errs.CodeOf(s3Err(err)) == errs.EAUTH {
			answer = "refused"
		}
	}
	if e.g.JSON {
		e.json(map[string]any{"bucket": one(opts, "--to"), "answer": answer})
		return code
	}
	fmt.Fprintf(e.out, "%s: %s\n", one(opts, "--to"), answer)
	return code
}
