package laws

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/core/release"
)

// LAW-06 for offline upgrades (SB-1804): an image archive is loaded only if it is exactly
// the image the admitted release names. Any byte changed in any file of the archive, or
// an archive of another image, is refused before Docker sees it.

type imageCheck func(r *bytes.Reader, id string) error

func lawHex(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// lawImage is a random image's files in docker save's layout, and its id.
func lawImage(r *rand.Rand) ([][2][]byte, string) {
	var files [][2][]byte
	var layers []string
	for i := 0; i <= r.Intn(3); i++ {
		l := make([]byte, 16+r.Intn(64))
		r.Read(l)
		files = append(files, [2][]byte{[]byte("blobs/sha256/" + lawHex(l)), l})
		layers = append(layers, `{"digest":"sha256:`+lawHex(l)+`"}`)
	}
	cfg := []byte(fmt.Sprintf(`{"n":%d}`, r.Int()))
	files = append(files, [2][]byte{[]byte("blobs/sha256/" + lawHex(cfg)), cfg})
	man := []byte(`{"config":{"digest":"sha256:` + lawHex(cfg) + `"},"layers":[` + join(layers) + `]}`)
	id := lawHex(man)
	files = append(files, [2][]byte{[]byte("blobs/sha256/" + id), man},
		[2][]byte{[]byte("index.json"), []byte(`{"manifests":[{"digest":"sha256:` + id + `"}]}`)})
	return files, id
}

func join(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ","
		}
		out += x
	}
	return out
}

func lawTar(files [][2][]byte) *bytes.Reader {
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	for _, f := range files {
		_ = w.WriteHeader(&tar.Header{Name: string(f[0]), Mode: 0o644, Size: int64(len(f[1])), Typeflag: tar.TypeReg})
		_, _ = w.Write(f[1])
	}
	_ = w.Close()
	return bytes.NewReader(buf.Bytes())
}

func law06Bundle(r *rand.Rand, check imageCheck) error {
	files, id := lawImage(r)
	if err := check(lawTar(files), id); err != nil {
		return fmt.Errorf("the release's own image was refused: %v", err)
	}
	// One byte of one file changed.
	i := r.Intn(len(files))
	f := append([]byte(nil), files[i][1]...)
	j := r.Intn(len(f))
	f[j] ^= byte(1 + r.Intn(255))
	bad := append([][2][]byte(nil), files...)
	bad[i] = [2][]byte{files[i][0], f}
	if check(lawTar(bad), id) == nil {
		return fmt.Errorf("an archive with byte %d of %s changed was accepted", j, files[i][0])
	}
	// Another image entirely.
	other, _ := lawImage(r)
	if check(lawTar(other), id) == nil {
		return fmt.Errorf("another image's archive was accepted as this one")
	}
	return nil
}

func realImageCheck(r *bytes.Reader, id string) error { return release.VerifyImageTar(r, id) }
