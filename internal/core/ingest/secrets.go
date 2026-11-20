package ingest

import (
	"regexp"
	"sort"
	"unicode/utf8"
)

// The secret scanner (sb-ingest(7), Secrets never stored; SB-1103, SB-1401). A chunk that
// matches any pattern, or overlaps a secret found in the whole document, is dropped before
// it is embedded or written, and only the pattern's name is reported (LAW-07). Patterns err
// on the side of dropping.
var secretPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	// A whole PEM private key, header to footer (or to the end of the text), so a key split
	// across chunks is still found in every chunk it touches. JSON-escaped newlines count.
	{"private-key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP |ENCRYPTED )?PRIVATE KEY(?: BLOCK)?-----(?:[\s\S]*?-----END (?:RSA |EC |OPENSSH |DSA |PGP |ENCRYPTED )?PRIVATE KEY(?: BLOCK)?-----|[\s\S]*$)`)},
	{"aws-access-key", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"github-token", regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,})\b`)},
	{"gitlab-token", regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`)},
	{"slack-token", regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}\b`)},
	{"stripe-key", regexp.MustCompile(`\b(?:sk|rk)_live_[A-Za-z0-9]{20,}\b`)},
	{"model-api-key", regexp.MustCompile(`\bsk-(?:ant-|proj-)?[A-Za-z0-9_-]{20,}`)},
	{"google-api-key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
	{"jwt", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},
	{"bearer-token", regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{20,}`)},
	{"url-credentials", regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]{1,20}://[^\s:/@]+:[^\s@/]{3,}@`)},
	// A credential assigned to a name that says so, also inside a longer name
	// (aws_secret_access_key, DB_PASSWORD), quoted (spaces allowed) or not.
	{"assigned-secret", regexp.MustCompile(`(?i)[a-z0-9_.-]*(?:password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|client[_-]?secret)[a-z0-9_.-]*["']?\s*[:=]\s*(?:"[^"\n]{8,}"|'[^'\n]{8,}'|[^\s"']{12,})`)},
}

// SecretIn returns the name of the first secret pattern the text matches, or "".
func SecretIn(text string) string {
	for _, p := range secretPatterns {
		if p.re.MatchString(text) {
			return p.name
		}
	}
	return ""
}

// SecretSpan is a secret found in a whole document, in rune offsets, and its pattern.
type SecretSpan struct {
	Start, End int
	Pattern    string
}

// SecretSpans finds every secret in a whole document, so a secret a chunk boundary splits
// is still dropped from every chunk it touches (SB-1401).
func SecretSpans(text string) []SecretSpan {
	var out []SecretSpan
	for _, p := range secretPatterns {
		for _, m := range p.re.FindAllStringIndex(text, -1) {
			out = append(out, SecretSpan{utf8.RuneCountInString(text[:m[0]]), utf8.RuneCountInString(text[:m[1]]), p.name})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// overlaps returns the pattern of the first span overlapping [start, end), or "".
func overlaps(spans []SecretSpan, start, end int) string {
	for _, s := range spans {
		if s.Start < end && start < s.End {
			return s.Pattern
		}
	}
	return ""
}
