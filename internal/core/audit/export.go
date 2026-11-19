package audit

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// AnchorEvery is how many rows an export holds between anchors.
const AnchorEvery = 10000

// Anchor is a signed statement that row Seq has hash Hash (sb-audit(5), Anchors).
type Anchor struct {
	Anchor   bool   `json:"anchor"`
	Seq      int64  `json:"seq"`
	Hash     string `json:"hash"`
	SignedAt string `json:"signed_at"`
	Sig      string `json:"sig"`
	Key      string `json:"key"`
}

// Signer signs anchors with the cell's own key.
type Signer struct {
	Tenant string
	priv   ed25519.PrivateKey
}

// NewSigner makes the cell's signer from its 32-byte seed. The seed is a data key wrapped
// by the cell's root key (ports.Keys); callers unwrap it, never a factory-wide key.
func NewSigner(tenant string, seed []byte) (Signer, error) {
	if len(seed) != ed25519.SeedSize {
		return Signer{}, fmt.Errorf("audit: anchor seed must be %d bytes", ed25519.SeedSize)
	}
	return Signer{Tenant: tenant, priv: ed25519.NewKeyFromSeed(seed)}, nil
}

// Key is the public key as written in anchors and custody records.
func (s Signer) Key() string {
	return "ed25519:" + base64.StdEncoding.EncodeToString(s.priv.Public().(ed25519.PublicKey))
}

func anchorMessage(tenant string, seq int64, hash, at string) []byte {
	return []byte("sb-audit-anchor:v1:" + tenant + ":" + strconv.FormatInt(seq, 10) + ":" + hash + ":" + at)
}

// Sign anchors row seq with hash.
func (s Signer) Sign(seq int64, hash string, at time.Time) Anchor {
	ts := at.UTC().Format("2006-01-02T15:04:05.000Z")
	sig := ed25519.Sign(s.priv, anchorMessage(s.Tenant, seq, hash, ts))
	return Anchor{Anchor: true, Seq: seq, Hash: hash, SignedAt: ts, Sig: base64.StdEncoding.EncodeToString(sig), Key: s.Key()}
}

// Export writes rows from fromSeq to the end as NDJSON (sb-audit(5), Export): first a
// signed anchor on the row before fromSeq (seq 0 and Zero for a full export), then the
// rows with an anchor every AnchorEvery rows, and an anchor on the last row.
func Export(ctx context.Context, st Store, s Signer, fromSeq int64, now time.Time, w io.Writer) (int, error) {
	if fromSeq < 1 {
		fromSeq = 1
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	var start Anchor
	if fromSeq == 1 {
		start = s.Sign(0, Zero, now)
	} else {
		prev, err := st.Range(ctx, fromSeq-1, 1)
		if err != nil {
			return 0, err
		}
		if len(prev) != 1 || prev[0].Seq != fromSeq-1 {
			return 0, fmt.Errorf("audit: no row %d to start the export from", fromSeq-1)
		}
		start = s.Sign(prev[0].Seq, prev[0].Hash, now)
	}
	if err := enc.Encode(start); err != nil {
		return 0, err
	}
	n, next := 0, fromSeq
	var last Row
	for {
		rows, err := st.Range(ctx, next, 1000)
		if err != nil {
			return n, err
		}
		for _, r := range rows {
			if err := enc.Encode(r); err != nil {
				return n, err
			}
			n++
			last = r
			if n%AnchorEvery == 0 {
				if err := enc.Encode(s.Sign(r.Seq, r.Hash, now)); err != nil {
					return n, err
				}
			}
		}
		if len(rows) < 1000 {
			break
		}
		next = rows[len(rows)-1].Seq + 1
	}
	if n > 0 && n%AnchorEvery != 0 {
		if err := enc.Encode(s.Sign(last.Seq, last.Hash, now)); err != nil {
			return n, err
		}
	}
	return n, nil
}

// ErrChain is a broken chain; the message names the first bad seq or line.
var ErrChain = errors.New("audit: ECHAIN")

// Report is what Verify found in an intact export.
type Report struct {
	Rows      int
	FirstSeq  int64
	LastSeq   int64
	Anchors   int
	Key       string
	KeyPinned bool
}

// Verify checks an export offline (sb-audit(1), verify --export). pinned is the cell's
// public key ("ed25519:…"), or "" to use the file's own key (reported as not pinned).
func Verify(r io.Reader, tenant, pinned string) (Report, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	var rep Report
	var prevSeq int64 = -1
	prevHash := ""
	lastAnchored, lastRowSeq := int64(-1), int64(-1)
	line := 0
	bad := func(format string, a ...any) (Report, error) {
		return Report{}, fmt.Errorf("%w: line %d: %s", ErrChain, line, fmt.Sprintf(format, a...))
	}
	for sc.Scan() {
		line++
		text := sc.Bytes()
		if len(strings.TrimSpace(string(text))) == 0 {
			continue
		}
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(text, &probe); err != nil {
			return bad("not JSON")
		}
		if _, isAnchor := probe["anchor"]; isAnchor {
			var a Anchor
			if err := strictDecode(text, &a); err != nil || !a.Anchor {
				return bad("malformed anchor")
			}
			key := pinned
			if key == "" {
				key = a.Key
			}
			if rep.Key == "" {
				rep.Key = a.Key
			} else if a.Key != rep.Key {
				return bad("anchors signed with different keys")
			}
			if !verifyAnchor(tenant, key, a) {
				return bad("anchor signature does not verify for seq %d", a.Seq)
			}
			if prevSeq == -1 { // the start anchor
				prevSeq, prevHash = a.Seq, a.Hash
				if a.Seq == 0 && a.Hash != Zero {
					return bad("start anchor for seq 0 must carry the zero hash")
				}
			} else if a.Seq != prevSeq || a.Hash != prevHash {
				return bad("anchor for seq %d doesn't match the row before it", a.Seq)
			}
			lastAnchored = a.Seq
			rep.Anchors++
			continue
		}
		if prevSeq == -1 {
			return bad("export must start with a signed anchor")
		}
		var row Row
		if err := strictDecode(text, &row); err != nil {
			return bad("malformed row: %v", err)
		}
		if row.Seq != prevSeq+1 {
			return bad("seq %d follows seq %d", row.Seq, prevSeq)
		}
		if row.Tenant != tenant {
			return bad("seq %d belongs to tenant %q", row.Seq, row.Tenant)
		}
		if row.Prev != prevHash {
			return bad("seq %d: prev doesn't match the hash of seq %d", row.Seq, prevSeq)
		}
		if row.ID != fmt.Sprintf("aud_%012d", row.Seq) || !Kinds[row.Kind] {
			return bad("seq %d: bad id or kind", row.Seq)
		}
		want, err := HashOf(row)
		if err != nil || want != row.Hash {
			return bad("seq %d: hash doesn't match its contents", row.Seq)
		}
		if rep.Rows == 0 {
			rep.FirstSeq = row.Seq
		}
		rep.Rows++
		prevSeq, prevHash, lastRowSeq = row.Seq, row.Hash, row.Seq
		rep.LastSeq = row.Seq
	}
	if err := sc.Err(); err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrChain, err)
	}
	if prevSeq == -1 {
		return Report{}, fmt.Errorf("%w: empty export", ErrChain)
	}
	if lastRowSeq != -1 && lastAnchored != lastRowSeq {
		return Report{}, fmt.Errorf("%w: the last row (seq %d) has no anchor: the export may be cut off", ErrChain, lastRowSeq)
	}
	rep.KeyPinned = pinned != ""
	return rep, nil
}

func verifyAnchor(tenant, key string, a Anchor) bool {
	raw, ok := strings.CutPrefix(key, "ed25519:")
	if !ok {
		return false
	}
	pub, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(a.Sig)
	if err != nil {
		return false
	}
	return ed25519.Verify(pub, anchorMessage(tenant, a.Seq, a.Hash, a.SignedAt), sig)
}

func strictDecode(b []byte, v any) error {
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
