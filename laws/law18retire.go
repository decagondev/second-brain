package laws

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/box"
)

// LAW-11 and LAW-18 for a retired box (SB-1907): the statement names the last row of the
// cell's own audit chain, and only an export that verifies under the cell's key; it
// verifies under the key printed for it and no other, and not at all after any change;
// and each statement has its own key, so no key on the destroyed box (and no earlier
// statement's) can have signed it.

// Retirer is the system under test (internal/core/box).
type Retirer struct {
	LastAnchor func(export []byte, tenant, cellKey string) (audit.Anchor, int, error)
	Sign       func(r box.Retirement, rnd io.Reader) (box.Retirement, error)
	Verify     func(b []byte, key string) (box.Retirement, error)
}

// RealRetirer is the real one.
var RealRetirer = Retirer{
	LastAnchor: box.LastAnchor,
	Sign:       func(r box.Retirement, rnd io.Reader) (box.Retirement, error) { return r.Sign(rnd, time.Now()) },
	Verify:     box.VerifyRetirement,
}

func retireExport(r *rand.Rand, rows int) ([]byte, string, error) {
	st := auditmem.New()
	al := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: st, Now: time.Now}
	for i := 0; i < rows; i++ {
		if _, err := al.Write(context.Background(), audit.Event{Actor: "sys_jobs", Kind: "config", Subject: fmt.Sprintf("s%d", r.Intn(1000)), Reason: "r"}); err != nil {
			return nil, "", err
		}
	}
	seed := make([]byte, 32)
	r.Read(seed)
	s, err := audit.NewSigner("acme-logistics", seed)
	if err != nil {
		return nil, "", err
	}
	var b bytes.Buffer
	if _, err := audit.Export(context.Background(), st, s, 1, time.Now(), &b); err != nil {
		return nil, "", err
	}
	return b.Bytes(), s.Key(), nil
}

func law18Retire(r *rand.Rand, sys Retirer) error {
	exp, cellKey, err := retireExport(r, 1+r.Intn(30))
	if err != nil {
		return err
	}
	last, n, err := sys.LastAnchor(exp, "acme-logistics", cellKey)
	if err != nil {
		return fmt.Errorf("a good export: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(exp)), "\n")
	if n != len(lines)-2 || last.Seq != int64(n) {
		return fmt.Errorf("the statement would name row %d of %d", last.Seq, len(lines)-2)
	}
	// A broken chain, or another cell's key, names nothing.
	row := 1 + r.Intn(len(lines)-2)
	broken := append([]string(nil), lines...)
	broken[row] = strings.Replace(broken[row], `"reason":"r"`, `"reason":"x"`, 1)
	if _, _, err := sys.LastAnchor([]byte(strings.Join(broken, "\n")+"\n"), "acme-logistics", cellKey); err == nil {
		return fmt.Errorf("an export with row %d changed was accepted", row)
	}
	_, otherKey, _ := retireExport(r, 1)
	if _, _, err := sys.LastAnchor(exp, "acme-logistics", otherKey); err == nil {
		return fmt.Errorf("an export checked under another cell's key was accepted")
	}
	st := box.Retirement{Tenant: "acme-logistics", Host: "cell.acme.example", Destroyed: []string{"volumes", "keys"}, Kept: []string{"sealed copies"}, LastAnchor: last, AuditRows: n}
	a, err := sys.Sign(st, cryptorand)
	if err != nil {
		return err
	}
	b, err := sys.Sign(st, cryptorand)
	if err != nil {
		return err
	}
	if a.Key == b.Key {
		return fmt.Errorf("two statements share a key: a key is kept somewhere")
	}
	if a.Key == cellKey {
		return fmt.Errorf("the statement is signed with the cell's audit key, which the retirement destroys")
	}
	raw := a.JSON()
	if _, err := sys.Verify(raw, a.Key); err != nil {
		return fmt.Errorf("the statement doesn't verify under its own key: %v", err)
	}
	if _, err := sys.Verify(raw, b.Key); err == nil {
		return fmt.Errorf("the statement verifies under another statement's key")
	}
	// Any one changed field is refused.
	mut := []func(*box.Retirement){
		func(x *box.Retirement) { x.LastAnchor.Hash = strings.Repeat("0", 64) },
		func(x *box.Retirement) { x.LastAnchor.Seq-- },
		func(x *box.Retirement) { x.Destroyed = x.Destroyed[1:] },
		func(x *box.Retirement) { x.Kept = append(x.Kept, "the vault") },
		func(x *box.Retirement) { x.At = "2020-01-01T00:00:00Z" },
		func(x *box.Retirement) { x.Tenant = "beta-freight" },
	}
	m := r.Intn(len(mut))
	c := a
	c.Destroyed = append([]string(nil), a.Destroyed...)
	c.Kept = append([]string(nil), a.Kept...)
	mut[m](&c)
	if _, err := sys.Verify(c.JSON(), a.Key); err == nil {
		return fmt.Errorf("a statement with change %d still verifies", m)
	}
	return nil
}

var cryptorand = cryptoReader{}

type cryptoReader struct{}

func (cryptoReader) Read(p []byte) (int, error) { return crand.Read(p) }
