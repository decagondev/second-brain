package ask

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// SB-4307: the sheet is rebuilt from its chunks whole, or not at all.
func TestWholeText(t *testing.T) {
	text := ingest.Normalise("Sheet: Invoices\nCustomer\tAmount\nacme-logistics\t1200\nbeta-freight\t300.50\n\nSheet: Other\nA\tB\n1\t2")
	rs := []rune(text)
	ai, oi := strings.Index(text, "acme"), strings.Index(text, "\n\nSheet: Other")
	if ai < 0 || oi < 0 {
		t.Fatal("fixture")
	}
	i, j := len([]rune(text[:ai])), len([]rune(text[:oi]))
	var chunks []ports.Chunk
	for _, sp := range []ingest.Span{{Start: 0, End: i}, {Start: i, End: j}, {Start: j + 2, End: len(rs)}} {
		chunks = append(chunks, ports.Chunk{Start: sp.Start, End: sp.End, Text: string(rs[sp.Start:sp.End])})
	}
	got, ok := wholeText([]ports.Chunk{chunks[2], chunks[0], chunks[1]})
	if !ok {
		t.Fatal("not whole")
	}
	shs := sheetsOf(got)
	if len(shs) != 2 || shs[0].Name != "Invoices" || len(shs[0].Rows) != 3 || shs[1].Name != "Other" {
		t.Fatalf("sheets %+v from %q", shs, got)
	}
	if _, ok := wholeText([]ports.Chunk{chunks[0], chunks[2]}); ok {
		t.Error("a missing chunk was taken as whole")
	}
}

func TestRunTable(t *testing.T) {
	sh := sheet{Name: "Invoices", Rows: [][]string{{"Customer", "Amount"}, {"acme-logistics", "1,200"}, {"beta-freight", "300.50"},
		{"Acme-Logistics ", "800"}, {"gamma-parts", "n/a"}}}
	for _, c := range []struct {
		plan tablePlan
		want float64
		rows int
	}{
		{tablePlan{Op: OpSum, Column: "amount"}, 2300.5, 3},
		{tablePlan{Op: OpSum, Column: "Amount", Where: []TableFilter{{Column: "customer", Equals: "acme-logistics"}}}, 2000, 2},
		{tablePlan{Op: OpCount}, 4, 4},
		{tablePlan{Op: OpAverage, Column: "Amount", Where: []TableFilter{{Column: "Customer", Equals: "acme-logistics"}}}, 1000, 2},
		{tablePlan{Op: OpMax, Column: "Amount"}, 1200, 3},
		{tablePlan{Op: OpMin, Column: "Amount"}, 300.5, 3},
	} {
		r, ok := runTable(sh, c.plan)
		if !ok || r.Value != c.want || r.Rows != c.rows || r.Range == "" {
			t.Errorf("%+v: %+v %v", c.plan, r, ok)
		}
	}
	if r, _ := runTable(sh, tablePlan{Op: OpSum, Column: "Amount"}); r.Range != "Invoices!B2:B5" {
		t.Errorf("range %q", r.Range)
	}
	if _, ok := runTable(sh, tablePlan{Op: OpSum, Column: "Price"}); ok {
		t.Error("a missing column ran")
	}
	if colName(0) != "A" || colName(25) != "Z" || colName(26) != "AA" {
		t.Error("column names")
	}
}

func TestReadPlan(t *testing.T) {
	if _, ok := readPlan(`{"doc":"docs:x.csv","sheet":"S","op":"SUM","column":"Amount"}`); !ok {
		t.Error("a good plan was refused")
	}
	for _, bad := range []string{`{"op":""}`, `nope`, `{"doc":"d","sheet":"S","op":"median","column":"A"}`, `{"doc":"d","sheet":"S","op":"sum"}`,
		`{"doc":"d","sheet":"S","op":"count","where":[{},{},{},{}]}`} {
		if _, ok := readPlan(bad); ok {
			t.Errorf("%s was taken", bad)
		}
	}
}

// The composer check (LAW-14): a number on a sheet's claim must be in its passages or be
// the tool's value; a claim on another document is left alone.
func TestTableCheck(t *testing.T) {
	sheetChunk := ports.Chunk{Cite: "docs:inv.csv@1", SourceID: "inv.csv", Start: 0, Text: "Sheet: Invoices\nCustomer\tAmount\nacme\t1200\nbeta\t300.5"}
	tool := sheetChunk
	tool.Text = "Table tool result: sum of Amount, over Invoices!B2:B3: 1500.5 (2 rows)."
	doc := ports.Chunk{Cite: "docs:memo@2", SourceID: "memo", Start: 0, Text: "The memo."}
	cite := func(c ports.Chunk) []answer.Cite { return []answer.Cite{{Cite: c.Cite, Span: [2]int{c.Start, c.End}}} }
	tr := &TableResult{Value: 1500.5}
	claims := []answer.Claim{
		{Text: "The invoices total 1,500.50.", Cites: cite(tool)},
		{Text: "acme owes 1200.", Cites: cite(sheetChunk)},
		{Text: "The invoices total 1600.", Cites: cite(sheetChunk)},
		{Text: "The memo says 42.", Cites: cite(doc)},
	}
	kept, dropped := tableCheck(claims, []ports.Chunk{tool, sheetChunk, doc}, tr)
	if dropped != 1 || len(kept) != 3 || kept[2].Text != "The memo says 42." {
		t.Fatalf("kept %+v dropped %d", kept, dropped)
	}
	if kept, dropped := tableCheck(claims[2:3], []ports.Chunk{sheetChunk}, nil); dropped != 1 || !kept[0].Uncited {
		t.Fatalf("an answer of only a made-up number: %+v", kept)
	}
	if !matchesValue("33.33", 33.3333) || matchesValue("33.3", 33.36) || !matchesValue("2400", 2400) {
		t.Error("precision")
	}
}

func TestNumericQuestion(t *testing.T) {
	if !NumericQuestion("What is the total amount?") || !NumericQuestion("How many invoices are open?") || NumericQuestion("Who approves refunds?") {
		t.Error("numeric questions")
	}
}
