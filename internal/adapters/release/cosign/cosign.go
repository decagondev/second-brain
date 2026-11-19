// Package cosign verifies keyless release signatures made in CI (sb-release(7), Signed;
// SB-504, SB-506) by running `cosign verify-blob` against the sigstore bundle, pinned to
// one certificate identity and the GitHub Actions OIDC issuer.
package cosign

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"example.com/second-brain/internal/core/release"
)

// Issuer is GitHub Actions' OIDC issuer.
const Issuer = "https://token.actions.githubusercontent.com"

// Verifier trusts exactly one workflow identity, e.g.
// https://github.com/OWNER/REPO/.github/workflows/release.yml@refs/heads/main.
type Verifier struct {
	Identity string
	Cosign   string // the cosign binary; "" means "cosign" on PATH
}

var _ release.Verifier = Verifier{}

// Args is the verify-blob command line for a message file and a bundle file.
func (v Verifier) Args(msg, bundle string) []string {
	return []string{"verify-blob", "--bundle", bundle, "--certificate-identity", v.Identity, "--certificate-oidc-issuer", Issuer, msg}
}

// Verify implements release.Verifier.
func (v Verifier) Verify(ctx context.Context, digest string, s release.Signature) error {
	if s.Kind != "cosign" || s.Key != v.Identity || v.Identity == "" {
		return fmt.Errorf("%w: not signed by the trusted release workflow", release.ErrUnsigned)
	}
	proof, err := base64.StdEncoding.DecodeString(s.Proof)
	if err != nil || len(proof) == 0 {
		return fmt.Errorf("%w: no sigstore bundle", release.ErrUnsigned)
	}
	dir, err := os.MkdirTemp("", "sb-cosign-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	msg, bundle := filepath.Join(dir, "sign.txt"), filepath.Join(dir, "bundle.json")
	if err := os.WriteFile(msg, []byte("sb-release:v1:"+digest), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(bundle, proof, 0o600); err != nil {
		return err
	}
	bin := v.Cosign
	if bin == "" {
		bin = "cosign"
	}
	out, err := exec.CommandContext(ctx, bin, v.Args(msg, bundle)...).CombinedOutput()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return fmt.Errorf("%w: cosign: %s", release.ErrUnsigned, firstLine(out))
	case err != nil:
		return fmt.Errorf("%w: cosign unavailable: %v", release.ErrUnsigned, err) // no verifier, no start
	}
	return nil
}

func firstLine(b []byte) string {
	for i, c := range b {
		if c == '\n' {
			return string(b[:i])
		}
	}
	return string(b)
}
