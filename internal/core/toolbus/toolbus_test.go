package toolbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

const tenant = "acme-logistics"

// trace records the order of everything the bus does.
type trace struct {
	mu  sync.Mutex
	evs []string
}

func (t *trace) add(s string) { t.mu.Lock(); t.evs = append(t.evs, s); t.mu.Unlock() }

type auditor struct {
	t    *trace
	fail bool
	rows []audit.Event
}

func (a *auditor) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	if a.fail {
		return audit.Row{}, errors.New("audit down")
	}
	a.rows = append(a.rows, e)
	a.t.add("audit:" + e.State)
	return audit.Row{}, nil
}

type checker struct{ killed bool }

func (c *checker) Check(op kill.Op) error {
	if c.killed {
		return fmt.Errorf("%w: test", kill.ErrKilled)
	}
	return nil
}

// allow is a policy engine that allows everything except the forbids the bus relies on
// policy for; separation of duties is checked by the bus itself too.
type allow struct{}

func (allow) Evaluate(_ context.Context, r policy.Request) (policy.Result, error) {
	return policy.Result{Decision: policy.Allow, Reason: "test"}, nil
}

type rec struct{}

func (rec) RecordDecision(context.Context, policy.Record) error { return nil }

type store struct {
	mu sync.Mutex
	m  map[string]approvals.Action
}

func (s *store) Put(_ context.Context, a approvals.Action) error {
	s.mu.Lock()
	s.m[a.ID] = a
	s.mu.Unlock()
	return nil
}
func (s *store) Get(_ context.Context, _, id string) (approvals.Action, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.m[id]
	return a, ok, nil
}
func (s *store) Move(_ context.Context, _, id string, from approvals.State, f func(*approvals.Action) error) (approvals.Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.m[id]
	if a.State != from {
		return a, approvals.ErrMoved
	}
	if err := f(&a); err != nil {
		return a, err
	}
	s.m[id] = a
	return a, nil
}

type target struct {
	ports.Connector
	t       *trace
	receipt string
	err     error
	keys    []ports.IdempotencyKey
	got     []string
}

func (c *target) Act(_ context.Context, a ports.Action, k ports.IdempotencyKey) (ports.Receipt, error) {
	c.t.add("act")
	c.keys = append(c.keys, k)
	c.got = append(c.got, string(a.Payload))
	return ports.Receipt{TargetID: c.receipt}, c.err
}

type fixture struct {
	bus   *Bus
	tr    *trace
	aud   *auditor
	kill  *checker
	tgt   *target
	clock *time.Time
}

func newFixture() *fixture {
	tr := &trace{}
	now := time.Unix(1_800_000_000, 0)
	f := &fixture{tr: tr, aud: &auditor{t: tr}, kill: &checker{}, tgt: &target{t: tr, receipt: "jira:comment/1"}, clock: &now}
	n := 0
	f.bus = &Bus{Tenant: tenant, Kill: f.kill, Policy: policy.Kernel{Engine: allow{}, Recorder: rec{}}, Audit: f.aud,
		Actions: &store{m: map[string]approvals.Action{}}, Connectors: map[string]ports.Connector{"tickets": f.tgt},
		Now: func() time.Time { return *f.clock }, NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Tools: map[string]Tool{
			"lookup": {Name: "lookup", Connector: "tickets"},
			"update": {Name: "update", Connector: "tickets", Consequential: true, Workflow: "ticket-update", Approvers: "grp_it_leads"},
		}}
	return f
}

func sc(t *testing.T, user string) tenancy.Scope {
	s, err := tenancy.NewIssuer(tenant).Issue(tenant, user, []string{"grp_it_leads"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var payload = json.RawMessage(`{"comment":"Root cause: disk full","n":3}`)

func TestOrderForAPlainCall(t *testing.T) {
	f := newFixture()
	a, err := f.bus.Invoke(context.Background(), sc(t, "usr_dana"), Call{Agent: "ask", Tool: "lookup", Target: "tickets:OPS-1", Payload: payload})
	if err != nil || a.State != approvals.Ran || a.Receipt != "jira:comment/1" {
		t.Fatalf("%+v %v", a, err)
	}
	if got := strings.Join(f.tr.evs, " "); got != "audit:Executing act audit:Ran" {
		t.Fatalf("order %q", got)
	}
}

func TestConsequentialNeedsApprovalBoundToHash(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	a, err := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Agent: "ticket-update", Tool: "update", Target: "tickets:OPS-1182", Payload: payload})
	if err != nil || a.State != approvals.PendingApproval || len(f.tgt.got) != 0 {
		t.Fatalf("%+v %v", a, err)
	}
	if _, err := f.bus.Execute(ctx, a.ID); !errors.Is(err, ErrNoApproval) {
		t.Fatalf("executed without approval: %v", err)
	}
	if _, err := f.bus.Approve(ctx, sc(t, "usr_dana"), a.ID, a.PayloadSHA256, ""); !errors.Is(err, ErrNoApproval) {
		t.Fatalf("self-approval: %v", err)
	}
	if _, err := f.bus.Approve(ctx, sc(t, "usr_erin"), a.ID, strings.Repeat("0", 64), ""); !errors.Is(err, ErrNoApproval) {
		t.Fatalf("wrong hash: %v", err)
	}
	old := a.PayloadSHA256
	a, err = f.bus.Edit(ctx, sc(t, "usr_dana"), a.ID, json.RawMessage(`{"comment":"Root cause: DNS"}`))
	if err != nil || a.State != approvals.PendingApproval || a.PayloadSHA256 == old {
		t.Fatalf("edit: %+v %v", a, err)
	}
	if _, err := f.bus.Approve(ctx, sc(t, "usr_erin"), a.ID, old, ""); !errors.Is(err, ErrNoApproval) {
		t.Fatalf("stale hash after edit: %v", err)
	}
	f.tr.evs = nil
	a, err = f.bus.Approve(ctx, sc(t, "usr_erin"), a.ID, a.PayloadSHA256, "ok")
	if err != nil || a.State != approvals.Ran || a.ApprovedBy != "usr_erin" {
		t.Fatalf("approve: %+v %v", a, err)
	}
	if got := strings.Join(f.tr.evs, " "); got != "audit:Approved audit:Executing act audit:Ran" {
		t.Fatalf("order %q", got)
	}
	if len(f.tgt.got) != 1 || f.tgt.got[0] != `{"comment":"Root cause: DNS"}` || string(f.tgt.keys[0]) != a.Approval {
		t.Fatalf("executed %v with keys %v", f.tgt.got, f.tgt.keys)
	}
	if _, err := f.bus.Approve(ctx, sc(t, "usr_finn"), a.ID, a.PayloadSHA256, ""); !errors.Is(err, ErrNoApproval) {
		t.Fatalf("approved twice: %v", err)
	}
}

func TestNothingRunsUnaudited(t *testing.T) {
	f := newFixture()
	f.aud.fail = true
	if _, err := f.bus.Invoke(context.Background(), sc(t, "usr_dana"), Call{Tool: "lookup", Target: "tickets:OPS-1", Payload: payload}); !errors.Is(err, ErrUnaudited) {
		t.Fatalf("%v", err)
	}
	if len(f.tgt.got) != 0 {
		t.Fatal("acted without an audit row")
	}
}

func TestKillAndExpiryAndReceipts(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	f.kill.killed = true
	if _, err := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Tool: "lookup", Target: "tickets:OPS-1", Payload: payload}); !errors.Is(err, kill.ErrKilled) {
		t.Fatalf("killed: %v", err)
	}
	f.kill.killed = false
	a, _ := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Tool: "update", Target: "tickets:OPS-2", Payload: payload})
	*f.clock = f.clock.Add(approvals.TTL)
	if _, err := f.bus.Approve(ctx, sc(t, "usr_erin"), a.ID, a.PayloadSHA256, ""); !errors.Is(err, ErrNoApproval) {
		t.Fatalf("expired: %v", err)
	}
	if got, _, _ := f.bus.Actions.Get(ctx, tenant, a.ID); got.State != approvals.Expired {
		t.Fatalf("state %s", got.State)
	}
	f.tgt.receipt = ""
	a, err := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Tool: "lookup", Target: "tickets:OPS-3", Payload: payload})
	if err != nil || a.State != approvals.DidNotRun || !strings.Contains(a.Reason, "ENORECEIPT") {
		t.Fatalf("no receipt: %+v %v", a, err)
	}
	f.tgt.receipt, f.tgt.err = "x", errors.New("timeout\nwith a newline")
	a, err = f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Tool: "lookup", Target: "tickets:OPS-4", Payload: payload})
	if err != nil || a.State != approvals.DidNotRun || strings.Contains(a.Reason, "\n") {
		t.Fatalf("act error: %+v %v", a, err)
	}
	if _, err := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Tool: "delete_everything", Target: "x", Payload: payload}); !errors.Is(err, ErrPerm) {
		t.Fatalf("unregistered tool: %v", err)
	}
}

func TestApproveKillRecheck(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	a, _ := f.bus.Invoke(ctx, sc(t, "usr_dana"), Call{Tool: "update", Target: "tickets:OPS-5", Payload: payload})
	f.kill.killed = true
	if _, err := f.bus.Approve(ctx, sc(t, "usr_erin"), a.ID, a.PayloadSHA256, ""); !errors.Is(err, kill.ErrKilled) || len(f.tgt.got) != 0 {
		t.Fatalf("approve while killed: %v", err)
	}
}
