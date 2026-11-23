package laws

import (
	"math/rand"
	"testing"
)

// leakyGlossary gives every space the glossary of the first space that has one.
type leakyGlossary struct{}

func (leakyGlossary) Glossary(spaces map[string]map[string]string) func(string) map[string]string {
	return func(string) map[string]string { return spaces["ops"] }
}

// The glossary laws pass on the cell's glossary and catch one used outside its space.
func TestSelf_GlossaryLaw(t *testing.T) {
	if violates(60, func(r *rand.Rand) error { return law19Glossary(r, RealGlossary{}) }) {
		t.Fatal("the real glossary breaks the law")
	}
	if !violates(60, func(r *rand.Rand) error { return law19Glossary(r, leakyGlossary{}) }) {
		t.Error("a glossary used in every space was not caught")
	}
}
