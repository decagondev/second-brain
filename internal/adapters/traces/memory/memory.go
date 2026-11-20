// Package memory is an in-memory ask.Store for development and tests: traces and
// conversations for one cell. A Postgres store replaces it with the cell runtime.
package memory

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/retention"
)

// Store is safe for concurrent use.
type Store struct {
	mu       sync.Mutex
	traces   map[string]ask.Trace
	convs    map[string]conv
	feedback map[string]ask.Feedback // tenant/trace
	// Now stamps conversations for retention; nil is time.Now.
	Now func() time.Time
}

type conv struct {
	tenant, user string
	questions    []string
	at           time.Time // the last question
}

// New returns an empty store.
func New() *Store {
	return &Store{traces: map[string]ask.Trace{}, convs: map[string]conv{}, feedback: map[string]ask.Feedback{}}
}

// PutTrace implements ask.Store.
func (s *Store) PutTrace(_ context.Context, t ask.Trace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.traces[t.ID]; dup {
		return errors.New("traces: duplicate trace id")
	}
	s.traces[t.ID] = t
	return nil
}

// GetTrace implements ask.Store.
func (s *Store) GetTrace(_ context.Context, tenant, id string) (ask.Trace, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.traces[id]
	if !ok || t.Tenant != tenant {
		return ask.Trace{}, false, nil
	}
	return t, true, nil
}

// Questions implements ask.Store.
func (s *Store) Questions(_ context.Context, tenant, user, id string) ([]string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.convs[id]
	if !ok || c.tenant != tenant || c.user != user {
		return nil, false, nil
	}
	return append([]string(nil), c.questions...), true, nil
}

// AppendQuestion implements ask.Store.
func (s *Store) AppendQuestion(_ context.Context, tenant, user, id, q string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.convs[id]
	if ok && (c.tenant != tenant || c.user != user) {
		return errors.New("traces: conversation belongs to someone else")
	}
	c.tenant, c.user = tenant, user
	c.questions = append(c.questions, q)
	c.at = time.Now()
	if s.Now != nil {
		c.at = s.Now()
	}
	s.convs[id] = c
	return nil
}

// PutFeedback implements ask.FeedbackStore; a later feedback on a trace replaces it.
func (s *Store) PutFeedback(_ context.Context, f ask.Feedback) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.feedback[f.Tenant+"/"+f.Trace] = f
	return nil
}

// FeedbackBetween implements ask.FeedbackStore: the tenant's feedback in [from, to), oldest first.
func (s *Store) FeedbackBetween(_ context.Context, tenant string, from, to time.Time) ([]ask.Feedback, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ask.Feedback
	for _, f := range s.feedback {
		if f.Tenant == tenant && !f.At.Before(from) && f.At.Before(to) {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At) || (out[i].At.Equal(out[j].At) && out[i].Trace < out[j].Trace)
	})
	return out, nil
}

// Retention returns the store's three retention stores: traces (answers and drafts),
// conversations and feedback (sb-tenant(5), Retention and deletion).
func (s *Store) Retention() []retention.Store {
	// Feedback first: it takes its space from its trace, which must still be there (SB-1401).
	return []retention.Store{view{s, "feedback"}, view{s, "conversations"}, view{s, "traces"}}
}

type view struct {
	s    *Store
	name string
}

func (v view) Name() string { return v.name }

// Expire implements retention.Store: a trace carries its space; feedback takes its trace's;
// a conversation's space isn't kept, so any hold on its user, or on a space, keeps it.
func (v view) Expire(_ context.Context, tenant string, before time.Time, held retention.Held) (int, int, error) {
	n, k := v.drop(func(t, _, _ string, at time.Time) bool { return t == tenant && at.Before(before) }, held)
	return n, k, nil
}

// Erase implements retention.Store.
func (v view) Erase(_ context.Context, tenant, user string) (int, error) {
	n, _ := v.drop(func(t, u, _ string, _ time.Time) bool { return t == tenant && u == user }, nil)
	return n, nil
}

func (v view) drop(match func(tenant, user, space string, at time.Time) bool, held retention.Held) (int, int) {
	s := v.s
	s.mu.Lock()
	defer s.mu.Unlock()
	n, kept := 0, 0
	try := func(tenant, user, space string, at time.Time) bool {
		if !match(tenant, user, space, at) {
			return false
		}
		if held.Is(space, user) {
			kept++
			return false
		}
		n++
		return true
	}
	switch v.name {
	case "traces":
		for k, t := range s.traces {
			if try(t.Tenant, t.User, spaceOf(t.Space), t.At) {
				delete(s.traces, k)
			}
		}
	case "conversations":
		for k, c := range s.convs {
			if try(c.tenant, c.user, "", c.at) {
				delete(s.convs, k)
			}
		}
	case "feedback":
		for k, f := range s.feedback {
			space := ""
			if t, ok := s.traces[f.Trace]; ok && t.Tenant == f.Tenant {
				space = spaceOf(t.Space)
			}
			if try(f.Tenant, f.User, space, f.At) {
				delete(s.feedback, k)
			}
		}
	}
	return n, kept
}

// spaceOf is a record's space for holds: "" for one asked company-wide, which any hold on
// a space might still cover through its sources, so it is treated as unknown.
func spaceOf(s string) string {
	if s == "company" {
		return ""
	}
	return s
}
