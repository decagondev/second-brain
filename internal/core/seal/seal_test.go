package seal

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/ports"
)

// plainKeys "wraps" by copying; enough to test sealing, which never sees the root key.
type plainKeys struct{}

func (plainKeys) EnsureRootKey(context.Context, ports.CellResource) (string, error) { return "k", nil }
func (plainKeys) Wrap(_ context.Context, _ string, k []byte) ([]byte, error) {
	return append([]byte(nil), k...), nil
}
func (plainKeys) Unwrap(_ context.Context, _ string, w []byte) ([]byte, error) {
	return append([]byte(nil), w...), nil
}

func TestSealOpenShred(t *testing.T) {
	ctx := context.Background()
	st := &keywrap.MemoryStore{}
	c := keywrap.Custodian{Name: "factory", Keys: plainKeys{}, KeyID: "k"}
	for _, sp := range []string{"spc_finance", "spc_eng"} {
		if err := keywrap.Seed(ctx, "acme-logistics", st, c, keywrap.PurposeSpace(sp)); err != nil {
			t.Fatal(err)
		}
	}
	s := Sealer{Tenant: "acme-logistics", Wraps: st, Custodians: []keywrap.Custodian{c}}
	plain := []byte("Q3 forecast: confidential")
	a, err := s.Seal(ctx, "spc_finance", plain)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.Seal(ctx, "spc_finance", plain)
	if bytes.Equal(a, b) || bytes.Contains(a, plain) {
		t.Fatal("sealing must use a fresh nonce and hide the text")
	}
	if got, err := s.Open(ctx, "spc_finance", a); err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("open: %q %v", got, err)
	}
	if _, err := s.Open(ctx, "spc_eng", a); !errors.Is(err, ErrOpen) {
		t.Fatalf("another space's key opened it: %v", err)
	}
	other := s
	other.Tenant = "beta-freight"
	if err := keywrap.Seed(ctx, "beta-freight", st, c, keywrap.PurposeSpace("spc_finance")); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Open(ctx, "spc_finance", a); !errors.Is(err, ErrOpen) {
		t.Fatalf("another cell opened it: %v", err)
	}
	tampered := append([]byte(nil), a...)
	tampered[len(tampered)-1] ^= 1
	if _, err := s.Open(ctx, "spc_finance", tampered); !errors.Is(err, ErrOpen) {
		t.Fatalf("tampered: %v", err)
	}
	if err := keywrap.Shred(ctx, "acme-logistics", st, keywrap.PurposeSpace("spc_finance")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(ctx, "spc_finance", a); !errors.Is(err, ErrShredded) {
		t.Fatalf("after shred: %v", err)
	}
	if _, err := s.Seal(ctx, "spc_finance", plain); !errors.Is(err, ErrShredded) {
		t.Fatalf("sealing into a shredded space: %v", err)
	}
	if e, _ := s.Seal(ctx, "spc_eng", plain); e == nil {
		t.Fatal("shredding one space broke another")
	}
}
