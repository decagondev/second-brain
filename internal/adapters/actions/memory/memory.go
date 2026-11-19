// Package memory is an in-process approvals.Store (development and tests). Move is a
// compare-and-set on the state, so two approvers can't both move one action.
package memory

import (
	"context"
	"fmt"
	"sync"

	"example.com/second-brain/internal/core/approvals"
)

// Store holds actions by tenant and id.
type Store struct {
	mu sync.Mutex
	m  map[string]approvals.Action
}

// New returns an empty store.
func New() *Store { return &Store{m: map[string]approvals.Action{}} }

func key(tenant, id string) string { return tenant + "/" + id }

// Put implements approvals.Store; ids are never reused.
func (s *Store) Put(_ context.Context, a approvals.Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.m[key(a.Tenant, a.ID)]; dup {
		return fmt.Errorf("actions: duplicate id %s", a.ID)
	}
	s.m[key(a.Tenant, a.ID)] = a
	return nil
}

// Get implements approvals.Store.
func (s *Store) Get(_ context.Context, tenant, id string) (approvals.Action, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.m[key(tenant, id)]
	return a, ok, nil
}

// Move implements approvals.Store.
func (s *Store) Move(_ context.Context, tenant, id string, from approvals.State, change func(*approvals.Action) error) (approvals.Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.m[key(tenant, id)]
	if !ok {
		return approvals.Action{}, fmt.Errorf("actions: no action %s", id)
	}
	if a.State != from {
		return a, approvals.ErrMoved
	}
	old := a.State
	if err := change(&a); err != nil {
		return a, err
	}
	if a.State != old && !approvals.CanMove(old, a.State) {
		return a, fmt.Errorf("actions: %s -> %s is not a transition in sb-approvals(7)", old, a.State)
	}
	s.m[key(tenant, id)] = a
	return a, nil
}
