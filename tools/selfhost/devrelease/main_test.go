package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/core/release"
)

// A demo release is admitted by a cell that trusts the demo key, and by no other.
func TestDemoReleaseIsAdmittedOnlyWithItsKey(t *testing.T) {
	dir := t.TempDir()
	b := release.Bundle{Code: map[string]string{"image/api": strings.Repeat("a", 64)}, Prompts: map[string]string{"answer": release.SHA256([]byte("p"))},
		Policies: map[string]string{"policy/platform/base.cedar": release.SHA256([]byte("b"))}, Model: release.Model{Provider: "openaicompat", Model: "m", MaxTokens: 1},
		Gate: release.SHA256([]byte("g")), Created: "2027-03-02"}
	m, err := b.Seal()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(m)
	bp, rp, kp := filepath.Join(dir, "bundle.json"), filepath.Join(dir, "results.txt"), filepath.Join(dir, "key")
	_ = os.WriteFile(bp, raw, 0o600)
	_ = os.WriteFile(rp, []byte("make check: all green"), 0o600)
	store := filepath.Join(dir, "store")
	if err := run(bp, store, kp, rp, "stable"); err != nil {
		t.Fatal(err)
	}
	k, err := key(kp)
	if err != nil {
		t.Fatal(err)
	}
	st := relfs.Dir{Root: store}
	id, err := st.Channel(context.Background(), "stable")
	if err != nil || id != m.ID {
		t.Fatalf("channel %q %v", id, err)
	}
	if _, _, err := release.Admit(context.Background(), st, id, release.Ed25519Verifier{Trusted: k.Public()}, nil); err != nil {
		t.Fatalf("not admitted with its own key: %v", err)
	}
	other := release.Ed25519Verifier{Trusted: "ed25519:" + strings.Repeat("A", 43) + "="}
	if _, _, err := release.Admit(context.Background(), st, id, other, nil); err == nil {
		t.Fatal("admitted by a cell that trusts another key")
	}
}
