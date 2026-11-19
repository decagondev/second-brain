package release

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"example.com/second-brain/internal/core/canonjson"
)

// Signature is a release or gate signature (sb-release(7), Signed).
type Signature struct {
	Kind  string `json:"kind"`  // "ed25519" (development) or "cosign" (CI, keyless)
	Sig   string `json:"sig"`   // base64
	Key   string `json:"key"`   // ed25519: "ed25519:<base64 public key>"; cosign: the certificate identity
	Proof string `json:"proof"` // cosign: the sigstore bundle (certificate and Rekor entry), base64
}

// Signer signs a digest (hex sha256).
type Signer interface {
	Sign(ctx context.Context, digest string) (Signature, error)
}

// Verifier checks a signature over a digest for the identity a cell trusts.
type Verifier interface {
	Verify(ctx context.Context, digest string, s Signature) error
}

// Ed25519 is the development signer and verifier.
type Ed25519 struct{ Priv ed25519.PrivateKey }

// Public is the trusted key for verifiers.
func (e Ed25519) Public() string {
	return "ed25519:" + base64.StdEncoding.EncodeToString(e.Priv.Public().(ed25519.PublicKey))
}

func message(digest string) []byte { return []byte("sb-release:v1:" + digest) }

// Sign implements Signer.
func (e Ed25519) Sign(_ context.Context, digest string) (Signature, error) {
	return Signature{Kind: "ed25519", Sig: base64.StdEncoding.EncodeToString(ed25519.Sign(e.Priv, message(digest))), Key: e.Public()}, nil
}

// Ed25519Verifier trusts exactly one public key.
type Ed25519Verifier struct{ Trusted string }

// Verify implements Verifier. The signature's own key must be the trusted one: a
// signature that brings its own key proves nothing.
func (v Ed25519Verifier) Verify(_ context.Context, digest string, s Signature) error {
	if s.Kind != "ed25519" || s.Key != v.Trusted {
		return fmt.Errorf("%w: not signed by the trusted release key", ErrUnsigned)
	}
	raw, err := base64.StdEncoding.DecodeString(s.Key[len("ed25519:"):])
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: bad key", ErrUnsigned)
	}
	sig, err := base64.StdEncoding.DecodeString(s.Sig)
	if err != nil || !ed25519.Verify(raw, message(digest), sig) {
		return fmt.Errorf("%w: the signature doesn't verify", ErrUnsigned)
	}
	return nil
}

// GateRecord is the eval gate's verdict for one release (sb-release(7), Gated).
type GateRecord struct {
	Release       string `json:"release"`
	Passed        bool   `json:"passed"`
	ResultsSHA256 string `json:"results_sha256"`
}

// Digest is the sha256 of the record's canonical JSON.
func (g GateRecord) Digest() (string, error) {
	raw, err := json.Marshal(g)
	if err != nil {
		return "", err
	}
	c, err := canonjson.Encode(raw)
	if err != nil {
		return "", err
	}
	return SHA256(c), nil
}

// Store is the release store a cell pulls from (cell-initiated; the factory never
// connects into a cell).
type Store interface {
	Get(ctx context.Context, release, name string) ([]byte, error)
}

// ErrUnsigned is any admission failure (sb-errors(7) EUNSIGNED).
var ErrUnsigned = errors.New("release: EUNSIGNED")

// Admit pulls a release and checks it in order: id matches contents, bundle signature,
// every artefact's digest, and a signed passing gate record for this id. It returns the
// manifest and the artefacts' bytes only if all pass (sb-release(7), What a cell admits).
func Admit(ctx context.Context, st Store, id string, v Verifier, run []string) (Manifest, map[string][]byte, error) {
	var m Manifest
	raw, err := st.Get(ctx, id, "bundle.json")
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("%w: no bundle: %v", ErrUnsigned, err)
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, nil, fmt.Errorf("%w: bundle.json isn't valid", ErrUnsigned)
	}
	if m.ID != id {
		return Manifest{}, nil, fmt.Errorf("%w: asked for %s, the store has %s", ErrUnsigned, id, m.ID)
	}
	if err := m.Check(); err != nil { // 1
		return Manifest{}, nil, fmt.Errorf("%w: %v", ErrUnsigned, err)
	}
	digest, err := m.Digest()
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("%w: %v", ErrUnsigned, err)
	}
	if err := verifyFile(ctx, st, id, "bundle.sig", digest, v); err != nil { // 2
		return Manifest{}, nil, err
	}
	out := map[string][]byte{}
	for _, name := range run { // 3
		want, listed := m.Code[name]
		if !listed {
			return Manifest{}, nil, fmt.Errorf("%w: artefact %q isn't in the bundle", ErrUnsigned, name)
		}
		data, err := st.Get(ctx, id, "artefacts/"+name)
		if err != nil || SHA256(data) != want {
			return Manifest{}, nil, fmt.Errorf("%w: artefact %q doesn't match its digest", ErrUnsigned, name)
		}
		out[name] = data
	}
	graw, err := st.Get(ctx, id, "gate.json") // 4
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("%w: no gate record", ErrUnsigned)
	}
	var g GateRecord
	if err := json.Unmarshal(graw, &g); err != nil || g.Release != id || !g.Passed {
		return Manifest{}, nil, fmt.Errorf("%w: the gate didn't pass this release", ErrUnsigned)
	}
	gd, err := g.Digest()
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("%w: %v", ErrUnsigned, err)
	}
	if err := verifyFile(ctx, st, id, "gate.sig", gd, v); err != nil {
		return Manifest{}, nil, err
	}
	return m, out, nil
}

func verifyFile(ctx context.Context, st Store, id, name, digest string, v Verifier) error {
	raw, err := st.Get(ctx, id, name)
	if err != nil {
		return fmt.Errorf("%w: no %s", ErrUnsigned, name)
	}
	var s Signature
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("%w: %s isn't valid", ErrUnsigned, name)
	}
	if err := v.Verify(ctx, digest, s); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
