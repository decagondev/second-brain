// Package memory is an in-process directory for one cell (development and tests). A
// production cell keeps its directory in the cell database; the port is the same.
package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Dir is a mutex-guarded directory.
type Dir struct {
	mu     sync.Mutex
	users  map[string]ports.User  // tenant/id
	groups map[string]ports.Group // tenant/id
}

var _ ports.DirectoryStore = (*Dir)(nil)

func New() *Dir { return &Dir{users: map[string]ports.User{}, groups: map[string]ports.Group{}} }

func k(tenant, id string) string { return tenant + "/" + id }

func (d *Dir) Active(_ context.Context, tenant, user string) (bool, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	u, ok := d.users[k(tenant, user)]
	return ok && u.Active, ok, nil
}

func (d *Dir) PutUser(_ context.Context, u ports.User) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.users[k(u.Tenant, u.ID)] = u
	return nil
}

func (d *Dir) User(_ context.Context, tenant, id string) (ports.User, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	u, ok := d.users[k(tenant, id)]
	return u, ok, nil
}

func (d *Dir) UserByName(_ context.Context, tenant, name string) (ports.User, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, u := range d.users {
		if u.Tenant == tenant && u.UserName == name {
			return u, true, nil
		}
	}
	return ports.User{}, false, nil
}

// UsersByEmail implements ports.UserFinder: every user of the tenant with that email,
// compared case-insensitively.
func (d *Dir) UsersByEmail(_ context.Context, tenant, email string) ([]ports.User, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []ports.User
	for _, u := range d.users {
		if u.Tenant != tenant {
			continue
		}
		for _, e := range u.Emails {
			if strings.EqualFold(e, email) {
				out = append(out, u)
				break
			}
		}
	}
	return out, nil
}

func (d *Dir) PutGroup(_ context.Context, g ports.Group) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.groups[k(g.Tenant, g.ID)] = g
	return nil
}

func (d *Dir) Group(_ context.Context, tenant, id string) (ports.Group, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	g, ok := d.groups[k(tenant, id)]
	return g, ok, nil
}

func (d *Dir) DeleteGroup(_ context.Context, tenant, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.groups, k(tenant, id))
	return nil
}

// GroupsOf returns the names of the user's groups, sorted.
func (d *Dir) GroupsOf(_ context.Context, tenant, user string) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []string
	for _, g := range d.groups {
		if g.Tenant != tenant {
			continue
		}
		for _, m := range g.Members {
			if m == user {
				out = append(out, g.Name)
				break
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

var _ ports.People = (*Dir)(nil)

// Person finds a user of the scope's tenant only; other tenants' ids are not found.
func (d *Dir) Person(_ context.Context, s tenancy.Scope, id string) (ports.User, bool, error) {
	if err := s.Check(); err != nil {
		return ports.User{}, false, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	u, ok := d.users[k(s.Tenant(), id)]
	return u, ok, nil
}

// MyGroups lists the scope user's groups in the scope's tenant.
func (d *Dir) MyGroups(ctx context.Context, s tenancy.Scope) ([]string, error) {
	if err := s.Check(); err != nil {
		return nil, err
	}
	return d.GroupsOf(ctx, s.Tenant(), s.User())
}

// Members lists the user ids of a tenant's group by name, sorted; a group no one has
// created is an error, never an empty group.
func (d *Dir) Members(_ context.Context, tenant, name string) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, g := range d.groups {
		if g.Tenant == tenant && g.Name == name {
			out := append([]string{}, g.Members...)
			sort.Strings(out)
			return out, nil
		}
	}
	return nil, fmt.Errorf("directory: no group %q", name)
}

// ActiveUsers counts a tenant's active users (seats, for the weekly report).
func (d *Dir) ActiveUsers(_ context.Context, tenant string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, u := range d.users {
		if u.Tenant == tenant && u.Active {
			n++
		}
	}
	return n, nil
}
