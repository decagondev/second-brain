package folder

import (
	"strings"
	"testing"
	"unicode/utf8"

	"example.com/second-brain/internal/core/ingest"
)

// SB-5205: a PDF's pages and a deck's slides start where their text starts in Text, and a
// passage's page is the one it starts on.
func TestPagesOfPDFsAndDecks(t *testing.T) {
	two := []byte(strings.Replace(string(pdf([]byte("BT (First page) Tj ET"), false)), "%%EOF", "", 1) +
		"5 0 obj\n<< /Length 22 >>\nstream\nBT (Second page) Tj ET\nendstream\nendobj\n%%EOF\n")
	text, _ := Text("a.pdf", two)
	pages := Pages("a.pdf", two)
	if text != "First page\nSecond page" || len(pages) != 2 || pages[0] != 0 || pages[1] != utf8.RuneCountInString("First page\n") {
		t.Fatalf("pdf %q %v", text, pages)
	}
	if ingest.PageAt(pages, strings.Index(text, "Second")) != 2 || ingest.PageAt(pages, 0) != 1 || ingest.PageAt(nil, 5) != 0 {
		t.Fatal("PageAt")
	}
	deck := archive(t, "ppt/slides/slide2.xml", slideXML("Second"), "ppt/slides/slide1.xml", slideXML("First"))
	text, _ = Text("deck.pptx", deck)
	pages = Pages("deck.pptx", deck)
	if len(pages) != 2 || !strings.HasPrefix(string([]rune(text)[pages[1]:]), "Second") {
		t.Fatalf("deck %q %v", text, pages)
	}
	if Pages("notes.md", []byte("# x")) != nil {
		t.Fatal("a Markdown file has pages")
	}
}
