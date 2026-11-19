package laws

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
)

// LAW-09 against the pen test's corpus (SB-612): every payload in
// security/attacks/injection.txt, planted anywhere in a benign document, is flagged on the
// chunk that carries it or stripped as hidden text. Nothing in the corpus is indexed as
// plain, unflagged data.

// InjectionCorpus reads the corpus file (path relative to the laws package).
func InjectionCorpus(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the corpus %s is empty", path)
	}
	return out, nil
}

func law09Corpus(r *rand.Rand, w SanitiserWorld, corpus []string) error {
	for _, payload := range corpus {
		var paras []string
		for i := 0; i < 1+r.Intn(4); i++ {
			ws := make([]string, 6+r.Intn(10))
			for j := range ws {
				ws[j] = benign[r.Intn(len(benign))]
			}
			paras = append(paras, strings.Join(ws, " ")+".")
		}
		at := r.Intn(len(paras) + 1)
		// A payload is its own paragraph, as a tester would plant it.
		paras = append(paras[:at], append([]string{payload}, paras[at:]...)...)
		chunks, err := w.Ingest(strings.Join(paras, "\n\n"))
		if err != nil {
			return err
		}
		for _, c := range chunks {
			if len(c.Flags) > 0 {
				continue
			}
			for _, word := range attackWords(payload) {
				if strings.Contains(c.Text, word) {
					return fmt.Errorf("payload %q reached an unflagged chunk:\n%s", payload, c.Text)
				}
			}
		}
	}
	return nil
}

// attackWords are the payload's distinctive words: any of them in an unflagged chunk means
// the payload (or part of it) was indexed as plain data.
func attackWords(p string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(p, func(r rune) bool { return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') }) {
		lw := strings.ToLower(w)
		if len(lw) >= 5 && !isBenign(lw) {
			out = append(out, w)
		}
	}
	return out
}

func isBenign(w string) bool {
	for _, b := range benign {
		if b == w {
			return true
		}
	}
	return false
}
