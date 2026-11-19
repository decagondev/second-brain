package memory

import (
	"context"
	"testing"

	"example.com/second-brain/internal/core/ask"
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
