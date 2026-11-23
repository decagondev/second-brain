package folder

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// archive builds a zip with the given parts, in order.
func archive(t testing.TB, parts ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for i := 0; i+1 < len(parts); i += 2 {
		f, err := w.Create(parts[i])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(parts[i+1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const wordDoc = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
<w:body>
<w:p><w:r><w:t>Fuel cards are</w:t></w:r><w:r><w:t xml:space="preserve"> issued by the depot lead.</w:t></w:r></w:p>
<w:p><w:r><w:t>Report a lost card</w:t></w:r><w:r><w:tab/><w:t>the same day.</w:t></w:r></w:p>
<w:p><w:r><w:object><w:t>EMBEDDED OBJECT TEXT</w:t></w:object></w:r></w:p>
<w:p><w:r><w:delText>DELETED TEXT</w:delText></w:r></w:p>
<w:p><mc:AlternateContent><mc:Choice><w:r><w:t>Box text.</w:t></w:r></mc:Choice><mc:Fallback><w:r><w:t>Box text.</w:t></w:r></mc:Fallback></mc:AlternateContent></w:p>
</w:body></w:document>`

func slideXML(text string) string {
	return `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">` +
		`<p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>` + text + `</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`
}

const odtContent = `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0">
<office:scripts><office:script>SCRIPT TEXT</office:script></office:scripts>
<office:body><office:text>
<text:h>Loading bay</text:h>
<text:p>Bays open at<text:s/>six.<text:line-break/>Close at ten.</text:p>
<draw:frame><draw:object><text:p>EMBEDDED OBJECT TEXT</text:p></draw:object></draw:frame>
<office:annotation><text:p>COMMENT TEXT</text:p></office:annotation>
</office:text></office:body></office:document-content>`

const page = `<!DOCTYPE html>
<html><head><title>TITLE TEXT</title><meta charset="utf-8"><style>p { color: red } /* STYLE TEXT */</style>
<script>if (a < b && c) { document.write("SCRIPT TEXT") }</script></head>
<body><h1>Night shift</h1>
<p>Doors lock at 22:00 &amp; reopen at 05:00.<br>Ring the bell.
<p>Keys are with the <b>shift lead</b>.
<!-- COMMENT TEXT -->
<noscript>NOSCRIPT TEXT</noscript><iframe src="x">IFRAME TEXT</iframe><embed src=movie.swf><object data=x>OBJECT TEXT</object>
<svg><text>SVG TEXT</text></svg>
<ul><li>Badge<li>Vest</ul>
</body></html>`

func TestOfficeText(t *testing.T) {
	for _, c := range []struct {
		name string
		b    []byte
		want []string
	}{
		{"card.docx", archive(t, "[Content_Types].xml", "<Types/>", "word/document.xml", wordDoc, "word/vbaProject.bin", "MACRO TEXT",
			"word/embeddings/oleObject1.bin", "EMBEDDED OBJECT TEXT"),
			[]string{"Fuel cards are issued by the depot lead.\nReport a lost card\tthe same day.", "Box text."}},
		{"deck.pptx", archive(t, "ppt/slides/slide10.xml", slideXML("Tenth"), "ppt/slides/slide2.xml", slideXML("Second"),
			"ppt/notesSlides/notesSlide1.xml", slideXML("NOTES TEXT"), "ppt/slides/slide1.xml", slideXML("First")),
			[]string{"First\n\nSecond\n\nTenth"}},
		{"bay.odt", archive(t, "mimetype", "application/vnd.oasis.opendocument.text", "content.xml", odtContent, "Basic/Standard/Module1.xml", "MACRO TEXT"),
			[]string{"Loading bay\nBays open at six.\nClose at ten."}},
		{"night.html", []byte(page), []string{"Night shift", "Doors lock at 22:00 & reopen at 05:00.\nRing the bell.", "Keys are with the shift lead.", "Badge", "Vest"}},
		{"NIGHT.HTM", []byte(page), []string{"Night shift"}},
	} {
		got := OfficeText(c.name, c.b)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: %q lacks %q", c.name, got, w)
			}
		}
		for _, never := range []string{"MACRO", "EMBEDDED", "DELETED", "SCRIPT", "STYLE", "COMMENT", "NOTES", "NOSCRIPT", "IFRAME", "OBJECT", "SVG", "TITLE"} {
			if strings.Contains(got, never) {
				t.Errorf("%s: read %s text: %q", c.name, never, got)
			}
		}
		if strings.Count(got, "Box text.") > 1 {
			t.Errorf("%s: alternate content read twice: %q", c.name, got)
		}
	}
}

// An archive with too many entries, or text parts past the limit, is skipped whole; so is
// one that doesn't open, a part that doesn't parse, and a page in another charset.
func TestOfficeTextSkipsWhole(t *testing.T) {
	many := []string{"word/document.xml", wordDoc}
	for i := 0; i <= MaxOfficeEntries; i++ {
		many = append(many, "x/"+strconv.Itoa(i), "")
	}
	big := strings.Repeat("<w:p><w:r><w:t>"+strings.Repeat("A", 1000)+"</w:t></w:r></w:p>", MaxOfficeText/1000/2)
	var bomb bytes.Buffer
	w := zip.NewWriter(&bomb)
	for i := 1; i <= 3; i++ {
		f, _ := w.Create("ppt/slides/slide" + strconv.Itoa(i) + ".xml")
		_, _ = f.Write([]byte(slideXML(big)))
	}
	_ = w.Close()
	for name, b := range map[string][]byte{
		"many.docx":   archive(t, many...),
		"bomb.pptx":   bomb.Bytes(),
		"broken.docx": []byte("PK\x03\x04 not really"),
		"bad.odt":     archive(t, "content.xml", "<office:document-content><text:p>half"),
		"latin.html":  []byte("<p>caf\xe9</p>"),
	} {
		if got := OfficeText(name, b); got != "" {
			t.Errorf("%s: half-read %d bytes", name, len(got))
		}
	}
}

// A part that declares a small size but inflates past the limit is caught by the reader.
func TestOfficeTextBoundsALyingSize(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.CreateHeader(&zip.FileHeader{Name: "content.xml", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("<a><text:p xmlns:text=\"t\">" + strings.Repeat("A", MaxOfficeText+10) + "</text:p></a>"))
	_ = w.Close()
	b := buf.Bytes()
	// Rewrite the declared uncompressed size in the central directory to 1 byte.
	cd := bytes.LastIndex(b, []byte("PK\x01\x02"))
	b[cd+24], b[cd+25], b[cd+26], b[cd+27] = 1, 0, 0, 0
	if got := OfficeText("lie.odt", b); got != "" {
		t.Fatalf("a lying archive was read: %d bytes", len(got))
	}
}

// The connector lists and fetches office files and pages like any document.
func TestOfficeFilesAreDocuments(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "card.docx"), archive(t, "word/document.xml", wordDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "night.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	for id, want := range map[string]string{"card.docx": "issued by the depot lead", "night.html": "Ring the bell"} {
		if !Document(id) {
			t.Fatalf("%s isn't a document", id)
		}
		d, err := c.Fetch(context.Background(), ports.SourceRef{Kind: "docs", ID: id})
		if err != nil || !strings.Contains(d.Text, want) {
			t.Errorf("%s: %q %v", id, d.Text, err)
		}
	}
}

func FuzzOfficeText(f *testing.F) {
	f.Add("a.docx", archive(f, "word/document.xml", wordDoc))
	f.Add("a.pptx", archive(f, "ppt/slides/slide1.xml", slideXML("x")))
	f.Add("a.odt", archive(f, "content.xml", odtContent))
	f.Add("a.html", []byte(page))
	f.Fuzz(func(t *testing.T, name string, b []byte) {
		for _, ext := range []string{".docx", ".pptx", ".odt", ".html"} {
			if s := OfficeText(name+ext, b); len(s) > MaxOfficeText+len(b) {
				t.Fatalf("%s: %d bytes from %d", ext, len(s), len(b))
			}
		}
	})
}
