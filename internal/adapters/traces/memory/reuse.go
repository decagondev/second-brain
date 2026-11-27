package memory

import (
	"context"

	"example.com/second-brain/internal/core/ask"
)

var _ ask.ReuseStore = (*Store)(nil)

// PutReuse implements ask.ReuseStore (SB-5603): the newest per key.
func (s *Store) PutReuse(_ context.Context, e ask.ReuseEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reuse == nil {
		s.reuse = map[string]ask.ReuseEntry{}
	}
	s.reuse[e.Tenant+"/"+e.Key] = e
	return nil
}

// Reuse implements ask.ReuseStore.
func (s *Store) Reuse(_ context.Context, tenant, key string) (ask.ReuseEntry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.reuse[tenant+"/"+key]
	return e, ok, nil
}
