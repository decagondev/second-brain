package folder

import (
	"archive/zip"
	"bytes"
	"testing"
)

// SB-6101, SB-6102: an image is its own picture; a PDF with no text layer gives its JPEGs as
// pages; one with text gives none; a deck's pictures carry their slide; a recording is one
// part; text files have none.
func TestImagesAndMedia(t *testing.T) {
	if ps := Images("scans/board.PNG", []byte("png")); len(ps) != 1 || ps[0].MIME != "image/png" || ps[0].Name != "board.PNG" {
		t.Fatalf("png: %+v", ps)
	}
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 1, 2, 3}
	pdf := []byte("%PDF-1.4\n1 0 obj << /Type /XObject /Subtype /Image /Filter /DCTDecode /Length 7 >>\nstream\n" + string(jpeg) + "\nendstream\n" +
		"2 0 obj << /Type /XObject /Subtype /Image /Filter /DCTDecode /Length 7 >>\nstream\n" + string(jpeg) + "\nendstream\n")
	ps := Images("contract.pdf", pdf)
	if len(ps) != 2 || ps[0].Page != 1 || ps[1].Page != 2 || !bytes.Equal(ps[1].Data, jpeg) {
		t.Fatalf("scanned pdf: %+v", ps)
	}
	withText := append([]byte("1 0 obj << /Length 20 >>\nstream\nBT (Hello there) Tj ET\nendstream\n"), pdf...)
	if ps := Images("memo.pdf", withText); len(ps) != 0 {
		t.Fatalf("a pdf with text gave pictures: %d", len(ps))
	}
	var deck bytes.Buffer
	z := zip.NewWriter(&deck)
	for name, body := range map[string]string{
		"ppt/slides/slide1.xml":            "<p/>",
		"ppt/slides/_rels/slide3.xml.rels": `<Relationships><Relationship Target="../media/image1.png"/></Relationships>`,
		"ppt/media/image1.png":             "png-bytes",
		"ppt/media/notes.txt":              "not a picture",
	} {
		w, _ := z.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = z.Close()
	if ps := Images("plan.pptx", deck.Bytes()); len(ps) != 1 || ps[0].Page != 3 || ps[0].Name != "plan.pptx image1.png" {
		t.Fatalf("deck: %+v", ps)
	}
	if ms := Media("calls/standup.m4a", []byte("audio")); len(ms) != 1 || ms[0].MIME != "audio/mp4" {
		t.Fatalf("media: %+v", ms)
	}
	if Images("notes.md", []byte("x")) != nil || Media("notes.md", []byte("x")) != nil {
		t.Fatal("a text file has parts")
	}
	if !Document("scans/board.jpg") || !Document("calls/standup.mp3") {
		t.Fatal("pictures and recordings are documents")
	}
}
