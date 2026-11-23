package folder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

const workbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Rates" sheetId="1" r:id="rId1"/><sheet name="Chart" sheetId="2" r:id="rId3"/><sheet name="Depots" sheetId="3" r:id="rId2"/></sheets></workbook>`

const workbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="/xl/worksheets/sheet2.xml"/>
<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/chartsheet" Target="chartsheets/sheet1.xml"/>
</Relationships>`

const sharedXML = `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>Lane</t></si><si><t>Rate</t></si>` +
	`<si><r><t>Hamburg</t></r><r><t xml:space="preserve"> to Lyon</t></r><rPh><t>PHONETIC</t></rPh></si><si><t>Depot</t></si><si><t>Bay 4
night</t></si></sst>`

const sheet1 = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
<row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2"><f>SUM(C2:C9)*FORMULATEXT</f><v>412.5</v></c><c r="D2" t="b"><v>1</v></c></row>
<row r="3"/>
<row r="5"><c r="C5" t="inlineStr"><is><t>inline note</t></is></c></row>
</sheetData></worksheet>`

const sheet2 = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
<row r="1"><c r="A1" t="s"><v>3</v></c></row><row r="2"><c r="A2" t="s"><v>4</v></c><c r="B2" t="str"><f>CONCAT("x")</f><v>cached text</v></c></row>
</sheetData></worksheet>`

func xlsxFile(t testing.TB, extra ...string) []byte {
	parts := []string{"[Content_Types].xml", "<Types/>", "xl/workbook.xml", workbook, "xl/_rels/workbook.xml.rels", workbookRels,
		"xl/sharedStrings.xml", sharedXML, "xl/worksheets/sheet1.xml", sheet1, "xl/worksheets/sheet2.xml", sheet2,
		"xl/chartsheets/sheet1.xml", "<chartsheet>CHART TEXT</chartsheet>", "xl/vbaProject.bin", "MACRO TEXT",
		"xl/comments1.xml", "<comments><comment><text><t>COMMENT TEXT</t></text></comment></comments>",
		"xl/externalLinks/externalLink1.xml", "<externalLink>EXTERNAL TEXT</externalLink>"}
	return archive(t, append(parts, extra...)...)
}

const odsContentXML = `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" xmlns:draw="d">
<office:scripts><office:script>MACRO TEXT</office:script></office:scripts>
<office:body><office:spreadsheet>
<table:table table:name="Rates">
<table:table-row><table:table-cell office:value-type="string"><text:p>Lane</text:p></table:table-cell><table:table-cell><text:p>Rate</text:p></table:table-cell><table:table-cell table:number-columns-repeated="1022"/></table:table-row>
<table:table-row><table:table-cell><text:p>Hamburg<text:s text:c="2"/>to Lyon</text:p><office:annotation><text:p>COMMENT TEXT</text:p></office:annotation></table:table-cell><table:table-cell table:formula="of:=SUM([.C2:.C9])" office:value-type="float" office:value="412.5"><text:p>412.50</text:p></table:table-cell><table:table-cell/><table:table-cell office:value-type="boolean" office:boolean-value="true"/><table:table-cell table:number-columns-repeated="1020"/></table:table-row>
<table:table-row table:number-rows-repeated="1048570"><table:table-cell table:number-columns-repeated="1024"/></table:table-row>
</table:table>
<table:table table:name="Depots"><table:table-row table:number-rows-repeated="2"><table:table-cell table:number-columns-repeated="2"><text:p>x</text:p></table:table-cell><table:table-cell><draw:frame><draw:object><text:p>OBJECT TEXT</text:p></draw:object></draw:frame></table:table-cell></table:table-row></table:table>
</office:spreadsheet></office:body></office:document-content>`

func TestSheetText(t *testing.T) {
	for _, c := range []struct {
		name string
		b    []byte
		want string
	}{
		{"rates.xlsx", xlsxFile(t), "Sheet: Rates\nLane\tRate\nHamburg to Lyon\t412.5\t\tTRUE\n\t\tinline note\n\nSheet: Depots\nDepot\nBay 4 night\tcached text"},
		{"rates.ods", archive(t, "mimetype", "application/vnd.oasis.opendocument.spreadsheet", "content.xml", odsContentXML, "Basic/Standard/Module1.xml", "MACRO TEXT"),
			"Sheet: Rates\nLane\tRate\nHamburg  to Lyon\t412.50\t\ttrue\n\nSheet: Depots\nx\tx\nx\tx"},
		{"lanes.csv", []byte("\xef\xbb\xbfLane,Rate,Note\r\nHamburg to Lyon,=SUM(B2:B9),\"two\nlines, quoted\"\r\n,,\r\nBay 4,12\r\n"),
			"Sheet: lanes\nLane\tRate\tNote\nHamburg to Lyon\t=SUM(B2:B9)\ttwo lines, quoted\nBay 4\t12"},
		{"tarife.csv", []byte("Strecke;Preis\nHamburg, Lyon;412,50\n"), "Sheet: tarife\nStrecke\tPreis\nHamburg, Lyon\t412,50"},
		{"tabs.csv", []byte("a\tb\tc\n1\t2\t3\n"), "Sheet: tabs\na\tb\tc\n1\t2\t3"},
	} {
		got := OfficeText(c.name, c.b)
		if got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
		for _, hidden := range []string{"MACRO", "COMMENT", "CHART", "EXTERNAL", "OBJECT", "PHONETIC", "FORMULATEXT", "of:="} {
			if strings.Contains(got, hidden) {
				t.Errorf("%s: read a part it must never read: %q", c.name, hidden)
			}
		}
	}
}

func TestSheetPastTheBoundIsSkippedWhole(t *testing.T) {
	var rows strings.Builder
	for i := 1; i <= MaxSheetRows+1; i++ {
		fmt.Fprintf(&rows, `<row r="%d"><c r="A%d"><v>%d</v></c></row>`, i, i, i)
	}
	big := `<worksheet><sheetData>` + rows.String() + `</sheetData></worksheet>`
	b := archive(t, "xl/workbook.xml", workbook, "xl/_rels/workbook.xml.rels", workbookRels, "xl/sharedStrings.xml", sharedXML,
		"xl/worksheets/sheet1.xml", big, "xl/worksheets/sheet2.xml", sheet2)
	if got, want := OfficeText("big.xlsx", b), "Sheet: Depots\nDepot\nBay 4 night\tcached text"; got != want {
		t.Errorf("a sheet past %d rows isn't skipped whole, the others kept:\n got %q\nwant %q", MaxSheetRows, got, want)
	}
	odsBig := strings.Replace(odsContentXML, `<table:table-row><table:table-cell office:value-type="string">`,
		fmt.Sprintf(`<table:table-row table:number-rows-repeated="%d"><table:table-cell><text:p>r</text:p></table:table-cell></table:table-row><table:table-row><table:table-cell office:value-type="string">`, MaxSheetRows), 1)
	if got := OfficeText("big.ods", archive(t, "content.xml", odsBig)); got != "Sheet: Depots\nx\tx\nx\tx" {
		t.Errorf("an .ods sheet past the bound: %q", got)
	}
	var csvRows strings.Builder
	for i := 0; i <= MaxSheetRows; i++ {
		fmt.Fprintf(&csvRows, "%d,x\n", i)
	}
	if got := OfficeText("big.csv", []byte(csvRows.String())); got != "" {
		t.Errorf("a CSV past the bound is read: %d bytes", len(got))
	}
	if got := OfficeText("ok.csv", []byte(strings.Repeat("1,x\n", MaxSheetRows))); strings.Count(got, "\n") != MaxSheetRows {
		t.Errorf("a CSV of exactly %d rows isn't read whole", MaxSheetRows)
	}
}

func TestSheetTextSkipsWhole(t *testing.T) {
	wide := `<worksheet><sheetData><row><c r="XFE1"><v>1</v></c></row></sheetData></worksheet>`
	huge := `<worksheet><sheetData><row><c r="A1" t="inlineStr"><is><t>` + strings.Repeat("A", MaxOfficeText+1) + `</t></is></c></row></sheetData></worksheet>`
	repeated := `<office:document-content xmlns:office="o" xmlns:table="t" xmlns:text="x"><office:body><office:spreadsheet><table:table table:name="S"><table:table-row table:number-rows-repeated="5000"><table:table-cell table:number-columns-repeated="16000"><text:p>` +
		strings.Repeat("A", 64) + `</text:p></table:table-cell></table:table-row></table:table></office:spreadsheet></office:body></office:document-content>`
	for name, b := range map[string][]byte{
		"wide.xlsx":       archive(t, "xl/workbook.xml", workbook, "xl/_rels/workbook.xml.rels", workbookRels, "xl/worksheets/sheet1.xml", wide),
		"huge.xlsx":       archive(t, "xl/workbook.xml", workbook, "xl/_rels/workbook.xml.rels", workbookRels, "xl/worksheets/sheet1.xml", huge),
		"noworkbook.xlsx": archive(t, "xl/worksheets/sheet1.xml", sheet1),
		"broken.xlsx":     []byte("PK\x03\x04 not really"),
		"bad.ods":         archive(t, "content.xml", "<office:document-content><table:table><table:table-row>half"),
		"repeated.ods":    archive(t, "content.xml", repeated),
		"latin1.csv":      []byte("Stra\xdfe,Preis\n"),
	} {
		if got := OfficeText(name, b); got != "" {
			t.Errorf("%s is half-read: %.80q", name, got)
		}
	}
}

func TestSpreadsheetsAreDocuments(t *testing.T) {
	dir := t.TempDir()
	for name, b := range map[string][]byte{"rates.xlsx": xlsxFile(t), "lanes.csv": []byte("Lane,Rate\nHamburg,412\n")} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".acl"), []byte("read ann@acme.example\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"rates.xlsx", "lanes.csv", "x.ods"} {
		if !Document(p) {
			t.Errorf("%s isn't a document", p)
		}
	}
	c, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	for id, want := range map[string]string{"rates.xlsx": "Hamburg to Lyon\t412.5", "lanes.csv": "Sheet: lanes\nLane\tRate"} {
		d, err := c.Fetch(context.Background(), ports.SourceRef{Kind: "docs", ID: id})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(d.Text, want) {
			t.Errorf("%s: %q", id, d.Text)
		}
	}
}

func FuzzSheetText(f *testing.F) {
	f.Add("a.xlsx", xlsxFile(f))
	f.Add("a.ods", archive(f, "content.xml", odsContentXML))
	f.Add("a.csv", []byte("a,b\n\"c\nd\",e\n"))
	f.Add("a.csv", []byte("x;y;z\n1;2;3\n"))
	f.Fuzz(func(t *testing.T, name string, b []byte) {
		for _, ext := range []string{".xlsx", ".ods", ".csv"} {
			s := OfficeText(name+ext, b)
			if len(s) > MaxOfficeText {
				t.Fatalf("%s: %d bytes of text past the bound", ext, len(s))
			}
			if s != "" && !strings.HasPrefix(s, "Sheet:") {
				t.Fatalf("%s: text that isn't a sheet: %.40q", ext, s)
			}
			if strings.Count(s, "\n") > 4*(MaxSheetRows+2)*(1+strings.Count(s, "\n\nSheet: ")) {
				t.Fatalf("%s: more rows than the bound", ext)
			}
		}
	})
}
