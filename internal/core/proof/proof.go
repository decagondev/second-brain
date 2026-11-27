// Package proof is a ticket's proof on a real system (SB-6001; sb(1), sb prove): each
// acceptance criterion's pass or fail with counts, timings and hashes, signed with the
// cell's anchor key. A proof holds no document text, no name and no unhashed id: Check
// refuses one that does, and Seal refuses to sign it.
package proof

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/canonjson"
)

// Kind is the signed document kind (sb-audit(5), Signed documents).
const Kind = "proof"

// Criterion is one acceptance criterion's result. Name is fixed text from the runner, never
// from the system under test.
type Criterion struct {
	Name   string         `json:"name"`
	Pass   bool           `json:"pass"`
	Counts map[string]int `json:"counts,omitempty"`
	Seen   []string       `json:"seen,omitempty"` // hashed ids, at most MaxSeen
	Millis int64          `json:"millis"`
}

// MaxSeen is how many hashed ids a criterion keeps.
const MaxSeen = 20

// Proof is one run of one ticket's criteria.
type Proof struct {
	Ticket   string      `json:"ticket"`
	Tenant   string      `json:"tenant"`
	At       string      `json:"at"`
	API      string      `json:"api"` // the vendor's API version, such as graph-v1.0
	Criteria []Criterion `json:"criteria"`
	Passed   bool        `json:"passed"`
}

// Signed is a proof with its signature.
type Signed struct {
	Proof    Proof  `json:"proof"`
	SHA256   string `json:"sha256"`
	SignedAt string `json:"signed_at"`
	Sig      string `json:"sig"`
	Key      string `json:"key"`
}

var (
	ticketRe = regexp.MustCompile(`^SB-[0-9]{3,4}$`)
	tenantRe = regexp.MustCompile(`^[a-z0-9-]{1,63}$`)
	apiRe    = regexp.MustCompile(`^[a-z0-9.-]{1,40}$`)
	nameRe   = regexp.MustCompile(`^[a-z][a-z ,()'-]{2,99}$`)
	countRe  = regexp.MustCompile(`^[a-z_]{1,40}$`)
	hashRe   = regexp.MustCompile(`^h:[0-9a-f]{16}$`)
)

// ErrContent is a proof that holds something other than counts, hashes and fixed names.
var ErrContent = errors.New("proof: holds content")

// Check refuses a proof with any value that could be content: names are fixed lower-case
// words, counts are keyed by short words, and ids are hashes.
func Check(p Proof) error {
	bad := func(what string) error { return fmt.Errorf("%w: %s", ErrContent, what) }
	switch {
	case !ticketRe.MatchString(p.Ticket):
		return bad("ticket")
	case !tenantRe.MatchString(p.Tenant):
		return bad("tenant")
	case !apiRe.MatchString(p.API):
		return bad("api")
	case len(p.Criteria) == 0:
		return bad("no criteria")
	}
	if _, err := time.Parse(time.RFC3339, p.At); err != nil {
		return bad("at")
	}
	all := true
	for i, c := range p.Criteria {
		if !nameRe.MatchString(c.Name) {
			return bad(fmt.Sprintf("criterion %d name", i+1))
		}
		for k := range c.Counts {
			if !countRe.MatchString(k) {
				return bad(fmt.Sprintf("criterion %d count key", i+1))
			}
		}
		if len(c.Seen) > MaxSeen {
			return bad(fmt.Sprintf("criterion %d keeps too many ids", i+1))
		}
		for _, h := range c.Seen {
			if !hashRe.MatchString(h) {
				return bad(fmt.Sprintf("criterion %d id isn't hashed", i+1))
			}
		}
		all = all && c.Pass
	}
	if all != p.Passed {
		return bad("passed doesn't match its criteria")
	}
	return nil
}

// Hasher hashes ids with a key made for one proof and never kept, so a hash means the same
// id within the proof and nothing outside it.
type Hasher struct{ key []byte }

// NewHasher makes a hasher with a fresh key.
func NewHasher() Hasher {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	return Hasher{key: k}
}

// ID is id's hash.
func (h Hasher) ID(id string) string {
	m := hmac.New(sha256.New, h.key)
	m.Write([]byte(id))
	return "h:" + hex.EncodeToString(m.Sum(nil))[:16]
}

// IDs hashes up to MaxSeen ids, sorted.
func (h Hasher) IDs(ids []string) []string {
	var out []string
	for _, id := range ids {
		out = append(out, h.ID(id))
	}
	sort.Strings(out)
	if len(out) > MaxSeen {
		out = out[:MaxSeen]
	}
	return out
}

// New is a proof of ticket at now from criteria; Passed is whether every one passed.
func New(ticket, tenant, api string, now time.Time, cs []Criterion) Proof {
	p := Proof{Ticket: ticket, Tenant: tenant, API: api, At: now.UTC().Format(time.RFC3339), Criteria: cs, Passed: len(cs) > 0}
	for _, c := range cs {
		p.Passed = p.Passed && c.Pass
	}
	return p
}

func digest(p Proof) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	c, err := canonjson.EncodeDocument(raw)
	if err != nil {
		return "", err
	}
	s := sha256.Sum256(c)
	return hex.EncodeToString(s[:]), nil
}

// Seal checks p and signs it with sg.
func Seal(p Proof, sg audit.Signer, now time.Time) (Signed, error) {
	if err := Check(p); err != nil {
		return Signed{}, err
	}
	sha, err := digest(p)
	if err != nil {
		return Signed{}, err
	}
	at, sig, key := sg.SignDocument(Kind, sha, now)
	return Signed{Proof: p, SHA256: sha, SignedAt: at, Sig: sig, Key: key}, nil
}

// ErrSignature is a proof whose hash or signature doesn't verify.
var ErrSignature = errors.New("proof: the signature doesn't verify")

// Verify checks a proof file: its content rules, its hash, its signature, and its key
// against pinned when set.
func Verify(doc []byte, pinned string) (Signed, error) {
	var s Signed
	d := json.NewDecoder(strings.NewReader(string(doc)))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return Signed{}, fmt.Errorf("%w: %v", ErrSignature, err)
	}
	if err := Check(s.Proof); err != nil {
		return Signed{}, err
	}
	sha, err := digest(s.Proof)
	if err != nil || sha != s.SHA256 {
		return Signed{}, fmt.Errorf("%w: the hash doesn't match", ErrSignature)
	}
	if pinned != "" && s.Key != pinned {
		return Signed{}, fmt.Errorf("%w: signed by another key", ErrSignature)
	}
	if !audit.VerifyDocument(Kind, s.Proof.Tenant, s.Key, sha, s.SignedAt, s.Sig) {
		return Signed{}, ErrSignature
	}
	return s, nil
}

// FileName is where a proof is kept under state/proofs: TICKET-YYYYMMDD-HHMMSS.json.
func FileName(p Proof) string {
	t, _ := time.Parse(time.RFC3339, p.At)
	return p.Ticket + "-" + t.UTC().Format("20060102-150405") + ".json"
}
