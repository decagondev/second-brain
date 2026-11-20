package keywrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// xorKeys is a toy key store: wrapping XORs with the key id's bytes and appends a tag.
type xorKeys struct{ fail bool }

func (x xorKeys) mask(keyID string, b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[i] ^ keyID[i%len(keyID)]
	}
	return out
}
func (x xorKeys) EnsureRootKey(context.Context, ports.CellResource) (string, error) { return "", nil }
func (x xorKeys) Wrap(_ context.Context, keyID string, k []byte) ([]byte, error) {
	if x.fail {
		return nil, errors.New("kms down")
	}
	return append(x.mask(keyID, k), []byte(keyID)...), nil
}
func (x xorKeys) Unwrap(_ context.Context, keyID string, w []byte) ([]byte, error) {
	if x.fail || len(w) < len(keyID) || !bytes.HasSuffix(w, []byte(keyID)) {
		return nil, fmt.Errorf("unwrap under %s failed", keyID)
	}
	return x.mask(keyID, w[:len(w)-len(keyID)]), nil
}

var ctx = context.Background()

func custodians() (Custodian, Custodian) {
	return Custodian{Name: "factory", Keys: xorKeys{}, KeyID: "factory-key"},
		Custodian{Name: "tenant", Keys: xorKeys{}, KeyID: "tenant-key-1"}
}

// The ordering test the ticket asks for: destroying the factory wrap before the tenant's
// exists is refused, and nothing is lost.
func TestDestroyBeforeRewrapIsRefused(t *testing.T) {
	st := &MemoryStore{}
	f, tn := custodians()
	if err := Seed(ctx, "gamma-parts", st, f, PurposeAuditSigning); err != nil {
		t.Fatal(err)
	}
	if err := Destroy(ctx, "gamma-parts", st, f, tn); !errors.Is(err, ErrOrphan) {
		t.Fatalf("destroy before re-wrap: %v", err)
	}
	if ws, _ := st.List(ctx, "gamma-parts"); len(ws) != 1 || ws[0].Custodian != "factory" {
		t.Fatalf("the factory wrap was lost: %+v", ws)
	}
	n, err := Rewrap(ctx, "gamma-parts", st, f, tn)
	if err != nil || n != 1 {
		t.Fatalf("rewrap: %d %v", n, err)
	}
	if n, _ := Rewrap(ctx, "gamma-parts", st, f, tn); n != 0 {
		t.Fatal("a second re-wrap wrapped again")
	}
	if err := Destroy(ctx, "gamma-parts", st, f, tn); err != nil {
		t.Fatal(err)
	}
	inv, _ := Inventory(ctx, "gamma-parts", st, map[string]ports.Keys{"tenant": tn.Keys})
	if len(inv) != 1 || fmt.Sprint(inv[0].Working) != "[tenant]" || Orphaned(inv) {
		t.Fatalf("inventory: %+v", inv)
	}
}

func TestDestroyChecksNow(t *testing.T) {
	st := &MemoryStore{}
	f, tn := custodians()
	_ = Seed(ctx, "gamma-parts", st, f, PurposeAuditSigning)
	_, _ = Rewrap(ctx, "gamma-parts", st, f, tn)
	broken := tn
	broken.Keys = xorKeys{fail: true} // the tenant's KMS is unreachable at destroy time
	if err := Destroy(ctx, "gamma-parts", st, f, broken); !errors.Is(err, ErrOrphan) {
		t.Fatalf("destroy while the tenant can't unwrap: %v", err)
	}
	if err := Abort(ctx, "gamma-parts", st, tn, f); err != nil {
		t.Fatal(err)
	}
	if ws, _ := st.List(ctx, "gamma-parts"); len(ws) != 1 || ws[0].Custodian != "factory" {
		t.Fatalf("abort: %+v", ws)
	}
}

func TestRewrapVerifies(t *testing.T) {
	st := &MemoryStore{}
	f, tn := custodians()
	_ = Seed(ctx, "gamma-parts", st, f, PurposeAuditSigning)
	tn.Keys = lossy{}
	if _, err := Rewrap(ctx, "gamma-parts", st, f, tn); !errors.Is(err, ErrMatch) {
		t.Fatalf("a wrap that gives back a different key: %v", err)
	}
	if ws, _ := st.List(ctx, "gamma-parts"); len(ws) != 1 {
		t.Fatalf("a bad wrap was stored: %+v", ws)
	}
}

// lossy wraps fine but unwraps to the wrong bytes.
type lossy struct{ xorKeys }

func (lossy) Unwrap(_ context.Context, _ string, w []byte) ([]byte, error) { return w[:1], nil }

// SB-1003: only a space's key shreds, and it shreds for every custodian.
func TestShredOnlySpaces(t *testing.T) {
	st := &MemoryStore{}
	f, tn := custodians()
	for _, p := range []string{PurposeAuditSigning, PurposeSpace("spc_finance"), PurposeSpace("spc_eng")} {
		if err := Seed(ctx, "gamma-parts", st, f, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Rewrap(ctx, "gamma-parts", st, f, tn); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{PurposeAuditSigning, "space/", "backup"} {
		if err := Shred(ctx, "gamma-parts", st, p); !errors.Is(err, ErrNotSpace) {
			t.Fatalf("shred %q: %v", p, err)
		}
	}
	if err := Shred(ctx, "gamma-parts", st, PurposeSpace("spc_finance")); err != nil {
		t.Fatal(err)
	}
	if _, found, err := DataKey(ctx, "gamma-parts", st, PurposeSpace("spc_finance"), f, tn); found || err != nil {
		t.Fatalf("shredded key: found %v err %v", found, err)
	}
	for _, p := range []string{PurposeAuditSigning, PurposeSpace("spc_eng")} {
		if k, found, err := DataKey(ctx, "gamma-parts", st, p, tn); !found || err != nil || len(k) != 32 {
			t.Fatalf("%s after a shred: %v %v", p, found, err)
		}
	}
	if _, found, err := DataKey(ctx, "gamma-parts", st, PurposeAuditSigning, Custodian{Name: "tenant", Keys: xorKeys{fail: true}, KeyID: "tenant-key-1"}); !found || err == nil {
		t.Fatalf("a key that doesn't unwrap must be an error, not missing: %v %v", found, err)
	}
}
