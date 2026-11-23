package canary

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Learn learns every secret a stored value holds (sb-selfhost(7), Support bundle; SB-1808):
// the value itself, each line of a multi-line value (a PEM key, say), and the secret-named
// string fields of a JSON object, so a log line that carries only the object's secret_key
// is found too. Fields like access_key or bucket names are not secrets and aren't learnt.
func (s *Scanner) Learn(v string) {
	v = strings.TrimSpace(v)
	if !strings.Contains(v, "\n") {
		s.Register(v)
	}
	for _, line := range strings.Split(v, "\n") {
		line = strings.TrimSpace(line)
		if len(line) >= 16 && !strings.HasPrefix(line, "-----") {
			s.Register(line)
		}
	}
	var obj map[string]any
	if strings.HasPrefix(v, "{") && json.Unmarshal([]byte(v), &obj) == nil {
		for k, x := range obj {
			str, ok := x.(string)
			if ok && secretField(k) {
				s.Register(str)
			}
		}
	}
}

func secretField(k string) bool {
	k = strings.ToLower(k)
	if k == "access_key" || k == "access_key_id" {
		return false
	}
	for _, w := range []string{"secret", "password", "token", "private", "key"} {
		if strings.Contains(k, w) {
			return true
		}
	}
	return false
}

// shapes are secrets the scanner finds by their look, known or not: a private key, a URL
// that carries a password, and a bearer token.
var shapes = []struct {
	id string
	re *regexp.Regexp
}{
	{"shape_private_key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"shape_url_password", regexp.MustCompile(`[a-z][a-z0-9+.-]*://[^\s/:@]+:[^\s/@]+@`)},
	{"shape_bearer", regexp.MustCompile(`(?i)\bbearer [A-Za-z0-9._~+/=-]{20,}`)},
}

// ScanAll is Scan plus the shapes: every known value and every secret-shaped string in
// text, line by line, by id only.
func (s *Scanner) ScanAll(where, text string) []Finding {
	out := s.Scan(where, text)
	for i, line := range strings.Split(text, "\n") {
		for _, sh := range shapes {
			if sh.re.MatchString(line) {
				out = append(out, Finding{Where: where, Line: i + 1, ID: sh.id})
			}
		}
	}
	return out
}
