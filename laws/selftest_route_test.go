package laws

import (
	"math/rand"
	"testing"

	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/route"
)

func TestLAW13_SpaceRoutingIsDerived(t *testing.T) {
	Check(t, "LAW-13", runs, func(r *rand.Rand) error { return law13Spaces(r, route.Route) })
}

func TestLAW20_IngestNeverLabelsBelowRestricted(t *testing.T) {
	Check(t, "LAW-20", runs, func(r *rand.Rand) error { return law13Spaces(r, route.Route) })
}

// LAW-20's ingest check catches a router that takes the first holder's default instead
// of the strictest; LAW-13's catches one that keeps routing by the configuration it saw first.
func TestSelf_RoutingMutationsAreCaught(t *testing.T) {
	firstHolder := func(sp []ports.Space, kind, loc, src string) ([]string, string) {
		h, l := route.Route(sp, kind, loc, src)
		if src == "" && len(h) > 1 {
			for _, s := range sp {
				if s.ID == h[0] {
					return h, s.Label
				}
			}
		}
		return h, l
	}
	if !violates(runs, func(r *rand.Rand) error { return law13Spaces(r, firstHolder) }) {
		t.Error("routing: a router that ignores the stricter label was not caught")
	}
	stale := func() Router {
		var first []ports.Space
		return func(sp []ports.Space, kind, loc, src string) ([]string, string) {
			if first == nil {
				first = sp
			}
			return route.Route(first, kind, loc, src)
		}
	}
	if !violates(runs, func(r *rand.Rand) error { return law13Spaces(r, stale()) }) {
		t.Error("routing: a router that keeps an old configuration was not caught")
	}
}

func TestLAW20_ChatChannelsAskOnlyInTheirSpace(t *testing.T) {
	Check(t, "LAW-20", runs, func(r *rand.Rand) error {
		return law20Chat(r, func(st *spacemem.Store) ports.SpaceStore { return st })
	})
}

// LAW-20's chat test catches a registry read that loses its answer, which would turn a
// space's channel into a company-wide one.
func TestSelf_LAW20ChatMutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error {
		return law20Chat(r, func(st *spacemem.Store) ports.SpaceStore { return forgetful{st} })
	}) {
		t.Error("LAW-20 (chat): a registry that answers nothing was not caught")
	}
}
