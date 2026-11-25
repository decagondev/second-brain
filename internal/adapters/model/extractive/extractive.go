// Package extractive is a local model for development, tests and evals (sb-cell(5),
// sb-eval(1)): it never calls out, and answers by quoting its sources, so every answer is
// checkable against the context it was given.
package extractive

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// Model answers each data block's first sentence, citing it; with Shift, it cites
// the next block instead (misattribution, for the self-test). Blocks whose first sentences
// open with the same three words are one claim citing them all, marked disagree when the
// sentences differ (SB-3304).
type Model struct{ Shift bool }

var block = regexp.MustCompile(`(?s)<<data cite="(C[0-9]+) [^"]*" flags="[^"]*">>\n(.*?)\n<</data>>`)

func (m Model) Complete(_ context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	last := req.Messages[len(req.Messages)-1].Content
	if strings.Contains(req.System, "for the table tool") { // SB-4307: plan from the words alone
		return ports.ModelResponse{Text: tablePlan(last)}, nil
	}
	bs := block.FindAllStringSubmatch(last, -1)
	type cl struct {
		Text     string   `json:"text"`
		Cites    []string `json:"cites"`
		Disagree bool     `json:"disagree,omitempty"`
	}
	var cs []cl
	opened := map[string]int{} // a claim's first three words -> its index (SB-3304)
	for i, b := range bs {
		label := b[1]
		if m.Shift {
			label = bs[(i+1)%len(bs)][1]
			if len(bs) == 1 {
				label = "C99"
			}
		}
		text := firstSentence(b[2])
		if k := opening(text); k != "" && !m.Shift {
			if j, ok := opened[k]; ok { // two blocks on one subject: one claim citing both
				cs[j].Cites = append(cs[j].Cites, label)
				cs[j].Disagree = cs[j].Disagree || !strings.EqualFold(body(cs[j].Text), body(text))
				continue
			}
			opened[k] = len(cs)
		}
		cs = append(cs, cl{Text: text, Cites: []string{label}})
	}
	if len(cs) == 0 {
		cs = append(cs, cl{Text: "I don't know.", Cites: nil})
	}
	raw, err := json.Marshal(map[string]any{"claims": cs})
	return ports.ModelResponse{Text: string(raw)}, err
}

func firstSentence(t string) string {
	t = strings.TrimSpace(t)
	for _, sep := range []string{". ", ".\n", "? ", "! "} {
		if i := strings.Index(t, sep); i >= 0 {
			return t[:i+1]
		}
	}
	return t
}

// body is a sentence without the title line a chunk starts with.
func body(t string) string {
	if i := strings.LastIndex(t, "\n"); i >= 0 {
		t = t[i+1:]
	}
	return strings.TrimSpace(t)
}

// opening is a sentence's first three words, lower-cased, or "" for a shorter one.
func opening(t string) string {
	ws := strings.Fields(strings.ToLower(body(t)))
	if len(ws) < 4 {
		return ""
	}
	return strings.Join(ws[:3], " ")
}

var (
	sheetBlock = regexp.MustCompile(`(?s)<<data cite="([^"]+)" flags="[^"]*">>\n(.*?)\n<</data>>`)
	sheetLine  = regexp.MustCompile(`(?m)^Sheet: (.*)\nColumns: (.*)$`)
	planOps    = []struct{ word, op string }{{"average", "average"}, {"mean", "average"}, {"total", "sum"}, {"sum", "sum"},
		{"how much", "sum"}, {"highest", "max"}, {"largest", "max"}, {"maximum", "max"}, {"lowest", "min"},
		{"smallest", "min"}, {"minimum", "min"}, {"how many", "count"}, {"count", "count"}}
)

// tablePlan plans the table tool from the question's words (SB-4307): the operation its
// first matching word names, over the longest column it names, in the first sheet with
// one; with no filters. Nothing it can match is {"op": ""}.
func tablePlan(msg string) string {
	_, q, _ := strings.Cut(msg, "\nQuestion: ")
	lq := strings.ToLower(q)
	op := ""
	for _, o := range planOps {
		if strings.Contains(lq, o.word) {
			op = o.op
			break
		}
	}
	for _, b := range sheetBlock.FindAllStringSubmatch(msg, -1) {
		for _, sh := range sheetLine.FindAllStringSubmatch(b[2], -1) {
			col := ""
			for _, c := range strings.Split(sh[2], " | ") {
				if c = strings.TrimSpace(c); c != "" && strings.Contains(lq, strings.ToLower(c)) && len(c) > len(col) {
					col = c
				}
			}
			if op != "" && (col != "" || op == "count") {
				raw, _ := json.Marshal(map[string]any{"doc": b[1], "sheet": strings.TrimSpace(sh[1]), "op": op, "column": col, "where": []any{}})
				return string(raw)
			}
		}
	}
	return `{"op": ""}`
}
