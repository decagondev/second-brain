package ingest

import "regexp"

// The secret scanner (sb-ingest(7), Secrets never stored; SB-1103). A chunk that matches
// any pattern is dropped before it is embedded or written, and only the pattern's name is
// reported (LAW-07). Patterns err on the side of dropping.
var secretPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"private-key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP |ENCRYPTED )?PRIVATE KEY(?: BLOCK)?-----`)},
	{"aws-access-key", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"github-token", regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,})\b`)},
	{"slack-token", regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}\b`)},
	{"stripe-key", regexp.MustCompile(`\b(?:sk|rk)_live_[A-Za-z0-9]{20,}\b`)},
	{"google-api-key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
	{"assigned-secret", regexp.MustCompile(`(?i)(?:password|passwd|secret|api[_-]?key|access[_-]?token|client[_-]?secret)["']?\s*[:=]\s*["']?[^\s"']{12,}`)},
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
