package ask

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/prompts"
	"example.com/second-brain/internal/core/tenancy"
)

// Ask in one language, find another (sb-ask(3), Languages; SB-6103): with sb-cell(5)
// languages naming more than one, a question is also searched in each of them, translated
// by the small tier while the evals job doesn't hold it (else the main one, LAW-06) with the
// space's glossary words (SB-6104). The translations join the search as the glossary's
// meanings do; cites stay in their own language and the answer is in the UI's (SB-3003).
// The trace keeps the translated queries, which the context view shows.

// MaxTranslated bounds the languages one question is translated into.
const MaxTranslated = 6

type translatedKey struct{}

// withTranslated gives retrieve a place to leave the question's translations.
func withTranslated(ctx context.Context, into *[]string) context.Context {
	return context.WithValue(ctx, translatedKey{}, into)
}

// translations are q in each of the cell's languages, with the glossary's words; none with
// one language, no model, or a translation that fails (the question is still searched as
// asked). A translation that reads like an instruction is dropped (LAW-09).
func (a Agent) translations(ctx context.Context, q string, terms []Term) []string {
	if len(a.Languages) < 2 || utf8.RuneCountInString(q) > 2000 {
		return nil
	}
	m := a.Model
	if a.Small != nil && a.SmallHeld != nil && !a.SmallHeld() {
		m = a.Small
	}
	if m == nil {
		return nil
	}
	pr, err := prompts.Get("translate")
	if err != nil {
		return nil
	}
	langs := a.Languages
	if len(langs) > MaxTranslated {
		langs = langs[:MaxTranslated]
	}
	var b strings.Builder
	b.WriteString("Languages: " + strings.Join(langs, ", ") + "\n")
	for _, t := range terms {
		for _, l := range sortedKeys(t.Words) {
			b.WriteString("Glossary: " + t.Term + " is " + t.Words[l] + " in " + l + "\n")
		}
	}
	b.WriteString("<<question>>\n" + q + "\n<</question>>")
	resp, err := m.Complete(ctx, ports.ModelRequest{System: pr.Text, JSON: true, MaxTokens: 600,
		Messages: []ports.Message{{Role: "user", Content: b.String()}}})
	if err != nil {
		return nil
	}
	var out map[string]string
	if json.Unmarshal([]byte(strings.TrimSpace(resp.Text)), &out) != nil {
		return nil
	}
	var res []string
	seen := map[string]bool{strings.ToLower(strings.TrimSpace(q)): true}
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, l := range keys {
		t := strings.TrimSpace(out[l])
		if t == "" || seen[strings.ToLower(t)] || !contains(langs, l) || instructionLike(t) || utf8.RuneCountInString(t) > 4000 {
			continue
		}
		seen[strings.ToLower(t)] = true
		res = append(res, t)
	}
	return res
}

// Translate a cited passage (sb-ask(3), Translate a passage; SB-6105): the passage as the
// reader may open it now (LAW-26), at most MaxTranslatedPassage characters, translated into
// lang by the same tier as a question and marked; it is kept nowhere.

// MaxTranslatedPassage is the most characters of a passage translated.
const MaxTranslatedPassage = 600

// Translation is a passage in another language, marked as a model's (LAW-28).
type Translation struct {
	Language string `json:"language"`
	Text     string `json:"text"`
	ReadBy   string `json:"read_by"` // always "translate"
}

// ReadByTranslate is the mark of a passage a model translated.
const ReadByTranslate = "translate"

// TranslatePassage is Passage with its translation into lang.
func (a Agent) TranslatePassage(ctx context.Context, s tenancy.Scope, trace, cite string, start int, lang string) (Passage, *Translation, error) {
	if !validLanguage(lang) {
		return Passage{}, nil, fmt.Errorf("%w: language %q isn't one of %s", ErrInvalid, lang, strings.Join(TranslateTo, ", "))
	}
	p, err := a.Passage(ctx, s, trace, cite, start)
	if err != nil || p.Gone || p.Text == "" {
		return p, nil, err
	}
	m := a.Model
	if a.Small != nil && a.SmallHeld != nil && !a.SmallHeld() {
		m = a.Small
	}
	if m == nil {
		return p, nil, fmt.Errorf("%w: no model to translate with", ErrUnavailable)
	}
	pr, err := prompts.Get("translate-passage")
	if err != nil {
		return p, nil, err
	}
	text := p.Text
	if rs := []rune(text); len(rs) > MaxTranslatedPassage {
		text = string(rs[:MaxTranslatedPassage])
	}
	resp, err := m.Complete(ctx, ports.ModelRequest{System: pr.Text, MaxTokens: 1200, User: s.User(),
		Messages: []ports.Message{{Role: "user", Content: "Language: " + lang + "\n<<passage>>\n" + text + "\n<</passage>>"}}})
	if err != nil {
		return p, nil, fmt.Errorf("%w: the model couldn't translate: %v", ErrUnavailable, err)
	}
	return p, &Translation{Language: lang, Text: strings.TrimSpace(resp.Text), ReadBy: ReadByTranslate}, nil
}

// TranslateTo are the languages a passage may be translated into: the UI's.
var TranslateTo = []string{"en", "de", "fr", "es", "it", "nl"}

func validLanguage(l string) bool {
	for _, x := range TranslateTo {
		if x == l {
			return true
		}
	}
	return false
}
