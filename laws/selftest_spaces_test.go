package laws

import (
	"math/rand"
	"testing"
	"time"
)

// refSpaces is a reference cell with spaces, written from sb-permissions(7), Spaces, labels
// and barriers: membership, label, barrier, then the source's ACL. bug breaks one step.
type refSpaces struct {
	f   spaceFacts
	bug string // "", "skip-acl", "restricted-company-wide", "ignore-barriers"
}

func (s *refSpaces) Load(sp SpaceSpec) error { s.f = factsOf(sp); return nil }

func (s *refSpaces) Ask(user, space string) ([]string, bool, error) {
	if space != "company" && !s.f.memberOf(user, space) {
		return nil, true, nil // ESPACE
	}
	var used []string
	for _, d := range s.f.sp.Docs {
		member := false
		for _, h := range d.Spaces {
			member = member || s.f.memberOf(user, h)
		}
		if space != "company" {
			if !contains(d.Spaces, space) {
				continue
			}
		} else if !member && d.Label != "public" && d.Label != "internal" && s.bug != "restricted-company-wide" {
			continue
		}
		if (d.Label == "confidential" || d.Label == "restricted") && !member && s.bug != "restricted-company-wide" {
			continue
		}
		if s.bug != "ignore-barriers" && s.f.barrierBetween(user, d) {
			continue
		}
		if s.bug != "skip-acl" && !contains(d.Viewers, user) {
			continue
		}
		used = append(used, d.ID)
	}
	return used, false, nil
}

func TestSelf_SpaceReferencePassesLAW19LAW20LAW22(t *testing.T) {
	usedAny := false
	for seed := int64(1); seed <= runs; seed++ {
		for name, law := range map[string]func(*rand.Rand, SpaceSystem) error{"LAW-19": law19, "LAW-20": law20, "LAW-22": law22} {
			if err := law(rand.New(rand.NewSource(seed)), &refSpaces{}); err != nil {
				t.Fatalf("seed %d: %s on the reference: %v", seed, name, err)
			}
		}
		ref := &refSpaces{}
		r := rand.New(rand.NewSource(seed))
		sp := genSpaces(r)
		_ = ref.Load(sp)
		for _, a := range asks(sp) {
			if u, _, _ := ref.Ask(a[0], a[1]); len(u) > 0 {
				usedAny = true
			}
		}
	}
	if !usedAny {
		t.Fatal("the reference never used a document: the space laws would pass vacuously")
	}
}

func TestSelf_SpaceMutationsAreCaught(t *testing.T) {
	for bug, law := range map[string]func(*rand.Rand, SpaceSystem) error{
		"skip-acl": law19, "restricted-company-wide": law20, "ignore-barriers": law22,
	} {
		if !violates(runs, func(r *rand.Rand) error { return law(r, &refSpaces{bug: bug}) }) {
			t.Errorf("space bug %q was not caught", bug)
		}
	}
}

// refMoves is a reference directory feed: removals apply within lag, unconfirmed joins never.
type refMoves struct {
	now    time.Time
	space  map[string]string          // space -> group
	member map[string]map[string]bool // group -> confirmed users
	leaves map[string]time.Time       // user/group -> when removed
	lag    time.Duration
}

func newRefMoves(lag time.Duration) *refMoves {
	return &refMoves{now: time.Unix(1_800_000_000, 0), space: map[string]string{}, member: map[string]map[string]bool{}, leaves: map[string]time.Time{}, lag: lag}
}
func (m *refMoves) Space(n, g string) error {
	m.space[n] = g
	m.member[g] = map[string]bool{}
	return nil
}
func (m *refMoves) Join(u, g string) error               { m.member[g][u] = true; return nil }
func (m *refMoves) Leave(u, g string) error              { m.leaves[u+"/"+g] = m.now; return nil }
func (m *refMoves) JoinUnconfirmed(string, string) error { return nil }
func (m *refMoves) Advance(d time.Duration)              { m.now = m.now.Add(d) }
func (m *refMoves) Allowed(u, space string) (bool, error) {
	g := m.space[space]
	if !m.member[g][u] {
		return false, nil
	}
	if at, left := m.leaves[u+"/"+g]; left && m.now.Sub(at) >= m.lag {
		return false, nil
	}
	return true, nil
}

func TestSelf_LAW21ReferenceAndMutation(t *testing.T) {
	for seed := int64(1); seed <= runs; seed++ {
		if err := law21(rand.New(rand.NewSource(seed)), newRefMoves(time.Minute)); err != nil {
			t.Fatalf("seed %d: LAW-21 on the reference: %v", seed, err)
		}
	}
	if !violates(runs, func(r *rand.Rand) error { return law21(r, newRefMoves(15*time.Minute)) }) {
		t.Error("LAW-21: a 15-minute revocation was not caught")
	}
}

// refGrants is a reference aggregate-only grant; bug "keep-after-revoke" or "any-metric".
type refGrants struct {
	now     time.Time
	space   string
	metrics map[string]bool
	revoked time.Time
	bug     string
}

func (g *refGrants) Grant(space string, ms []string) (string, error) {
	g.now, g.space, g.metrics = time.Unix(1_800_000_000, 0), space, map[string]bool{}
	for _, m := range ms {
		g.metrics[m] = true
	}
	return "grant_1", nil
}
func (g *refGrants) Revoke(string) error          { g.revoked = g.now; return nil }
func (g *refGrants) Advance(d time.Duration)      { g.now = g.now.Add(d) }
func (g *refGrants) Content(string) (bool, error) { return false, nil }
func (g *refGrants) Aggregate(space, m string) (bool, error) {
	if space != g.space {
		return false, nil
	}
	if !g.revoked.IsZero() && g.bug != "keep-after-revoke" {
		return false, nil
	}
	return g.metrics[m] || g.bug == "any-metric", nil
}

func TestSelf_LAW23ReferenceAndMutations(t *testing.T) {
	for seed := int64(1); seed <= runs; seed++ {
		if err := law23(rand.New(rand.NewSource(seed)), &refGrants{}); err != nil {
			t.Fatalf("seed %d: LAW-23 on the reference: %v", seed, err)
		}
	}
	for _, bug := range []string{"keep-after-revoke", "any-metric"} {
		if !violates(runs, func(r *rand.Rand) error { return law23(r, &refGrants{bug: bug}) }) {
			t.Errorf("LAW-23: grant bug %q was not caught", bug)
		}
	}
}
