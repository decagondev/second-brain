package release

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// ErrImageTar is an image archive that isn't exactly the image a release names (sb-errors(7)
// EUNSIGNED): checked before `docker load` ever sees it (sb-selfhost(7), Offline upgrades;
// SB-1804).
var ErrImageTar = fmt.Errorf("%w: image archive", ErrUnsigned)

// MaxImageTar bounds what an archive may hold, so a hostile one can't fill the disk.
const MaxImageTar = 8 << 30

type ociIndex struct {
	Manifests []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
	} `json:"manifests"`
}

type ociManifest struct {
	MediaType string `json:"mediaType"`
	Config    struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		Digest string `json:"digest"`
	} `json:"layers"`
	Manifests []struct {
		Digest string `json:"digest"`
	} `json:"manifests"` // an image index
}

type dockerManifest []struct {
	Config string   `json:"Config"`
	Layers []string `json:"Layers"`
}

// VerifyImageTar checks that an image archive written by `docker save` (an OCI layout) is
// the image whose id is id (hex, the manifest digest the release records as image/NAME)
// and nothing else: every blob's content hashes to its name, index.json names exactly that
// one image, and every blob the image reaches is present. A layer, config or manifest
// that was changed, an extra image smuggled into the index, or a file outside the layout
// is refused.
func VerifyImageTar(r io.Reader, id string) error {
	if !hexRe.MatchString(id) {
		return fmt.Errorf("%w: %q isn't an image id", ErrImageTar, id)
	}
	blobs := map[string][]byte{} // small blobs (manifests, configs, indexes) by digest
	have := map[string]bool{}
	var index, legacy []byte
	tr := tar.NewReader(r)
	var total int64
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrImageTar, err)
		}
		name := path.Clean(h.Name)
		if h.Typeflag == tar.TypeDir {
			if name != "blobs" && name != "blobs/sha256" {
				return fmt.Errorf("%w: unexpected directory %q", ErrImageTar, h.Name)
			}
			continue
		}
		if h.Typeflag != tar.TypeReg {
			return fmt.Errorf("%w: %q isn't a regular file", ErrImageTar, h.Name)
		}
		total += h.Size
		if total > MaxImageTar {
			return fmt.Errorf("%w: larger than %d bytes", ErrImageTar, int64(MaxImageTar))
		}
		switch {
		case name == "index.json", name == "manifest.json", name == "oci-layout", name == "repositories":
			b, err := io.ReadAll(io.LimitReader(tr, 1<<20))
			if err != nil {
				return fmt.Errorf("%w: %v", ErrImageTar, err)
			}
			switch name {
			case "index.json":
				index = b
			case "manifest.json":
				legacy = b
			}
		case strings.HasPrefix(name, "blobs/sha256/"):
			want := strings.TrimPrefix(name, "blobs/sha256/")
			if !hexRe.MatchString(want) {
				return fmt.Errorf("%w: %q isn't named by its digest", ErrImageTar, h.Name)
			}
			hs := sha256.New()
			var keep []byte
			if h.Size <= 1<<20 { // manifests and configs are small; layers are streamed
				b, err := io.ReadAll(tr)
				if err != nil {
					return fmt.Errorf("%w: %v", ErrImageTar, err)
				}
				keep = b
				hs.Write(b)
			} else if _, err := io.Copy(hs, tr); err != nil {
				return fmt.Errorf("%w: %v", ErrImageTar, err)
			}
			if got := hex.EncodeToString(hs.Sum(nil)); got != want {
				return fmt.Errorf("%w: blob %s… doesn't match its content", ErrImageTar, want[:12])
			}
			have[want] = true
			if keep != nil {
				blobs[want] = keep
			}
		default:
			return fmt.Errorf("%w: unexpected file %q", ErrImageTar, h.Name)
		}
	}
	return checkImageLayout(id, index, legacy, blobs, have)
}

// checkImageLayout checks, after every blob's content was hashed, that index.json names
// exactly the release's image, that the image reaches only present blobs, and that
// docker's manifest.json names nothing else.
func checkImageLayout(id string, index, legacy []byte, blobs map[string][]byte, have map[string]bool) error {
	// index.json names exactly the release's image.
	var ix ociIndex
	if err := json.Unmarshal(index, &ix); err != nil || len(ix.Manifests) != 1 {
		return fmt.Errorf("%w: index.json must name exactly one image", ErrImageTar)
	}
	if ix.Manifests[0].Digest != "sha256:"+id {
		return fmt.Errorf("%w: the archive holds %s, not the release's image sha256:%s", ErrImageTar, ix.Manifests[0].Digest, id)
	}
	// Everything the image reaches is present (its content was checked above).
	reached := map[string]bool{}
	var walk func(d string, depth int) error
	walk = func(d string, depth int) error {
		h := strings.TrimPrefix(d, "sha256:")
		if !have[h] {
			return fmt.Errorf("%w: blob %s… is missing", ErrImageTar, h[:min(12, len(h))])
		}
		reached[h] = true
		b, small := blobs[h]
		if !small || depth > 2 {
			return nil
		}
		var m ociManifest
		if !json.Valid(b) || json.Unmarshal(b, &m) != nil {
			return nil //nolint:nilerr // a blob that isn't a manifest is a config, not an error
		}
		if m.Config.Digest != "" {
			if err := walk(m.Config.Digest, depth+1); err != nil {
				return err
			}
		}
		for _, l := range m.Layers {
			if err := walk(l.Digest, depth+1); err != nil {
				return err
			}
		}
		for _, sub := range m.Manifests {
			if err := walk(sub.Digest, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk("sha256:"+id, 0); err != nil {
		return err
	}
	// docker's own manifest.json may name only blobs the image reaches.
	if legacy != nil {
		var dm dockerManifest
		if err := json.Unmarshal(legacy, &dm); err != nil || len(dm) != 1 {
			return fmt.Errorf("%w: manifest.json must name exactly one image", ErrImageTar)
		}
		for _, p := range append([]string{dm[0].Config}, dm[0].Layers...) {
			if h := strings.TrimPrefix(path.Clean(p), "blobs/sha256/"); !reached[h] {
				return fmt.Errorf("%w: manifest.json names %q, which the image doesn't", ErrImageTar, p)
			}
		}
	}
	return nil
}
