package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-19 and LAW-02 at the request context (SB-1002): a request in a space is in that
// space only if this tenant's registry holds it and the asker is a member. An unknown
// space, a non-member, another tenant's space and a registry that can't answer are all
// refused, never turned into a company-wide request, which would widen what it reaches.

// SpaceSeed is one space in the shared registry.
type SpaceSeed struct{ Tenant, Name, Group string }

// SpaceCell is one tenant's cell with a space registry shared with other tenants.
type SpaceCell interface {
	// Request sends a request with these ?space= values as user; it returns the scope's
	// space, or "" if the request was refused.
	Request(user string, space []string) string
	// Break makes the registry fail (true) or answer (false).
	Break(bool)
}

// NewSpaceCell builds the system under test: the real auth middleware (SB-1002) with the
// cell's one Issuer and spaces.Resolve over a shared memory registry.
var NewSpaceCell = func(cell string, users map[string]string, seed []SpaceSeed) SpaceCell {
	ctx := context.Background()
	dir, st := dirmem.New(), spacemem.New()
	for u, g := range users {
		_ = dir.PutUser(ctx, ports.User{ID: u, Tenant: cell, UserName: u, Active: true})
		_ = dir.PutGroup(ctx, ports.Group{ID: "grp-" + u, Tenant: cell, Name: g, Members: []string{u}})
	}
	for _, s := range seed {
		_ = st.PutSpace(ctx, ports.Space{ID: spaces.ID(s.Name), Tenant: s.Tenant, Name: s.Name, Members: []string{"group:" + s.Group}})
	}
	now := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	c := &auth.Cell{Tenant: cell, Sessions: sessmem.New(), Directory: dir, Now: func() time.Time { return now },
		Issuer: tenancy.NewIssuer(cell), GroupsOf: dir.GroupsOf, Spaces: st}
	return &spaceCell{cell: c, st: st, now: now}
}

type spaceCell struct {
	cell *auth.Cell
	st   *spacemem.Store
	now  time.Time
	n    int
}

func (s *spaceCell) Break(b bool) {
	s.st.Fail = nil
	if b {
		s.st.Fail = errors.New("registry timeout")
	}
}

func (s *spaceCell) Request(user string, space []string) string {
	s.n++
	id := fmt.Sprintf("s%d", s.n)
	_ = s.cell.Sessions.Put(context.Background(), ports.Session{ID: id, Tenant: s.cell.Tenant, User: user, SignedIn: s.now, Validated: s.now})
	q := url.Values{}
	for _, v := range space {
		q.Add("space", v)
	}
	r := httptest.NewRequest("GET", "/v1/answers?"+q.Encode(), nil)
	r.AddCookie(&http.Cookie{Name: auth.Cookie, Value: id})
	got := ""
	s.cell.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if sc, err := tenancy.FromContext(r.Context()); err == nil {
			got = sc.Space()
		}
	})).ServeHTTP(httptest.NewRecorder(), r)
	return got
}

func law19Context(r *rand.Rand, mk func(string, map[string]string, []SpaceSeed) SpaceCell) error {
	const cell = "acme-logistics"
	groups := []string{"finance", "eng", "people"}
	users := map[string]string{"usr_1": "finance", "usr_2": "eng", "usr_3": "people"}
	var seed []SpaceSeed
	held := map[string]string{} // this tenant's space -> member group
	for _, name := range []string{"finance", "eng", "people", "legal"} {
		if r.Intn(3) != 0 {
			g := groups[r.Intn(len(groups))]
			seed = append(seed, SpaceSeed{cell, name, g})
			held[name] = g
		}
		if r.Intn(2) == 0 { // another tenant's space of the same name, open to every group
			for _, g := range groups {
				seed = append(seed, SpaceSeed{"beta-freight", name, g})
			}
		}
	}
	sys := mk(cell, users, seed)
	for i := 0; i < 16; i++ {
		user := []string{"usr_1", "usr_2", "usr_3"}[r.Intn(3)]
		var asked []string
		switch r.Intn(6) {
		case 0:
		case 1:
			asked = []string{"company"}
		case 2:
			asked = []string{"finance", "eng"}
		case 3:
			asked = []string{"nowhere"}
		default:
			asked = []string{[]string{"finance", "eng", "people", "legal"}[r.Intn(4)]}
		}
		broken := r.Intn(4) == 0
		sys.Break(broken)
		got := sys.Request(user, asked)
		want := ""
		switch {
		case len(asked) == 0 || (len(asked) == 1 && asked[0] == "company"):
			want = "company"
		case len(asked) > 1 || broken:
		case held[asked[0]] != "" && held[asked[0]] == users[user]:
			want = spaces.ID(asked[0])
		}
		if got != want {
			return fmt.Errorf("%s (group %s) asking in %v (registry broken: %v) got space %q, want %q (this tenant holds %v)",
				user, users[user], asked, broken, got, want, held)
		}
	}
	return nil
}
