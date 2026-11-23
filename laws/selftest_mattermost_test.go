package laws

import (
	"math/rand"
	"testing"
)

// The Mattermost LAW-12 test catches a mapping that answers someone deprovisioned.
func TestSelf_MattermostDeprovisionedIsCaught(t *testing.T) {
	if !violates(30, func(r *rand.Rand) error { return law12Mattermost(Gen(r), true) }) {
		t.Fatal("LAW-12: an answer to a deprovisioned Mattermost user was not caught")
	}
}

// The Mattermost Share law passes on the real app and catches a share that carries claims.
func TestSelf_MattermostShareLaw(t *testing.T) {
	if violates(30, func(r *rand.Rand) error { return law01MattermostShare(Gen(r), false) }) {
		t.Fatal("the real Mattermost Share breaks the law")
	}
	if !violates(30, func(r *rand.Rand) error { return law01MattermostShare(Gen(r), true) }) {
		t.Fatal("a share carrying the claims was not caught")
	}
}
