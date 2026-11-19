package cosign

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/release"
)

const id = "https://github.com/decagondev/second-brain/.github/workflows/release.yml@refs/heads/main"

func TestArgsPinIdentityAndIssuer(t *testing.T) {
	got := strings.Join(Verifier{Identity: id}.Args("m", "b"), " ")
	want := "verify-blob --bundle b --certificate-identity " + id + " --certificate-oidc-issuer " + Issuer + " m"
	if got != want {
		t.Fatalf("%s", got)
	}
}

// A fake cosign: it records its arguments and the message, and succeeds or fails.
func fake(t *testing.T, ok bool) string {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell")
	}
	p := filepath.Join(t.TempDir(), "cosign")
	code := "0"
	if !ok {
		code = "1"
	}
	script := "#!/bin/sh\necho \"$@\" > \"$(dirname \"$0\")/args\"\nfor last; do :; done\ncat \"$last\" > \"$(dirname \"$0\")/msg\"\necho 'Error: none of the expected identities matched' >&2\nexit " + code + "\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestVerify(t *testing.T) {
	sig := release.Signature{Kind: "cosign", Key: id, Proof: base64.StdEncoding.EncodeToString([]byte(`{"mediaType":"x"}`))}
	bin := fake(t, true)
	if err := (Verifier{Identity: id, Cosign: bin}).Verify(context.Background(), "abc123", sig); err != nil {
		t.Fatal(err)
	}
	msg, _ := os.ReadFile(filepath.Join(filepath.Dir(bin), "msg"))
	if string(msg) != "sb-release:v1:abc123" {
		t.Fatalf("message %q", msg)
	}
	if err := (Verifier{Identity: id, Cosign: fake(t, false)}).Verify(context.Background(), "abc123", sig); !errors.Is(err, release.ErrUnsigned) {
		t.Fatalf("a failing cosign: %v", err)
	}
	if err := (Verifier{Identity: id, Cosign: "/nonexistent/cosign"}).Verify(context.Background(), "abc123", sig); !errors.Is(err, release.ErrUnsigned) {
		t.Fatalf("no cosign must refuse: %v", err)
	}
	other := sig
	other.Key = "https://github.com/someone/else/.github/workflows/x.yml@refs/heads/main"
	if err := (Verifier{Identity: id, Cosign: bin}).Verify(context.Background(), "abc123", other); !errors.Is(err, release.ErrUnsigned) {
		t.Fatalf("another identity: %v", err)
	}
}
