package laws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	sessmem "example.com/second-brain/internal/adapters/sessions/memory"
	spacemem "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/auth"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/scim"
)

// LAW-21 through the real SCIM endpoint and the real request path (SB-1205): the IdP moves
// people between groups with SCIM, and an ask in a space goes through the auth middleware,
// which reads groups from the directory at each request. Time is a fake clock.

const moveTenant = "acme-logistics"

// flakyDir is the directory; while failing, group writes fail (an add the directory
// didn't commit).
type flakyDir struct {
	*dirmem.Dir
	failing bool
}

func (d *flakyDir) PutGroup(ctx context.Context, g ports.Group) error {
	if d.failing {
		return errors.New("directory unavailable")
	}
	return d.Dir.PutGroup(ctx, g)
}

type scimMoves struct {
	now    time.Time
	dir    *flakyDir
	sess   *sessmem.Store
	scim   http.Handler
	cell   *auth.Cell
	reg    *spacemem.Store
	users  map[string]string   // name -> SCIM id
	groups map[string]string   // group:NAME -> SCIM id
	claims map[string][]string // SCIM id -> groups the IdP claims
	// mutations
	cacheGroups bool                // groups read once and kept
	trustClaims bool                // the IdP's claimed groups count
	cache       map[string][]string // for cacheGroups
}

func newSCIMMoves() *scimMoves {
	m := &scimMoves{now: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC), dir: &flakyDir{Dir: dirmem.New()}, sess: sessmem.New(),
		reg: spacemem.New(), users: map[string]string{}, groups: map[string]string{}, claims: map[string][]string{}, cache: map[string][]string{}}
	m.scim = (&scim.Endpoint{Tenant: moveTenant, Directory: m.dir, Sessions: m.sess,
		Token: func(context.Context) (string, error) { return "tok", nil }}).Handler()
	m.cell = &auth.Cell{Tenant: moveTenant, Sessions: m.sess, Directory: m.dir, Now: func() time.Time { return m.now },
		Issuer: tenancy.NewIssuer(moveTenant), Spaces: m.reg, GroupsOf: m.groupsOf, Attributes: auth.DirectoryAttributes(m.dir)}
	return m
}

func (m *scimMoves) groupsOf(ctx context.Context, tenant, user string) ([]string, error) {
	if m.cacheGroups {
		if gs, ok := m.cache[user]; ok {
			return gs, nil
		}
	}
	gs, err := m.dir.GroupsOf(ctx, tenant, user)
	if err != nil {
		return nil, err
	}
	if m.trustClaims {
		for _, g := range m.claims[user] {
			gs = append(gs, strings.TrimPrefix(g, "group:"))
		}
	}
	if m.cacheGroups {
		m.cache[user] = gs
	}
	return gs, nil
}

func (m *scimMoves) do(method, path string, body any) (int, map[string]any) {
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, strings.NewReader(string(b)))
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	m.scim.ServeHTTP(w, r)
	var v map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	return w.Code, v
}

func (m *scimMoves) Space(name, group string) error {
	code, v := m.do("POST", "/scim/v2/Groups", map[string]any{"displayName": strings.TrimPrefix(group, "group:")})
	if code != 201 {
		return fmt.Errorf("SCIM group %s: %d %v", group, code, v)
	}
	m.groups[group] = v["id"].(string)
	return m.reg.PutSpace(context.Background(), ports.Space{ID: spaces.ID(name), Tenant: moveTenant, Name: name, Members: []string{group}, Label: "restricted"})
}

func (m *scimMoves) user(name string) (string, error) {
	if id, ok := m.users[name]; ok {
		return id, nil
	}
	code, v := m.do("POST", "/scim/v2/Users", map[string]any{"userName": name, "emails": []map[string]string{{"value": name + "@acme.example"}},
		"title": "Analyst", "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": map[string]any{"department": "People", "costCenter": "CC-100"}})
	if code != 201 {
		return "", fmt.Errorf("SCIM user %s: %d %v", name, code, v)
	}
	id := v["id"].(string)
	m.users[name] = id
	return id, m.sess.Put(context.Background(), ports.Session{ID: "s-" + name, Tenant: moveTenant, User: id, SignedIn: m.now, Validated: m.now})
}

func (m *scimMoves) patch(user, group, op string) (int, error) {
	id, err := m.user(user)
	if err != nil {
		return 0, err
	}
	body := map[string]any{"Operations": []map[string]any{{"op": op, "path": "members", "value": []map[string]string{{"value": id}}}}}
	if op == "remove" {
		body = map[string]any{"Operations": []map[string]any{{"op": "remove", "path": `members[value eq "` + id + `"]`}}}
	}
	code, _ := m.do("PATCH", "/scim/v2/Groups/"+m.groups[group], body)
	return code, nil
}

func (m *scimMoves) Join(user, group string) error {
	code, err := m.patch(user, group, "add")
	if err == nil && code != 200 {
		err = fmt.Errorf("SCIM add %s to %s: %d", user, group, code)
	}
	return err
}

func (m *scimMoves) Leave(user, group string) error {
	code, err := m.patch(user, group, "remove")
	if err == nil && code != 200 {
		err = fmt.Errorf("SCIM remove %s from %s: %d", user, group, code)
	}
	return err
}

// JoinUnconfirmed: the IdP believes user is in group (it says so in its token, and sent
// an add), but the directory didn't commit the add.
func (m *scimMoves) JoinUnconfirmed(user, group string) error {
	id, err := m.user(user)
	if err != nil {
		return err
	}
	m.claims[id] = append(m.claims[id], group) // the IdP's view, by directory id
	m.dir.failing = true
	defer func() { m.dir.failing = false }()
	code, err := m.patch(user, group, "add")
	if err == nil && code == 200 {
		err = fmt.Errorf("a failed directory write was acknowledged to the IdP")
	}
	return err
}

func (m *scimMoves) Advance(d time.Duration) { m.now = m.now.Add(d) }

func (m *scimMoves) Allowed(user, space string) (bool, error) {
	if _, err := m.user(user); err != nil {
		return false, err
	}
	var seen tenancy.Scope
	h := m.cell.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = tenancy.FromContext(r.Context())
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest("GET", "/v1/ask?space="+space, nil)
	r.AddCookie(&http.Cookie{Name: auth.Cookie, Value: "s-" + user})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 200 && seen.Attrs()["department"] != "People" {
		return false, fmt.Errorf("the request's scope lacks the directory's attributes: %v", seen.Attrs())
	}
	return w.Code == 200 && seen.Space() == spaces.ID(space), nil
}
