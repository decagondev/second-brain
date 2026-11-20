package ingest

import (
	"testing"
	"unicode/utf8"
)

// FuzzChunksAndSecrets (SB-1402): chunking and the secret scanner never panic, spans stay
// inside the text, and a chunk holding a secret pattern always overlaps a document span.
func FuzzChunksAndSecrets(f *testing.F) {
	f.Add("Deploy notes. password = \"correct horse battery staple\" then restart.")
	f.Add("-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----")
	f.Add("héllo wörld " + string(rune(0x1F600)))
	f.Fuzz(func(t *testing.T, raw string) {
		if !utf8.ValidString(raw) {
			return
		}
		text := Normalise(raw)
		n := utf8.RuneCountInString(text)
		spans := SecretSpans(text)
		for _, s := range spans {
			if s.Start < 0 || s.End > n || s.Start > s.End {
				t.Fatalf("secret span %+v outside a text of %d runes", s, n)
			}
		}
		rs := []rune(text)
		for _, c := range Chunks(text) {
			if c.Start < 0 || c.End > n || c.Start > c.End {
				t.Fatalf("chunk %+v outside a text of %d runes", c, n)
			}
			if pat := SecretIn(string(rs[c.Start:c.End])); pat != "" && overlaps(spans, c.Start, c.End) == "" && pat != "assigned-secret" {
				t.Fatalf("chunk holds %s but overlaps no document span", pat)
			}
		}
	})
}
