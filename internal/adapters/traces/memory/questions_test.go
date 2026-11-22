package memory

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/retention"
)

// SB-1703: saved questions and shares are kept under retention and erased with the person.
func TestQuestionsRetentionAndErasure(t *testing.T) {
	ctx := context.Background()
	s := New()
	old, now := time.Unix(1_700_000_000, 0), time.Unix(1_800_000_000, 0)
	for _, q := range []ask.SavedQuestion{
		{ID: "sq_1", Tenant: "acme-logistics", User: "alice", Question: "a", Space: "company", At: old},
		{ID: "sq_2", Tenant: "acme-logistics", User: "alice", Question: "b", Space: "spc_finance", At: now},
		{ID: "sq_3", Tenant: "acme-logistics", User: "bob", Question: "c", Space: "company", At: now},
	} {
		if err := s.PutSaved(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.PutShare(ctx, ask.Share{ID: "sh_1", Tenant: "acme-logistics", User: "alice", Question: "a", Space: "company", At: now}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.SavedOf(ctx, "acme-logistics", "alice"); len(got) != 2 || got[0].ID != "sq_2" {
		t.Fatalf("newest first: %+v", got)
	}
	var n int
	for _, r := range s.Retention() {
		k, _, err := r.Expire(ctx, "acme-logistics", now.Add(-time.Hour), retention.Held(nil))
		if err != nil {
			t.Fatal(err)
		}
		n += k
	}
	if n != 1 {
		t.Fatalf("expired %d, want the one old saved question", n)
	}
	for _, r := range s.Retention() {
		if _, err := r.Erase(ctx, "acme-logistics", "alice"); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := s.SavedOf(ctx, "acme-logistics", "alice"); len(got) != 0 {
		t.Fatalf("erased person still has %v", got)
	}
	if _, ok, _ := s.GetShare(ctx, "acme-logistics", "sh_1"); ok {
		t.Fatal("a share survived its sharer's erasure")
	}
	if got, _ := s.SavedOf(ctx, "acme-logistics", "bob"); len(got) != 1 {
		t.Fatal("erasure took someone else's question")
	}
	if found, _ := s.DeleteSaved(ctx, "acme-logistics", "alice", "sq_3"); found {
		t.Fatal("alice deleted bob's question")
	}
}
