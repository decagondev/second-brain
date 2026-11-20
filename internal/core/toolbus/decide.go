// Approving and rejecting: every check a decision takes (sb-approvals(7), Approving, exactly; SB-1204, SB-1401).

package toolbus

import (
	"context"
	"fmt"
	"strings"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"
)

// acrossBarrier reads an approval context: the requester's groups meet the viewer's barred
// groups.
func acrossBarrier(actx map[string]any) bool {
	barred, _ := actx["barred_groups"].([]string)
	req, _ := actx["requester_groups"].([]string)
	for _, g := range req {
		for _, x := range barred {
			if g == x {
				return true
			}
		}
	}
	return false
}

// mayDecide is every check an approval or a rejection shares (SB-1204, SB-1401): not in
// the requester's reporting line, and policy allows deciding for the group whose turn it is
// (the workflow's first, the duty's second), with the barrier facts. Under a duty, a cell
// that can't read reporting lines refuses.
func (b *Bus) mayDecide(ctx context.Context, s tenancy.Scope, a approvals.Action) error {
	switch {
	case b.Managers == nil && a.Needs == 2:
		return fmt.Errorf("%w: segregation of duties needs the directory's reporting lines, and this cell has none", ErrNoApproval)
	case b.Managers != nil:
		line, err := approvals.ReportingLine(ctx, b.Managers, a.Requester)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrNoApproval, err)
		}
		for _, m := range line {
			if m == s.User() {
				return fmt.Errorf("%w: the requester reports to you", ErrNoApproval)
			}
		}
	}
	group := a.Approvers
	if a.Needs == 2 && a.FirstApprovedBy != "" {
		group = a.SecondApprovers // the duty's group decides second
	}
	actx, err := b.approveContext(ctx, s, a)
	if err != nil {
		return err
	}
	if err := b.Policy.Require(ctx, s, "approve", policy.Entity{Type: "ProposedAction", ID: a.ID, Tenant: b.Tenant,
		Attrs: map[string]any{"workflow": a.Workflow, "requester": policy.Ref{Type: "User", ID: a.Requester},
			"approvers": policy.Ref{Type: "Group", ID: group}}}, actx); err != nil {
		return fmt.Errorf("%w: %v", ErrNoApproval, err)
	}
	return nil
}

// approveContext is the barrier context of deciding on a's approval.
func (b *Bus) approveContext(ctx context.Context, s tenancy.Scope, a approvals.Action) (map[string]any, error) {
	if b.Barriers == nil {
		return nil, nil
	}
	x, err := b.Barriers.Approval(ctx, s, a.Requester)
	if err != nil {
		return nil, fmt.Errorf("%w: EBARRIER the requester can't be placed: %v", ErrNoApproval, err)
	}
	return x, nil
}

// Approve checks, in order: PendingApproval, not expired, echoed hash, not the
// requester, policy, kill; records the approval (audited first); then executes.
func (b *Bus) Approve(ctx context.Context, s tenancy.Scope, id, echoedHash, note string) (approvals.Action, error) {
	a, err := b.load(ctx, s, id)
	if err != nil {
		return a, err
	}
	switch {
	case a.State != approvals.PendingApproval:
		return a, fmt.Errorf("%w: the action is %s, not PendingApproval", ErrNoApproval, a.State)
	case !b.Now().Before(a.ExpiresAt):
		b.expire(ctx, a)
		return a, fmt.Errorf("%w: expired", ErrNoApproval)
	case echoedHash != a.PayloadSHA256:
		return a, fmt.Errorf("%w: the hash shown doesn't match the payload", ErrNoApproval)
	case s.User() == a.Requester:
		return a, fmt.Errorf("%w: you can't approve your own action", ErrNoApproval)
	case a.FirstApprovedBy != "" && s.User() == a.FirstApprovedBy:
		return a, fmt.Errorf("%w: a second approval must come from someone else", ErrNoApproval)
	case a.FirstApprovedBy != "" && echoedHash != a.FirstHash:
		return a, fmt.Errorf("%w: the hash shown isn't the one first approved", ErrNoApproval)
	}
	first := a.Needs == 2 && a.FirstApprovedBy == ""
	if err := b.mayDecide(ctx, s, a); err != nil {
		return a, err
	}
	tool := b.Tools[a.Tool]
	if err := b.Kill.Check(kill.Op{Connector: tool.Connector, Consequential: true, Space: strings.TrimPrefix(a.Space, "spc_")}); err != nil {
		return a, err
	}
	apr := b.NewID("apr_")
	if first {
		// The first of two approvals: recorded and audited; the action waits for the second.
		if _, err := b.Audit.Write(ctx, audit.Event{Actor: s.User(), Space: s.Space(), Kind: "approval", Subject: a.ID, State: "first_approval",
			Approval: apr, PayloadSHA256: a.PayloadSHA256, Reason: "first approval: " + a.Workflow}); err != nil {
			return a, ErrUnaudited
		}
		a, err = b.Actions.Move(ctx, b.Tenant, id, approvals.PendingApproval, func(x *approvals.Action) error {
			if x.PayloadSHA256 != echoedHash || x.FirstApprovedBy != "" {
				return fmt.Errorf("%w: the action changed", ErrNoApproval)
			}
			x.FirstApproval, x.FirstApprovedBy, x.FirstHash = apr, s.User(), echoedHash
			return nil
		})
		if err == nil {
			b.notify(ctx, a, "first_approval")
		}
		return a, err
	}
	if _, err := b.Audit.Write(ctx, audit.Event{Actor: s.User(), Space: s.Space(), Kind: "approval", Subject: a.ID, State: string(approvals.Approved),
		Approval: apr, PayloadSHA256: a.PayloadSHA256, Reason: "approve: " + a.Workflow}); err != nil {
		return a, ErrUnaudited
	}
	a, err = b.Actions.Move(ctx, b.Tenant, id, approvals.PendingApproval, func(x *approvals.Action) error {
		if x.PayloadSHA256 != echoedHash || (x.Needs == 2 && (x.FirstApprovedBy == "" || x.FirstApprovedBy == s.User())) {
			return fmt.Errorf("%w: the payload or its approvals changed", ErrNoApproval)
		}
		x.State, x.Approval, x.ApprovedBy, x.ApprovedHash, x.Note = approvals.Approved, apr, s.User(), echoedHash, note
		return nil
	})
	if err != nil {
		return a, err
	}
	return b.Execute(ctx, a.ID)
}

// Reject ends a pending action with a note (which may be empty).
func (b *Bus) Reject(ctx context.Context, s tenancy.Scope, id, note string) (approvals.Action, error) {
	a, err := b.load(ctx, s, id)
	if err != nil {
		return a, err
	}
	if a.State != approvals.PendingApproval {
		return a, fmt.Errorf("%w: the action is %s", ErrNoApproval, a.State)
	}
	if s.User() == a.Requester {
		return a, fmt.Errorf("%w: you can't decide on your own action", ErrNoApproval)
	}
	if err := b.mayDecide(ctx, s, a); err != nil {
		return a, err
	}
	if _, err := b.Audit.Write(ctx, audit.Event{Actor: s.User(), Space: s.Space(), Kind: "approval", Subject: a.ID, State: string(approvals.Rejected),
		PayloadSHA256: a.PayloadSHA256, Reason: "reject: " + a.Workflow}); err != nil {
		return a, ErrUnaudited
	}
	return b.Actions.Move(ctx, b.Tenant, id, approvals.PendingApproval, func(x *approvals.Action) error {
		x.State, x.Note = approvals.Rejected, note
		return nil
	})
}

func (b *Bus) expire(ctx context.Context, a approvals.Action) {
	if _, err := b.Audit.Write(ctx, audit.Event{Actor: "sys_toolbus", Space: a.Space, Kind: "tool_call", Subject: a.ID, State: string(approvals.Expired),
		PayloadSHA256: a.PayloadSHA256, Reason: "expired"}); err != nil {
		return
	}
	_, _ = b.Actions.Move(ctx, b.Tenant, a.ID, a.State, func(x *approvals.Action) error { x.State = approvals.Expired; return nil })
}
