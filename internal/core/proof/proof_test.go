package proof

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// SB-6001: a proof with a name, a free-text count key or an unhashed id isn't signed; a
// signed proof verifies, and one changed after signing doesn't.
func TestSealRefusesContentAndVerifies(t *testing.T) {
	sg, _ := audit.NewSigner("acme-logistics", make([]byte, 32))
	now := time.Date(2026, 11, 27, 22, 0, 0, 0, time.UTC)
	h := NewHasher()
	good := New("SB-5901", "acme-logistics", "graph-v1.0", now, []Criterion{{Name: "the named drives list their items", Pass: true, Counts: map[string]int{"items": 2}, Seen: h.IDs([]string{"a", "b"})}})
	for name, c := range map[string]Criterion{
		"a name":         {Name: "ann@acme.example reads", Pass: true},
		"a count key":    {Name: "items", Pass: true, Counts: map[string]int{"Quarterly Close": 1}},
		"an unhashed id": {Name: "items", Pass: true, Seen: []string{"item-123"}},
	} {
		if _, err := Seal(New("SB-5901", "acme-logistics", "graph-v1.0", now, []Criterion{c}), sg, now); !errors.Is(err, ErrContent) {
			t.Errorf("%s: sealed (%v)", name, err)
		}
	}
	s, err := Seal(good, sg, now)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(s)
	if _, err := Verify(b, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(b, "ed25519:other"); err == nil {
		t.Fatal("verified against another pinned key")
	}
	if _, err := Verify([]byte(strings.Replace(string(b), `"items":2`, `"items":3`, 1)), ""); err == nil {
		t.Fatal("a changed proof verified")
	}
	once, again := h.ID("a"), h.ID("a")
	if once != again || NewHasher().ID("a") == once {
		t.Fatal("a hash isn't stable within a proof, or is the same across proofs")
	}
	if FileName(good) != "SB-5901-20261127-220000.json" {
		t.Fatal(FileName(good))
	}
}
