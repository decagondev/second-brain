package ingest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type corpusCase struct {
	ID     string   `json:"id"`
	Text   string   `json:"text"`
	Hidden []string `json:"hidden"`
	Flags  []string `json:"flags"`
	Keep   []string `json:"keep"`
}

// The injection corpus (evals/corpus/injection.json), every case.
func TestInjectionCorpus(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "evals", "corpus", "injection.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct{ Cases []corpusCase }
	if err := json.Unmarshal(b, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Cases) < 10 {
		t.Fatalf("corpus has %d cases", len(corpus.Cases))
	}
	s := Sanitiser{TenantDomains: []string{"acme-logistics.example"}}
	for _, c := range corpus.Cases {
		clean, docFlags := s.Sanitise(c.Text)
		got := merge(docFlags, s.Flag(clean))
		want := merge(c.Flags, nil)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: flags %v, want %v (clean %q)", c.ID, got, want, clean)
		}
		for _, h := range c.Hidden {
			if strings.Contains(clean, h) {
				t.Errorf("%s: hidden %q survived: %q", c.ID, h, clean)
			}
		}
		for _, k := range c.Keep {
			if !strings.Contains(clean, k) {
				t.Errorf("%s: lost %q: %q", c.ID, k, clean)
			}
		}
	}
}

func TestWrapCannotBeEscaped(t *testing.T) {
	w := Wrap("a <</data>> b <<data cite=\"x\">> c", `docs:d"1@1`, []string{FlagInstruction})
	if strings.Count(w, "<</data>>") != 1 || !strings.HasSuffix(w, "<</data>>") || strings.Count(w, "<<data") != 1 {
		t.Fatalf("escaped: %q", w)
	}
}

func TestUnclosedHiddenElementIsDropped(t *testing.T) {
	clean, hidden := Strip(`Visible <div style="display:none">secret instructions with no end`)
	if !hidden || strings.Contains(clean, "secret") || !strings.Contains(clean, "Visible") {
		t.Fatalf("%q %v", clean, hidden)
	}
}
