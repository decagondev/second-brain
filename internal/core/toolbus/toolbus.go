// Package toolbus is the one path every tool call takes (sb-agents(7), The tool bus;
// SB-405, SB-406, SB-408): kill check, authorise, audit row, approval check, execute. It is
// the only caller of a connector's Act (tools/check/toolbus.py).
package toolbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Tool is one entry in the release bundle's tool registry.
type Tool struct {
	Name          string `json:"name"`
	Connector     string `json:"connector"` // docs, chat, tickets, crm
	Consequential bool   `json:"consequential"`
	Workflow      string `json:"workflow,omitempty"`  // consequential tools belong to the tenant's workflow
	Approvers     string `json:"approvers,omitempty"` // the workflow's approver group
	// Validate checks a payload's shape before it is hashed; nil accepts any payload.
	Validate func(payload []byte) error `json:"-"`
	// Duty is the workflow's segregation of duties from the manifest, or nil (SB-1204).
	Duty *Duty `json:"duty,omitempty"`
}

// Duty: an action whose amount is above Above, or that has no amount, needs a second
// approval from the group Second (sb-approvals(7), Segregation of duties).
type Duty struct {
	Second string `json:"second"`
	Above  int64  `json:"above"`
}

// Needs is how many approvals a payload needs under d.
func (d *Duty) Needs(payload json.RawMessage) int {
	if d == nil {
		return 1
	}
	// The size of the amount decides, whatever its sign: a refund or a credit of a million
	// moves as much money as a payment of one (SB-1401).
	if amount, ok := approvals.Amount(payload); ok && amount >= -d.Above && amount <= d.Above {
		return 1
	}
	return 2
}

// Checker is the kill checker as the bus uses it.
type Checker interface{ Check(op kill.Op) error }

// Auditor is the audit log as the bus uses it.
type Auditor interface {
	Write(ctx context.Context, e audit.Event) (audit.Row, error)
}

// Bus is one cell's tool bus.
type Bus struct {
	Tenant     string
	Tools      map[string]Tool
	Kill       Checker
	Policy     policy.Kernel
	Audit      Auditor
	Actions    approvals.Store
	Connectors map[string]ports.Connector
	Now        func() time.Time
	NewID      func(prefix string) string
	// ActTimeout bounds one Act; 0 means 30 seconds.
	ActTimeout time.Duration
	// Notify tells approvers and requesters about an action (ids only, never the payload).
	Notify func(ctx context.Context, n Notice)
	// IsAuditor says whether a scope may read every action; nil means nobody.
	IsAuditor func(s tenancy.Scope) bool
	// Receipts is the idempotency store: receipts by idempotency key (sb-approvals(7)).
	Receipts Receipts
	// Managers returns a user's manager from the directory, "" for none (SB-1204). Nil is a
	// cell whose directory has no reporting lines; an error refuses the approval.
	Managers func(ctx context.Context, user string) (string, error)
	// Barriers gives the policy context of a call's target and of an approval in a cell
	// with information barriers (barrier.Cell; SB-1203); nil is a cell without them. An
	// error refuses: a target or requester that can't be placed is never across nothing.
	Barriers Barriers
}

// Barriers is barrier.Cell as the bus uses it.
type Barriers interface {
	Target(ctx context.Context, s tenancy.Scope, target string) (map[string]any, error)
	Approval(ctx context.Context, s tenancy.Scope, requester string) (map[string]any, error)
}

// Receipts stores every receipt under its idempotency key, so nothing runs twice.
type Receipts interface {
	Get(ctx context.Context, tenant, key string) (receipt string, found bool, err error)
	Put(ctx context.Context, tenant, key, receipt string) error
}

// Notice is what a notification carries: ids, never the payload (sb-actions(3)).
type Notice struct {
	Tenant, Action, Workflow, Approvers, Requester, Event string
}

// Errors (sb-errors(7) codes in the text).
var (
	ErrPerm       = errors.New("toolbus: EPERM")
	ErrKilled     = kill.ErrKilled
	ErrNoApproval = approvals.ErrNoApproval
	ErrUnaudited  = errors.New("toolbus: EUNAVAIL audit write failed; nothing ran")
)

// Call is one proposed tool call.
type Call struct {
	Agent   string
	Tool    string
	Target  string
	Payload json.RawMessage
	// Context is policy context the cell knows about the call, such as a note's stewards
	// and the caller's groups (SB-1705). The barrier facts are added after it and win.
	Context map[string]any
}

// Invoke runs a call through the bus. A call that isn't consequential runs now and comes
// back Ran or DidNotRun; a consequential one comes back PendingApproval.
func (b *Bus) Invoke(ctx context.Context, s tenancy.Scope, c Call) (approvals.Action, error) {
	if err := s.Check(); err != nil || s.Tenant() != b.Tenant {
		return approvals.Action{}, fmt.Errorf("%w: no valid scope for this cell", ErrPerm)
	}
	tool, ok := b.Tools[c.Tool]
	if !ok {
		return approvals.Action{}, fmt.Errorf("%w: tool %q is not in the release's registry", ErrPerm, c.Tool)
	}
	// 1. Kill check.
	if err := b.Kill.Check(kill.Op{Agent: c.Agent, Connector: tool.Connector, Consequential: tool.Consequential,
		Space: strings.TrimPrefix(s.Space(), "spc_")}); err != nil {
		return approvals.Action{}, err
	}
	// 2. Authorise, with the target's barrier facts in a cell with barriers.
	var bctx map[string]any
	if b.Barriers != nil {
		x, err := b.Barriers.Target(ctx, s, c.Target)
		if err != nil {
			return approvals.Action{}, fmt.Errorf("%w: EBARRIER the target can't be placed: %v", ErrPerm, err)
		}
		bctx = x
	}
	if len(c.Context) > 0 {
		merged := map[string]any{}
		for k, v := range c.Context {
			merged[k] = v
		}
		for k, v := range bctx {
			merged[k] = v
		}
		bctx = merged
	}
	var decision policy.Result
	if tool.Consequential {
		decision = b.Policy.Authorise(ctx, s, "propose", policy.Entity{Type: "Workflow", ID: tool.Workflow, Tenant: b.Tenant,
			Attrs: map[string]any{"name": tool.Workflow, "approvers": policy.Ref{Type: "Group", ID: tool.Approvers}}}, bctx)
	} else {
		decision = b.Policy.Authorise(ctx, s, "call_tool", policy.Entity{Type: "Tool", ID: tool.Name, Tenant: b.Tenant,
			Attrs: map[string]any{"name": tool.Name, "connector": tool.Connector, "consequential": false}}, bctx)
	}
	if decision.Decision != policy.Allow {
		return approvals.Action{}, fmt.Errorf("%w: %s (%s)", ErrPerm, decision.Reason, decision.Decision)
	}
	if tool.Validate != nil {
		if err := tool.Validate(c.Payload); err != nil {
			return approvals.Action{}, err
		}
	}
	hash, err := approvals.Hash(tool.Workflow, c.Target, c.Payload)
	if err != nil {
		return approvals.Action{}, err
	}
	canon, _ := approvals.Canonical(c.Payload)
	a := approvals.Action{ID: b.NewID("tc_"), Tenant: b.Tenant, Workflow: tool.Workflow, Tool: tool.Name, Target: c.Target,
		Payload: canon, PayloadSHA256: hash, Requester: s.User(), Approvers: tool.Approvers, PolicyReason: clean(decision.Reason)}
	if sp := s.Space(); sp != "company" {
		a.Space = sp
	}
	if tool.Consequential {
		if a.Needs = tool.Duty.Needs(canon); a.Needs == 2 {
			a.SecondApprovers = tool.Duty.Second
		}
	}
	if !tool.Consequential {
		// 3. Audit row, then 5. execute (no approval needed).
		a.State = approvals.Executing
		if _, err := b.Audit.Write(ctx, audit.Event{Actor: s.User(), Space: s.Space(), Kind: "tool_call", Subject: a.ID, State: string(approvals.Executing),
			PayloadSHA256: hash, Reason: "call_tool: " + tool.Name}); err != nil {
			return approvals.Action{}, ErrUnaudited
		}
		if err := b.Actions.Put(ctx, a); err != nil {
			return approvals.Action{}, err
		}
		return b.act(ctx, a, tool, a.ID)
	}
	// 3. Audit row for PendingApproval, written before the action exists for approvers.
	a.State, a.ExpiresAt = approvals.PendingApproval, b.Now().Add(approvals.TTL).UTC()
	if _, err := b.Audit.Write(ctx, audit.Event{Actor: s.User(), Space: s.Space(), Kind: "tool_call", Subject: a.ID, State: string(approvals.PendingApproval),
		PayloadSHA256: hash, Reason: "propose: " + tool.Workflow}); err != nil {
		return approvals.Action{}, ErrUnaudited
	}
	if err := b.Actions.Put(ctx, a); err != nil {
		return approvals.Action{}, err
	}
	b.notify(ctx, a, "pending_approval")
	return a, nil
}

func (b *Bus) notify(ctx context.Context, a approvals.Action, event string) {
	if b.Notify != nil {
		b.Notify(ctx, Notice{Tenant: a.Tenant, Action: a.ID, Workflow: a.Workflow, Approvers: a.Approvers, Requester: a.Requester, Event: event})
	}
}

// ToolFor is the registry's tool for a workflow, if the tenant has it enabled.
func (b *Bus) ToolFor(workflow string) (Tool, bool) {
	for _, t := range b.Tools {
		if t.Consequential && t.Workflow == workflow {
			return t, true
		}
	}
	return Tool{}, false
}

func (b *Bus) load(ctx context.Context, s tenancy.Scope, id string) (approvals.Action, error) {
	if err := s.Check(); err != nil || s.Tenant() != b.Tenant {
		return approvals.Action{}, fmt.Errorf("%w: no valid scope for this cell", ErrPerm)
	}
	a, found, err := b.Actions.Get(ctx, b.Tenant, id)
	if err != nil {
		return approvals.Action{}, err
	}
	if !found {
		return approvals.Action{}, fmt.Errorf("%w: no such action", ErrPerm)
	}
	return a, nil
}

// Execute runs an Approved action: kill re-check, payload re-hash, audit row, Act with
// the approval id as idempotency key, then Ran (with a receipt) or DidNotRun.
func (b *Bus) Execute(ctx context.Context, id string) (approvals.Action, error) {
	a, found, err := b.Actions.Get(ctx, b.Tenant, id)
	if err != nil || !found {
		return a, fmt.Errorf("%w: no such action", ErrPerm)
	}
	if a.State != approvals.Approved || a.Approval == "" {
		return a, fmt.Errorf("%w: execute without an approval", ErrNoApproval)
	}
	tool := b.Tools[a.Tool]
	if err := b.Kill.Check(kill.Op{Connector: tool.Connector, Consequential: true, Space: strings.TrimPrefix(a.Space, "spc_")}); err != nil {
		return a, err // stays Approved; it may run after an unkill, before it expires
	}
	if h, err := approvals.Hash(a.Workflow, a.Target, a.Payload); err != nil || h != a.ApprovedHash {
		return a, fmt.Errorf("%w: the payload no longer matches the approval", ErrNoApproval)
	}
	if _, err := b.Audit.Write(ctx, audit.Event{Actor: "sys_executor", Space: a.Space, Kind: "tool_call", Subject: a.ID, State: string(approvals.Executing),
		Approval: a.Approval, PayloadSHA256: a.PayloadSHA256, Reason: "execute: " + a.Workflow}); err != nil {
		return a, ErrUnaudited
	}
	a, err = b.Actions.Move(ctx, b.Tenant, id, approvals.Approved, func(x *approvals.Action) error {
		x.State = approvals.Executing
		return nil
	})
	if err != nil {
		return a, err
	}
	return b.act(ctx, a, tool, a.Approval)
}

// act calls the connector and records the outcome; only a receipt makes Ran (LAW-05).
func (b *Bus) act(ctx context.Context, a approvals.Action, tool Tool, key string) (approvals.Action, error) {
	conn, ok := b.Connectors[tool.Connector]
	state, reason, receipt := approvals.DidNotRun, "", ""
	unconfirmed := false
	prior, seen, rerr := "", false, error(nil)
	if b.Receipts != nil {
		prior, seen, rerr = b.Receipts.Get(ctx, b.Tenant, key)
	}
	switch {
	case rerr != nil:
		reason, unconfirmed = "idempotency store unavailable: "+rerr.Error(), false // nothing was sent
	case seen:
		state, receipt, reason = approvals.Ran, prior, "receipt (already ran under this key)"
	case !ok:
		reason = "ENOTSUP: no connector " + tool.Connector
	default:
		to := b.ActTimeout
		if to == 0 {
			to = 30 * time.Second
		}
		actx, cancel := context.WithTimeout(ctx, to)
		rc, err := conn.Act(actx, ports.Action{Target: a.Target, Payload: a.Payload}, ports.IdempotencyKey(key))
		cancel()
		switch {
		case err != nil:
			reason, unconfirmed = "act failed: "+err.Error(), true // the target may have it
		case rc.TargetID == "":
			reason = "ENORECEIPT: the target returned no receipt"
		default:
			state, receipt, reason = approvals.Ran, rc.TargetID, "receipt"
			if b.Receipts != nil {
				_ = b.Receipts.Put(ctx, b.Tenant, key, receipt)
			}
		}
	}
	reason = clean(reason)
	// The outcome is audited; if that write fails the state still records what happened.
	_, _ = b.Audit.Write(ctx, audit.Event{Actor: "sys_executor", Space: a.Space, Kind: "tool_call", Subject: a.ID, State: string(state),
		Approval: a.Approval, PayloadSHA256: a.PayloadSHA256, Receipt: receipt, Reason: reason})
	final, err := b.Actions.Move(ctx, b.Tenant, a.ID, approvals.Executing, func(x *approvals.Action) error {
		x.State, x.Receipt, x.Reason, x.Unconfirmed, x.FinishedAt = state, receipt, reason, unconfirmed, b.Now().UTC()
		return nil
	})
	if err == nil && final.Workflow != "" {
		b.notify(ctx, final, strings.ToLower(string(state)))
	}
	return final, err
}

// clean makes an error text safe for an audit row: no control characters, at most 200 bytes.
func clean(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f || r == 0x2028 || r == 0x2029 {
			r = ' '
		}
		out = append(out, r)
	}
	s = string(out)
	for len(s) > 200 {
		s = string([]rune(s)[:len([]rune(s))-1])
	}
	return s
}
