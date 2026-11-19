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
	return []retention.Store{view{s, "traces"}, view{s, "conversations"}, view{s, "feedback"}}
}

type view struct {
	s    *Store
	name string
}

func (v view) Name() string { return v.name }

// Expire implements retention.Store.
func (v view) Expire(_ context.Context, tenant string, before time.Time) (int, error) {
	return v.drop(func(t, _ string, at time.Time) bool { return t == tenant && at.Before(before) }), nil
}

// Erase implements retention.Store.
func (v view) Erase(_ context.Context, tenant, user string) (int, error) {
	return v.drop(func(t, u string, _ time.Time) bool { return t == tenant && u == user }), nil
}

func (v view) drop(match func(tenant, user string, at time.Time) bool) int {
	s := v.s
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	switch v.name {
	case "traces":
		for k, t := range s.traces {
			if match(t.Tenant, t.User, t.At) {
				delete(s.traces, k)
				n++
			}
		}
	case "conversations":
		for k, c := range s.convs {
			if match(c.tenant, c.user, c.at) {
				delete(s.convs, k)
				n++
			}
		}
	case "feedback":
		for k, f := range s.feedback {
			if match(f.Tenant, f.User, f.At) {
				delete(s.feedback, k)
				n++
			}
		}
	}
	return n
}
