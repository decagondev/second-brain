package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-03 at the policy kernel (SB-109): Unknown, in every form, is refusal. The engine
// is wrapped with injected faults; the kernel may allow only when the engine cleanly said
// Allow, for a resource of the caller's own tenant.

// PolicyFault is how the wrapped engine misbehaves on one call.
type PolicyFault int

const (
	FaultClean   PolicyFault = iota // the engine answers normally
	FaultError                      // the engine returns an error
	FaultSlow                       // the engine answers after the deadline
	FaultPanic                      // the engine panics
	FaultGarbage                    // the engine returns a value that is neither Allow nor Deny
)

// PolicyCase is one authorisation attempt.
type PolicyCase struct {
	EngineSays  string // "allow" or "deny": what a clean engine would answer
	Fault       PolicyFault
	OtherTenant bool // the resource belongs to another tenant
	NoScope     bool // the caller has no valid scope
}

// PolicyKernel runs one case through the kernel and reports whether it was allowed.
type PolicyKernel func(c PolicyCase) (allowed bool, err error)

// NewPolicyKernel builds the system under test: the real kernel (SB-109) calling
// policy.Require, in front of an engine that misbehaves as the case says.
var NewPolicyKernel = func() PolicyKernel {
	return func(c PolicyCase) (bool, error) {
		s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_1", []string{"it-leads"})
		if err != nil {
			return false, err
		}
		if c.NoScope {
			s = tenancy.Scope{}
		}
		res := policy.Entity{Type: "Chunk", ID: "c1"}
		if c.OtherTenant {
			res.Tenant = "beta-freight"
		}
		return policy.Kernel{Engine: faulty{c}, Recorder: okRecorder{}}.Require(context.Background(), s, "read", res, nil) == nil, nil
	}
}

type faulty struct{ c PolicyCase }

func (f faulty) Evaluate(ctx context.Context, _ policy.Request) (policy.Result, error) {
	clean := policy.Result{Decision: policy.Deny, Reason: "test"}
	if f.c.EngineSays == "allow" {
		clean.Decision = policy.Allow
	}
	switch f.c.Fault {
	case FaultClean:
		return clean, nil
	case FaultError:
		return clean, errors.New("engine broke")
	case FaultSlow:
		select {
		case <-time.After(2 * policy.Deadline):
		case <-ctx.Done():
		}
		return clean, nil
	case FaultPanic:
		panic("engine panic")
	case FaultGarbage:
		return policy.Result{Decision: policy.Decision(7)}, nil
	}
	return clean, nil
}

func law03Policy(r *rand.Rand, k PolicyKernel) error {
	for i := 0; i < 10; i++ {
		c := PolicyCase{
			EngineSays:  []string{"allow", "deny"}[r.Intn(2)],
			Fault:       PolicyFault(r.Intn(5)),
			OtherTenant: r.Intn(4) == 0,
			NoScope:     r.Intn(6) == 0,
		}
		got, err := k(c)
		if err != nil {
			return err
		}
		want := c.EngineSays == "allow" && c.Fault == FaultClean && !c.OtherTenant && !c.NoScope
		if got != want {
			return fmt.Errorf("case %+v: allowed=%v, want %v", c, got, want)
		}
	}
	return nil
}

type okRecorder struct{}

func (okRecorder) RecordDecision(context.Context, policy.Record) error { return nil }
