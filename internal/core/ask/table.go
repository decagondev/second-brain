package ask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/prompts"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Numbers from tables (SB-4307): the ask agent does a sheet's arithmetic in code. The model
// sees only the sheets' names and column names and plans at most one operation; the table
// tool runs it over the chunks the asker may open, and its result is a passage with the
// sheet's own cite. A claim that cites a sheet and states a number neither its passages nor
// the tool hold is dropped (LAW-14).

// Table tool operations and limits.
const (
	OpCount   = "count"
	OpSum     = "sum"
	OpAverage = "average"
	OpMin     = "min"
	OpMax     = "max"
	// MaxTableFilters is the most filters a plan may have, and MaxTableSheets the most
	// sheet documents the planner is shown.
	MaxTableFilters = 3
	MaxTableSheets  = 3
)

var tableOps = map[string]bool{OpCount: true, OpSum: true, OpAverage: true, OpMin: true, OpMax: true}

// TableFilter keeps the rows whose column equals a value.
type TableFilter struct {
	Column string `json:"column"`
	Equals string `json:"equals"`
}

// TableResult is the table tool's result, as the answer carries it.
type TableResult struct {
	Cite   string        `json:"cite"`
	Sheet  string        `json:"sheet"`
	Op     string        `json:"op"`
	Column string        `json:"column,omitempty"`
	Where  []TableFilter `json:"where"`
	Value  float64       `json:"value"`
	Rows   int           `json:"rows"`
	Range  string        `json:"range"`
}

// tablePlan is the model's plan: one operation over one sheet.
type tablePlan struct {
	Doc    string        `json:"doc"`
	Sheet  string        `json:"sheet"`
	Where  []TableFilter `json:"where"`
	Op     string        `json:"op"`
	Column string        `json:"column"`
}

var numericQuestion = regexp.MustCompile(`(?i)\b(how many|how much|total|sum|average|mean|count|minimum|maximum|highest|lowest|largest|smallest)\b`)

// NumericQuestion is whether a question asks for a number the table tool can compute.
func NumericQuestion(q string) bool { return numericQuestion.MatchString(q) }

// IsSheet is whether a chunk is of a spreadsheet, by its source id's extension.
func IsSheet(c ports.Chunk) bool {
	switch strings.ToLower(path.Ext(c.SourceID)) {
	case ".xlsx", ".ods", ".csv":
		return true
	}
	return false
}

// sheet is one sheet of a document, rebuilt: its name and rows, the first the columns.
type sheet struct {
	Name string
	Rows [][]string
}

// wholeText rebuilds one version's normalised text from its chunks, in order: a gap of one
// rune was a space the chunker cut at, a longer one a paragraph break. A gap longer than
// maxGap means a chunk isn't there (a secret dropped it), so the text isn't whole.
func wholeText(chunks []ports.Chunk) (string, bool) {
	const maxGap = 4
	cs := append([]ports.Chunk(nil), chunks...)
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].Start < cs[j].Start })
	var b strings.Builder
	at := 0
	for i, c := range cs {
		if i > 0 && c.Start < at {
			return "", false // overlapping chunks: not one version's
		}
		gap := c.Start - at
		switch {
		case i == 0 && c.Start != 0, gap > maxGap:
			return "", false
		case i > 0 && gap == 1:
			b.WriteString(" ")
		case i > 0 && gap > 1:
			b.WriteString("\n\n")
		}
		b.WriteString(c.Text)
		at = c.End
	}
	return b.String(), len(cs) > 0
}

// sheetsOf reads "Sheet: NAME" blocks, each row a line of tab-separated cells.
func sheetsOf(text string) []sheet {
	var out []sheet
	for _, block := range strings.Split(text, "\n\n") {
		lines := strings.Split(block, "\n")
		name, ok := strings.CutPrefix(lines[0], "Sheet: ")
		if !ok {
			continue
		}
		sh := sheet{Name: strings.TrimSpace(name)}
		for _, l := range lines[1:] {
			sh.Rows = append(sh.Rows, strings.Split(l, "\t"))
		}
		if len(sh.Rows) > 0 {
			out = append(out, sh)
		}
	}
	return out
}

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func columnOf(header []string, name string) int {
	for i, h := range header {
		if norm(h) == norm(name) && norm(name) != "" {
			return i
		}
	}
	return -1
}

// colName is a column's letters, from 0 (A, B, … Z, AA).
func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

// cellNumber reads a cell as a number: thousands commas and a currency sign are allowed.
func cellNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "£$€")
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// runTable runs a plan over one sheet; false when a column isn't there or nothing matched.
func runTable(sh sheet, p tablePlan) (TableResult, bool) {
	header := sh.Rows[0]
	col := -1
	if p.Op != OpCount || p.Column != "" {
		if col = columnOf(header, p.Column); col < 0 {
			return TableResult{}, false
		}
	}
	type filt struct {
		col int
		val string
	}
	var fs []filt
	for _, f := range p.Where {
		c := columnOf(header, f.Column)
		if c < 0 {
			return TableResult{}, false
		}
		fs = append(fs, filt{c, norm(f.Equals)})
	}
	res := TableResult{Sheet: sh.Name, Op: p.Op, Where: p.Where, Column: p.Column}
	if res.Where == nil {
		res.Where = []TableFilter{}
	}
	var vals []float64
	for _, row := range sh.Rows[1:] {
		keep := true
		for _, f := range fs {
			if norm(cell(row, f.col)) != f.val {
				keep = false
				break
			}
		}
		if !keep {
			continue
		}
		if p.Op == OpCount {
			res.Rows++
			continue
		}
		if v, ok := cellNumber(cell(row, col)); ok {
			vals = append(vals, v)
			res.Rows++
		}
	}
	switch p.Op {
	case OpCount:
		res.Value = float64(res.Rows)
	case OpSum, OpAverage:
		for _, v := range vals {
			res.Value += v
		}
		if p.Op == OpAverage {
			if len(vals) == 0 {
				return TableResult{}, false
			}
			res.Value /= float64(len(vals))
		}
	case OpMin, OpMax:
		if len(vals) == 0 {
			return TableResult{}, false
		}
		res.Value = vals[0]
		for _, v := range vals[1:] {
			if (p.Op == OpMin && v < res.Value) || (p.Op == OpMax && v > res.Value) {
				res.Value = v
			}
		}
	}
	res.Value = math.Round(res.Value*1e6) / 1e6
	c := colName(max(col, 0))
	res.Range = fmt.Sprintf("%s!%s2:%s%d", sh.Name, c, c, len(sh.Rows))
	return res, true
}

// tableText is the tool's result as a passage.
func tableText(r TableResult) string {
	what := r.Op
	if r.Column != "" {
		what += " of " + r.Column
	} else {
		what += " of rows"
	}
	var ws []string
	for _, w := range r.Where {
		ws = append(ws, w.Column+" is "+w.Equals)
	}
	if len(ws) > 0 {
		what += " where " + strings.Join(ws, " and ")
	}
	return fmt.Sprintf("Table tool result: %s, over %s: %s (%d rows).", what, r.Range, formatNumber(r.Value), r.Rows)
}

func formatNumber(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// sheetDocs are the spreadsheet documents among the allowed chunks, by cite, in rank order.
func sheetDocs(chunks []ports.Chunk) []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range chunks {
		if IsSheet(c) && !seen[c.Cite] {
			seen[c.Cite] = true
			out = append(out, c.Cite)
			if len(out) == MaxTableSheets {
				break
			}
		}
	}
	return out
}

// sheetChunks are every stored chunk of one version the asker may open now, checked live.
func (a Agent) sheetChunks(ctx context.Context, s tenancy.Scope, cite string) ([]ports.Chunk, error) {
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	doc, _, _ := strings.Cut(cite, "@")
	kind, id, _ := strings.Cut(doc, ":")
	all, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var mine []ports.Chunk
	for _, c := range all {
		if c.Cite == cite {
			mine = append(mine, c)
		}
	}
	if len(mine) == 0 {
		return nil, nil
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return nil, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, mine, len(mine))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(res.Context) != len(mine) {
		return nil, nil // not every chunk is open to them: not the whole sheet
	}
	return res.Context, nil
}

// table runs the table tool for a numeric question over the allowed chunks: its result and
// the passage that carries it, or nil when it doesn't apply or can't run whole.
func (a Agent) table(ctx context.Context, s tenancy.Scope, q string, allowed []ports.Chunk) (*TableResult, *ports.Chunk, error) {
	if !NumericQuestion(q) {
		return nil, nil, nil
	}
	cites := sheetDocs(allowed)
	if len(cites) == 0 {
		return nil, nil, nil
	}
	docs := map[string][]sheet{}
	byDoc := map[string]ports.Chunk{}
	var blocks []string
	for _, cite := range cites {
		chunks, err := a.sheetChunks(ctx, s, cite)
		if err != nil {
			return nil, nil, err
		}
		text, whole := wholeText(chunks)
		if !whole {
			continue
		}
		shs := sheetsOf(text)
		if len(shs) == 0 {
			continue
		}
		doc, _, _ := strings.Cut(cite, "@")
		docs[doc] = shs
		last := chunks[0]
		for _, c := range chunks {
			if c.End > last.End {
				last = c
			}
		}
		first := chunks[0]
		first.End = last.End
		byDoc[doc] = first
		var heads []string
		for _, sh := range shs {
			heads = append(heads, "Sheet: "+sh.Name+"\nColumns: "+strings.Join(sh.Rows[0], " | "))
		}
		blocks = append(blocks, ingest.Wrap("Document "+doc+" ("+first.Title+")\n"+strings.Join(heads, "\n"), doc, nil))
	}
	if len(blocks) == 0 {
		return nil, nil, nil
	}
	p, err := prompts.Get("table")
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	out, err := a.Model.Complete(ctx, ports.ModelRequest{System: p.Text, JSON: true, User: s.User(), Space: s.Space(),
		Messages: []ports.Message{{Role: "user", Content: strings.Join(blocks, "\n") + "\nQuestion: " + q}}})
	if err != nil {
		if errors.Is(err, budget.ErrBudget) {
			return nil, nil, fmt.Errorf("%w: %v", ErrBudget, err)
		}
		return nil, nil, fmt.Errorf("%w: model: %v", ErrUnavailable, err)
	}
	plan, ok := readPlan(out.Text)
	if !ok {
		return nil, nil, nil
	}
	plan.Doc, _, _ = strings.Cut(strings.TrimSpace(plan.Doc), "@")
	shs, ok := docs[plan.Doc]
	if !ok {
		return nil, nil, nil
	}
	for _, sh := range shs {
		if norm(sh.Name) != norm(plan.Sheet) {
			continue
		}
		res, ok := runTable(sh, plan)
		if !ok {
			return nil, nil, nil
		}
		ch := byDoc[plan.Doc]
		res.Cite = ch.Cite
		ch.ID = "table_" + ch.ID
		ch.Text = tableText(res)
		ch.Section = ""
		return &res, &ch, nil
	}
	return nil, nil, nil
}

// readPlan reads the model's plan; anything malformed is no plan.
func readPlan(text string) (tablePlan, bool) {
	var p tablePlan
	if json.Unmarshal([]byte(strings.TrimSpace(text)), &p) != nil {
		return p, false
	}
	p.Op = norm(p.Op)
	if !tableOps[p.Op] || p.Doc == "" || p.Sheet == "" || len(p.Where) > MaxTableFilters {
		return p, false
	}
	if p.Op != OpCount && strings.TrimSpace(p.Column) == "" {
		return p, false
	}
	return p, true
}

var numberRe = regexp.MustCompile(`-?\d[\d,]*(?:\.\d+)?`)

// numbersIn are the numbers a text states, commas between thousands dropped.
func numbersIn(text string) []string {
	var out []string
	for _, m := range numberRe.FindAllString(text, -1) {
		out = append(out, strings.TrimLeft(strings.ReplaceAll(m, ",", ""), "-"))
	}
	return out
}

// matchesValue is whether n, as written, is v at n's own precision.
func matchesValue(n string, v float64) bool {
	f, err := strconv.ParseFloat(n, 64)
	if err != nil {
		return false
	}
	dec := 0
	if _, frac, ok := strings.Cut(n, "."); ok {
		dec = len(frac)
	}
	p := math.Pow(10, float64(dec))
	return math.Round(math.Abs(v)*p)/p == f
}

// tableCheck drops each claim that cites a spreadsheet and states a number that is neither
// in the text of the passages it cites nor the tool's value (LAW-14), and counts them.
func tableCheck(claims []answer.Claim, chunks []ports.Chunk, tr *TableResult) ([]answer.Claim, int) {
	texts := map[string][]string{} // cite and start -> the passages' texts
	sheets := map[string]bool{}
	for _, c := range chunks {
		k := c.Cite + "|" + strconv.Itoa(c.Start)
		texts[k] = append(texts[k], c.Text)
		if IsSheet(c) {
			sheets[c.Cite] = true
		}
	}
	var kept []answer.Claim
	dropped := 0
	for _, cl := range claims {
		onSheet := false
		var have []string
		for _, ci := range cl.Cites {
			onSheet = onSheet || sheets[ci.Cite]
			for _, t := range texts[ci.Cite+"|"+strconv.Itoa(ci.Span[0])] {
				have = append(have, numbersIn(t)...)
			}
		}
		if onSheet && !numbersBacked(numbersIn(cl.Text), have, tr) {
			dropped++
			continue
		}
		kept = append(kept, cl)
	}
	if len(kept) == 0 && dropped > 0 {
		kept = []answer.Claim{{Text: NothingToCite, Cites: []answer.Cite{}, Uncited: true}}
	}
	return kept, dropped
}

func numbersBacked(stated, have []string, tr *TableResult) bool {
	in := map[string]bool{}
	for _, h := range have {
		if f, err := strconv.ParseFloat(h, 64); err == nil {
			in[formatNumber(f)] = true
		}
	}
	for _, n := range stated {
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			continue
		}
		if in[formatNumber(f)] || (tr != nil && matchesValue(n, tr.Value)) {
			continue
		}
		return false
	}
	return true
}
