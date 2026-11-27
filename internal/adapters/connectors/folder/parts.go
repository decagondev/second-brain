package folder

import (
	"archive/zip"
	"bytes"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// The parts of a file the folder connector can't read as text (sb-connector(3), Scans,
// images and recordings; SB-6101, SB-6102): an image file, a PDF with no text layer (each
// page a picture), the pictures in a .docx or .pptx (with their slide), and an audio or
// video file. Ingest has them read by the gated vision and transcribe tiers.

// MaxMediaSize is the largest recording read (an audio or video file).
const MaxMediaSize = 200 << 20

// MaxImageSize is the largest image read, alone or inside a document.
const MaxImageSize = 10 << 20

var imageMIME = map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp"}

var mediaMIME = map[string]string{".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".wav": "audio/wav", ".mp4": "video/mp4", ".webm": "video/webm"}

// IsImage and IsMedia say whether a file is an image or a recording, by its extension.
func IsImage(name string) bool { _, ok := imageMIME[strings.ToLower(path.Ext(name))]; return ok }
func IsMedia(name string) bool { _, ok := mediaMIME[strings.ToLower(path.Ext(name))]; return ok }

// Images are a file's pictures to be read: the file itself for an image, each picture of a
// PDF with no text layer as its pages in order, and the pictures in a .docx (page 0) or
// a .pptx (its slide).
func Images(name string, b []byte) []ports.Part {
	ext := strings.ToLower(path.Ext(name))
	base := path.Base(name)
	switch {
	case IsImage(name):
		if len(b) > MaxImageSize {
			return nil
		}
		return []ports.Part{{Name: base, MIME: imageMIME[ext], Data: b}}
	case ext == ".pdf":
		if strings.TrimSpace(PDFText(b)) != "" {
			return nil // it has a text layer: its own text is read
		}
		var out []ports.Part
		for i, img := range pdfJPEGs(b) {
			out = append(out, ports.Part{Name: base, MIME: "image/jpeg", Page: i + 1, Data: img})
		}
		return out
	case ext == ".docx" || ext == ".pptx":
		return officeImages(base, ext, b)
	}
	return nil
}

// Media is a recording as the one part to transcribe.
func Media(name string, b []byte) []ports.Part {
	if !IsMedia(name) || len(b) == 0 || len(b) > MaxMediaSize {
		return nil
	}
	return []ports.Part{{Name: path.Base(name), MIME: mediaMIME[strings.ToLower(path.Ext(name))], Data: b}}
}

// pdfJPEGs are a PDF's JPEG images (DCTDecode streams), in the order they appear: in a
// scanned PDF, one a page.
func pdfJPEGs(b []byte) [][]byte {
	var out [][]byte
	for len(b) > 0 {
		i := bytes.Index(b, []byte("stream"))
		if i < 0 {
			break
		}
		if i >= 3 && string(b[i-3:i]) == "end" {
			b = b[i+6:]
			continue
		}
		dict := b[:i]
		if d := bytes.LastIndex(dict, []byte("<<")); d >= 0 {
			dict = dict[d:]
		}
		body := bytes.TrimPrefix(bytes.TrimPrefix(b[i+6:], []byte("\r")), []byte("\n"))
		end := bytes.Index(body, []byte("endstream"))
		if end < 0 {
			break
		}
		data := bytes.TrimRight(body[:end], "\r\n")
		b = body[end+9:]
		if bytes.Contains(dict, []byte("/Image")) && bytes.Contains(dict, []byte("/DCTDecode")) && len(data) <= MaxImageSize && bytes.HasPrefix(data, []byte{0xFF, 0xD8}) {
			out = append(out, append([]byte(nil), data...))
		}
		if len(out) >= 50 {
			break
		}
	}
	return out
}

var slideRel = regexp.MustCompile(`^ppt/slides/_rels/slide([0-9]+)\.xml\.rels$`)
var relTarget = regexp.MustCompile(`Target="\.\./media/([^"]+)"`)

// officeImages are the pictures under word/media or ppt/media; a deck's carry the first
// slide that shows them.
func officeImages(base, ext string, b []byte) []ports.Part {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil
	}
	dir := "word/media/"
	if ext == ".pptx" {
		dir = "ppt/media/"
	}
	slideOf := map[string]int{}
	if ext == ".pptx" {
		for _, f := range z.File {
			m := slideRel.FindStringSubmatch(f.Name)
			if m == nil {
				continue
			}
			n, _ := strconv.Atoi(m[1])
			rc, err := f.Open()
			if err != nil {
				continue
			}
			x, _ := io.ReadAll(io.LimitReader(rc, 1<<20))
			_ = rc.Close()
			for _, t := range relTarget.FindAllStringSubmatch(string(x), -1) {
				if s, ok := slideOf[t[1]]; !ok || n < s {
					slideOf[t[1]] = n
				}
			}
		}
	}
	var out []ports.Part
	for _, f := range z.File {
		mime, ok := imageMIME[strings.ToLower(path.Ext(f.Name))]
		if !ok || !strings.HasPrefix(f.Name, dir) || f.UncompressedSize64 > MaxImageSize {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(rc, MaxImageSize+1))
		_ = rc.Close()
		if err != nil || len(data) > MaxImageSize {
			continue
		}
		out = append(out, ports.Part{Name: base + " " + path.Base(f.Name), MIME: mime, Page: slideOf[path.Base(f.Name)], Data: data})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Page < out[j].Page })
	return out
}
