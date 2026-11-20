package laws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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

// LAW-15 for segregation of duties (SB-1204), through the real tool bus and policies: an
// action above its duty's threshold (or with no amount) runs only after two approvals by
// different people, one from each group; one at or below runs after one; and no approval
// ever comes from the requester or anyone above them in the reporting line.

const (
	dutyFirst  = "grp_it_leads"
	dutySecond = "grp_finance_leads"
	dutyAbove  = 1000
)

// DutyBus builds the bus under test from the world's reporting lines.
type DutyBus func(managers map[string]string) (*toolbus.Bus, error)

// RealDutyBus is the shipped bus: the duty on the tool and the directory's managers.
func RealDutyBus(managers map[string]string) (*toolbus.Bus, error) {
	b, err := dutyBus()
	if err != nil {
		return nil, err
	}
	b.Managers = func(_ context.Context, u string) (string, error) { return managers[u], nil }
	return b, nil
}

type receiptAct struct{ viewers }

func (receiptAct) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{TargetID: "linear:comment/1"}, nil
}

func dutyBus() (*toolbus.Bus, error) {
	if askPoliciesErr != nil {
		return nil, askPoliciesErr
	}
	now := time.Unix(1_800_000_000, 0)
	clock := func() time.Time { return now }
	checker := &kill.Checker{Tenant: spaceTenant, Store: killmem.New(), Now: clock}
	if err := checker.Refresh(context.Background()); err != nil {
		return nil, err
	}
	log := &audit.Log{Tenant: spaceTenant, Release: func() string { return "rel_law" }, Store: auditmem.New(), Now: clock}
	n := 0
	return &toolbus.Bus{Tenant: spaceTenant, Tools: map[string]toolbus.Tool{
		"update": {Name: "update", Connector: "tickets", Consequential: true, Workflow: "ticket-update", Approvers: dutyFirst,
			Duty: &toolbus.Duty{Second: dutySecond, Above: dutyAbove}},
	}, Kill: checker, Policy: policy.Kernel{Engine: askPolicies, Recorder: log}, Audit: log, Actions: actmem.New(),
		Connectors: map[string]ports.Connector{"tickets": receiptAct{}}, Now: clock,
		NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) }, Receipts: actmem.NewReceipts()}, nil
}

// dutyRuns counts actions that ran, so the self-test can show the law test isn't vacuous.
var dutyRuns struct{ one, two int }

func law15Duties(r *rand.Rand, mk DutyBus) error {
	ctx := context.Background()
	users := []string{"usr_a", "usr_b", "usr_c", "usr_d", "usr_e"}
	groups := map[string][]string{}
	in := func(u, g string) bool { return contains(groups[u], g) }
	for _, u := range users {
		if r.Intn(2) == 0 {
			groups[u] = append(groups[u], dutyFirst)
		}
		if r.Intn(2) == 0 {
			groups[u] = append(groups[u], dutySecond)
		}
	}
	managers := map[string]string{}
	for i, u := range users {
		if i > 0 && r.Intn(3) > 0 {
			managers[u] = users[r.Intn(i)] // managers come earlier, so no loops
		}
	}
	above := func(m, u string) bool {
		for cur, k := managers[u], 0; cur != "" && k < 10; cur, k = managers[cur], k+1 {
			if cur == m {
				return true
			}
		}
		return false
	}
	bus, err := mk(managers)
	if err != nil {
		return err
	}
	scope := func(u string) tenancy.Scope {
		s, _ := tenancy.NewIssuer(spaceTenant).Issue(spaceTenant, u, groups[u])
		return s
	}
	requester := users[r.Intn(len(users))]
	pl := map[string]any{"comment": "pay the invoice"}
	needs := 2
	switch r.Intn(4) {
	case 0: // no amount: both approvals
	case 1:
		pl["amount"] = 1 + r.Intn(dutyAbove) // at or below: one
		needs = 1
	case 2:
		pl["amount"] = dutyAbove + 1 + r.Intn(100000)
	default:
		pl["amount"] = "a lot" // not a whole number: both
	}
	payload, _ := json.Marshal(pl)
	a, err := bus.Invoke(ctx, scope(requester), toolbus.Call{Agent: "ticket-update", Tool: "update", Target: "tickets:FIN-1", Payload: payload})
	if err != nil {
		return fmt.Errorf("propose: %w", err)
	}
	var approvedBy []string
	for i := 0; i < 8; i++ {
		u := users[r.Intn(len(users))]
		got, err := bus.Approve(ctx, scope(u), a.ID, a.PayloadSHA256, "")
		if err != nil {
			if !errors.Is(err, toolbus.ErrNoApproval) && !errors.Is(err, toolbus.ErrPerm) {
				return fmt.Errorf("approve: %w", err)
			}
			continue
		}
		stage := len(approvedBy) + 1
		switch {
		case u == requester:
			return fmt.Errorf("%s approved their own action", u)
		case above(u, requester):
			return fmt.Errorf("%s approved for %s, who reports to them (managers %v)", u, requester, managers)
		case stage == 1 && !in(u, dutyFirst):
			return fmt.Errorf("first approval by %s, who isn't in %s", u, dutyFirst)
		case stage == 2 && (!in(u, dutySecond) || u == approvedBy[0]):
			return fmt.Errorf("second approval by %s (groups %v; first was %s)", u, groups[u], approvedBy[0])
		case stage > 2:
			return fmt.Errorf("a third approval was accepted")
		}
		approvedBy = append(approvedBy, u)
		ran := got.State == approvals.Ran || got.State == approvals.Executing || got.State == approvals.DidNotRun
		switch {
		case ran && len(approvedBy) < needs:
			return fmt.Errorf("the action (payload %s) ran after %d approval(s); it needs %d", payload, len(approvedBy), needs)
		case !ran && len(approvedBy) == needs:
			return fmt.Errorf("the action is %s after the %d approvals it needs", got.State, needs)
		}
		if ran {
			if needs == 1 {
				dutyRuns.one++
			} else {
				dutyRuns.two++
			}
			break
		}
	}
	return nil
}
