// Package memory is an in-process kill.Store (development and tests). Down makes every
// call fail, to exercise the checker's fail-safe.
package memory

import (
	"context"
	"errors"
	"sort"
	"sync"

	"example.com/second-brain/internal/core/kill"
)

// Store holds switches per tenant.
type Store struct {
	mu   sync.Mutex
	sw   map[string]map[string]kill.Switch
	down bool
}

// ErrDown is an unreachable store.
var ErrDown = errors.New("kill store unreachable")

// New returns an empty store.
func New() *Store { return &Store{sw: map[string]map[string]kill.Switch{}} }

// SetDown makes the store (un)reachable.
func (s *Store) SetDown(down bool) { s.mu.Lock(); s.down = down; s.mu.Unlock() }

// Set implements kill.Store.
func (s *Store) Set(_ context.Context, tenant string, sw kill.Switch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sw[tenant] == nil {
		s.sw[tenant] = map[string]kill.Switch{}
	}
	s.sw[tenant][sw.Scope] = sw
	return nil
}

// Clear implements kill.Store.
func (s *Store) Clear(_ context.Context, tenant, scope string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sw[tenant], scope)
	return nil
}

// List implements kill.Store.
func (s *Store) List(_ context.Context, tenant string) ([]kill.Switch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.down {
		return nil, ErrDown
	}
	var out []kill.Switch
	for _, v := range s.sw[tenant] {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scope < out[j].Scope })
	return out, nil
}
