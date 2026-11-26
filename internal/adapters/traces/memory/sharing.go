package memory

import (
	"context"
	"sort"
	"time"

	"example.com/second-brain/internal/core/ask"
)

var (
	_ ask.WideConfirmStore = (*Store)(nil)
	_ ask.SharingStore     = (*Store)(nil)
)

// PutWideConfirm implements ask.WideConfirmStore (SB-5402): one per document.
func (s *Store) PutWideConfirm(_ context.Context, c ask.WideConfirm) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.wide == nil {
		s.wide = map[string]ask.WideConfirm{}
	}
	s.wide[c.Tenant+"/"+c.Doc] = c
	return nil
}

// WideConfirm implements ask.WideConfirmStore.
func (s *Store) WideConfirm(_ context.Context, tenant, doc string) (ask.WideConfirm, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.wide[tenant+"/"+doc]
	return c, ok, nil
}

// PutSharingEvent implements ask.SharingStore (SB-5403).
func (s *Store) PutSharingEvent(_ context.Context, e ask.SharingEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sharing = append(s.sharing, e)
	return nil
}

// SharingEvents implements ask.SharingStore: those at or after since, oldest first.
func (s *Store) SharingEvents(_ context.Context, tenant string, since time.Time) ([]ask.SharingEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ask.SharingEvent
	for _, e := range s.sharing {
		if e.Tenant == tenant && !e.At.Before(since) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out, nil
}
