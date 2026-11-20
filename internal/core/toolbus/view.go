// Reading and editing an action (sb-actions(3)).

package toolbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Change is one top-level payload field that differs from the target's current state.
type Change struct {
	Field  string          `json:"field"`
	Before json.RawMessage `json:"before"`
	After  json.RawMessage `json:"after"`
}

// Current is implemented by connectors that can read a target's current state, so
// approvers see a diff (sb-actions(3)).
type Current interface {
	Current(ctx context.Context, target string) (json.RawMessage, error)
}

// View is an action as GET /v1/actions/{id} returns it.
type View struct {
	approvals.Action
	Diff []Change `json:"diff"`
}

// Get returns an action to its requester, its approver group, or an auditor.
func (b *Bus) Get(ctx context.Context, s tenancy.Scope, id string) (View, error) {
	a, err := b.load(ctx, s, id)
	if err != nil {
		return View{}, err
	}
	member := false
	for _, g := range s.Groups() {
		member = member || g == a.Approvers || (a.Needs == 2 && g == a.SecondApprovers)
	}
	if a.Requester != s.User() && !member && (b.IsAuditor == nil || !b.IsAuditor(s)) {
		return View{}, fmt.Errorf("%w: no such action", ErrPerm)
	}
	// Someone across a barrier from the requester never sees the payload or the target,
	// even as an approver (sb-permissions(7), Barriers beyond reads; SB-1401).
	if a.Requester != s.User() && b.Barriers != nil {
		actx, err := b.Barriers.Approval(ctx, s, a.Requester)
		if err != nil || acrossBarrier(actx) {
			return View{}, fmt.Errorf("%w: no such action", ErrPerm)
		}
	}
	v := View{Action: a}
	if cur, ok := b.Connectors[b.Tools[a.Tool].Connector].(Current); ok {
		if before, err := cur.Current(ctx, a.Target); err == nil {
			v.Diff = diff(before, a.Payload)
		}
	}
	return v, nil
}

func diff(before, after json.RawMessage) []Change {
	var bm, am map[string]json.RawMessage
	if json.Unmarshal(before, &bm) != nil || json.Unmarshal(after, &am) != nil {
		return nil
	}
	keys := map[string]bool{}
	for k := range bm {
		keys[k] = true
	}
	for k := range am {
		keys[k] = true
	}
	var names []string
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	out := []Change{}
	for _, k := range names {
		bc, _ := approvals.Canonical(orNull(bm[k]))
		ac, _ := approvals.Canonical(orNull(am[k]))
		if string(bc) != string(ac) {
			out = append(out, Change{Field: k, Before: bc, After: ac})
		}
	}
	return out
}

func orNull(r json.RawMessage) json.RawMessage {
	if len(r) == 0 {
		return json.RawMessage("null")
	}
	return r
}

// Edit replaces a pending or approved action's payload: back to Drafted, then proposed
// again with the new hash. Any approval is void (LAW-15). Only the requester edits.
func (b *Bus) Edit(ctx context.Context, s tenancy.Scope, id string, payload json.RawMessage) (approvals.Action, error) {
	a, err := b.load(ctx, s, id)
	if err != nil {
		return a, err
	}
	if a.Requester != s.User() {
		return approvals.Action{}, fmt.Errorf("%w: only the requester edits an action", ErrPerm)
	}
	if v := b.Tools[a.Tool].Validate; v != nil {
		if err := v(payload); err != nil {
			return approvals.Action{}, err
		}
	}
	hash, err := approvals.Hash(a.Workflow, a.Target, payload)
	if err != nil {
		return approvals.Action{}, err
	}
	canon, _ := approvals.Canonical(payload)
	if _, err := b.Audit.Write(ctx, audit.Event{Actor: s.User(), Space: s.Space(), Kind: "tool_call", Subject: a.ID, State: string(approvals.Drafted),
		PayloadSHA256: hash, Reason: "edit: approval void"}); err != nil {
		return approvals.Action{}, ErrUnaudited
	}
	a, err = b.Actions.Move(ctx, b.Tenant, id, a.State, func(x *approvals.Action) error {
		if !approvals.CanMove(x.State, approvals.Drafted) {
			return fmt.Errorf("%w: %s can't be edited", ErrNoApproval, x.State)
		}
		x.State, x.Payload, x.PayloadSHA256 = approvals.Drafted, canon, hash
		x.Approval, x.ApprovedBy, x.ApprovedHash = "", "", ""
		x.FirstApproval, x.FirstApprovedBy, x.FirstHash = "", "", "" // an edit voids every approval
		// The new payload decides how many approvals it needs (SB-1401): an edit that raises
		// the amount above a duty's threshold needs both again.
		if tool := b.Tools[x.Tool]; tool.Consequential {
			x.Needs, x.SecondApprovers = tool.Duty.Needs(canon), ""
			if x.Needs == 2 {
				x.SecondApprovers = tool.Duty.Second
			}
		}
		return nil
	})
	if err != nil {
		return a, err
	}
	if _, err := b.Audit.Write(ctx, audit.Event{Actor: s.User(), Space: s.Space(), Kind: "tool_call", Subject: a.ID, State: string(approvals.PendingApproval),
		PayloadSHA256: hash, Reason: "propose: " + a.Workflow}); err != nil {
		return approvals.Action{}, ErrUnaudited
	}
	return b.Actions.Move(ctx, b.Tenant, id, approvals.Drafted, func(x *approvals.Action) error {
		// Expiry still counts from the proposal: edits never keep an action alive (SB-1401).
		x.State = approvals.PendingApproval
		return nil
	})
}
