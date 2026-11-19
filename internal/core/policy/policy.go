// Package policy is the authorisation kernel (sb-permissions(7), SB-109): one entry point,
// three answers, and Unknown is never allow (LAW-03).
package policy

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"example.com/second-brain/internal/core/tenancy"
)

// Decision is the kernel's answer. The zero value is Unknown, so a forgotten assignment refuses.
type Decision int

const (
	Unknown Decision = iota
	Allow
	Deny
)

func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case Deny:
		return "deny"
	case Unknown:
		return "unknown"
	}
	return "unknown"
}

// Deadline bounds one engine evaluation; slower is Unknown.
const Deadline = 50 * time.Millisecond

// Ref names another entity in an attribute, e.g. a ProposedAction's requester.
type Ref struct{ Type, ID string }

// Entity is a resource being authorised. Attrs values are string, bool, int64 or Ref.
type Entity struct {
	Type, ID, Tenant string
	Attrs            map[string]any
}

// Principal is who is asking, built only from a tenancy.Scope.
type Principal struct {
	User, Tenant string
	Groups       []string
}

// Request is one question for the engine.
type Request struct {
	Principal Principal
	Action    string
	Resource  Entity
	Context   map[string]any
}

// Result is an answer with a reason for the audit trail.
type Result struct {
	Decision Decision
	Reason   string
}

// Evaluator is a policy engine (the Cedar adapter in production).
type Evaluator interface {
	Evaluate(ctx context.Context, r Request) (Result, error)
}

// Record is one audited decision (sb-audit(7), Policy decisions): ids and policy ids only.
type Record struct {
	Tenant, Actor, Action, Subject string
	Decision                       Decision
	Reason                         string
}

// Recorder writes decision records to the audit log (SB-401 provides the chain).
type Recorder interface {
	RecordDecision(ctx context.Context, r Record) error
}

// DefaultReadSample is the share of read allows recorded (sb-audit(7)).
const DefaultReadSample = 0.01

// Kernel is the policy engine plus the audit of its decisions. Code asks only through it.
type Kernel struct {
	Engine     Evaluator
	Recorder   Recorder
	ReadSample float64        // share of read allows recorded, 0 to 1
	Rand       func() float64 // in [0,1); nil means never sample
}

// mustRecord says whether a decision is always recorded.
func mustRecord(action string, d Decision) bool { return action != "read" || d != Allow }

// Authorise is the only way code asks for permission. The decision is recorded, if it
// must or is sampled, before it is returned; a failed record makes it Unknown (LAW-04).
func (k Kernel) Authorise(ctx context.Context, s tenancy.Scope, action string, res Entity, rctx map[string]any) Result {
	r := decide(ctx, k.Engine, s, action, res, rctx)
	sampled := !mustRecord(action, r.Decision) && k.Rand != nil && k.Rand() < k.ReadSample
	if !mustRecord(action, r.Decision) && !sampled {
		return r
	}
	rec := Record{Tenant: s.Tenant(), Actor: s.User(), Action: action, Subject: res.Type + "::\"" + res.ID + "\"",
		Decision: r.Decision, Reason: action + ": " + r.Reason}
	if k.Recorder == nil {
		return Result{Unknown, "no decision recorder"}
	}
	if err := k.Recorder.RecordDecision(ctx, rec); err != nil {
		return Result{Unknown, "decision could not be audited: " + err.Error()}
	}
	return r
}

func decide(ctx context.Context, ev Evaluator, s tenancy.Scope, action string, res Entity, rctx map[string]any) Result {
	if err := s.Check(); err != nil {
		return Result{Unknown, "no valid scope"}
	}
	if res.Tenant != "" && res.Tenant != s.Tenant() {
		return Result{Deny, "resource belongs to another tenant"}
	}
	if res.Tenant == "" {
		res.Tenant = s.Tenant()
	}
	req := Request{Principal: Principal{User: s.User(), Tenant: s.Tenant(), Groups: s.Groups()}, Action: action, Resource: res, Context: rctx}
	ctx, cancel := context.WithTimeout(ctx, Deadline)
	defer cancel()
	done := make(chan Result, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				done <- Result{Unknown, fmt.Sprintf("engine panicked: %v", p)}
			}
		}()
		r, err := ev.Evaluate(ctx, req)
		if err != nil {
			done <- Result{Unknown, "engine error: " + err.Error()}
			return
		}
		done <- r
	}()
	select {
	case r := <-done:
		switch r.Decision {
		case Allow, Deny:
			return r
		case Unknown:
			return Result{Unknown, r.Reason}
		}
		return Result{Unknown, fmt.Sprintf("engine returned an invalid decision %d", int(r.Decision))}
	case <-ctx.Done():
		return Result{Unknown, "engine timed out"}
	}
}

// ErrRefused is returned by Require for Deny and for Unknown alike (EPERM to users).
var ErrRefused = errors.New("policy: refused")

// Require returns nil only for Allow.
func (k Kernel) Require(ctx context.Context, s tenancy.Scope, action string, res Entity, rctx map[string]any) error {
	r := k.Authorise(ctx, s, action, res, rctx)
	switch r.Decision {
	case Allow:
		return nil
	case Deny, Unknown:
		return fmt.Errorf("%w (%s): %s", ErrRefused, r.Decision, r.Reason)
	}
	return fmt.Errorf("%w (invalid decision): %s", ErrRefused, r.Reason)
}

// ReadSampleFrom reads SB_AUDIT_READ_SAMPLE (0 to 1) through getenv; unset is the default.
// A malformed or out-of-range value is an error, never a silent default.
func ReadSampleFrom(getenv func(string) string) (float64, error) {
	v := getenv("SB_AUDIT_READ_SAMPLE")
	if v == "" {
		return DefaultReadSample, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || f > 1 {
		return 0, fmt.Errorf("policy: SB_AUDIT_READ_SAMPLE must be a number from 0 to 1, got %q", v)
	}
	return f, nil
}
