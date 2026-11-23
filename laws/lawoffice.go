package laws

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"html"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
)

// LAW-09 and LAW-13 over office documents and web pages (SB-2003): the folder connector
// reads .docx, .pptx, .odt and .html files, and .xlsx, .ods and .csv (SB-2206), and the
// same laws hold as for text. Every file
// also carries a canary in the parts that must never be read (a macro, an embedded object,
// speaker notes, a script, a style): data that isn't the document's text never reaches the
// index, and text addressed to an AI in the body is flagged as in any other document.

// OfficeFormats are the formats written, by extension: from SB-2206 spreadsheets too, each
// line of the text a row, with a canary in a formula, a comment, a macro or a script.
var OfficeFormats = []string{".docx", ".pptx", ".odt", ".html", ".xlsx", ".ods", ".csv"}

// OfficeEncoder writes a document's text in a format, with canary in its hidden parts.
type OfficeEncoder func(ext, text, canary string) []byte

// EncodeOffice is the real encoder: the text in the body, the canary only in hidden parts.
var EncodeOffice OfficeEncoder = encodeOffice

// LeakyOffice puts the canary in the text too, as a reader that read macros or scripts
// would index it. Self-tests use it; the law never does.
var LeakyOffice OfficeEncoder = func(ext, text, canary string) []byte { return encodeOffice(ext, text+"\n\n"+canary, canary) }

func xmlEsc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func encodeOffice(ext, text, canary string) []byte {
	paras := strings.Split(text, "\n\n")
	var buf bytes.Buffer
	zipOf := func(parts ...string) []byte {
		w := zip.NewWriter(&buf)
		for i := 0; i+1 < len(parts); i += 2 {
			f, _ := w.Create(parts[i])
			_, _ = f.Write([]byte(parts[i+1]))
		}
		_ = w.Close()
		return buf.Bytes()
	}
	switch ext {
	case ".docx":
		var body strings.Builder
		for _, p := range paras {
			fmt.Fprintf(&body, `<w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, xmlEsc(p))
		}
		fmt.Fprintf(&body, `<w:p><w:r><w:object><w:t>%s</w:t></w:object></w:r></w:p>`, canary)
		return zipOf("word/document.xml", `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`+body.String()+`</w:body></w:document>`,
			"word/vbaProject.bin", "Sub AutoOpen() "+canary+" End Sub", "word/embeddings/oleObject1.bin", canary)
	case ".pptx":
		var parts []string
		for i, p := range paras {
			parts = append(parts, fmt.Sprintf("ppt/slides/slide%d.xml", i+1),
				`<p:sld xmlns:p="p" xmlns:a="a"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>`+xmlEsc(p)+`</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`)
		}
		parts = append(parts, "ppt/notesSlides/notesSlide1.xml", `<p:notes xmlns:p="p" xmlns:a="a"><a:p><a:r><a:t>`+canary+`</a:t></a:r></a:p></p:notes>`,
			"ppt/vbaProject.bin", canary)
		return zipOf(parts...)
	case ".odt":
		var body strings.Builder
		for _, p := range paras {
			fmt.Fprintf(&body, "<text:p>%s</text:p>", xmlEsc(p))
		}
		return zipOf("mimetype", "application/vnd.oasis.opendocument.text",
			"content.xml", `<office:document-content xmlns:office="o" xmlns:text="t" xmlns:draw="d"><office:scripts><office:script>`+canary+
				`</office:script></office:scripts><office:body><office:text>`+body.String()+`<draw:frame><draw:object><text:p>`+canary+
				`</text:p></draw:object></draw:frame></office:text></office:body></office:document-content>`,
			"Basic/Standard/Module1.xml", canary)
	case ".xlsx", ".ods", ".csv":
		return encodeSheet(ext, text, canary, zipOf)
	default: // .html
		var body strings.Builder
		for _, p := range paras {
			fmt.Fprintf(&body, "<p>%s</p>\n", html.EscapeString(p))
		}
		return []byte(`<!DOCTYPE html><html><head><title>` + canary + `</title><style>/* ` + canary + ` */</style><script>var a = 1 < 2; "` + canary +
			`"</script></head><body>` + body.String() + `<noscript>` + canary + `</noscript><object data=x>` + canary + `</object></body></html>`)
	}
}

// encodeSheet writes each line of the text as a row of one cell. In an .xlsx the canary is
// a formula's text, a comment and a macro; in an .ods a formula, an annotation and a
// script. A .csv has no hidden part.
func encodeSheet(ext, text, canary string, zipOf func(...string) []byte) []byte {
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	switch ext {
	case ".xlsx":
		var rows strings.Builder
		for i, l := range lines {
			fmt.Fprintf(&rows, `<row r="%d"><c r="A%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c></row>`, i+1, i+1, xmlEsc(l))
		}
		fmt.Fprintf(&rows, `<row r="%d"><c r="B%d"><f>%s</f><v>0</v></c></row>`, len(lines)+1, len(lines)+1, canary)
		return zipOf("xl/workbook.xml", `<workbook xmlns:r="r"><sheets><sheet name="Notes" sheetId="1" r:id="rId1"/></sheets></workbook>`,
			"xl/_rels/workbook.xml.rels", `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
			"xl/worksheets/sheet1.xml", `<worksheet><sheetData>`+rows.String()+`</sheetData></worksheet>`,
			"xl/comments1.xml", `<comments><commentList><comment ref="A1"><text><t>`+canary+`</t></text></comment></commentList></comments>`,
			"xl/vbaProject.bin", "Sub Auto_Open() "+canary+" End Sub")
	case ".ods":
		var rows strings.Builder
		for _, l := range lines {
			fmt.Fprintf(&rows, `<table:table-row><table:table-cell><text:p>%s</text:p></table:table-cell></table:table-row>`, xmlEsc(l))
		}
		return zipOf("mimetype", "application/vnd.oasis.opendocument.spreadsheet",
			"content.xml", `<office:document-content xmlns:office="o" xmlns:table="t" xmlns:text="x"><office:scripts><office:script>`+canary+
				`</office:script></office:scripts><office:body><office:spreadsheet><table:table table:name="Notes">`+rows.String()+
				`<table:table-row><table:table-cell table:formula="of:=`+canary+`()" office:value="0"><text:p>0</text:p><office:annotation><text:p>`+canary+
				`</text:p></office:annotation></table:table-cell></table:table-row></table:table></office:spreadsheet></office:body></office:document-content>`,
			"Basic/Standard/Module1.xml", canary)
	default: // .csv
		var b strings.Builder
		for _, l := range lines {
			b.WriteString(`"` + strings.ReplaceAll(l, `"`, `""`) + "\"\r\n")
		}
		return []byte(b.String())
	}
}

func extOf(id string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return OfficeFormats[int(h.Sum32())%len(OfficeFormats)]
}

// NewOfficeSanitiserWorld runs LAW-09 through a real share of office files: each document
// is written in a random format with a canary in its hidden parts.
func NewOfficeSanitiserWorld(dir string, r *rand.Rand, enc OfficeEncoder) (SanitiserWorld, string) {
	return &officeSanitiser{dir: dir, ext: OfficeFormats[r.Intn(len(OfficeFormats))], canary: fmt.Sprintf("HIDDENPART%x", r.Int63()), enc: enc,
		s: ingest.Sanitiser{TenantDomains: []string{"acme-logistics.example"}}}, ""
}

type officeSanitiser struct {
	dir, ext, canary string
	enc              OfficeEncoder
	s                ingest.Sanitiser
}

func (w *officeSanitiser) Ingest(text string) ([]SanitisedChunk, error) {
	share, err := os.MkdirTemp(w.dir, "share")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(share, "notes"+w.ext), w.enc(w.ext, text, w.canary), 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(share, "notes"+w.ext+".acl"), []byte("read ann@acme.example\n"), 0o600); err != nil {
		return nil, err
	}
	c, _, err := openFolder(share)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Close() }()
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: "acme-logistics", Connector: c, Embedder: hashing.Embedder{}, Index: idx, Sanitiser: w.s}
	if _, err := p.Sync(context.Background(), ""); err != nil {
		return nil, err
	}
	cs, err := idx.All(context.Background(), "acme-logistics")
	if err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, fmt.Errorf("the %s file was indexed with no text", w.ext)
	}
	out := make([]SanitisedChunk, len(cs))
	for i, c := range cs {
		if strings.Contains(c.Text, w.canary) {
			return nil, fmt.Errorf("LAW-09: a hidden part of a %s file (macro, object, notes, script or style) reached the index: %q", w.ext, c.Text)
		}
		out[i] = SanitisedChunk{Text: c.Text, Flags: c.Flags}
	}
	return out, nil
}

func (officeSanitiser) Wrap(text, cite string, flags []string) string {
	return ingest.Wrap(text, cite, flags)
}

// law09Office is LAW-09 over an office share.
func law09Office(r *rand.Rand, dir string, enc OfficeEncoder) error {
	w, _ := NewOfficeSanitiserWorld(dir, r, enc)
	return law09(r, w)
}

// NewOfficeIngestWorld runs LAW-13 over the folder connector on a share of office files.
func NewOfficeIngestWorld(dir string) IngestWorld {
	fi := NewFolderIngestWorld(dir).(*folderIngest)
	return &officeIngest{folderIngest: fi}
}

type officeIngest struct {
	*folderIngest
}

func (w *officeIngest) write(id, text string) (string, error) {
	ext := extOf(id)
	p := filepath.Join(w.w.dir, filepath.FromSlash(id)+ext)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return "", err
	}
	return p, os.WriteFile(p, EncodeOffice(ext, text, "HIDDENPART"), 0o600)
}

func (w *officeIngest) Load(docs []SourceDoc) error {
	for _, d := range docs {
		p, err := w.write(d.ID, d.Text)
		if err != nil {
			return err
		}
		var acl strings.Builder
		for _, v := range d.Viewers {
			if strings.Contains(v, "@") {
				fmt.Fprintf(&acl, "read %s\n", v)
			}
		}
		if err := os.WriteFile(p+".acl", []byte(acl.String()), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (w *officeIngest) Edit(id, text string) error {
	p, err := w.write(id, text)
	if err != nil {
		return err
	}
	w.edits++
	at := time.Now().Add(time.Duration(w.edits) * time.Minute)
	return os.Chtimes(p, at, at)
}
