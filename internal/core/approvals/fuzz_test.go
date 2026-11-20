package approvals

import (
	"bytes"
	"testing"
)

// FuzzCanonical (SB-1402): canonical JSON is a fixed point, and the payload hash depends
// only on it, so an approver and the executor always hash the same bytes.
func FuzzCanonical(f *testing.F) {
	for _, s := range []string{`{"comment":"x"}`, `{"b":1,"a":[1,2,{"z":null}]}`, `{"amount": -5}`, `"\u00e9"`, `[]`, `{"a":1e3}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		c, err := Canonical(raw)
		if err != nil {
			return
		}
		c2, err := Canonical(c)
		if err != nil || !bytes.Equal(c, c2) {
			t.Fatalf("canonical form isn't stable: %q -> %q (%v)", c, c2, err)
		}
		h1, err1 := Hash("w", "t:1", raw)
		h2, err2 := Hash("w", "t:1", c)
		if err1 != nil || err2 != nil || h1 != h2 {
			t.Fatalf("the hash depends on more than the canonical form: %v %v", err1, err2)
		}
		_, _ = Amount(raw)
	})
}
