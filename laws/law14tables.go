package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	memindex "example.com/second-brain/internal/adapters/index/memory"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-14 for numbers from tables (SB-4307): through the real ask path, with a model that
// plans a random operation and then states the tool's value and a number it made up, the
// answer's table value is the operation computed over the sheet, and every number a claim
// on the sheet states is in the sheet or is the tool's own (its value, rows or range).

// TableSheet is one generated sheet: customers and amounts.
type TableSheet struct {
	Rows [][2]string
}

// TableAsk asks one numeric question over a sheet, with the model's plan and made-up number.
type TableAsk func(sh TableSheet, plan string, madeUp string) (ask.Response, error)

// tableModel plans the table tool, then claims the tool's passage and a made-up number.
type tableModel struct{ plan, madeUp string }

var tableLabel = regexp.MustCompile(`<<data cite="(C[0-9]+) [^"]*" flags="[^"]*">>\n([^\n]*)`)

func (m *tableModel) Complete(_ context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	if strings.Contains(req.System, "for the table tool") {
		return ports.ModelResponse{Text: m.plan}, nil
	}
	type claim struct {
		Text  string   `json:"text"`
		Cites []string `json:"cites"`
	}
	var cs []claim
	sheet := ""
	for _, b := range tableLabel.FindAllStringSubmatch(req.Messages[len(req.Messages)-1].Content, -1) {
		if strings.HasPrefix(b[2], "Table tool result:") {
			cs = append(cs, claim{Text: b[2], Cites: []string{b[1]}})
		} else if sheet == "" {
			sheet = b[1]
		}
	}
	if sheet != "" {
		cs = append(cs, claim{Text: "The invoices come to " + m.madeUp + ".", Cites: []string{sheet}})
	}
	b, err := json.Marshal(map[string]any{"claims": cs})
	return ports.ModelResponse{Text: string(b)}, err
}

// RealTables is the system under test: the real ask agent over one sheet the asker may open.
func RealTables(sh TableSheet, plan, madeUp string) (ask.Response, error) {
	ctx := context.Background()
	ev, err := LoadPolicies()
	if err != nil {
		return ask.Response{}, err
	}
	text := "Sheet: Invoices\nCustomer\tAmount"
	for _, r := range sh.Rows {
		text += "\n" + r[0] + "\t" + r[1]
	}
	idx := memindex.New()
	c := ports.Chunk{ID: "chk_inv", Tenant: spaceTenant, Cite: "docs:invoices.csv@1", Kind: "docs", SourceID: "invoices.csv", Version: 1,
		Start: 0, End: len([]rune(text)), Text: text, ACLPrefilter: []string{"anyone"}, Title: "Invoices"}
	if err := idx.Replace(ctx, spaceTenant, ports.SourceRef{Kind: "docs", ID: "invoices.csv"}, 1, []ports.Chunk{c}); err != nil {
		return ask.Response{}, err
	}
	src := &askSpaces{}
	src.spec.Docs = []SpaceDoc{{ID: "invoices.csv", Viewers: []string{"usr_1"}}}
	n := 0
	agent := ask.Agent{Tenant: spaceTenant, Release: "rel_law", Index: idx.Bind(spaceTenant),
		Auth:  retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": viewers{src}}, Policy: policy.Kernel{Engine: ev, Recorder: okRecorder{}}},
		Model: &tableModel{plan: plan, madeUp: madeUp}, Store: tracemem.New(),
		Killed: func(context.Context) bool { return false },
		NewID:  func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Now:    time.Now}
	s, err := tenancy.NewIssuer(spaceTenant).Issue(spaceTenant, "usr_1", []string{"staff"})
	if err != nil {
		return ask.Response{}, err
	}
	return agent.Ask(ctx, s, ask.Request{Question: "What is the total Amount on the invoices?"})
}

var lawNumber = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?`)

func law14Tables(r *rand.Rand, run TableAsk) error {
	customers := []string{"acme-logistics", "beta-freight", "gamma-parts"}
	var sh TableSheet
	for i := 0; i < 2+r.Intn(8); i++ {
		sh.Rows = append(sh.Rows, [2]string{customers[r.Intn(3)], strconv.Itoa(1+r.Intn(5000)) + []string{"", ".5", ".25"}[r.Intn(3)]})
	}
	ops := []string{ask.OpCount, ask.OpSum, ask.OpAverage, ask.OpMin, ask.OpMax}
	op := ops[r.Intn(len(ops))]
	filter := ""
	if r.Intn(2) == 0 {
		filter = sh.Rows[r.Intn(len(sh.Rows))][0]
	}
	where := []map[string]string{}
	if filter != "" {
		where = append(where, map[string]string{"column": "Customer", "equals": filter})
	}
	plan, _ := json.Marshal(map[string]any{"doc": "docs:invoices.csv", "sheet": "Invoices", "op": op, "column": "Amount", "where": where})
	// The oracle: the operation over the rows the filter keeps.
	var vals []float64
	for _, row := range sh.Rows {
		if filter == "" || row[0] == filter {
			v, _ := strconv.ParseFloat(row[1], 64)
			vals = append(vals, v)
		}
	}
	want := float64(len(vals))
	switch op {
	case ask.OpSum, ask.OpAverage:
		want = 0
		for _, v := range vals {
			want += v
		}
		if op == ask.OpAverage {
			want /= float64(len(vals))
		}
	case ask.OpMin, ask.OpMax:
		want = vals[0]
		for _, v := range vals {
			if (op == ask.OpMin && v < want) || (op == ask.OpMax && v > want) {
				want = v
			}
		}
	}
	madeUp := strconv.Itoa(900000 + r.Intn(99999)) // never in a sheet of amounts up to 5,000
	resp, err := run(sh, string(plan), madeUp)
	if err != nil {
		return fmt.Errorf("ask: %v", err)
	}
	if resp.Table == nil || math.Abs(resp.Table.Value-want) > 1e-6 {
		return fmt.Errorf("%s over %d rows: the table tool says %+v, want %v", op, len(vals), resp.Table, want)
	}
	allowed := map[string]bool{}
	for _, row := range sh.Rows {
		allowed[canonNumber(row[1])] = true
	}
	allowed[canonNumber(strconv.FormatFloat(resp.Table.Value, 'f', -1, 64))] = true
	allowed[strconv.Itoa(resp.Table.Rows)] = true
	for _, n := range lawNumber.FindAllString(resp.Table.Range, -1) {
		allowed[canonNumber(n)] = true
	}
	for _, cl := range resp.Answer.Claims {
		onSheet := false
		for _, ci := range cl.Cites {
			onSheet = onSheet || strings.HasPrefix(ci.Cite, "docs:invoices.csv@")
		}
		if !onSheet {
			continue
		}
		for _, n := range lawNumber.FindAllString(cl.Text, -1) {
			if !allowed[canonNumber(n)] {
				return fmt.Errorf("a claim on the sheet states %s, which neither the sheet nor the table tool holds: %q", n, cl.Text)
			}
		}
	}
	return nil
}

// canonNumber is a number as a canonical string, commas dropped.
func canonNumber(n string) string {
	f, err := strconv.ParseFloat(strings.ReplaceAll(n, ",", ""), 64)
	if err != nil {
		return n
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
