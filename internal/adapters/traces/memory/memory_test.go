package memory

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/retention"
)

func TestOwnership(t *testing.T) {
	ctx := context.Background()
	s := New()
	if err := s.AppendQuestion(ctx, "acme-logistics", "alice", "cv_1", "q1"); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendQuestion(ctx, "acme-logistics", "bob", "cv_1", "q2"); err == nil {
		t.Error("bob appended to alice's conversation")
	}
	for _, c := range []struct{ tenant, user string }{{"acme-logistics", "bob"}, {"beta-freight", "alice"}} {
		if _, found, _ := s.Questions(ctx, c.tenant, c.user, "cv_1"); found {
			t.Errorf("%s/%s found alice's conversation", c.tenant, c.user)
		}
	}
	if qs, found, _ := s.Questions(ctx, "acme-logistics", "alice", "cv_1"); !found || len(qs) != 1 {
		t.Errorf("alice: %v %v", qs, found)
	}
	if err := s.PutTrace(ctx, ask.Trace{ID: "tr_1", Tenant: "acme-logistics"}); err != nil {
		t.Fatal(err)
	}
	if s.PutTrace(ctx, ask.Trace{ID: "tr_1", Tenant: "acme-logistics"}) == nil {
		t.Error("duplicate trace id accepted")
	}
	if _, found, _ := s.GetTrace(ctx, "beta-freight", "tr_1"); found {
		t.Error("another tenant read the trace")
	}
}

func TestRetention(t *testing.T) {
	ctx := context.Background()
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := old.Add(400 * 24 * time.Hour)
	at := old
	s := New()
	s.Now = func() time.Time { return at }
	_ = s.PutTrace(ctx, ask.Trace{ID: "tr_1", Tenant: "acme-logistics", User: "usr_a", At: old})
	_ = s.PutTrace(ctx, ask.Trace{ID: "tr_2", Tenant: "acme-logistics", User: "usr_a", At: recent})
	_ = s.PutTrace(ctx, ask.Trace{ID: "tr_3", Tenant: "beta-freight", User: "usr_a", At: old})
	_ = s.AppendQuestion(ctx, "acme-logistics", "usr_a", "cv_1", "q")
	at = recent
	_ = s.AppendQuestion(ctx, "acme-logistics", "usr_b", "cv_2", "q")
	_ = s.PutFeedback(ctx, ask.Feedback{Tenant: "acme-logistics", User: "usr_a", Trace: "tr_2", At: recent})
	got := map[string]int{}
	for _, r := range s.Retention() {
		n, _, err := r.Expire(ctx, "acme-logistics", recent.Add(-time.Hour), nil)
		if err != nil {
			t.Fatal(err)
		}
		got[r.Name()] = n
	}
	if got["traces"] != 1 || got["conversations"] != 1 || got["feedback"] != 0 {
		t.Fatalf("expire: %v", got)
	}
	if _, found, _ := s.GetTrace(ctx, "beta-freight", "tr_3"); !found {
		t.Fatal("another tenant's trace was expired")
	}
	for _, r := range s.Retention() {
		n, _ := r.Erase(ctx, "acme-logistics", "usr_a")
		got[r.Name()] = n
	}
	if got["traces"] != 1 || got["conversations"] != 0 || got["feedback"] != 1 {
		t.Fatalf("erase: %v", got)
	}
	if _, found, _ := s.Questions(ctx, "acme-logistics", "usr_b", "cv_2"); !found {
		t.Fatal("another user's conversation was erased")
	}
}

// A conversation asked in one space is kept by a hold on that space only.
func TestConversationSpaceForHolds(t *testing.T) {
	ctx := context.Background()
	s := New()
	old := time.Now().Add(-400 * 24 * time.Hour)
	s.Now = func() time.Time { return old }
	_ = s.AppendQuestion(ctx, "acme-logistics", "usr_a", "cv_fin", "q")
	_ = s.TagSpace(ctx, "acme-logistics", "cv_fin", "spc_finance")
	_ = s.AppendQuestion(ctx, "acme-logistics", "usr_a", "cv_eng", "q")
	_ = s.TagSpace(ctx, "acme-logistics", "cv_eng", "spc_eng")
	held := func(space, _ string) bool { return space == "" || space == "spc_finance" }
	var conv retention.Store
	for _, r := range s.Retention() {
		if r.Name() == "conversations" {
			conv = r
		}
	}
	n, k, err := conv.Expire(ctx, "acme-logistics", time.Now(), held)
	if err != nil || n != 1 || k != 1 {
		t.Fatalf("deleted %d, kept %d (%v); want the eng conversation deleted and the finance one kept", n, k, err)
	}
}
