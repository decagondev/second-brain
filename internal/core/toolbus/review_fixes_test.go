package toolbus

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/tenancy"
)

func dutyFixture() *fixture {
	f := newFixture()
	tool := f.bus.Tools["update"]
	tool.Duty = &Duty{Second: "grp_finance_leads", Above: 1000}
	f.bus.Tools["update"] = tool
	f.bus.Managers = func(context.Context, string) (string, error) { return "", nil }
	return f
}

// SB-1401: an edit that raises the amount above the duty needs both approvals again.
func TestEditRecomputesTheDuty(t *testing.T) {
	f := dutyFixture()
	ctx := context.Background()
	a, err := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Agent: "ticket-update", Tool: "update", Target: "tickets:FIN-1", Payload: json.RawMessage(`{"amount": 50}`)})
	if err != nil || a.Needs != 1 {
		t.Fatalf("propose: %+v %v", a, err)
	}
	a, err = f.bus.Edit(ctx, sc(t, "usr_dana"), a.ID, json.RawMessage(`{"amount": 900000000}`))
	if err != nil || a.Needs != 2 || a.SecondApprovers != "grp_finance_leads" {
		t.Fatalf("after the edit: needs %d (%q) %v", a.Needs, a.SecondApprovers, err)
	}
	a, err = f.bus.Approve(ctx, sc(t, "usr_erin"), a.ID, a.PayloadSHA256, "")
	if err != nil || a.State != approvals.PendingApproval {
		t.Fatalf("one approval ran a raised amount: %+v %v", a, err)
	}
}

// SB-1401: edits never move the expiry.
func TestEditKeepsTheExpiry(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	a, _ := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Agent: "ticket-update", Tool: "update", Target: "tickets:OPS-1", Payload: payload})
	exp := a.ExpiresAt
	*f.clock = f.clock.Add(20 * time.Hour)
	a, err := f.bus.Edit(ctx, sc(t, "usr_dana"), a.ID, json.RawMessage(`{"comment":"later"}`))
	if err != nil || !a.ExpiresAt.Equal(exp) {
		t.Fatalf("expiry moved from %v to %v (%v)", exp, a.ExpiresAt, err)
	}
}

// SB-1401: a large negative amount is as large as a positive one.
func TestNegativeAmountNeedsBoth(t *testing.T) {
	d := &Duty{Second: "g", Above: 1000}
	for p, want := range map[string]int{`{"amount": -5000000}`: 2, `{"amount": -999}`: 1, `{"amount": 1000}`: 1, `{"amount": 1001}`: 2} {
		if got := d.Needs(json.RawMessage(p)); got != want {
			t.Errorf("%s: needs %d, want %d", p, got, want)
		}
	}
}

// SB-1401: rejecting takes the same checks as approving: not your own, not your report's,
// and not without reporting lines under a duty.
func TestRejectTakesTheApprovalChecks(t *testing.T) {
	f := dutyFixture()
	f.bus.Managers = func(_ context.Context, u string) (string, error) {
		if u == "usr_dana" {
			return "usr_boss", nil
		}
		return "", nil
	}
	ctx := context.Background()
	a, _ := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Agent: "ticket-update", Tool: "update", Target: "tickets:FIN-1", Payload: json.RawMessage(`{"amount": 5000}`)})
	for _, who := range []string{"usr_dana", "usr_boss"} {
		if _, err := f.bus.Reject(ctx, sc(t, who), a.ID, "no"); !errors.Is(err, ErrNoApproval) {
			t.Errorf("%s rejected: %v", who, err)
		}
	}
	f.bus.Managers = nil
	if _, err := f.bus.Approve(ctx, sc(t, "usr_erin"), a.ID, a.PayloadSHA256, ""); !errors.Is(err, ErrNoApproval) {
		t.Fatalf("a duty was approved without reporting lines: %v", err)
	}
}

type fakeBarriers struct{ across bool }

func (fakeBarriers) Target(context.Context, tenancy.Scope, string) (map[string]any, error) {
	return nil, nil
}
func (f fakeBarriers) Approval(context.Context, tenancy.Scope, string) (map[string]any, error) {
	if f.across {
		return map[string]any{"barred_groups": []string{"group:deal"}, "requester_groups": []string{"group:deal"}}, nil
	}
	return map[string]any{"barred_groups": []string{}, "requester_groups": []string{"group:deal"}}, nil
}

// SB-1401: an approver across a barrier can't read the action.
func TestGetHidesTheActionAcrossABarrier(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	a, _ := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Agent: "ticket-update", Tool: "update", Target: "tickets:OPS-1", Payload: payload})
	f.bus.Barriers = fakeBarriers{across: true}
	if _, err := f.bus.Get(ctx, sc(t, "usr_erin"), a.ID); !errors.Is(err, ErrPerm) {
		t.Fatalf("read across a barrier: %v", err)
	}
	if _, err := f.bus.Get(ctx, sc(t, "usr_dana"), a.ID); err != nil {
		t.Fatalf("the requester can't read their own action: %v", err)
	}
	f.bus.Barriers = fakeBarriers{}
	if _, err := f.bus.Get(ctx, sc(t, "usr_erin"), a.ID); err != nil {
		t.Fatalf("an approver with no barrier can't read: %v", err)
	}
}
