package laws

import (
	"math/rand"
	"testing"
)

// The office LAW-09 test catches a reader that indexes a document's hidden parts.
func TestSelf_OfficeLawCatchesHiddenParts(t *testing.T) {
	dir := t.TempDir()
	if !violates(60, func(r *rand.Rand) error { return law09Office(r, dir, LeakyOffice) }) {
		t.Fatal("LAW-09: indexing macros, scripts or embedded objects was not caught")
	}
}
