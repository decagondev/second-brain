// Package proposals holds stewards' proposed space changes until an operator approves or
// rejects the exact manifest they would make (sb-spaces(7), Delegated changes; SB-1201).
package proposals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"

	"example.com/second-brain/internal/core/manifest"
)

// States of a proposal.
const (
	Open     = "open"
	Approved = "approved"
	Rejected = "rejected"
	Stale    = "stale" // the applied manifest changed before approval
)

// Proposal is one steward's proposed change to one space.
type Proposal struct {
	ID        string    `json:"id"`
	Space     string    `json:"space"`
	Requester string    `json:"requester"`
	Base      string    `json:"base"` // hash of the applied manifest it was made from
	Hash      string    `json:"hash"` // hash of the manifest it would make
	Manifest  string    `json:"manifest"`
	Diff      []string  `json:"diff"`
	State     string    `json:"state"`
	At        time.Time `json:"at"`
	DecidedBy string    `json:"decided_by,omitempty"`
}

// Hash identifies a manifest: SHA-256 of its canonical text.
func Hash(t manifest.Tenant) string {
	h := sha256.Sum256(manifest.Format(t))
	return hex.EncodeToString(h[:])
}

// Store keeps one cell's proposals.
type Store interface {
	Put(ctx context.Context, p Proposal) error
	Get(ctx context.Context, id string) (p Proposal, found bool, err error)
	List(ctx context.Context) ([]Proposal, error)
}

// ErrNotFound is a proposal id the store doesn't hold.
var ErrNotFound = errors.New("proposals: no such proposal")

// MemoryStore is an in-process Store (development and tests).
type MemoryStore struct {
	mu sync.Mutex
	m  map[string]Proposal
}

// Put implements Store.
func (s *MemoryStore) Put(_ context.Context, p Proposal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]Proposal{}
	}
	p.Diff = append([]string(nil), p.Diff...)
	s.m[p.ID] = p
	return nil
}

// Get implements Store.
func (s *MemoryStore) Get(_ context.Context, id string) (Proposal, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.m[id]
	return p, ok, nil
}

// List implements Store, oldest first.
func (s *MemoryStore) List(_ context.Context) ([]Proposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Proposal, 0, len(s.m))
	for _, p := range s.m {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
