package ask

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Terms the documents define (sb-ask(3), Terms the documents define; SB-3407).
const (
	MaxDefined     = 20
	DefinedMinDocs = 2
)

// DefinedTerm is a term the space's documents define, and in how many of them.
type DefinedTerm struct {
	Term    string `json:"term"`
	Meaning string `json:"meaning"`
	Docs    int    `json:"docs"`
}

var (
	longShort = regexp.MustCompile(`((?:[A-Za-z][a-z]* ){1,5}[A-Za-z][a-z]*) \(([A-Z]{2,6})\)`)
	shortLong = regexp.MustCompile(`\b([A-Z]{2,6}) \(([A-Za-z][a-z]*(?: [A-Za-z][a-z]*){1,5})\)`)
)

// definitions are the (short, long) pairs written in text, where the long form's words'
// initials spell the short form.
func definitions(text string) [][2]string {
	var out [][2]string
	for _, m := range longShort.FindAllStringSubmatch(text, -1) {
		ws := strings.Fields(m[1])
		if n := len(m[2]); len(ws) >= n {
			if long := strings.Join(ws[len(ws)-n:], " "); initials(long) == m[2] {
				out = append(out, [2]string{m[2], strings.ToLower(long)})
			}
		}
	}
	for _, m := range shortLong.FindAllStringSubmatch(text, -1) {
		if initials(m[2]) == m[1] {
			out = append(out, [2]string{m[1], strings.ToLower(m[2])})
		}
	}
	return out
}

func initials(long string) string {
	var b strings.Builder
	for _, w := range strings.Fields(long) {
		b.WriteString(strings.ToUpper(w[:1]))
	}
	return b.String()
}

// termsDefined is the terms at least DefinedMinDocs of the space's documents the steward can
// open now define, and that aren't in its glossary. A chunk flagged instruction_like is
// never read, and a term whose text would be flagged is skipped (LAW-09).
func (x Stewards) termsDefined(ctx context.Context, s tenancy.Scope, lk ports.DocLookup, name string) ([]DefinedTerm, error) {
	out := []DefinedTerm{}
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Space: SpaceKey(name)})
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	cs = latest(cs)
	firstOf := firsts(cs)
	if len(firstOf) == 0 {
		return out, nil
	}
	cand := make([]ports.Chunk, 0, len(firstOf))
	for _, c := range firstOf {
		cand = append(cand, c)
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].ID < cand[j].ID })
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, cand, len(cand))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	open := map[string]bool{}
	for _, c := range res.Context {
		open[c.Kind+":"+c.SourceID] = true
	}
	known := map[string]bool{}
	if x.Agent.Glossary != nil {
		for t, m := range x.Agent.Glossary(name) {
			known[strings.ToLower(t)], known[strings.ToLower(m)] = true, true
		}
	}
	docs := map[[2]string]map[string]bool{}
	for _, c := range cs {
		k := c.Kind + ":" + c.SourceID
		if !open[k] || flagged(c.Flags, ingest.FlagInstruction) {
			continue
		}
		for _, d := range definitions(c.Text) {
			if known[strings.ToLower(d[0])] || known[d[1]] || instructionShaped(d) {
				continue
			}
			if docs[d] == nil {
				docs[d] = map[string]bool{}
			}
			docs[d][k] = true
		}
	}
	for d, in := range docs {
		if len(in) >= DefinedMinDocs {
			out = append(out, DefinedTerm{Term: d[0], Meaning: d[1], Docs: len(in)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Docs != out[j].Docs {
			return out[i].Docs > out[j].Docs
		}
		return out[i].Term+out[i].Meaning < out[j].Term+out[j].Meaning
	})
	if len(out) > MaxDefined {
		out = out[:MaxDefined]
	}
	return out, nil
}

func flagged(flags []string, f string) bool {
	for _, x := range flags {
		if x == f {
			return true
		}
	}
	return false
}

// instructionShaped is a definition whose text sanitising would flag (sb-ingest(7)).
func instructionShaped(d [2]string) bool {
	return len((ingest.Sanitiser{}).Flag(d[1]+" ("+d[0]+")")) > 0
}
