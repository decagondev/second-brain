package attest

import (
	"bytes"
	"crypto/ed25519"
	"testing"
	"time"

	"example.com/second-brain/internal/core/release"
)

var t0 = time.Date(2027, 3, 2, 10, 0, 0, 0, time.UTC)

func setup() (*Checker, ed25519.PrivateKey) {
	priv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{3}, 32))
	m := release.Manifest{ID: "rel_2027-03-02_aaaaaa", Bundle: release.Bundle{Code: map[string]string{"sb": release.SHA256([]byte("sb")), "sbom": release.SHA256([]byte("sbom"))}}}
	c := &Checker{Keys: map[string]ed25519.PublicKey{"acme-logistics": priv.Public().(ed25519.PublicKey)},
		Deployed: func(string) string { return m.ID },
		Bundles:  func(id string) (release.Manifest, bool) { return m, id == m.ID }}
	return c, priv
}

func report(seq int64, rel string, digests map[string]string) Report {
	return Report{Tenant: "acme-logistics", Release: rel, Digests: digests, Seq: seq, At: t0.Add(time.Duration(seq) * time.Minute)}
}

func TestAccept(t *testing.T) {
	c, priv := setup()
	good := map[string]string{"sb": release.SHA256([]byte("sb"))}
	s, _ := Sign(priv, report(1, "rel_2027-03-02_aaaaaa", good))
	if f := c.Accept(s, t0); f != nil {
		t.Fatal(f)
	}
	other := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{4}, 32))
	cases := map[string]struct {
		s    Signed
		kind string
	}{
		"replay":        {s, "attestation_replay"},
		"wrong release": {must(Sign(priv, report(2, "rel_2027-01-01_bbbbbb", good))), "attestation_mismatch"},
		"wrong digest":  {must(Sign(priv, report(3, "rel_2027-03-02_aaaaaa", map[string]string{"sb": release.SHA256([]byte("evil"))}))), "attestation_mismatch"},
		"extra binary":  {must(Sign(priv, report(4, "rel_2027-03-02_aaaaaa", map[string]string{"miner": release.SHA256([]byte("x"))}))), "attestation_mismatch"},
		"forged":        {must(Sign(other, report(5, "rel_2027-03-02_aaaaaa", good))), "attestation_mismatch"},
	}
	for name, cs := range cases {
		f := c.Accept(cs.s, t0)
		if len(f) == 0 || f[0].Kind != cs.kind {
			t.Errorf("%s: %+v", name, f)
		}
	}
	tampered := must(Sign(priv, report(6, "rel_2027-03-02_aaaaaa", good)))
	tampered.Digests = map[string]string{"sb": release.SHA256([]byte("evil"))}
	if f := c.Accept(tampered, t0); len(f) == 0 {
		t.Error("a report edited after signing was accepted")
	}
	if r, ok := c.Attested("acme-logistics"); !ok || r.Seq != 1 {
		t.Fatalf("attested %+v", r)
	}
}

func TestSilence(t *testing.T) {
	c, priv := setup()
	if f := c.Silent(t0); len(f) != 1 || f[0].Kind != "attestation_missing" {
		t.Fatalf("never attested: %+v", f)
	}
	s, _ := Sign(priv, report(1, "rel_2027-03-02_aaaaaa", map[string]string{"sb": release.SHA256([]byte("sb"))}))
	_ = c.Accept(s, t0)
	if f := c.Silent(t0.Add(time.Hour)); len(f) != 0 {
		t.Fatalf("an hour is fine: %+v", f)
	}
	if f := c.Silent(t0.Add(MaxSilence + time.Minute)); len(f) != 1 {
		t.Fatalf("silent past 2h: %+v", f)
	}
}

func must(s Signed, err error) Signed {
	if err != nil {
		panic(err)
	}
	return s
}
