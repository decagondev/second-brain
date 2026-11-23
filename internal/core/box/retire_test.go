package box

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

type rows struct{ rs []audit.Row }

func (s *rows) Append(_ context.Context, build func(int64, string) (audit.Row, error)) (audit.Row, error) {
	seq, hash := int64(0), audit.Zero
	if n := len(s.rs); n > 0 {
		seq, hash = s.rs[n-1].Seq, s.rs[n-1].Hash
	}
	r, err := build(seq, hash)
	if err == nil {
		s.rs = append(s.rs, r)
	}
	return r, err
}

func (s *rows) Range(_ context.Context, from int64, limit int) ([]audit.Row, error) {
	var out []audit.Row
	for _, r := range s.rs {
		if r.Seq >= from && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}

// cellExport is a cell's audit export with n rows, and its key.
func cellExport(t testing.TB, n int) ([]byte, string) {
	t.Helper()
	st := &rows{}
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: st, Now: time.Now}
	for i := 0; i < n; i++ {
		if _, err := log.Write(context.Background(), audit.Event{Actor: "sys_jobs", Kind: "config", Subject: "x", Reason: "r"}); err != nil {
			t.Fatal(err)
		}
	}
	s, err := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if _, err := audit.Export(context.Background(), st, s, 1, time.Now(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes(), s.Key()
}

func TestRetirementStatement(t *testing.T) {
	exp, cellKey := cellExport(t, 5)
	a, n, err := LastAnchor(exp, "acme-logistics", cellKey)
	if err != nil || a.Seq != 5 || n != 5 {
		t.Fatalf("anchor %+v %d %v", a, n, err)
	}
	r, err := Retirement{Tenant: "acme-logistics", Host: "cell.acme.example", Destroyed: []string{"volumes: pgdata, worm-data", "keys: 12 files under build/selfhost"},
		Kept: []string{"sealed off-box copies, openable only with the recovery kit"}, LastAnchor: a, AuditRows: n}.Sign(rand.Reader, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	b := r.JSON()
	if _, err := VerifyRetirement(b, r.Key); err != nil {
		t.Fatalf("verify: %v", err)
	}
	for name, bad := range map[string][]byte{
		"a changed hash":   bytes.Replace(b, []byte(a.Hash), []byte(strings.Repeat("0", len(a.Hash))), 1),
		"a dropped entry":  bytes.Replace(b, []byte(`"volumes: pgdata, worm-data",`), nil, 1),
		"an unknown field": bytes.Replace(b, []byte(`"statement":`), []byte(`"extra": 1, "statement":`), 1),
	} {
		if _, err := VerifyRetirement(bad, r.Key); !errors.Is(err, ErrStatement) {
			t.Errorf("%s: %v", name, err)
		}
	}
	other, _ := Retirement{Tenant: "acme-logistics", Destroyed: []string{"x"}, LastAnchor: a}.Sign(rand.Reader, time.Now())
	if _, err := VerifyRetirement(b, other.Key); !errors.Is(err, ErrStatement) {
		t.Fatalf("another key: %v", err)
	}
	// A broken chain, or one checked under another cell's key, names no anchor.
	if _, _, err := LastAnchor(bytes.Replace(exp, []byte(`"reason":"r"`), []byte(`"reason":"s"`), 1), "acme-logistics", cellKey); !errors.Is(err, ErrStatement) {
		t.Fatalf("broken chain: %v", err)
	}
	if _, _, err := LastAnchor(exp, "acme-logistics", other.Key); !errors.Is(err, ErrStatement) {
		t.Fatalf("other key: %v", err)
	}
	if _, err := (Retirement{Tenant: "acme-logistics", LastAnchor: a}).Sign(rand.Reader, time.Now()); !errors.Is(err, ErrStatement) {
		t.Fatalf("nothing destroyed: %v", err)
	}
}
