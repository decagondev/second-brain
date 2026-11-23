package laws

import (
	"context"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// staleMembers keeps the space's members as they were at the first look: the review list
// never sees anyone leave.
type staleMembers struct{}

func (staleMembers) Stewards(dir string, members func() []string) (ask.Stewards, *folder.Connector, func() error, error) {
	first := members()
	return RealNoteNarrowing{}.Stewards(dir, func() []string { return first })
}

// fakeRepublish says Ran on a republish without writing it, once a note is out.
type fakeRepublish struct{}

type onceBus struct {
	real ask.NoteBus
	done bool
}

func (b *onceBus) Invoke(ctx context.Context, s tenancy.Scope, c toolbus.Call) (approvals.Action, error) {
	if b.done {
		return approvals.Action{ID: "tc_2", State: approvals.Ran, Receipt: c.Target + "@sha256:x"}, nil
	}
	b.done = true
	return b.real.Invoke(ctx, s, c)
}

func (fakeRepublish) Stewards(dir string, members func() []string) (ask.Stewards, *folder.Connector, func() error, error) {
	x, c, sync, err := RealNoteNarrowing{}.Stewards(dir, members)
	x.Bus = &onceBus{real: x.Bus}
	return x, c, sync, err
}

// The notes law passes on the real system, and catches a review list that doesn't see the
// space narrow and a republish that changes nothing on the share.
func TestSelf_NoteNarrowingLaw(t *testing.T) {
	dir := t.TempDir()
	if violates(60, func(r *rand.Rand) error { return law19Notes(r, dir, RealNoteNarrowing{}) }) {
		t.Fatal("the real system breaks the notes law")
	}
	for name, sys := range map[string]NoteNarrowing{"stale members": staleMembers{}, "a republish that writes nothing": fakeRepublish{}} {
		if !violates(60, func(r *rand.Rand) error { return law19Notes(r, dir, sys) }) {
			t.Errorf("LAW-19: %s was not caught", name)
		}
	}
}
