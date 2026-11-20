// Package memory is an in-process space registry for one cell (development and tests).
// A production cell keeps its spaces in the cell database (postgres/spaces); the port is
// the same.
package memory

import (
	"context"
	"sort"
	"sync"

	"example.com/second-brain/internal/core/ports"
)

// Store is a mutex-guarded space registry. Fail, if set, makes every call fail, for
// tests of a registry that can't answer.
type Store struct {
	mu     sync.Mutex
	spaces map[string]ports.Space // tenant/id
	Fail   error
}

var _ ports.SpaceStore = (*Store)(nil)

// New returns an empty registry.
func New() *Store { return &Store{spaces: map[string]ports.Space{}} }

func k(tenant, id string) string { return tenant + "/" + id }

func clone(s ports.Space) ports.Space {
	c := s
	c.Members = append([]string(nil), s.Members...)
	c.Stewards = append([]string(nil), s.Stewards...)
	c.Agents = append([]string(nil), s.Agents...)
	c.Channels = append([]string(nil), s.Channels...)
	c.Scope = map[string][]string{}
	for kind, ps := range s.Scope {
		c.Scope[kind] = append([]string(nil), ps...)
	}
	return c
}

// PutSpace creates or replaces a space.
func (st *Store) PutSpace(_ context.Context, s ports.Space) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.Fail != nil {
		return st.Fail
	}
	st.spaces[k(s.Tenant, s.ID)] = clone(s)
	return nil
}

// DeleteSpace removes a space; deleting one that isn't there is not an error.
func (st *Store) DeleteSpace(_ context.Context, tenant, id string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.Fail != nil {
		return st.Fail
	}
	delete(st.spaces, k(tenant, id))
	return nil
}

// Space finds one of the tenant's spaces; another tenant's ids are not found.
func (st *Store) Space(_ context.Context, tenant, id string) (ports.Space, bool, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.Fail != nil {
		return ports.Space{}, false, st.Fail
	}
	s, ok := st.spaces[k(tenant, id)]
	if !ok {
		return ports.Space{}, false, nil
	}
	return clone(s), true, nil
}

// Spaces lists the tenant's spaces by id.
func (st *Store) Spaces(_ context.Context, tenant string) ([]ports.Space, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.Fail != nil {
		return nil, st.Fail
	}
	var out []ports.Space
	for _, s := range st.spaces {
		if s.Tenant == tenant {
			out = append(out, clone(s))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
