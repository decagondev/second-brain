package box

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// RetireKind is a retirement statement's first field (sb-selfhost(7), Retiring a box;
// SB-1907).
const RetireKind = "sb-retire/v1"

// Retirement is the statement a retired box leaves: what was destroyed, when, the last
// audit anchor the cell signed, and what was kept, signed with a key made for this one
// statement whose private half is never written anywhere.
type Retirement struct {
	Kind      string   `json:"statement"`
	Tenant    string   `json:"tenant"`
	Host      string   `json:"host"`
	At        string   `json:"at"`
	Destroyed []string `json:"destroyed"`
	Kept      []string `json:"kept"`
	// LastAnchor is the cell's own signed anchor on its last audit row, from its export.
	LastAnchor audit.Anchor `json:"last_anchor"`
	AuditRows  int          `json:"audit_rows"`
	Key        string       `json:"key"` // ed25519:BASE64, the statement's own key
	Sig        string       `json:"sig"`
}

// ErrStatement is a statement that doesn't verify, or one that can't be made.
var ErrStatement = errors.New("retirement statement refused")

// LastAnchor verifies a cell's audit export (sb-audit(5), Export) and returns its last
// anchor and its row count. An export that doesn't verify, or ends without an anchor on
// its last row, is refused: the statement never names a hash the chain doesn't hold.
func LastAnchor(export []byte, tenant, cellKey string) (audit.Anchor, int, error) {
	rep, err := audit.Verify(bytes.NewReader(export), tenant, cellKey)
	if err != nil {
		return audit.Anchor{}, 0, fmt.Errorf("%w: the audit export doesn't verify: %v", ErrStatement, err)
	}
	var last audit.Anchor
	sc := bufio.NewScanner(bytes.NewReader(export))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		var a audit.Anchor
		if json.Unmarshal(sc.Bytes(), &a) == nil && a.Anchor {
			last = a
		}
	}
	if !last.Anchor || last.Seq != rep.LastSeq {
		return audit.Anchor{}, 0, fmt.Errorf("%w: the export's last row (seq %d) has no anchor", ErrStatement, rep.LastSeq)
	}
	return last, rep.Rows, nil
}

func (r Retirement) message() []byte {
	r.Sig = ""
	b, _ := json.Marshal(r)
	return append([]byte("sb-retire:v1\n"), b...)
}

// Sign fills in Kind, At, Key and Sig with a fresh Ed25519 key from rnd, and returns the
// statement. The private key is dropped when Sign returns.
func (r Retirement) Sign(rnd io.Reader, at time.Time) (Retirement, error) {
	if strings.TrimSpace(r.Tenant) == "" || len(r.Destroyed) == 0 || !r.LastAnchor.Anchor {
		return Retirement{}, fmt.Errorf("%w: a tenant, what was destroyed and the last anchor are required", ErrStatement)
	}
	pub, priv, err := ed25519.GenerateKey(rnd)
	if err != nil {
		return Retirement{}, fmt.Errorf("%w: %v", ErrStatement, err)
	}
	r.Kind = RetireKind
	r.At = at.UTC().Format(time.RFC3339)
	if r.Kept == nil {
		r.Kept = []string{}
	}
	r.Key = "ed25519:" + base64.StdEncoding.EncodeToString(pub)
	r.Sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, r.message()))
	return r, nil
}

// VerifyRetirement checks a statement against the key printed when it was made. A
// statement with any field changed, or checked under another key, is refused.
func VerifyRetirement(b []byte, key string) (Retirement, error) {
	var r Retirement
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil || r.Kind != RetireKind {
		return Retirement{}, fmt.Errorf("%w: not an %s statement", ErrStatement, RetireKind)
	}
	if key != r.Key {
		return Retirement{}, fmt.Errorf("%w: it was signed with %s, not the key given", ErrStatement, r.Key)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(key, "ed25519:"))
	sig, serr := base64.StdEncoding.DecodeString(r.Sig)
	if err != nil || serr != nil || len(raw) != ed25519.PublicKeySize || !strings.HasPrefix(key, "ed25519:") {
		return Retirement{}, fmt.Errorf("%w: the key or signature isn't readable", ErrStatement)
	}
	if !ed25519.Verify(raw, r.message(), sig) {
		return Retirement{}, fmt.Errorf("%w: the signature doesn't match the statement", ErrStatement)
	}
	return r, nil
}

// JSON is the statement as it is written: indented, newline-terminated.
func (r Retirement) JSON() []byte {
	b, _ := json.MarshalIndent(r, "", "  ")
	return append(b, '\n')
}
