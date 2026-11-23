package release

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

type tarFile struct {
	name string
	body []byte
	typ  byte
}

func imageTar(t *testing.T, files []tarFile) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	for _, f := range files {
		typ := f.typ
		if typ == 0 {
			typ = tar.TypeReg
		}
		h := &tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.body)), Typeflag: typ}
		if typ != tar.TypeReg {
			h.Size, h.Linkname = 0, "/etc/passwd"
		}
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if typ == tar.TypeReg {
			_, _ = w.Write(f.body)
		}
	}
	_ = w.Close()
	return &buf
}

// goodImage is a one-layer image in docker save's OCI layout, and its id.
func goodImage() ([]tarFile, string) {
	layer := []byte("layer bytes")
	config := []byte(`{"rootfs":{"diff_ids":["sha256:` + digest(layer) + `"]}}`)
	manifest := []byte(fmt.Sprintf(`{"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":"sha256:%s"},"layers":[{"digest":"sha256:%s"}]}`, digest(config), digest(layer)))
	id := digest(manifest)
	index := []byte(`{"manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:` + id + `"}]}`)
	legacy := []byte(`[{"Config":"blobs/sha256/` + digest(config) + `","RepoTags":["sb-api:local"],"Layers":["blobs/sha256/` + digest(layer) + `"]}]`)
	return []tarFile{
		{name: "blobs/", typ: tar.TypeDir}, {name: "blobs/sha256/", typ: tar.TypeDir},
		{name: "blobs/sha256/" + digest(layer), body: layer},
		{name: "blobs/sha256/" + digest(config), body: config},
		{name: "blobs/sha256/" + id, body: manifest},
		{name: "index.json", body: index},
		{name: "manifest.json", body: legacy},
		{name: "oci-layout", body: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
	}, id
}

func TestVerifyImageTar(t *testing.T) {
	files, id := goodImage()
	if err := VerifyImageTar(imageTar(t, files), id); err != nil {
		t.Fatalf("the release's own image refused: %v", err)
	}
	bad := map[string]func([]tarFile) []tarFile{
		"a changed layer": func(f []tarFile) []tarFile { f[2].body = []byte("evil bytes!"); return f },
		"a missing layer": func(f []tarFile) []tarFile { return append(f[:2:2], f[3:]...) },
		"another image": func(f []tarFile) []tarFile {
			other := []byte(`{"config":{"digest":"sha256:` + digest(f[3].body) + `"}}`)
			f = append(f, tarFile{name: "blobs/sha256/" + digest(other), body: other})
			f[5].body = []byte(`{"manifests":[{"digest":"sha256:` + digest(other) + `"}]}`)
			return f
		},
		"a second image in the index": func(f []tarFile) []tarFile {
			f[5].body = []byte(`{"manifests":[{"digest":"sha256:` + id + `"},{"digest":"sha256:` + digest(f[3].body) + `"}]}`)
			return f
		},
		"no index": func(f []tarFile) []tarFile { return append(f[:5:5], f[6:]...) },
		"a symlink": func(f []tarFile) []tarFile {
			return append(f, tarFile{name: "blobs/sha256/" + strings.Repeat("a", 64), typ: tar.TypeSymlink})
		},
		"a path outside the layout": func(f []tarFile) []tarFile { return append(f, tarFile{name: "../../etc/cron.d/x", body: []byte("x")}) },
		"manifest.json naming another blob": func(f []tarFile) []tarFile {
			x := []byte("x")
			f = append(f, tarFile{name: "blobs/sha256/" + digest(x), body: x})
			f[6].body = []byte(`[{"Config":"blobs/sha256/` + digest(x) + `","Layers":[]}]`)
			return f
		},
	}
	for name, mut := range bad {
		f, _ := goodImage()
		err := VerifyImageTar(imageTar(t, mut(f)), id)
		if err == nil || !errors.Is(err, ErrUnsigned) {
			t.Errorf("%s: %v, want EUNSIGNED", name, err)
		}
	}
	if err := VerifyImageTar(imageTar(t, files), "not-an-id"); err == nil {
		t.Fatal("a bad id")
	}
}
