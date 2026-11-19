package laws

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-02 at the request context (SB-108): inside a cell, data is reached only through a
// tenancy.Scope that the cell's own Issuer made for its own tenant. Sessions of another
// tenant get no scope; adversarial queries for another tenant's ids find nothing; a
// missing scope is an error, never "no filter".

// ScopedCell is one tenant's cell, seeded with users and groups of several tenants in a
// shared store (the worst case: a mistake would leak).
type ScopedCell interface {
	// SignIn tries to get a scope for a session of sessionTenant; ok false if refused.
	SignIn(sessionTenant, user string) (ok bool, err error)
	// Person looks up a user id with the last scope issued; found false if not visible.
	Person(id string) (found bool, err error)
	// MyGroups lists the groups visible to the last scope.
	MyGroups() ([]string, error)
	// NoScope calls Person with the zero Scope; it must return an error.
	NoScope(id string) error
}

// Seeded is one user in the shared store.
type Seeded struct{ Tenant, ID, Group string }

// NewScopedCell builds the system under test: the real auth middleware (SB-108) with the
// cell's one Issuer, in front of a directory shared by every seeded tenant.
var NewScopedCell = func(cell string, seed []Seeded) ScopedCell {
	ctx := context.Background()
	dir := dirmem.New()
	for _, s := range seed {
		_ = dir.PutUser(ctx, ports.User{ID: s.ID, Tenant: s.Tenant, UserName: s.ID, Active: true})
		_ = dir.PutGroup(ctx, ports.Group{ID: s.Tenant + "-" + s.ID, Tenant: s.Tenant, Name: s.Group, Members: []string{s.ID}})
	}
	now := time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)
	c := &auth.Cell{Tenant: cell, Sessions: sessmem.New(), Directory: dir, Now: func() time.Time { return now },
		Issuer: tenancy.NewIssuer(cell), GroupsOf: dir.GroupsOf}
	return &scoped{cell: c, dir: dir, now: now}
}

type scoped struct {
	cell  *auth.Cell
	dir   *dirmem.Dir
	now   time.Time
	scope tenancy.Scope
	n     int
}

func (s *scoped) SignIn(sessionTenant, user string) (bool, error) {
	s.n++
	id := fmt.Sprintf("s%d", s.n)
	_ = s.cell.Sessions.Put(context.Background(), ports.Session{ID: id, Tenant: sessionTenant, User: user, SignedIn: s.now, Validated: s.now})
	r := httptest.NewRequest("GET", "/v1/ask", nil)
	r.AddCookie(&http.Cookie{Name: auth.Cookie, Value: id})
	got := false
	s.cell.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		sc, err := tenancy.FromContext(r.Context())
		if err == nil {
			s.scope, got = sc, true
		}
	})).ServeHTTP(httptest.NewRecorder(), r)
	return got, nil
}

func (s *scoped) Person(id string) (bool, error) {
	_, found, err := s.dir.Person(context.Background(), s.scope, id)
	return found, err
}

func (s *scoped) MyGroups() ([]string, error) { return s.dir.MyGroups(context.Background(), s.scope) }

func (s *scoped) NoScope(id string) error {
	_, _, err := s.dir.Person(context.Background(), tenancy.Scope{}, id)
	return err
}

func law02Scope(r *rand.Rand, mk func(string, []Seeded) ScopedCell) error {
	tenants := []string{"acme-logistics", "beta-freight", "gamma-parts"}
	var seed []Seeded
	byTenant := map[string][]Seeded{}
	for i := 0; i < 6+r.Intn(6); i++ {
		t := tenants[r.Intn(len(tenants))]
		s := Seeded{Tenant: t, ID: fmt.Sprintf("usr_%d", i), Group: []string{"it-leads", "ops", "everyone"}[r.Intn(3)]}
		seed = append(seed, s)
		byTenant[t] = append(byTenant[t], s)
	}
	cell := tenants[r.Intn(len(tenants))]
	if len(byTenant[cell]) == 0 {
		return nil
	}
	sys := mk(cell, seed)
	for i := 0; i < 8; i++ {
		other := tenants[r.Intn(len(tenants))]
		if other != cell {
			ok, err := sys.SignIn(other, "usr_x")
			if err != nil {
				return err
			}
			if ok {
				return fmt.Errorf("a %s session got a scope in %s's cell", other, cell)
			}
		}
		me := byTenant[cell][r.Intn(len(byTenant[cell]))]
		ok, err := sys.SignIn(cell, me.ID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s's own user %s got no scope", cell, me.ID)
		}
		probe := seed[r.Intn(len(seed))]
		found, err := sys.Person(probe.ID)
		if err != nil {
			return err
		}
		if found != (probe.Tenant == cell) {
			return fmt.Errorf("in %s's cell, user %s of %s: found=%v", cell, probe.ID, probe.Tenant, found)
		}
		groups, err := sys.MyGroups()
		if err != nil {
			return err
		}
		if len(groups) > 1 || (len(groups) == 1 && groups[0] != me.Group) {
			return fmt.Errorf("%s of %s sees groups %v, want [%s]", me.ID, cell, groups, me.Group)
		}
		if sys.NoScope(probe.ID) == nil {
			return fmt.Errorf("a lookup with no scope succeeded")
		}
	}
	return nil
}
