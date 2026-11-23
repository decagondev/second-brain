package ask

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/tenancy"
)

// Term is one glossary entry a question used (sb-ask(3), Glossary; SB-2203).
type Term struct {
	Term  string `json:"term"`
	Means string `json:"means"`
}

// usable splits a space's glossary into the entries that may be used and the terms of those
// that read like an instruction, which are never used (LAW-09).
func usable(g map[string]string) (ok []Term, flagged []string) {
	keys := make([]string, 0, len(g))
	for k := range g {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		m := g[k]
		if instructionLike(k) || instructionLike(m) || instructionLike(k+" "+m) {
			flagged = append(flagged, k)
			continue
		}
		ok = append(ok, Term{Term: k, Means: m})
	}
	return ok, flagged
}

// instructionLike is text the sanitiser flags (sb-ingest(7), Content is data), or that could
// close a data block.
func instructionLike(s string) bool {
	return len(ingest.Sanitiser{}.Flag(s)) > 0 || strings.Contains(s, "<<") || strings.Contains(s, ">>")
}

// glossaryOf is the usable glossary of the scope's space: none for a company-wide ask, so a
// space's terms are used only in that space (LAW-19).
func (a Agent) glossaryOf(s tenancy.Scope) []Term {
	if a.Glossary == nil || !strings.HasPrefix(s.Space(), "spc_") {
		return nil
	}
	ok, _ := usable(a.Glossary(strings.TrimPrefix(s.Space(), "spc_")))
	return ok
}

// expand finds every term q uses as a whole word (as written for a term with a capital
// letter, in any case for one in lower case), and returns q with their meanings added.
func expand(q string, terms []Term) (string, []Term) {
	var used []Term
	out := q
	for _, t := range terms {
		pat := regexp.QuoteMeta(t.Term)
		if !hasUpper(t.Term) {
			pat = "(?i)" + pat
		}
		re := regexp.MustCompile(`(^|[^\pL\pN])` + pat + `($|[^\pL\pN])`)
		if re.MatchString(q) {
			used = append(used, t)
			out += " " + t.Means
		}
	}
	return out, used
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// glossaryData is the terms an answer used as one data block: the stewards' text, quoted as
// data like any chunk, never an instruction (LAW-09).
func glossaryData(terms []Term) string {
	if len(terms) == 0 {
		return ""
	}
	var b strings.Builder
	for _, t := range terms {
		b.WriteString(t.Term + ": " + t.Means + "\n")
	}
	return ingest.Wrap(strings.TrimSuffix(b.String(), "\n"), "glossary", nil) + "\n"
}

type termsKey struct{}

func withTerms(ctx context.Context, terms []Term) context.Context {
	return context.WithValue(ctx, termsKey{}, terms)
}

func termsOf(ctx context.Context) []Term {
	t, _ := ctx.Value(termsKey{}).([]Term)
	return t
}
