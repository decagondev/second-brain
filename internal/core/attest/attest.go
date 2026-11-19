// Package attest is running-build attestation (sb-release(7), Attestation; SB-509): cells
// push signed reports of what they run, and the controller compares them with the bundle
// it deployed. Mismatches, replays and silence are Sev-1.
package attest

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"example.com/second-brain/internal/core/canonjson"
	"example.com/second-brain/internal/core/release"
)

// MaxSilence is how long a cell may go without an accepted attestation.
const MaxSilence = 2 * time.Hour

// Report is what a cell attests.
type Report struct {
	Tenant  string            `json:"tenant"`
	Release string            `json:"release"`
	Digests map[string]string `json:"digests"`
	Seq     int64             `json:"seq"`
	At      time.Time         `json:"at"`
}

// Signed is a report with the cell's signature.
type Signed struct {
	Report
	Sig string `json:"sig"` // base64 Ed25519 over sb-attest:v1:<sha256 of canonical report>
}

func digest(r Report) (string, error) {
	raw, err := json.Marshal(struct {
		Tenant  string            `json:"tenant"`
		Release string            `json:"release"`
		Digests map[string]string `json:"digests"`
		Seq     int64             `json:"seq"`
		At      string            `json:"at"`
	}{r.Tenant, r.Release, r.Digests, r.Seq, r.At.UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return "", err
	}
	c, err := canonjson.Encode(raw)
	if err != nil {
		return "", err
	}
	return release.SHA256(c), nil
}

// Sign is the cell's side.
func Sign(priv ed25519.PrivateKey, r Report) (Signed, error) {
	d, err := digest(r)
	if err != nil {
		return Signed{}, err
	}
	return Signed{Report: r, Sig: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("sb-attest:v1:"+d)))}, nil
}

// Finding is one problem, named by kind; it is a Sev-1 alert.
type Finding struct {
	Tenant string
	Kind   string // attestation_mismatch, attestation_replay, attestation_missing
	Detail string // names artefacts and releases, never content
}

// Bundles looks up a release's manifest.
type Bundles func(release string) (release.Manifest, bool)

// Checker is the controller's side.
type Checker struct {
	Keys     map[string]ed25519.PublicKey // tenant -> the cell's attestation key
	Deployed func(tenant string) string   // the release the controller deployed there
	Bundles  Bundles

	mu       sync.Mutex
	lastSeq  map[string]int64
	lastSeen map[string]time.Time
	attested map[string]Report
}

// Accept checks one pushed attestation; nil findings means it was accepted.
func (c *Checker) Accept(s Signed, now time.Time) []Finding {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lastSeq == nil {
		c.lastSeq, c.lastSeen, c.attested = map[string]int64{}, map[string]time.Time{}, map[string]Report{}
	}
	bad := func(kind, f string, a ...any) []Finding {
		return []Finding{{Tenant: s.Tenant, Kind: kind, Detail: fmt.Sprintf(f, a...)}}
	}
	key, ok := c.Keys[s.Tenant]
	if !ok {
		return bad("attestation_mismatch", "no attestation key for this cell")
	}
	d, err := digest(s.Report)
	sig, err2 := base64.StdEncoding.DecodeString(s.Sig)
	if err != nil || err2 != nil || !ed25519.Verify(key, []byte("sb-attest:v1:"+d), sig) {
		return bad("attestation_mismatch", "signature doesn't verify for this cell's key")
	}
	if s.Seq <= c.lastSeq[s.Tenant] {
		return bad("attestation_replay", "seq %d after %d", s.Seq, c.lastSeq[s.Tenant])
	}
	if want := c.Deployed(s.Tenant); s.Release != want {
		return bad("attestation_mismatch", "runs %s, deployed %s", s.Release, want)
	}
	m, ok := c.Bundles(s.Release)
	if !ok {
		return bad("attestation_mismatch", "no bundle for %s", s.Release)
	}
	var problems []Finding
	names := make([]string, 0, len(s.Digests))
	for n := range s.Digests {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		want, listed := m.Code[n]
		switch {
		case !listed:
			problems = append(problems, Finding{s.Tenant, "attestation_mismatch", "artefact " + n + " isn't in the bundle"})
		case want != s.Digests[n]:
			problems = append(problems, Finding{s.Tenant, "attestation_mismatch", "artefact " + n + " doesn't match the bundle's digest"})
		}
	}
	if len(problems) > 0 {
		return problems
	}
	c.lastSeq[s.Tenant], c.lastSeen[s.Tenant], c.attested[s.Tenant] = s.Seq, now, s.Report
	return nil
}

// Silent lists cells with no accepted attestation for MaxSilence (run hourly).
func (c *Checker) Silent(now time.Time) []Finding {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []Finding
	names := make([]string, 0, len(c.Keys))
	for t := range c.Keys {
		names = append(names, t)
	}
	sort.Strings(names)
	for _, t := range names {
		seen, ok := c.lastSeen[t]
		if !ok || now.Sub(seen) > MaxSilence {
			out = append(out, Finding{Tenant: t, Kind: "attestation_missing", Detail: "no accepted attestation for 2h"})
		}
	}
	return out
}

// Attested is the last accepted report for a tenant (sb release status).
func (c *Checker) Attested(tenant string) (Report, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r, ok := c.attested[tenant]
	return r, ok
}
