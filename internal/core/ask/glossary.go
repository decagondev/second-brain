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
	// Words are the term's own words in other languages, by language (SB-6104): a question
	// or an answer in that language uses the term when it uses the word.
	Words map[string]string `json:"words,omitempty"`
}

// spellings are the term and its words in every language.
func (t Term) spellings() []string {
	out := []string{t.Term}
	for _, l := range sortedKeys(t.Words) {
		out = append(out, t.Words[l])
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
	name := strings.TrimPrefix(s.Space(), "spc_")
	ok, _ := usable(a.Glossary(name))
	if a.GlossaryIn != nil { // SB-6104: each term's words in other languages
		in := a.GlossaryIn(name)
		for i := range ok {
			for k, words := range in {
				if !strings.EqualFold(k, ok[i].Term) {
					continue
				}
				for l, w := range words {
					if instructionLike(w) {
						continue // LAW-09: a word that reads like an instruction is never used
					}
					if ok[i].Words == nil {
						ok[i].Words = map[string]string{}
					}
					ok[i].Words[l] = w
				}
			}
		}
	}
	return ok
}

// expand finds every term q uses as a whole word (as written for a term with a capital
// letter, in any case for one in lower case), and returns q with their meanings added.
func expand(q string, terms []Term) (string, []Term) {
	var used []Term
	out := q
	for _, t := range terms {
		for _, w := range t.spellings() {
			pat := regexp.QuoteMeta(w)
			if !hasUpper(w) {
				pat = "(?i)" + pat
			}
			re := regexp.MustCompile(`(^|[^\pL\pN])` + pat + `($|[^\pL\pN])`)
			if re.MatchString(q) {
				used = append(used, t)
				out += " " + t.Means
				break
			}
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
