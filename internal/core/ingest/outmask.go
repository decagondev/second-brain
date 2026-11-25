package ingest

import (
	"bytes"
	"encoding/json"
	"regexp"
)

// Nothing sensitive goes out (sb-ingest(7), Nothing sensitive goes out; SB-5102): one filter
// on every way text leaves the cell. Credentials are replaced whole; payment cards, IBANs and
// national ids keep their last four characters, as at ingest (SB-1904).

// Removed is what a credential is replaced by on its way out.
const Removed = "[removed]"

// MaskOut is text as it may leave the cell, and whether anything was masked.
func MaskOut(text string) (string, bool) {
	out := text
	for _, p := range secretPatterns {
		out = p.re.ReplaceAllString(out, Removed)
	}
	out = MaskPII(out, false)
	return out, out != text
}

var jsonString = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

// MaskJSON is a JSON document with every string masked by MaskOut, its structure, order and
// other bytes as they were, and how many strings changed. A string that can't be read as
// JSON is left alone: it was never a string.
func MaskJSON(b []byte) ([]byte, int) {
	n := 0
	out := jsonString.ReplaceAllFunc(b, func(lit []byte) []byte {
		var s string
		if err := json.Unmarshal(lit, &s); err != nil {
			return lit
		}
		m, changed := MaskOut(s)
		if !changed {
			return lit
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf) // the same escaping as the API's own encoder
		if err := enc.Encode(m); err != nil {
			return lit
		}
		n++
		return bytes.TrimRight(buf.Bytes(), "\n")
	})
	return out, n
}
