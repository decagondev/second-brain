package laws

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

// forgetfulAccess remembers no request, so every ask sends again.
type forgetfulAccess struct{}

func (forgetfulAccess) Store(ask.AccessStore) ask.AccessStore { return forgetStore{} }

type forgetStore struct{}

func (forgetStore) AccessRequested(context.Context, string, string) (bool, error) { return false, nil }
func (forgetStore) PutAccessRequest(context.Context, string, string, string, time.Time) error {
	return nil
}

// The access law passes on the cell's store and catches one that forgets requests.
func TestSelf_AccessLaw(t *testing.T) {
	if violates(60, func(r *rand.Rand) error { return law01Access(r, RealAccess{}) }) {
		t.Fatal("the real access requests break the law")
	}
	if !violates(60, func(r *rand.Rand) error { return law01Access(r, forgetfulAccess{}) }) {
		t.Error("a store that forgets requests was not caught")
	}
}
