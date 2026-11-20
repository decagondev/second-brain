package audit

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/policy"
)

type store struct {
	rows []Row
	fail error
}

func (s *store) Append(_ context.Context, build func(int64, string) (Row, error)) (Row, error) {
	if s.fail != nil {
		return Row{}, s.fail
	}
	seq, h := int64(0), Zero
	if n := len(s.rows); n > 0 {
		seq, h = s.rows[n-1].Seq, s.rows[n-1].Hash
	}
	r, err := build(seq, h)
	if err == nil {
		s.rows = append(s.rows, r)
	}
	return r, err
}

func (s *store) Range(_ context.Context, from int64, limit int) ([]Row, error) {
	var out []Row
	for _, r := range s.rows {
		if r.Seq >= from && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}

var t0 = time.Date(2027, 3, 2, 10, 41, 7, 512_000_000, time.UTC)

func newLog() (*Log, *store) {
	st := &store{}
	return &Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: st, Now: func() time.Time { return t0 }}, st
}

func signer(t *testing.T, b byte) Signer {
	s, err := NewSigner("acme-logistics", bytes.Repeat([]byte{b}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCanonical(t *testing.T) {
	r := Row{Seq: 12, ID: "aud_000000000012", At: "2027-03-02T10:41:07.512Z", Tenant: "acme-logistics", Actor: "usr_1",
		Kind: "decision", Subject: `Chunk::"c<1>&"`, Release: "rel_x", Reason: "read: ok", Prev: Zero}
	got, err := Canonical(r)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"actor":"usr_1","at":"2027-03-02T10:41:07.512Z","id":"aud_000000000012","kind":"decision","prev":"` + Zero +
		`","reason":"read: ok","release":"rel_x","seq":12,"subject":"Chunk::\"c<1>&\"","tenant":"acme-logistics"}`
	if string(got) != want {
		t.Fatalf("canonical\n got %s\nwant %s", got, want)
	}
	for _, bad := range []string{"a\nb", "x ", strings.Repeat("x", MaxField+1)} {
		r.Reason = bad
		if _, err := Canonical(r); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestWriteChainsAndRefuses(t *testing.T) {
	l, st := newLog()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := l.Write(ctx, Event{Actor: "usr_1", Kind: "tool_call", Subject: "tc_1", State: "Executing", Reason: "ok"}); err != nil {
			t.Fatal(err)
		}
	}
	if st.rows[0].Prev != Zero || st.rows[1].Prev != st.rows[0].Hash || st.rows[2].ID != "aud_000000000003" {
		t.Fatalf("chain %+v", st.rows)
	}
	for _, e := range []Event{{Actor: "u", Kind: "gossip", Subject: "s", Reason: "r"}, {Kind: "kill", Subject: "s", Reason: "r"}, {Actor: "u", Kind: "kill", Subject: "s"}} {
		if _, err := l.Write(ctx, e); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v accepted", e)
		}
	}
	st.fail = errors.New("disk full")
	if _, err := l.Write(ctx, Event{Actor: "u", Kind: "kill", Subject: "s", Reason: "r"}); err == nil {
		t.Error("a failed store write was reported as written")
	}
}

func TestDecisionRows(t *testing.T) {
	l, st := newLog()
	if err := l.RecordDecision(context.Background(), policy.Record{Tenant: "acme-logistics", Actor: "usr_1", Action: "call_tool",
		Subject: `Tool::"x"`, Decision: policy.Deny, Reason: "call_tool: no permit"}); err != nil {
		t.Fatal(err)
	}
	r := st.rows[0]
	if r.Kind != "decision" || r.State != "deny" || r.Reason != "call_tool: no permit" || r.Actor != "usr_1" {
		t.Fatalf("%+v", r)
	}
}

func TestExportVerify(t *testing.T) {
	l, st := newLog()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, _ = l.Write(ctx, Event{Actor: "usr_1", Kind: "access", Subject: "s", Reason: "r"})
	}
	s := signer(t, 1)
	var full bytes.Buffer
	if n, err := Export(ctx, st, s, 1, t0, &full); err != nil || n != 5 {
		t.Fatal(n, err)
	}
	rep, err := Verify(bytes.NewReader(full.Bytes()), "acme-logistics", s.Key())
	if err != nil || rep.Rows != 5 || rep.Anchors != 2 || !rep.KeyPinned {
		t.Fatalf("%+v %v", rep, err)
	}
	if rep, err := Verify(bytes.NewReader(full.Bytes()), "acme-logistics", ""); err != nil || rep.KeyPinned {
		t.Fatalf("unpinned: %+v %v", rep, err)
	}
	if _, err := Verify(bytes.NewReader(full.Bytes()), "acme-logistics", signer(t, 2).Key()); !errors.Is(err, ErrChain) {
		t.Error("a different pinned key verified")
	}
	if _, err := Verify(bytes.NewReader(full.Bytes()), "beta-freight", s.Key()); !errors.Is(err, ErrChain) {
		t.Error("another tenant's export verified")
	}
	var part bytes.Buffer
	if _, err := Export(ctx, st, s, 3, t0, &part); err != nil {
		t.Fatal(err)
	}
	if rep, err := Verify(bytes.NewReader(part.Bytes()), "acme-logistics", s.Key()); err != nil || rep.FirstSeq != 3 || rep.Rows != 3 {
		t.Fatalf("partial: %+v %v", rep, err)
	}
	// Dropping the first row of a partial export breaks the link to its start anchor.
	lines := strings.SplitN(part.String(), "\n", 3)
	if _, err := Verify(strings.NewReader(lines[0]+"\n"+lines[2]), "acme-logistics", s.Key()); !errors.Is(err, ErrChain) {
		t.Error("a dropped first row went unnoticed")
	}
	if _, err := Verify(strings.NewReader(""), "acme-logistics", s.Key()); !errors.Is(err, ErrChain) {
		t.Error("an empty export verified")
	}
}

// SB-1008: a row's space is hashed, company is never recorded, and InSpace filters.
func TestSpaceField(t *testing.T) {
	base := Row{Seq: 1, ID: "aud_000000000001", At: "2026-11-20T09:00:00.000Z", Tenant: "acme-logistics", Actor: "usr_1",
		Kind: "decision", Subject: "Chunk::\"c\"", Release: "rel_x", Reason: "read", Prev: Zero}
	plain, _ := HashOf(base)
	in := base
	in.Space = "spc_finance"
	h, _ := HashOf(in)
	if plain == h {
		t.Fatal("the space must be part of the hash")
	}
	if !InSpace(in, "finance") || InSpace(base, "finance") || !InSpace(Row{Kind: "kill", Subject: "space:finance"}, "finance") {
		t.Fatal("InSpace")
	}
}
