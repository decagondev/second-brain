// Package holds keeps a cell's legal holds (sb-spaces(7), Legal holds, exactly; SB-1207):
// a hold on a space, on custodians, or both stops deletion in its scope, and only there.
package holds

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// Hold is one legal hold.
type Hold struct {
	ID            string    `json:"id"`
	Matter        string    `json:"matter"`
	Space         string    `json:"space,omitempty"` // a space name, or "" for every space
	Users         []string  `json:"users"`           // custodians, or none for everyone in the space
	Reason        string    `json:"reason"`
	PlacedBy      string    `json:"placed_by"`
	PlacedAt      time.Time `json:"placed_at"`
	Released      bool      `json:"released"`
	ReleasedBy    string    `json:"released_by,omitempty"`
	ReleasedAt    time.Time `json:"released_at,omitempty"`
	ReleaseReason string    `json:"release_reason,omitempty"`
}

// Covers reports whether h covers a record in spaceID (spc_NAME, or "" when the store
// doesn't know it) about user ("" when unknown). Unknown is covered: it never deletes.
func (h Hold) Covers(spaceID, user string) bool {
	if h.Released {
		return false
	}
	inSpace := h.Space == "" || spaceID == "" || "spc_"+h.Space == spaceID
	ofUser := len(h.Users) == 0 || user == ""
	for _, u := range h.Users {
		ofUser = ofUser || u == user
	}
	return inSpace && ofUser
}

// spaceID is the hold's space as an audit row names it, "" for a hold on every space.
func (h Hold) spaceID() string {
	if h.Space == "" {
		return ""
	}
	return "spc_" + h.Space
}

// Set is the active holds.
type Set []Hold

// Held is retention's question: is a record in spaceID about user held by any hold.
func (s Set) Held(spaceID, user string) bool {
	for _, h := range s {
		if h.Covers(spaceID, user) {
			return true
		}
	}
	return false
}

// HeldSpace reports whether space spaceID can't be removed: a hold on it, or a hold on
// custodians in every space.
func (s Set) HeldSpace(spaceID string) bool {
	for _, h := range s {
		if !h.Released && (h.Space == "" || "spc_"+h.Space == spaceID) {
			return true
		}
	}
	return false
}

// Store keeps one cell's holds.
type Store interface {
	Put(ctx context.Context, h Hold) error
	Get(ctx context.Context, id string) (h Hold, found bool, err error)
	List(ctx context.Context) ([]Hold, error)
}

// Active reads the active holds. An error is an error: the caller refuses, never deletes.
func Active(ctx context.Context, st Store) (Set, error) {
	all, err := st.List(ctx)
	if err != nil {
		return nil, err
	}
	var out Set
	for _, h := range all {
		if !h.Released {
			out = append(out, h)
		}
	}
	return out, nil
}

// MemoryStore is an in-process Store.
type MemoryStore struct {
	mu sync.Mutex
	m  map[string]Hold
}

func clone(h Hold) Hold { h.Users = append([]string{}, h.Users...); return h }

// Put implements Store.
func (s *MemoryStore) Put(_ context.Context, h Hold) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]Hold{}
	}
	s.m[h.ID] = clone(h)
	return nil
}

// Get implements Store.
func (s *MemoryStore) Get(_ context.Context, id string) (Hold, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.m[id]
	return clone(h), ok, nil
}

// List implements Store, by id.
func (s *MemoryStore) List(_ context.Context) ([]Hold, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Hold, 0, len(s.m))
	for _, h := range s.m {
		out = append(out, clone(h))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Auditor writes audit rows (audit.Log).
type Auditor interface {
	Write(ctx context.Context, e audit.Event) (audit.Row, error)
}

// Service places and releases holds, auditing each first (LAW-11).
type Service struct {
	Store Store
	Audit Auditor
	NewID func(prefix string) string
	Now   func() time.Time
}

// Errors.
var (
	ErrInvalid = errors.New("holds: EINVAL")
	ErrAudit   = errors.New("holds: the audit row can't be written, so nothing changed")
	ErrState   = errors.New("holds: no such active hold")
)

var userRe = regexp.MustCompile(`^usr_[A-Za-z0-9_]{1,64}$`)

func scopeText(h Hold) string {
	sp := h.Space
	if sp == "" {
		sp = "*"
	}
	us := "*"
	if len(h.Users) > 0 {
		us = strings.Join(h.Users, ",")
	}
	return "space=" + sp + " users=" + us
}

// Place records a hold: audited first, then stored. A hold names a space, custodians, or
// both, and a matter and a reason.
func (s Service) Place(ctx context.Context, by, matter, space string, users []string, reason string) (Hold, error) {
	matter, reason = strings.TrimSpace(matter), strings.TrimSpace(reason)
	switch {
	case matter == "" || reason == "":
		return Hold{}, fmt.Errorf("%w: a hold needs a matter and a reason", ErrInvalid)
	case space == "" && len(users) == 0:
		return Hold{}, fmt.Errorf("%w: a hold needs a space, custodians, or both", ErrInvalid)
	}
	for _, u := range users {
		if !userRe.MatchString(u) {
			return Hold{}, fmt.Errorf("%w: %q is not a user id", ErrInvalid, u)
		}
	}
	h := Hold{ID: s.NewID("hold_"), Matter: matter, Space: space, Users: append([]string{}, users...), Reason: reason,
		PlacedBy: by, PlacedAt: s.Now().UTC()}
	if _, err := s.Audit.Write(ctx, audit.Event{Actor: by, Kind: "retention", Subject: "hold:" + h.ID, State: "placed",
		Reason: "matter " + matter + " " + scopeText(h) + ": " + reason, Space: h.spaceID()}); err != nil {
		return Hold{}, fmt.Errorf("%w: %v", ErrAudit, err)
	}
	if err := s.Store.Put(ctx, h); err != nil {
		return Hold{}, err
	}
	return h, nil
}

// Release ends a hold: audited first, then stored; it stops protecting at once.
func (s Service) Release(ctx context.Context, by, id, reason string) (Hold, error) {
	if strings.TrimSpace(reason) == "" {
		return Hold{}, fmt.Errorf("%w: a release needs a reason", ErrInvalid)
	}
	h, found, err := s.Store.Get(ctx, id)
	if err != nil {
		return Hold{}, err
	}
	if !found || h.Released {
		return Hold{}, ErrState
	}
	if _, err := s.Audit.Write(ctx, audit.Event{Actor: by, Kind: "retention", Subject: "hold:" + h.ID, State: "released",
		Reason: "matter " + h.Matter + " " + scopeText(h) + ": " + reason, Space: h.spaceID()}); err != nil {
		return Hold{}, fmt.Errorf("%w: %v", ErrAudit, err)
	}
	h.Released, h.ReleasedBy, h.ReleasedAt, h.ReleaseReason = true, by, s.Now().UTC(), reason
	if err := s.Store.Put(ctx, h); err != nil {
		return Hold{}, err
	}
	return h, nil
}
