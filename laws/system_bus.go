package laws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	killmem "example.com/second-brain/internal/adapters/kill/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// The real tool bus (SB-405, SB-406, SB-408) under LAW-04, LAW-05 and LAW-15: kill
// checker, cedar policies, the audit log, the approval store and a target connector.

const approverGroup = "grp_approvers"

var busTools = map[string]toolbus.Tool{
	"lookup": {Name: "lookup", Connector: "tickets"},
	"update": {Name: "update", Connector: "tickets", Consequential: true, Workflow: "ticket-update", Approvers: approverGroup},
}

// tapStore is the audit store; it also tells the world each committed Executing row.
type tapStore struct {
	inner  audit.Store
	onExec func(subject, approval string)
	drop   bool // mutation: tool_call rows are acknowledged but never stored
}

func (t tapStore) Append(ctx context.Context, b func(int64, string) (audit.Row, error)) (audit.Row, error) {
	var built audit.Row
	if t.drop {
		var err error
		built, err = b(0, audit.Zero)
		if err == nil && built.Kind == "tool_call" {
			return built, nil
		}
	}
	r, err := t.inner.Append(ctx, b)
	if err == nil && r.Kind == "tool_call" && r.State == string(approvals.Executing) && t.onExec != nil {
		t.onExec(r.Subject, r.Approval)
	}
	return r, err
}

func (t tapStore) Range(ctx context.Context, from int64, limit int) ([]audit.Row, error) {
	return t.inner.Range(ctx, from, limit)
}

// actConnector is the target system: Act runs f with the action and returns its receipt.
type actConnector struct {
	worldConnector
	f func(a ports.Action, key ports.IdempotencyKey) string
}

func (c actConnector) Act(_ context.Context, a ports.Action, key ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{TargetID: c.f(a, key)}, nil
}

type busEnv struct {
	bus    *toolbus.Bus
	tenant string
}

func newBus(tenant string, act func(ports.Action, ports.IdempotencyKey) string, onExec func(subject, approval string), drop bool, newID func(string) string) (*busEnv, error) {
	if askPoliciesErr != nil {
		return nil, askPoliciesErr
	}
	now := time.Unix(1_800_000_000, 0)
	clock := func() time.Time { return now }
	ks := killmem.New()
	checker := &kill.Checker{Tenant: tenant, Store: ks, Now: clock}
	if err := checker.Refresh(context.Background()); err != nil {
		return nil, err
	}
	log := &audit.Log{Tenant: tenant, Release: func() string { return "rel_law" }, Store: tapStore{inner: auditmem.New(), onExec: onExec, drop: drop}, Now: clock}
	b := &toolbus.Bus{Tenant: tenant, Tools: busTools, Kill: checker, Policy: policy.Kernel{Engine: askPolicies, Recorder: log},
		Audit: log, Actions: actmem.New(), Connectors: map[string]ports.Connector{"tickets": actConnector{f: act}}, Now: clock, NewID: newID,
		Receipts: actmem.NewReceipts()}
	return &busEnv{bus: b, tenant: tenant}, nil
}

func (e *busEnv) scope(user string, groups ...string) tenancy.Scope {
	s, _ := tenancy.NewIssuer(e.tenant).Issue(e.tenant, user, groups)
	return s
}

// Call runs one law tool call through the real bus.
func (s askSystem) Call(w *World, userID string, tc ToolCall) (Outcome, error) {
	u := w.user(userID)
	var approver User
	for _, x := range w.Users {
		if x.Tenant == u.Tenant && x.ID != u.ID {
			approver = x
			break
		}
	}
	calls := map[string]string{} // action id -> law call id
	n := 0
	newID := func(p string) string {
		if p == "apr_" {
			return "apr_ok" // the world's one valid approval id
		}
		n++
		id := fmt.Sprintf("%s%s_%d", p, tc.ID, n)
		calls[id] = tc.ID // known before the bus writes any row for it
		return id
	}
	env, err := newBus(u.Tenant,
		func(a ports.Action, _ ports.IdempotencyKey) string {
			var p struct{ Call string }
			_ = json.Unmarshal(a.Payload, &p)
			return w.Act(p.Call)
		},
		func(subject, approval string) { w.Audit(calls[subject], approval) },
		s.dropAudit, newID)
	if err != nil {
		return Outcome{}, err
	}
	ctx := context.Background()
	tool := "lookup"
	if tc.Consequential {
		tool = "update"
	}
	payload, _ := json.Marshal(map[string]string{"call": tc.ID})
	a, err := env.bus.Invoke(ctx, env.scope(u.ID, u.Groups...), toolbus.Call{Agent: "ticket-update", Tool: tool, Target: "tickets:" + tc.ID, Payload: payload})
	if err != nil {
		return refused(err)
	}
	if tc.Consequential {
		switch {
		case tc.Approval == "apr_ok" && approver.ID != "":
			a, err = env.bus.Approve(ctx, env.scope(approver.ID, approverGroup), a.ID, a.PayloadSHA256, "")
		case tc.Approval == "apr_forged" && approver.ID != "":
			if _, err = env.bus.Approve(ctx, env.scope(approver.ID, approverGroup), a.ID, strings.Repeat("0", 64), ""); err == nil {
				return Outcome{}, fmt.Errorf("a made-up hash was approved")
			}
			a, err = env.bus.Execute(ctx, a.ID)
		default:
			a, err = env.bus.Execute(ctx, a.ID)
		}
		if err != nil {
			return refused(err)
		}
	}
	return Outcome{State: string(a.State), Receipt: a.Receipt}, nil
}

func refused(err error) (Outcome, error) {
	if errors.Is(err, toolbus.ErrPerm) || errors.Is(err, toolbus.ErrNoApproval) || errors.Is(err, toolbus.ErrKilled) {
		return Outcome{State: "Refused"}, nil
	}
	return Outcome{}, err
}

// approvalWorld drives LAW-15 through the real bus.
type approvalWorld struct {
	env      *busEnv
	executed map[string][]string // target -> payloads
	targets  map[string]string   // action id -> target
	n        int
	// ignoreEcho is a mutation: approval uses the stored hash, not the one the approver saw.
	ignoreEcho bool
}

func newApprovalWorld() *approvalWorld {
	w := &approvalWorld{executed: map[string][]string{}, targets: map[string]string{}}
	k := 0
	env, err := newBus("acme-logistics", func(a ports.Action, _ ports.IdempotencyKey) string {
		var p struct{ Comment string }
		_ = json.Unmarshal(a.Payload, &p)
		w.executed[a.Target] = append(w.executed[a.Target], p.Comment)
		return "rcpt_" + a.Target
	}, nil, false, func(p string) string { k++; return fmt.Sprintf("%s%d", p, k) })
	if err != nil {
		panic(err)
	}
	w.env = env
	return w
}

func (w *approvalWorld) Propose(requester, payload string) (string, string, error) {
	w.n++
	target := fmt.Sprintf("tickets:OPS-%d", w.n)
	p, _ := json.Marshal(map[string]string{"comment": payload})
	a, err := w.env.bus.Invoke(context.Background(), w.env.scope(requester, approverGroup), toolbus.Call{Agent: "ticket-update", Tool: "update", Target: target, Payload: p})
	if err != nil {
		return "", "", err
	}
	w.targets[a.ID] = target
	return a.ID, a.PayloadSHA256, nil
}

func (w *approvalWorld) Edit(id, editor, payload string) (string, error) {
	p, _ := json.Marshal(map[string]string{"comment": payload})
	a, err := w.env.bus.Edit(context.Background(), w.env.scope(editor, approverGroup), id, p)
	return a.PayloadSHA256, err
}

func (w *approvalWorld) Approve(id, approver, echo string) (bool, error) {
	if w.ignoreEcho {
		if cur, ok, _ := w.env.bus.Actions.Get(context.Background(), "acme-logistics", id); ok {
			echo = cur.PayloadSHA256
		}
	}
	a, err := w.env.bus.Approve(context.Background(), w.env.scope(approver, approverGroup), id, echo, "")
	if o, rerr := refused(err); rerr == nil && o.State == "Refused" {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return a.State == approvals.Ran || a.State == approvals.DidNotRun, nil
}

func (w *approvalWorld) Executed(id string) []string { return w.executed[w.targets[id]] }

func init() {
	NewApprovalWorld = func() ApprovalWorld { return newApprovalWorld() }
}
