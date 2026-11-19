package ingest

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Flags recorded on chunks (sb-ingest(7), Content is data).
const (
	FlagHidden      = "hidden_text_removed"
	FlagInstruction = "instruction_like"
)

var (
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	styledOpen  = regexp.MustCompile(`(?is)<([a-z][a-z0-9]*)\b[^>]*\bstyle\s*=\s*"([^"]*)"[^>]*>`)
	hiddenStyle = regexp.MustCompile(`(?i)(display\s*:\s*none|visibility\s*:\s*hidden|font-size\s*:\s*0(px|pt|em|%)?\s*(;|$)|(left|top)\s*:\s*-\d{3,}px)`)
	longAttr    = regexp.MustCompile(`(?i)\s(alt|title)\s*=\s*"[^"]{101,}"`)

	instructionLike = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(ignore|forget|override)\s+(all\s+|any\s+)?(the\s+|your\s+)?(previous|prior|above|earlier|preceding)\s+(instructions|rules|prompts?|directions)`),
		regexp.MustCompile(`(?i)\bdisregard\s+(all\s+|the\s+|any\s+)*(above|previous|prior|earlier|instructions|rules)`),
		regexp.MustCompile(`(?i)\byou\s+are\s+now\b`),
		regexp.MustCompile(`(?i)\b(act|behave)\s+as\s+(the|a|an)\s`),
		regexp.MustCompile(`(?i)\bpretend\s+to\s+be\b`),
		regexp.MustCompile(`(?im)^\s*(?:[#>*_-]+\s*)*(system|assistant|developer)\s*(?:[*_]+\s*)?:`), // also as a Markdown heading, quote or bold label (SB-612)
		regexp.MustCompile(`(?i)<\|im_(start|end)\|>|\[/?INST\]`),
		regexp.MustCompile(`\{\s*"(tool|function|tool_name|name)"\s*:\s*"[^"]+"\s*,\s*"(args|arguments|parameters|input)"\s*:`),
		regexp.MustCompile(`(?i)\b(do\s+not|don't|never)\s+(tell|inform|mention\s+(this\s+)?to|show)\s+the\s+user\b`),
		regexp.MustCompile(`(?i)\bwithout\s+(telling|asking|informing)\s+(the\s+user|anyone)\b`),
	}
	linkWithQuery = regexp.MustCompile(`(?i)https?://([a-z0-9.-]+)[^\s?#]*\?[^\s#]*=`)
)

// Sanitiser applies the strip and flag rules. TenantDomains are the tenant's own domains:
// links to them may carry query strings without being flagged.
type Sanitiser struct{ TenantDomains []string }

// Strip removes hidden text and reports whether anything was removed.
func Strip(text string) (string, bool) {
	before := text
	text = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) { // zero-width, bidi controls, BOM and other invisible format characters
			return -1
		}
		return r
	}, text)
	text = htmlComment.ReplaceAllString(text, "")
	for from := 0; from < len(text); {
		rel := styledOpen.FindStringSubmatchIndex(text[from:])
		if rel == nil {
			break
		}
		loc := make([]int, len(rel))
		for i, x := range rel {
			loc[i] = x + from
		}
		if !hiddenStyle.MatchString(text[loc[4]:loc[5]]) {
			from = loc[1] // a visible styled element: keep it, look further
			continue
		}
		tag := strings.ToLower(text[loc[2]:loc[3]])
		end := strings.Index(strings.ToLower(text[loc[1]:]), "</"+tag+">")
		if end < 0 {
			text = text[:loc[0]] // unclosed hidden element: drop the rest, never keep it
			break
		}
		text = text[:loc[0]] + text[loc[1]+end+len(tag)+3:]
	}
	text = longAttr.ReplaceAllString(text, "")
	return text, text != before
}

// Flag returns the flags for one span of sanitised text.
func (s Sanitiser) Flag(text string) []string {
	for _, re := range instructionLike {
		if re.MatchString(text) {
			return []string{FlagInstruction}
		}
	}
	for _, m := range linkWithQuery.FindAllStringSubmatch(text, -1) {
		if !s.own(strings.ToLower(m[1])) {
			return []string{FlagInstruction} // data smuggled out through a link
		}
	}
	return nil
}

func (s Sanitiser) own(host string) bool {
	for _, d := range s.TenantDomains {
		d = strings.ToLower(d)
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// Sanitise strips the whole text and returns it with the document-level flags; Flag is
// then applied per chunk.
func (s Sanitiser) Sanitise(text string) (string, []string) {
	clean, hidden := Strip(text)
	if hidden {
		return clean, []string{FlagHidden}
	}
	return clean, nil
}

func merge(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range append(append([]string(nil), a...), b...) {
		set[x] = true
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

// Wrap is how a chunk enters a prompt: a quoted data block with provenance. Any << or >>
// in the text is broken up first, so the content can't close its own block (LAW-09).
func Wrap(text, cite string, flags []string) string {
	esc := func(s string) string {
		s = strings.ReplaceAll(s, "<<", "< <")
		s = strings.ReplaceAll(s, ">>", "> >")
		return strings.ReplaceAll(s, `"`, "'")
	}
	return `<<data cite="` + esc(cite) + `" flags="` + esc(strings.Join(flags, ",")) + `">>` + "\n" + esc(text) + "\n<</data>>"
}
