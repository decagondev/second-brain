package laws

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// placer is a barrier cell whose Target is replaced.
type placer struct {
	toolbus.Barriers
	target func(ctx context.Context, s tenancy.Scope, target string) (map[string]any, error)
}

func (p placer) Target(ctx context.Context, s tenancy.Scope, target string) (map[string]any, error) {
	return p.target(ctx, s, target)
}

// The notes law (SB-2001) passes on the real placement, and catches a cell that refuses
// every note (the W16 behaviour), one that places every note in no space (so the barrier
// never sees it), and one that ignores an unreadable registry.
func TestSelf_NotesLawCatchesBadPlacement(t *testing.T) {
	ok := func(r *rand.Rand) error { return law22Notes(r, t.TempDir(), RealNotePlacer) }
	if violates(60, ok) {
		t.Fatal("the real note placement breaks the notes law")
	}
	bad := map[string]NotePlacer{
		"refuses every note": func(sp func(context.Context) ([]ports.Space, [][2]string, error)) toolbus.Barriers {
			return placer{RealNotePlacer(sp), func(context.Context, tenancy.Scope, string) (map[string]any, error) {
				return nil, errors.New("can't be placed")
			}}
		},
		"places a note nowhere": func(sp func(context.Context) ([]ports.Space, [][2]string, error)) toolbus.Barriers {
			real := RealNotePlacer(sp)
			return placer{real, func(ctx context.Context, s tenancy.Scope, target string) (map[string]any, error) {
				f, err := real.Target(ctx, s, target)
				if f != nil {
					f["target_spaces"] = []string{}
				}
				return f, err
			}}
		},
		"ignores an unreadable registry": func(sp func(context.Context) ([]ports.Space, [][2]string, error)) toolbus.Barriers {
			real := RealNotePlacer(sp)
			return placer{real, func(ctx context.Context, s tenancy.Scope, target string) (map[string]any, error) {
				f, err := real.Target(ctx, s, target)
				if err != nil {
					return map[string]any{"barred_spaces": []string{}, "barred_groups": []string{}, "target_spaces": []string{}, "target_owners": []string{}}, nil //nolint:nilerr // the mutation under test swallows the error
				}
				return f, nil
			}}
		},
	}
	for name, p := range bad {
		if !violates(60, func(r *rand.Rand) error { return law22Notes(r, t.TempDir(), p) }) {
			t.Errorf("LAW-22: a cell that %s was not caught", name)
		}
	}
}
