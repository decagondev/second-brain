package laws

import (
	"fmt"
	"math/rand"
	"sort"
)

// LAW-19, LAW-20 and LAW-22 over department spaces (SB-804; sb-spaces(7)). A random company
// with spaces, labels, barriers and source ACLs; every ask, in a space or company-wide, may
// use only chunks that pass the four checks of sb-permissions(7), Spaces, labels and barriers:
//   LAW-19 Spaces only narrow: nothing used that the source's ACL denies.
//   LAW-20 Restricted stays home: restricted content only for members of a holding space.
//   LAW-22 Barriers hold: nothing used across a barrier, in either direction.

// SpaceDef is one space: its member groups and label default.
type SpaceDef struct {
	Name    string
	Members []string
	Label   string
}

// SpaceDoc is one document: its label, the users the source lets open it, the spaces
// whose scope covers it, and its owner at the source ("" for none; SB-1203).
type SpaceDoc struct {
	ID      string
	Label   string
	Viewers []string
	Spaces  []string
	Owner   string
}

// SpaceSpec is a whole company.
type SpaceSpec struct {
	Users    map[string][]string // user -> groups
	Spaces   []SpaceDef
	Docs     []SpaceDoc
	Barriers [][2]string
}

// SpaceSystem is a cell with spaces.
type SpaceSystem interface {
	Load(spec SpaceSpec) error
	// Ask returns the documents an answer may use for user, in space ("company" for a
	// company-wide ask); refused is true when the whole ask is refused (ESPACE).
	Ask(user, space string) (used []string, refused bool, err error)
}

// NewSpaceSystem builds the system under test: asks in a cell with spaces, through
// retrieval (SB-1006). See law20ask.go.
var NewSpaceSystem = func() SpaceSystem { return newAskSpaces() }

// NewBarrierSystem is the system for LAW-22: barriers as forbids for reads, drafts,
// actions and approvals, with ownership from the source (SB-1203). See law22barriers.go.
var NewBarrierSystem = func() SpaceSystem { return newBarrierSpaces() }

var spaceLabels = []string{"public", "internal", "confidential", "restricted"}

// genSpaces makes a random company: 3 to 6 users, 2 to 4 spaces, 4 to 12 documents, 0 to 2
// barriers.
func genSpaces(r *rand.Rand) SpaceSpec {
	groups := []string{"group:eng", "group:people-team", "group:finance", "group:deal-team", "group:everyone"}
	sp := SpaceSpec{Users: map[string][]string{}}
	users := []string{"usr_ann", "usr_bob", "usr_cy", "usr_dee", "usr_eve", "usr_fay"}[:3+r.Intn(4)]
	for _, u := range users {
		gs := []string{"group:everyone"}
		for _, g := range groups[:4] {
			if r.Intn(3) == 0 {
				gs = append(gs, g)
			}
		}
		sp.Users[u] = gs
	}
	names := []string{"engineering", "people", "finance", "executive"}[:2+r.Intn(3)]
	for i, n := range names {
		sp.Spaces = append(sp.Spaces, SpaceDef{Name: n, Members: []string{groups[i%4]}, Label: spaceLabels[r.Intn(4)]})
	}
	for i := 0; i < 4+r.Intn(9); i++ {
		d := SpaceDoc{ID: fmt.Sprintf("d%d", i), Label: spaceLabels[r.Intn(4)]}
		for _, u := range users {
			if r.Intn(2) == 0 {
				d.Viewers = append(d.Viewers, u)
			}
		}
		for _, s := range names {
			if r.Intn(3) == 0 {
				d.Spaces = append(d.Spaces, s)
			}
		}
		if r.Intn(2) == 0 {
			d.Owner = users[r.Intn(len(users))]
		}
		sp.Docs = append(sp.Docs, d)
	}
	for i := 0; i < r.Intn(3); i++ {
		a, b := groups[r.Intn(4)], groups[r.Intn(5)]
		if a != b {
			sp.Barriers = append(sp.Barriers, [2]string{a, b})
		}
	}
	return sp
}

// spaceFacts answers the laws' questions about a spec.
type spaceFacts struct {
	sp      SpaceSpec
	members map[string]map[string]bool // space -> group set
	docs    map[string]SpaceDoc
}

func factsOf(sp SpaceSpec) spaceFacts {
	f := spaceFacts{sp: sp, members: map[string]map[string]bool{}, docs: map[string]SpaceDoc{}}
	for _, s := range sp.Spaces {
		f.members[s.Name] = map[string]bool{}
		for _, g := range s.Members {
			f.members[s.Name][g] = true
		}
	}
	for _, d := range sp.Docs {
		f.docs[d.ID] = d
	}
	return f
}

func (f spaceFacts) memberOf(user, space string) bool {
	for _, g := range f.sp.Users[user] {
		if f.members[space][g] {
			return true
		}
	}
	return false
}

func (f spaceFacts) barrierBetween(user string, d SpaceDoc) bool {
	for _, b := range f.sp.Barriers {
		for _, g := range f.sp.Users[user] {
			for _, s := range d.Spaces {
				for mg := range f.members[s] {
					if (g == b[0] && mg == b[1]) || (g == b[1] && mg == b[0]) {
						return true
					}
				}
			}
			if d.Owner != "" && f.groupsCross(g, f.sp.Users[d.Owner], b) {
				return true // owned at the source by someone across the barrier (SB-1203)
			}
		}
	}
	return false
}

// groupsCross: g on one side of barrier b and one of others on the other.
func (f spaceFacts) groupsCross(g string, others []string, b [2]string) bool {
	for _, o := range others {
		if (g == b[0] && o == b[1]) || (g == b[1] && o == b[0]) {
			return true
		}
	}
	return false
}

// peopleCross: a barrier separates a group of u from a group of v.
func (f spaceFacts) peopleCross(u, v string) bool {
	for _, b := range f.sp.Barriers {
		for _, g := range f.sp.Users[u] {
			if f.groupsCross(g, f.sp.Users[v], b) {
				return true
			}
		}
	}
	return false
}

func asks(sp SpaceSpec) [][2]string {
	var out [][2]string
	users := make([]string, 0, len(sp.Users))
	for u := range sp.Users {
		users = append(users, u)
	}
	sort.Strings(users)
	for _, u := range users {
		out = append(out, [2]string{u, "company"})
		for _, s := range sp.Spaces {
			out = append(out, [2]string{u, s.Name})
		}
	}
	return out
}

func spaceLaw(r *rand.Rand, sys SpaceSystem, check func(f spaceFacts, user, space string, d SpaceDoc) error) error {
	sp := genSpaces(r)
	if err := sys.Load(sp); err != nil {
		return err
	}
	f := factsOf(sp)
	for _, a := range asks(sp) {
		used, _, err := sys.Ask(a[0], a[1])
		if err != nil {
			return err
		}
		for _, id := range used {
			d, ok := f.docs[id]
			if !ok {
				return fmt.Errorf("%s asking in %s used %s, which doesn't exist", a[0], a[1], id)
			}
			if err := check(f, a[0], a[1], d); err != nil {
				return err
			}
		}
	}
	return nil
}

func law19(r *rand.Rand, sys SpaceSystem) error {
	return spaceLaw(r, sys, func(_ spaceFacts, user, space string, d SpaceDoc) error {
		if !contains(d.Viewers, user) {
			return fmt.Errorf("%s asking in %s used %s, which the source doesn't let them open (viewers %v)", user, space, d.ID, d.Viewers)
		}
		return nil
	})
}

func law20(r *rand.Rand, sys SpaceSystem) error {
	return spaceLaw(r, sys, func(f spaceFacts, user, space string, d SpaceDoc) error {
		if d.Label != "restricted" {
			return nil
		}
		for _, s := range d.Spaces {
			if f.memberOf(user, s) {
				return nil
			}
		}
		return fmt.Errorf("%s asking in %s used restricted %s held by %v, and is a member of none", user, space, d.ID, d.Spaces)
	})
}

func law22(r *rand.Rand, sys SpaceSystem) error {
	err := spaceLaw(r, sys, func(f spaceFacts, user, space string, d SpaceDoc) error {
		if f.barrierBetween(user, d) {
			return fmt.Errorf("%s (groups %v) asking in %s used %s across a barrier %v (holders %v, owner %q)", user, f.sp.Users[user], space, d.ID, f.sp.Barriers, d.Spaces, d.Owner)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return law22Acts(r, sys) // drafts, actions and approvals, for a system that has them
}
