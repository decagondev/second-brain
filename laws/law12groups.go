package laws

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"time"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/groupsync"
	"example.com/second-brain/internal/core/ports"
)

// LAW-12 for a company without SCIM (SB-907): its groups are the manifest's groups:
// list, and someone removed from every group is refused within 5 minutes of the apply,
// with or without the revocation fan-out. Someone still in another group keeps working.

// GroupApply applies a manifest's groups to the cell (groupsync.Apply in the real system).
type GroupApply func(ctx context.Context, dir ports.DirectoryStore, sess ports.Sessions, tenant string, prev, next map[string][]string) error

func applyGroups(ctx context.Context, dir ports.DirectoryStore, sess ports.Sessions, tenant string, prev, next map[string][]string) error {
	_, err := groupsync.Apply(ctx, dir, sess, tenant, prev, next)
	return err
}

type groupCell struct {
	now   time.Time
	dir   *dirmem.Dir
	store ports.Sessions
	sess  ports.Sessions // what apply revokes through
	cell  *auth.Cell
	n     int
}

func newGroupCell(lost bool) *groupCell {
	g := &groupCell{now: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC), dir: dirmem.New()}
	store := sessmem.New()
	g.store, g.sess = store, store
	if lost {
		g.sess = lostRevocations{store}
	}
	g.cell = &auth.Cell{Tenant: "delta-tools", Sessions: store, Directory: g.dir, Now: func() time.Time { return g.now }}
	return g
}

func (g *groupCell) signIn(email string) (string, error) {
	g.n++
	s := ports.Session{ID: fmt.Sprintf("s%d", g.n), Tenant: "delta-tools", User: groupsync.UserID(email), SignedIn: g.now, Validated: g.now}
	return s.ID, g.store.Put(context.Background(), s)
}

func (g *groupCell) served(session string) bool {
	r := httptest.NewRequest("GET", "/v1/ask", nil)
	r.AddCookie(&http.Cookie{Name: auth.Cookie, Value: session})
	_, err := g.cell.Authenticate(r)
	return err == nil
}

func law12Groups(r *rand.Rand, apply GroupApply) error {
	ctx := context.Background()
	lost := r.Intn(2) == 0
	g := newGroupCell(lost)
	people := []string{"ana@delta-tools.example", "bo@delta-tools.example", "cy@delta-tools.example", "di@delta-tools.example"}[:2+r.Intn(3)]
	prev := map[string][]string{}
	for _, p := range people {
		for _, grp := range []string{"group:finance", "group:support"} {
			if r.Intn(2) == 0 {
				prev[grp] = append(prev[grp], p)
			}
		}
		if !inAny(prev, p) {
			prev["group:finance"] = append(prev["group:finance"], p)
		}
	}
	if err := apply(ctx, g.dir, g.sess, "delta-tools", nil, prev); err != nil {
		return err
	}
	sessions := map[string]string{}
	for _, p := range people {
		s, err := g.signIn(p)
		if err != nil {
			return err
		}
		sessions[p] = s
	}
	g.now = g.now.Add(time.Duration(r.Intn(600)) * time.Second)
	leaver := people[r.Intn(len(people))]
	next := map[string][]string{}
	for grp, ms := range prev {
		for _, m := range ms {
			if m != leaver {
				next[grp] = append(next[grp], m)
			}
		}
	}
	if r.Intn(2) == 0 { // sometimes another person only changes group, and stays
		if other := people[r.Intn(len(people))]; other != leaver {
			next["group:ops"] = append(next["group:ops"], other)
		}
	}
	for grp := range next {
		sort.Strings(next[grp])
	}
	if err := apply(ctx, g.dir, g.sess, "delta-tools", prev, next); err != nil {
		return err
	}
	var since time.Duration
	for i := 0; i < 15; i++ {
		step := time.Duration(r.Intn(90)) * time.Second
		g.now = g.now.Add(step)
		since += step
		p := people[r.Intn(len(people))]
		ok := g.served(sessions[p])
		switch {
		case p == leaver && ok && since >= deprovBound:
			return fmt.Errorf("%s, removed from groups:, was served %v after the apply (lost fan-out: %v)", p, since, lost)
		case p != leaver && !ok:
			return fmt.Errorf("%s, still in groups:, was refused %v after another person was removed", p, since)
		}
	}
	return nil
}

func inAny(groups map[string][]string, p string) bool {
	for _, ms := range groups {
		for _, m := range ms {
			if m == p {
				return true
			}
		}
	}
	return false
}
