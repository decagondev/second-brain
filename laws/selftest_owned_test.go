package laws

import (
	"math/rand"
	"testing"
)

// The owned laws pass on the cell's code, and catch a directory check that never finds an
// owner gone and an index that keeps no text hash.
func TestSelf_OwnedLaws(t *testing.T) {
	if violates(60, func(r *rand.Rand) error { return law12Orphans(r, ownedSystem{}) }) {
		t.Fatal("the real review list breaks LAW-12")
	}
	if violates(60, func(r *rand.Rand) error { return law13Copies(r, ownedSystem{}) }) {
		t.Fatal("the real retrieval breaks LAW-13")
	}
	if !violates(60, func(r *rand.Rand) error { return law12Orphans(r, ownedSystem{neverGone: true}) }) {
		t.Error("a review list that never finds an owner gone was not caught")
	}
	if !violates(60, func(r *rand.Rand) error { return law13Copies(r, ownedSystem{noHash: true}) }) {
		t.Error("an index without a text hash was not caught")
	}
}
