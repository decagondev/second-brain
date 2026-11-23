// Package memory is an in-process session store for one cell (development and tests).
// A production cell keeps sessions in its own database; each cell has its own store, so
// a session id means nothing in another cell.
package memory

import (
	"context"
	"sort"
	"sync"

	"example.com/second-brain/internal/core/ports"
)

// Store is a mutex-guarded map of sessions.
type Store struct {
	mu sync.Mutex
	m  map[string]ports.Session
}

var _ ports.Sessions = (*Store)(nil)

func New() *Store { return &Store{m: map[string]ports.Session{}} }

func (s *Store) Put(_ context.Context, x ports.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[x.ID] = x
	return nil
}

func (s *Store) Get(_ context.Context, id string) (ports.Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.m[id]
	return x, ok, nil
}

func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
	return nil
}

func (s *Store) RevokeUser(_ context.Context, tenant, user string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, x := range s.m {
		if x.Tenant == tenant && x.User == user {
			delete(s.m, id)
			n++
		}
	}
	return n, nil
}

func (s *Store) ListUser(_ context.Context, tenant, user string) ([]ports.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ports.Session
	for _, x := range s.m {
		if x.Tenant == tenant && x.User == user {
			out = append(out, x)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SignedIn.Before(out[j].SignedIn) })
	return out, nil
}
