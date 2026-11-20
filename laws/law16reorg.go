package laws

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/reorg"
)

// LAW-16 for the reorg preview (SB-1206): a preview changes nothing (the manifest and the
// snapshot are the same after it), and what it reports is exact: per person, the spaces
// gained and lost under the manifest's membership rule, and the flags for losing every
// space, gaining a restricted one, or straddling a barrier.

// ReorgDiff is the preview under test (reorg.Diff in the real system).
type ReorgDiff func(t manifest.Tenant, s reorg.Snapshot) reorg.Report

func law16Reorg(r *rand.Rand, diff ReorgDiff) error {
	groups := []string{"group:eng", "group:people-team", "group:finance", "group:deal-team", "group:sales"}
	labels := []string{"public", "internal", "confidential", "restricted"}
	t := manifest.Tenant{Spaces: map[string]manifest.Space{}}
	for _, name := range []string{"engineering", "people", "finance", "sales"}[:2+r.Intn(3)] {
		var ms []string
		for _, g := range groups {
			if r.Intn(3) == 0 {
				ms = append(ms, g)
			}
		}
		if len(ms) == 0 {
			ms = []string{groups[r.Intn(len(groups))]}
		}
		t.Spaces[name] = manifest.Space{Members: ms, Label: labels[r.Intn(len(labels))]}
	}
	if r.Intn(2) == 0 {
		t.Barriers = [][2]string{{"group:deal-team", groups[r.Intn(3)]}}
	}
	pick := func() []string {
		var out []string
		for _, g := range groups {
			if r.Intn(3) == 0 {
				out = append(out, g)
			}
		}
		return out
	}
	var snap reorg.Snapshot
	snap.Version = 1
	for i := 0; i < 2+r.Intn(6); i++ {
		snap.People = append(snap.People, reorg.Person{User: fmt.Sprintf("p%d@acme.example", i), Before: pick(), After: pick()})
	}
	tCopy, sCopy := deepTenant(t), deepSnap(snap)
	got := diff(t, snap)
	if !reflect.DeepEqual(t, tCopy) || !reflect.DeepEqual(snap, sCopy) {
		return fmt.Errorf("the preview changed its inputs")
	}
	// The oracle.
	can := func(gs []string) map[string]bool {
		out := map[string]bool{}
		for name, sp := range t.Spaces {
			for _, m := range sp.Members {
				for _, g := range gs {
					if m == g {
						out[name] = true
					}
				}
			}
		}
		return out
	}
	byUser := map[string]reorg.Change{}
	for _, c := range got.People {
		byUser[c.User] = c
	}
	unchanged := 0
	for _, p := range snap.People {
		b, a := can(p.Before), can(p.After)
		var gains, loses, flags []string
		for s := range a {
			if !b[s] {
				gains = append(gains, s)
			}
		}
		for s := range b {
			if !a[s] {
				loses = append(loses, s)
			}
		}
		sort.Strings(gains)
		sort.Strings(loses)
		if len(b) > 0 && len(a) == 0 {
			flags = append(flags, reorg.LosesAll)
		}
		for _, s := range gains {
			if t.Spaces[s].Label == "restricted" {
				flags = append(flags, reorg.GainsRestricted)
				break
			}
		}
		for _, br := range t.Barriers {
			if contains(p.After, br[0]) && contains(p.After, br[1]) {
				flags = append(flags, reorg.Barrier)
				break
			}
		}
		c, listed := byUser[p.User]
		if len(gains)+len(loses)+len(flags) == 0 {
			unchanged++
			if listed {
				return fmt.Errorf("%s is listed but nothing changes for them", p.User)
			}
			continue
		}
		if !listed {
			return fmt.Errorf("%s gains %v, loses %v, flags %v, but isn't listed", p.User, gains, loses, flags)
		}
		if !sameSet(c.Gains, gains) || !sameSet(c.Loses, loses) || !sameSet(c.Flags, flags) {
			return fmt.Errorf("%s: preview gains %v loses %v flags %v; want %v %v %v (before %v, after %v)",
				p.User, c.Gains, c.Loses, c.Flags, gains, loses, flags, p.Before, p.After)
		}
	}
	if got.Unchanged != unchanged {
		return fmt.Errorf("%d unchanged, want %d", got.Unchanged, unchanged)
	}
	return nil
}

func deepTenant(t manifest.Tenant) manifest.Tenant {
	c := t
	c.Spaces = map[string]manifest.Space{}
	for k, v := range t.Spaces {
		v.Members = append([]string(nil), v.Members...)
		c.Spaces[k] = v
	}
	c.Barriers = append([][2]string(nil), t.Barriers...)
	return c
}

func deepSnap(s reorg.Snapshot) reorg.Snapshot {
	c := s
	c.People = nil
	for _, p := range s.People {
		c.People = append(c.People, reorg.Person{User: p.User, Before: append([]string(nil), p.Before...), After: append([]string(nil), p.After...)})
	}
	return c
}
