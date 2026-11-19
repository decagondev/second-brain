package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-04 at the policy kernel (SB-111): a decision that must be audited is recorded
// before the caller gets it, and if it can't be recorded it isn't Allow.

// DecisionCase is one authorisation with a recorder that may fail.
type DecisionCase struct {
	Action        string // "read", "call_tool", "approve", ...
	EngineAllows  bool
	RecorderFails bool
}

// DecisionOutcome is what the kernel did.
type DecisionOutcome struct {
	Allowed            bool
	RecordedBeforeExit bool // a record for this decision existed when the kernel returned
	RecordMatches      bool // the record names the right actor, subject, action and decision
}

// DecisionKernel runs one case through the kernel with read sampling turned off.
type DecisionKernel func(c DecisionCase) (DecisionOutcome, error)

// NewDecisionKernel builds the system under test: the real policy.Kernel (SB-111) with a
// clean engine and a recorder that captures rows, or fails.
var NewDecisionKernel = func() DecisionKernel {
	return func(c DecisionCase) (DecisionOutcome, error) {
		s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_1", nil)
		if err != nil {
			return DecisionOutcome{}, err
		}
		says := "deny"
		if c.EngineAllows {
			says = "allow"
		}
		rec := &capture{fail: c.RecorderFails}
		k := policy.Kernel{Engine: faulty{PolicyCase{EngineSays: says}}, Recorder: rec, ReadSample: 0}
		res := policy.Entity{Type: "Tool", ID: "t1"}
		got := k.Authorise(context.Background(), s, c.Action, res, nil)
		out := DecisionOutcome{Allowed: got.Decision == policy.Allow, RecordedBeforeExit: len(rec.rows) > 0}
		if out.RecordedBeforeExit {
			r := rec.rows[0]
			out.RecordMatches = r.Actor == "usr_1" && r.Tenant == "acme-logistics" && r.Subject == `Tool::"t1"` &&
				r.Action == c.Action && r.Decision == got.Decision && r.Reason != ""
		}
		return out, nil
	}
}

type capture struct {
	fail bool
	rows []policy.Record
}

func (c *capture) RecordDecision(_ context.Context, r policy.Record) error {
	if c.fail {
		return errors.New("audit store unavailable")
	}
	c.rows = append(c.rows, r)
	return nil
}

func law04Decisions(r *rand.Rand, k DecisionKernel) error {
	actions := []string{"read", "use_agent", "call_tool", "propose", "approve", "admin"}
	for i := 0; i < 12; i++ {
		c := DecisionCase{Action: actions[r.Intn(len(actions))], EngineAllows: r.Intn(2) == 0, RecorderFails: r.Intn(4) == 0}
		got, err := k(c)
		if err != nil {
			return err
		}
		mustRecord := c.Action != "read" || !c.EngineAllows // reads: refusals always, allows sampled (off here)
		switch {
		case c.RecorderFails && mustRecord && got.Allowed:
			return fmt.Errorf("%+v: allowed although its audit row could not be written", c)
		case !c.RecorderFails && mustRecord && !got.RecordedBeforeExit:
			return fmt.Errorf("%+v: decision returned before it was recorded", c)
		case got.RecordedBeforeExit && !got.RecordMatches:
			return fmt.Errorf("%+v: the record does not match the decision", c)
		case !c.RecorderFails && got.Allowed != c.EngineAllows:
			return fmt.Errorf("%+v: allowed=%v", c, got.Allowed)
		}
	}
	return nil
}
