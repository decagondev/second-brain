package laws

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// uncheckedDigest lists what the index says changed, without checking it as the person.
type uncheckedDigest struct{ sinceForever bool }

func (u uncheckedDigest) Changes(a ask.Agent, _ func(string) tenancy.Scope) func(context.Context, string, int) ([]string, int, error) {
	return func(ctx context.Context, _ string, _ int) ([]string, int, error) {
		since := a.Now().Add(-7 * 24 * time.Hour)
		if u.sinceForever {
			since = time.Time{}
		}
		cs, err := a.Index.(ports.DocLookup).Docs(ctx, a.Tenant, ports.DocQuery{SeenSince: since})
		seen := map[string]bool{}
		var out []string
		for _, c := range cs {
			if !seen[c.Title] {
				seen[c.Title] = true
				out = append(out, c.Title)
			}
		}
		return out, 0, err
	}
}

// The digest law passes on the cell's digest, and catches one that doesn't check as the
// person and one that lists everything.
func TestSelf_DigestLaw(t *testing.T) {
	if violates(60, func(r *rand.Rand) error { return law01Digest(r, RealDigest{}) }) {
		t.Fatal("the real digest breaks the law")
	}
	for name, sys := range map[string]DigestSystem{"unchecked": uncheckedDigest{}, "everything": uncheckedDigest{sinceForever: true}} {
		if !violates(60, func(r *rand.Rand) error { return law01Digest(r, sys) }) {
			t.Errorf("%s was not caught", name)
		}
	}
}
