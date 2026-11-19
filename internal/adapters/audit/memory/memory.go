// Package memory is an in-process audit.Store for development and tests. It is
// append-only by construction: there is no way to change or remove a row.
package memory

import (
	"context"
	"sync"

	"example.com/second-brain/internal/core/audit"
)

// Store holds one cell's rows.
type Store struct {
	mu   sync.Mutex
	rows []audit.Row
	// Fail makes every Append fail (to test that nothing acts unaudited).
	Fail error
}

// New returns an empty store.
func New() *Store { return &Store{} }

// Append implements audit.Store.
func (s *Store) Append(_ context.Context, build func(int64, string) (audit.Row, error)) (audit.Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Fail != nil {
		return audit.Row{}, s.Fail
	}
	seq, hash := int64(0), audit.Zero
	if n := len(s.rows); n > 0 {
		seq, hash = s.rows[n-1].Seq, s.rows[n-1].Hash
	}
	r, err := build(seq, hash)
	if err != nil {
		return audit.Row{}, err
	}
	s.rows = append(s.rows, r)
	return r, nil
}

// Range implements audit.Store.
func (s *Store) Range(_ context.Context, from int64, limit int) ([]audit.Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []audit.Row
	for _, r := range s.rows {
		if r.Seq >= from && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}
