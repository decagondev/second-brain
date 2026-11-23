package cli

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/core/release"
)

func hx(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// image writes a one-layer OCI image archive and returns its id.
func image(t *testing.T, path, layerText string) string {
	t.Helper()
	layer := []byte(layerText)
	config := []byte(`{"rootfs":{"diff_ids":["sha256:` + hx(layer) + `"]}}`)
	man := []byte(`{"config":{"digest":"sha256:` + hx(config) + `"},"layers":[{"digest":"sha256:` + hx(layer) + `"}]}`)
	id := hx(man)
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	for _, f := range []struct {
		n string
		b []byte
	}{{"blobs/sha256/" + hx(layer), layer}, {"blobs/sha256/" + hx(config), config}, {"blobs/sha256/" + id, man},
		{"index.json", []byte(`{"manifests":[{"digest":"sha256:` + id + `"}]}`)}} {
		_ = w.WriteHeader(&tar.Header{Name: f.n, Mode: 0o644, Size: int64(len(f.b)), Typeflag: tar.TypeReg})
		_, _ = w.Write(f.b)
	}
	_ = w.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return id
}

// offlineBundle lays out an unpacked bundle: images/ and a release signed by k naming them.
func offlineBundle(t *testing.T, k release.Ed25519) (string, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "images"), 0o755)
	ids := map[string]string{
		"api":    image(t, filepath.Join(dir, "images", "01-sb-api.tar"), "api binary"),
		"ingest": image(t, filepath.Join(dir, "images", "02-sb-ingest.tar"), "ingest binary"),
	}
	h := func(s string) string { return hx([]byte(s)) }
	b := release.Bundle{
		Code:     map[string]string{"image/api": ids["api"], "image/ingest": ids["ingest"]},
		Prompts:  map[string]string{"answer": h("p")},
		Policies: map[string]string{"policy/platform/base.cedar": h("b")},
		Model:    release.Model{Provider: "openaicompat", Model: "m", MaxTokens: 1},
		Gate:     h("g"), Created: "2027-03-02",
	}
	m, err := b.Seal()
	if err != nil {
		t.Fatal(err)
	}
	st := relfs.Dir{Root: filepath.Join(dir, "releases")}
	put := func(name string, v any) {
		raw, _ := json.Marshal(v)
		if err := st.Put(m.ID, name, raw); err != nil {
			t.Fatal(err)
		}
	}
	put("bundle.json", m)
	d, _ := m.Digest()
	s, _ := k.Sign(context.Background(), d)
	put("bundle.sig", s)
	g := release.GateRecord{Release: m.ID, Passed: true, ResultsSHA256: h("r")}
	gd, _ := g.Digest()
	gs, _ := k.Sign(context.Background(), gd)
	put("gate.json", g)
	put("gate.sig", gs)
	if err := st.SetChannel("stable", m.ID); err != nil {
		t.Fatal(err)
	}
	return dir, ids
}

func TestSelfhostVerifyBundle(t *testing.T) {
	k := release.Ed25519{Priv: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, 32))}
	other := release.Ed25519{Priv: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{6}, 32))}
	dir, _ := offlineBundle(t, k)
	var out, errb bytes.Buffer
	if code := Run([]string{"selfhost", "verify-bundle", dir, "--trusted", k.Public()}, &out, &errb); code != 0 {
		t.Fatalf("a good bundle: %d %s", code, errb.String())
	}
	if got := strings.Fields(out.String()); len(got) != 2 || !strings.HasSuffix(got[0], "01-sb-api.tar") {
		t.Fatalf("archives to load: %q", got)
	}
	refused := func(name string, args ...string) {
		t.Helper()
		var o, e bytes.Buffer
		if code := Run(append([]string{"selfhost", "verify-bundle"}, args...), &o, &e); code == 0 || o.Len() != 0 || !strings.Contains(e.String(), "EUNSIGNED") {
			t.Fatalf("%s: %d, out %q, err %q", name, code, o.String(), e.String())
		}
	}
	refused("another signer", dir, "--trusted", other.Public())
	// An image changed after the release was signed.
	dir2, _ := offlineBundle(t, k)
	image(t, filepath.Join(dir2, "images", "02-sb-ingest.tar"), "evil binary")
	refused("a changed image", dir2, "--trusted", k.Public())
	// No release at all.
	dir3, _ := offlineBundle(t, k)
	_ = os.RemoveAll(filepath.Join(dir3, "releases"))
	refused("no release", dir3, "--trusted", k.Public())
	var o, e bytes.Buffer
	if code := Run([]string{"selfhost", "verify-bundle", dir}, &o, &e); code != 2 {
		t.Fatalf("no key: %d", code)
	}
}
