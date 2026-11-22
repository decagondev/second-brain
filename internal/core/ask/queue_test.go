package ask

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"example.com/second-brain/internal/core/retrieve"
)

// SB-1609: the queue's items carry counts and decisions by verdict and reason, never
// cites; reviewing keeps the first review; a later feedback puts the item back.
func TestQueueAndReview(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	fs := &memFeedback{m: map[string]Feedback{}}
	tr := &memTraces{m: map[string]Trace{"tr_1": {ID: "tr_1", Tenant: "acme-logistics", Decisions: []retrieve.Decision{
		{Cite: "docs:secret@1", Verdict: "refuse", Reason: "source_denied"}, {Cite: "docs:a@2", Verdict: "allow", Reason: "allowed"},
		{Cite: "docs:b@2", Verdict: "allow", Reason: "allowed"}}}}}
	_ = fs.PutFeedback(ctx, Feedback{Tenant: "acme-logistics", Trace: "tr_1", Rating: "down", Comment: "wrong year", At: now.Add(-time.Hour), Claims: 2})
	_ = fs.PutFeedback(ctx, Feedback{Tenant: "acme-logistics", Trace: "tr_2", Rating: "up", At: now.Add(-time.Hour)})
	_ = fs.PutFeedback(ctx, Feedback{Tenant: "acme-logistics", Trace: "tr_old", Rating: "down", At: now.Add(-QueueWindow - time.Hour)})
	q, err := Queue(ctx, fs, tr, "acme-logistics", "down", false, now)
	if err != nil || len(q) != 1 || q[0].Trace != "tr_1" || q[0].Comment != "wrong year" {
		t.Fatalf("queue %+v %v", q, err)
	}
	if len(q[0].Decisions) != 2 || q[0].Decisions[0] != (DecisionCount{"allow", "allowed", 2}) || q[0].Decisions[1] != (DecisionCount{"refuse", "source_denied", 1}) {
		t.Fatalf("decisions %+v", q[0].Decisions)
	}
	if _, err := Queue(ctx, fs, tr, "acme-logistics", "meh", false, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("a bad rating: %v", err)
	}
	f, first, err := MarkReviewed(ctx, fs, fs, "acme-logistics", "tr_1", "usr_olga", now)
	if err != nil || !first || f.ReviewedBy != "usr_olga" {
		t.Fatalf("review %+v %v %v", f, first, err)
	}
	if _, first, _ := MarkReviewed(ctx, fs, fs, "acme-logistics", "tr_1", "usr_omar", now); first {
		t.Fatal("a second review replaced the first")
	}
	if q, _ := Queue(ctx, fs, tr, "acme-logistics", "down", false, now); len(q) != 0 {
		t.Fatalf("a reviewed item is still queued: %+v", q)
	}
	if q, _ := Queue(ctx, fs, tr, "acme-logistics", "down", true, now); len(q) != 1 || q[0].Reviewed == nil || q[0].Reviewed.By != "usr_olga" {
		t.Fatalf("reviewed=true: %+v", q)
	}
	_ = fs.PutFeedback(ctx, Feedback{Tenant: "acme-logistics", Trace: "tr_1", Rating: "down", Comment: "still wrong", At: now})
	if q, _ := Queue(ctx, fs, tr, "acme-logistics", "down", false, now); len(q) != 1 || q[0].Comment != "still wrong" {
		t.Fatalf("a later feedback didn't put it back: %+v", q)
	}
	if _, _, err := MarkReviewed(ctx, fs, fs, "acme-logistics", "tr_none", "usr_olga", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("no feedback: %v", err)
	}
}

type memFeedback struct{ m map[string]Feedback }

func (s *memFeedback) PutFeedback(_ context.Context, f Feedback) error { s.m[f.Trace] = f; return nil }
func (s *memFeedback) GetFeedback(_ context.Context, _, trace string) (Feedback, bool, error) {
	f, ok := s.m[trace]
	return f, ok, nil
}
func (s *memFeedback) FeedbackBetween(_ context.Context, tenant string, from, to time.Time) ([]Feedback, error) {
	var out []Feedback
	for _, f := range s.m {
		if f.Tenant == tenant && !f.At.Before(from) && f.At.Before(to) {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].At.Before(out[b].At) })
	return out, nil
}

type memTraces struct {
	Store
	m map[string]Trace
}

func (s *memTraces) GetTrace(_ context.Context, _, id string) (Trace, bool, error) {
	t, ok := s.m[id]
	return t, ok, nil
}
