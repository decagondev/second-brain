package folder

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"html"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Office documents and web pages (SB-2003): the text of a .docx, .pptx, .odt or .html
// file, read with the standard library only. Only the parts that hold the document's text
// are opened (word/document.xml, ppt/slides/slideN.xml, content.xml); macros, embedded
// objects, media and every other part of the archive are never read. In a web page,
// scripts, styles and embedded objects are skipped. An archive with too many entries, or
// whose text parts would grow past MaxOfficeText, is skipped whole, never half-read: the
// file is then indexed by name only, as a PDF with no text is.

// MaxOfficeEntries is the most entries an archive may list before it is skipped.
const MaxOfficeEntries = 4096

// MaxOfficeText is the most an archive's text parts may decompress to, in all.
const MaxOfficeText = 16 << 20

var errTooBig = errors.New("folder: the archive is too large to read")

// OfficeText is the text of an office document or web page, by its extension, or "" for
// a file it can't read whole.
func OfficeText(name string, b []byte) string {
	var s string
	var err error
	switch strings.ToLower(path.Ext(name)) {
	case ".docx":
		s, err = fromZip(b, func(n string) bool { return n == "word/document.xml" }, docx)
	case ".pptx":
		s, err = fromZip(b, slide, pptx)
	case ".odt":
		s, err = fromZip(b, func(n string) bool { return n == "content.xml" }, odt)
	case ".html", ".htm":
		s, err = htmlText(b)
	case ".xlsx", ".ods", ".csv":
		return SheetText(name, b)
	default:
		return ""
	}
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

var slideName = regexp.MustCompile(`^ppt/slides/slide([0-9]{1,5})\.xml$`)

func slide(n string) bool { return slideName.MatchString(n) }

// fromZip reads the parts want picks, in order (slides by number), through read.
func fromZip(b []byte, want func(string) bool, read func(io.Reader) (string, error)) (string, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", err
	}
	if len(z.File) > MaxOfficeEntries {
		return "", errTooBig
	}
	var parts []*zip.File
	var total uint64
	for _, f := range z.File {
		if !want(f.Name) {
			continue
		}
		if total += f.UncompressedSize64; total > MaxOfficeText {
			return "", errTooBig
		}
		parts = append(parts, f)
	}
	sort.SliceStable(parts, func(i, j int) bool { return partOrder(parts[i].Name) < partOrder(parts[j].Name) })
	var out strings.Builder
	left := int64(MaxOfficeText)
	for _, f := range parts {
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		// The sizes an archive declares can lie: the reader is bounded too, and a part that
		// runs past what is left fails the whole file.
		lr := &io.LimitedReader{R: rc, N: left + 1}
		s, err := read(lr)
		_ = rc.Close()
		if lr.N <= 0 {
			return "", errTooBig
		}
		if err != nil {
			return "", err
		}
		left = lr.N - 1
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(strings.TrimSpace(s))
	}
	return out.String(), nil
}

func partOrder(n string) int {
	if m := slideName.FindStringSubmatch(n); m != nil {
		i, _ := strconv.Atoi(m[1])
		return i
	}
	return 0
}

// xmlText walks an XML part: text inside a text element is kept, a block element's end is
// a line break, and nothing inside a skipped element is kept.
type xmlText struct {
	text, block, skip map[string]bool // by local name
	breaks            map[string]string
}

func (x xmlText) read(r io.Reader) (string, error) {
	d := xml.NewDecoder(r)
	var out strings.Builder
	inText, skipping := 0, 0
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return out.String(), nil
		}
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := t.Name.Local
			switch {
			case skipping > 0 || x.skip[n]:
				skipping++
			case x.text[n]:
				inText++
			case x.breaks[n] != "":
				out.WriteString(x.breaks[n])
			}
		case xml.EndElement:
			n := t.Name.Local
			switch {
			case skipping > 0:
				skipping--
			default:
				if x.text[n] {
					inText--
				}
				if x.block[n] {
					out.WriteString("\n")
				}
			}
		case xml.CharData:
			if inText > 0 && skipping == 0 {
				out.Write(t)
			}
		}
	}
}

// Word: runs' text (w:t) in paragraphs (w:p); embedded objects, drawings' alternate
// content and deleted text are skipped.
var docx = xmlText{
	text:   map[string]bool{"t": true},
	block:  map[string]bool{"p": true},
	skip:   map[string]bool{"object": true, "pict": true, "Fallback": true, "delText": true, "instrText": true},
	breaks: map[string]string{"tab": "\t", "br": "\n", "cr": "\n"},
}.read

// PowerPoint: a slide's text (a:t) in paragraphs (a:p).
var pptx = xmlText{
	text:   map[string]bool{"t": true},
	block:  map[string]bool{"p": true},
	skip:   map[string]bool{"Fallback": true, "oleObj": true},
	breaks: map[string]string{"br": "\n"},
}.read

// OpenDocument text: paragraphs and headings (text:p, text:h); scripts, forms, embedded
// objects and annotations are skipped.
var odt = xmlText{
	text:   map[string]bool{"p": true, "h": true},
	block:  map[string]bool{"p": true, "h": true},
	skip:   map[string]bool{"scripts": true, "script": true, "forms": true, "object": true, "object-ole": true, "annotation": true, "tracked-changes": true},
	breaks: map[string]string{"tab": "\t", "line-break": "\n", "s": " "},
}.read

// htmlSkip are the elements whose content is never text: scripts, styles, embedded
// objects, and the title (the file's name is the title already). Elements with no content
// (embed, img) need no skipping.
var htmlSkip = map[string]bool{"script": true, "style": true, "noscript": true, "template": true, "iframe": true,
	"object": true, "applet": true, "svg": true, "math": true, "title": true, "canvas": true}

var htmlBlock = map[string]bool{"p": true, "div": true, "li": true, "tr": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "section": true, "article": true, "blockquote": true, "pre": true, "table": true,
	"ul": true, "ol": true, "dd": true, "dt": true, "header": true, "footer": true, "main": true, "nav": true}

// rawText are the elements whose content an HTML parser reads as raw text; encoding/xml
// would try to parse it, so they are cut out before parsing.
var rawText = regexp.MustCompile(`(?is)<(script|style|textarea|xmp)\b.*?</(script|style|textarea|xmp)\s*>`)
var comment = regexp.MustCompile(`(?s)<!--.*?-->`)

// htmlText is a web page's visible text. Pages are rarely well-formed XML, so tags are
// scanned, not parsed: a tag runs from < to the next > outside quotes, and text between
// tags is kept unless it is inside a skipped element. Entities are decoded with html.
func htmlText(b []byte) (string, error) {
	if !utf8.Valid(b) {
		return "", errors.New("folder: only UTF-8 pages are read")
	}
	b = comment.ReplaceAll(b, nil)
	b = rawText.ReplaceAll(b, nil)
	var out strings.Builder
	skipName, skipDepth := "", 0
	for len(b) > 0 {
		i := bytes.IndexByte(b, '<')
		if i < 0 {
			i = len(b)
		}
		if skipDepth == 0 {
			out.WriteString(space.ReplaceAllString(html.UnescapeString(string(b[:i])), " "))
		}
		if i == len(b) {
			break
		}
		j := tagEnd(b, i+1)
		tag := b[i+1 : j]
		if j < len(b) {
			j++
		}
		b = b[j:]
		if len(tag) == 0 || tag[0] == '!' || tag[0] == '?' {
			continue
		}
		closing := tag[0] == '/'
		self := bytes.HasSuffix(tag, []byte("/"))
		name := tagName(bytes.TrimPrefix(tag, []byte("/")))
		switch {
		case skipDepth > 0:
			if name == skipName && closing {
				skipDepth--
			} else if name == skipName && !self {
				skipDepth++
			}
		case htmlSkip[name] && !closing && !self:
			skipName, skipDepth = name, 1
		case name == "br" || htmlBlock[name]:
			out.WriteString("\n")
		}
	}
	var lines []string
	for _, l := range strings.Split(out.String(), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n"), nil
}

var space = regexp.MustCompile(`[ \t\r\n\f]+`)

// tagEnd is the index of the > that ends the tag starting at i, outside quotes, or len(b).
func tagEnd(b []byte, i int) int {
	var q byte
	for ; i < len(b); i++ {
		switch c := b[i]; {
		case q != 0:
			if c == q {
				q = 0
			}
		case c == '"' || c == '\'':
			q = c
		case c == '>':
			return i
		}
	}
	return len(b)
}

// tagName is a tag's element name, lower case.
func tagName(t []byte) string {
	n := 0
	for n < len(t) && (t[n] >= 'a' && t[n] <= 'z' || t[n] >= 'A' && t[n] <= 'Z' || t[n] >= '0' && t[n] <= '9') {
		n++
	}
	return strings.ToLower(string(t[:n]))
}

// maxCustomProps is the most an Office file's custom properties part is read to.
const maxCustomProps = 64 << 10

// OfficeReviewBy is the value of a .docx or .pptx file's custom property named review-by
// (or expires), as written, or "" (SB-2705). Only docProps/custom.xml is opened.
func OfficeReviewBy(b []byte) string {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil || len(z.File) > MaxOfficeEntries {
		return ""
	}
	for _, f := range z.File {
		if f.Name != "docProps/custom.xml" || f.UncompressedSize64 > maxCustomProps {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ""
		}
		defer func() { _ = rc.Close() }()
		var props struct {
			Property []struct {
				Name  string `xml:"name,attr"`
				Inner struct {
					Text string `xml:",chardata"`
				} `xml:",any"`
			} `xml:"property"`
		}
		if xml.NewDecoder(io.LimitReader(rc, maxCustomProps)).Decode(&props) != nil {
			return ""
		}
		var expires string
		for _, p := range props.Property {
			switch strings.ToLower(p.Name) {
			case "review-by", "review_by":
				return strings.TrimSpace(p.Inner.Text)
			case "expires":
				expires = strings.TrimSpace(p.Inner.Text)
			}
		}
		return expires
	}
	return ""
}
