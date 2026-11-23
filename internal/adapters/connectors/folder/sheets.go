package folder

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Spreadsheets (SB-2206): the values of an .xlsx, .ods or .csv file, read with the
// standard library only. Each sheet is "Sheet: NAME" and then one line a row that holds a
// value, cells tab-separated in their columns. A formula is never evaluated: a cell is the
// value the file last saved for it, and a formula's text is never read. In an .xlsx only
// the workbook, its relationships, the shared strings and the worksheets are opened, so
// macros, comments, charts, drawings, pivot caches and external links are never read; in
// an .ods only content.xml's tables, without scripts, forms, annotations or objects. A
// sheet with more than MaxSheetRows rows is skipped whole; a file whose text would pass
// MaxOfficeText, or that doesn't open or parse, is indexed by name only.

// MaxSheetRows is the most rows holding a value a sheet may have before it is skipped.
const MaxSheetRows = 5000

// maxColumns is the widest a sheet may be (an .xlsx's own limit, XFD).
const maxColumns = 16384

var errTooManyRows = errors.New("folder: the sheet has too many rows")

// sheetOut writes sheets within the text budget.
type sheetOut struct {
	b      strings.Builder
	sheet  strings.Builder // the sheet being written
	rows   int
	over   bool // this sheet passed MaxSheetRows: it is dropped
	budget int
}

func newSheetOut() *sheetOut { return &sheetOut{budget: MaxOfficeText} }

func (o *sheetOut) start(name string) {
	o.sheet.Reset()
	o.rows, o.over = 0, false
	o.sheet.WriteString("Sheet: " + clean(name))
}

// row writes one row of (column, value) cells, columns from 1 and increasing; a row with
// no value is no row.
func (o *sheetOut) row(cells []sheetCell) error {
	if o.over {
		return nil
	}
	var line strings.Builder
	col, prev := 1, 0
	for _, c := range cells {
		if c.val == "" {
			continue
		}
		if c.col <= prev || c.col > maxColumns {
			return fmt.Errorf("folder: a cell out of its row's order")
		}
		if o.used()+line.Len()+(c.col-col)+len(c.val) > o.budget {
			return errTooBig
		}
		line.WriteString(strings.Repeat("\t", c.col-col))
		line.WriteString(c.val)
		col, prev = c.col, c.col
	}
	if line.Len() == 0 {
		return nil
	}
	if o.rows++; o.rows > MaxSheetRows {
		o.over = true
		o.sheet.Reset()
		return nil
	}
	o.sheet.WriteString("\n")
	o.sheet.WriteString(line.String())
	return nil
}

func (o *sheetOut) used() int { return o.b.Len() + o.sheet.Len() + 2 }

// end keeps the sheet unless it passed the bound or holds nothing.
func (o *sheetOut) end() error {
	if o.over || o.rows == 0 {
		return nil // past the bound, or no value at all
	}
	if o.used() > o.budget {
		return errTooBig
	}
	if o.b.Len() > 0 {
		o.b.WriteString("\n\n")
	}
	o.b.WriteString(o.sheet.String())
	o.sheet.Reset()
	return nil
}

type sheetCell struct {
	col int
	val string
}

// clean is a cell's value on one line: its tabs and line breaks are spaces.
func clean(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		switch r {
		case '\t', '\n', '\r', '\v', '\f':
			return ' '
		}
		return r
	}, s))
}

// SheetText is a spreadsheet's text by its extension, or "" for one it can't read whole.
func SheetText(name string, b []byte) string {
	var s string
	var err error
	switch strings.ToLower(path.Ext(name)) {
	case ".xlsx":
		s, err = xlsxText(b)
	case ".ods":
		s, err = odsText(b)
	case ".csv":
		base := path.Base(name)
		s, err = csvText(strings.TrimSuffix(base, path.Ext(base)), b)
	default:
		return ""
	}
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

// zipParts opens an archive within MaxOfficeEntries, and a reader that reads its parts
// within MaxOfficeText in all, whatever sizes they declare.
type zipParts struct {
	files map[string]*zip.File
	left  int64
}

func openZip(b []byte) (*zipParts, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}
	if len(z.File) > MaxOfficeEntries {
		return nil, errTooBig
	}
	p := &zipParts{files: map[string]*zip.File{}, left: MaxOfficeText}
	for _, f := range z.File {
		p.files[f.Name] = f
	}
	return p, nil
}

// read passes the named part to fn through a bounded reader; a part past what is left
// fails the whole file. A missing part is ok=false.
func (p *zipParts) read(name string, fn func(io.Reader) error) (bool, error) {
	f, ok := p.files[name]
	if !ok {
		return false, nil
	}
	if f.UncompressedSize64 > uint64(p.left) {
		return true, errTooBig
	}
	rc, err := f.Open()
	if err != nil {
		return true, err
	}
	defer func() { _ = rc.Close() }()
	lr := &io.LimitedReader{R: rc, N: p.left + 1}
	err = fn(lr)
	if lr.N <= 0 {
		return true, errTooBig
	}
	p.left = lr.N - 1
	return true, err
}

func attr(t xml.StartElement, local string) string {
	for _, a := range t.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// relAttr is the r:id of a workbook's sheet: the id attribute in the relationships space.
func relAttr(t xml.StartElement) string {
	for _, a := range t.Attr {
		if a.Name.Local == "id" && a.Name.Space != "" {
			return a.Value
		}
	}
	return ""
}

var worksheetPart = regexp.MustCompile(`^xl/worksheets/[A-Za-z0-9_.-]+\.xml$`)

// xlsxText reads a workbook's sheets in its order.
func xlsxText(b []byte) (string, error) {
	z, err := openZip(b)
	if err != nil {
		return "", err
	}
	type sheet struct{ name, rel string }
	var sheets []sheet
	ok, err := z.read("xl/workbook.xml", func(r io.Reader) error {
		return walk(r, func(t xml.StartElement) error {
			if t.Name.Local == "sheet" {
				sheets = append(sheets, sheet{attr(t, "name"), relAttr(t)})
			}
			return nil
		}, nil, nil)
	})
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("folder: no workbook")
	}
	targets := map[string]string{}
	if _, err := z.read("xl/_rels/workbook.xml.rels", func(r io.Reader) error {
		return walk(r, func(t xml.StartElement) error {
			if t.Name.Local == "Relationship" {
				tg := attr(t, "Target")
				if strings.HasPrefix(tg, "/") {
					tg = strings.TrimPrefix(tg, "/")
				} else {
					tg = path.Join("xl", tg)
				}
				targets[attr(t, "Id")] = tg
			}
			return nil
		}, nil, nil)
	}); err != nil {
		return "", err
	}
	var shared []string
	if _, err := z.read("xl/sharedStrings.xml", func(r io.Reader) error {
		shared, err = sharedStrings(r)
		return err
	}); err != nil {
		return "", err
	}
	out := newSheetOut()
	for _, s := range sheets {
		part := targets[s.rel]
		if !worksheetPart.MatchString(part) {
			continue // a chart sheet, or a part outside the worksheets: never read
		}
		out.start(s.name)
		if _, err := z.read(part, func(r io.Reader) error { return worksheet(r, shared, out) }); err != nil {
			return "", err
		}
		if err := out.end(); err != nil {
			return "", err
		}
	}
	return out.b.String(), nil
}

// walk streams an XML part to its handlers: start and end elements, and text.
func walk(r io.Reader, start func(xml.StartElement) error, end func(xml.EndElement) error, text func([]byte) error) error {
	d := xml.NewDecoder(r)
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if start != nil {
				if err := start(t); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if end != nil {
				if err := end(t); err != nil {
					return err
				}
			}
		case xml.CharData:
			if text != nil {
				if err := text(t); err != nil {
					return err
				}
			}
		}
	}
}

// sharedStrings is an .xlsx's string table: each item's text, its phonetic runs skipped.
func sharedStrings(r io.Reader) ([]string, error) {
	var out []string
	var cur strings.Builder
	inT, phon := 0, 0
	err := walk(r, func(t xml.StartElement) error {
		switch t.Name.Local {
		case "si":
			cur.Reset()
		case "t":
			inT++
		case "rPh":
			phon++
		}
		return nil
	}, func(t xml.EndElement) error {
		switch t.Name.Local {
		case "si":
			out = append(out, clean(cur.String()))
		case "t":
			inT--
		case "rPh":
			phon--
		}
		return nil
	}, func(b []byte) error {
		if inT > 0 && phon == 0 {
			cur.Write(b)
		}
		return nil
	})
	return out, err
}

// colOf is the column of a cell reference such as "BC12", from 1.
func colOf(ref string) (int, bool) {
	n, i := 0, 0
	for ; i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z'; i++ {
		if n = n*26 + int(ref[i]-'A'+1); n > maxColumns {
			return 0, false
		}
	}
	return n, i > 0
}

// worksheet reads one sheet's rows: each cell's saved value (v, or an inline string);
// formulas (f) and phonetic runs are skipped.
func worksheet(r io.Reader, shared []string, out *sheetOut) error {
	var cells []sheetCell
	var val strings.Builder
	var typ string
	col, next := 0, 1
	inV, inT, skip := 0, 0, 0
	return walk(r, func(t xml.StartElement) error {
		if skip > 0 {
			skip++
			return nil
		}
		switch t.Name.Local {
		case "row":
			cells, next = cells[:0], 1
		case "c":
			val.Reset()
			typ = attr(t, "t")
			col = next
			if ref := attr(t, "r"); ref != "" {
				c, ok := colOf(ref)
				if !ok {
					return fmt.Errorf("folder: a cell reference out of range")
				}
				col = c
			}
		case "v":
			inV++
		case "t":
			inT++
		case "f", "rPh", "extLst":
			skip = 1
		}
		return nil
	}, func(t xml.EndElement) error {
		if skip > 0 {
			skip--
			return nil
		}
		switch t.Name.Local {
		case "v":
			inV--
		case "t":
			inT--
		case "c":
			cells = append(cells, sheetCell{col, cellValue(typ, val.String(), shared)})
			next = col + 1
		case "row":
			return out.row(cells)
		}
		return nil
	}, func(b []byte) error {
		if skip == 0 && (inV > 0 || inT > 0) {
			val.Write(b)
		}
		return nil
	})
}

func cellValue(typ, v string, shared []string) string {
	switch typ {
	case "s":
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || i < 0 || i >= len(shared) {
			return ""
		}
		return shared[i]
	case "b":
		switch strings.TrimSpace(v) {
		case "1":
			return "TRUE"
		case "0":
			return "FALSE"
		}
		return ""
	}
	return clean(v)
}

// odsSkip are the elements of an .ods's content never read.
var odsSkip = map[string]bool{"scripts": true, "forms": true, "annotation": true, "frame": true, "object": true, "object-ole": true,
	"shapes": true, "event-listeners": true, "tracked-changes": true, "dde-links": true, "custom-shape": true, "g": true}

// odsText reads an OpenDocument spreadsheet's tables from content.xml.
func odsText(b []byte) (string, error) {
	z, err := openZip(b)
	if err != nil {
		return "", err
	}
	out := newSheetOut()
	ok, err := z.read("content.xml", func(r io.Reader) error { return odsContent(r, out) })
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("folder: no content.xml")
	}
	return out.b.String(), nil
}

func repeat(t xml.StartElement, name string) (int, error) {
	s := attr(t, name)
	if s == "" {
		return 1, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("folder: a bad %s", name)
	}
	return n, nil
}

// odsReader walks content.xml's tables.
type odsReader struct {
	out                *sheetOut
	depth, skip        int
	inTable            bool
	tableDepth, rowRep int
	cells              []sheetCell
	col, cellRep       int
	inCell, inP        bool
	val, p             strings.Builder
	fallback           string
}

func odsContent(r io.Reader, out *sheetOut) error {
	x := &odsReader{out: out}
	return walk(r, x.start, x.end, func(b []byte) error {
		if x.inP && x.skip == 0 {
			x.p.Write(b)
		}
		return nil
	})
}

func (x *odsReader) start(t xml.StartElement) error {
	x.depth++
	if x.skip > 0 {
		x.skip++
		return nil
	}
	n := t.Name.Local
	switch {
	case odsSkip[n]:
		x.skip = 1
	case n == "table" && !x.inTable:
		x.inTable, x.tableDepth = true, x.depth
		x.out.start(attr(t, "name"))
	case n == "table" && x.inTable:
		x.skip = 1 // a table inside a cell: not read
	case !x.inTable:
	case n == "table-row":
		rep, err := repeat(t, "number-rows-repeated")
		if err != nil {
			return err
		}
		x.rowRep, x.cells, x.col = rep, x.cells[:0], 1
	case n == "table-cell" || n == "covered-table-cell":
		return x.startCell(t)
	case x.inCell && n == "p":
		x.inP = true
		x.p.Reset()
	case x.inP && n == "s":
		c, err := repeat(t, "c")
		if err != nil || c > 1024 {
			return fmt.Errorf("folder: a bad space run")
		}
		x.p.WriteString(strings.Repeat(" ", c))
	case x.inP && (n == "tab" || n == "line-break"):
		x.p.WriteString(" ")
	}
	return nil
}

func (x *odsReader) startCell(t xml.StartElement) error {
	rep, err := repeat(t, "number-columns-repeated")
	if err != nil {
		return err
	}
	x.cellRep, x.inCell = rep, true
	x.val.Reset()
	x.fallback = ""
	for _, a := range []string{"value", "date-value", "time-value", "boolean-value", "string-value"} {
		if v := attr(t, a); v != "" {
			x.fallback = v
			break
		}
	}
	return nil
}

func (x *odsReader) end(t xml.EndElement) error {
	x.depth--
	if x.skip > 0 {
		x.skip--
		return nil
	}
	n := t.Name.Local
	switch {
	case x.inTable && n == "table" && x.depth == x.tableDepth-1:
		x.inTable = false
		return x.out.end()
	case x.inP && n == "p":
		x.inP = false
		if x.val.Len() > 0 {
			x.val.WriteString(" ")
		}
		x.val.WriteString(x.p.String())
	case x.inCell && (n == "table-cell" || n == "covered-table-cell"):
		x.inCell = false
		return x.endCell()
	case x.inTable && n == "table-row":
		return x.endRow()
	}
	return nil
}

func (x *odsReader) endCell() error {
	v := clean(x.val.String())
	if v == "" {
		v = clean(x.fallback)
	}
	if v == "" {
		if x.col += x.cellRep; x.col > maxColumns+1 {
			x.col = maxColumns + 1 // trailing empty columns; a value after them is an error
		}
		return nil
	}
	if x.col-1+x.cellRep > maxColumns {
		return fmt.Errorf("folder: a row wider than a sheet")
	}
	if x.cellRep*len(v) > MaxOfficeText {
		return errTooBig
	}
	for i := 0; i < x.cellRep; i++ {
		x.cells = append(x.cells, sheetCell{x.col, v})
		x.col++
	}
	return nil
}

// endRow writes the row as many times as it repeats; empty rows, however many, are no
// rows, and a repeat past the bound drops the sheet without writing it out.
func (x *odsReader) endRow() error {
	if len(x.cells) == 0 {
		return nil
	}
	if x.out.rows+x.rowRep > MaxSheetRows {
		x.out.rows, x.out.over = MaxSheetRows+1, true
		x.out.sheet.Reset()
		return nil
	}
	for i := 0; i < x.rowRep; i++ {
		if err := x.out.row(x.cells); err != nil {
			return err
		}
	}
	return nil
}

// csvText reads a CSV file as one sheet named after it.
func csvText(name string, b []byte) (string, error) {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(b) {
		return "", errors.New("folder: only UTF-8 CSV is read")
	}
	cr := csv.NewReader(bytes.NewReader(b))
	cr.Comma = delimiter(b)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.ReuseRecord = true
	out := newSheetOut()
	out.start(name)
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if len(rec) > maxColumns {
			return "", fmt.Errorf("folder: a row wider than a sheet")
		}
		cells := make([]sheetCell, 0, len(rec))
		for i, v := range rec {
			cells = append(cells, sheetCell{i + 1, clean(v)})
		}
		if err := out.row(cells); err != nil {
			return "", err
		}
		if out.over {
			return "", errTooManyRows // the only sheet passed the bound: name only
		}
	}
	if err := out.end(); err != nil {
		return "", err
	}
	return out.b.String(), nil
}

// delimiter is the one of comma, semicolon and tab the first line uses most, outside
// quotes; comma when none.
func delimiter(b []byte) rune {
	counts := map[byte]int{}
	q := false
	for _, c := range b {
		if c == '\n' && !q {
			break
		}
		switch {
		case c == '"':
			q = !q
		case !q && (c == ',' || c == ';' || c == '\t'):
			counts[c]++
		}
	}
	best, n := byte(','), counts[',']
	for _, c := range []byte{';', '\t'} {
		if counts[c] > n {
			best, n = c, counts[c]
		}
	}
	return rune(best)
}
