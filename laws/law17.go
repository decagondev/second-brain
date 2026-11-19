package laws

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// LAW-17 and LAW-18 through the hand-off controller (SB-701 specifies, SB-704 and SB-705
// build). Random runs of plan, approve, execute, failures, abort and replan must keep the
// model's invariants (proofs/tla/Handoff.tla) at every step:
//   LAW-17: from CutOver on, the factory holds only the retained grants; after Detach, none.
//   LAW-18: the root key always has a wrap that really unwraps, and the serving cell can
//           unwrap its own key.
//   Dual approval: past Planned, a factory operator other than the planner and the tenant
//           owner have approved the current plan hash.
//   Abort: Managed means the factory cell serves, factory wrap only, every path.

// HandoffState is what the laws inspect after each step.
type HandoffState struct {
	Phase     string
	Serving   string          // "factoryCell" or "tenantCell"
	Wraps     map[string]bool // party -> a wrap exists AND unwraps the root key now
	Access    []string        // the factory's access paths to the cell now
	Paths     []string        // every factory path at the start
	Retained  []string        // the plan's retained grants (paths that may stay)
	Plan      string
	Planner   string
	Approved  map[string]string // side -> the hash the latest valid approval was for
	Approvers map[string]string // side -> who gave that approval
}

// HandoffWorld is one cell's hand-off controller over fake accounts and KMSs.
type HandoffWorld interface {
	Plan(planner string, retain []string) (hash string, err error)
	Approve(side, who, hash string) error // side "factory" or "tenant"
	Step() error                          // run the next execute step
	Fail()                                // the next step's check fails
	Abort() error
	Detach() error
	State() HandoffState
}

// NewHandoffWorld builds the system under test: the hand-off controller (SB-705).
var NewHandoffWorld = func() HandoffWorld { return pendingHandoff{} }

type pendingHandoff struct{}

func (pendingHandoff) Plan(string, []string) (string, error) {
	return "", fmt.Errorf("%w: the hand-off controller arrives with SB-704 and SB-705", ErrPending)
}
func (pendingHandoff) Approve(string, string, string) error { return nil }
func (pendingHandoff) Step() error                          { return nil }
func (pendingHandoff) Fail()                                {}
func (pendingHandoff) Abort() error                         { return nil }
func (pendingHandoff) Detach() error                        { return nil }
func (pendingHandoff) State() HandoffState                  { return HandoffState{} }

var handoffGrants = []string{"publish-releases", "platform-kill", "support-metrics"}

// handoffRun drives one random hand-off and calls check after every operation.
func handoffRun(r *rand.Rand, w HandoffWorld, check func(op string, s HandoffState) error) error {
	retain := []string{"publish-releases"}
	for _, g := range handoffGrants[1:] {
		if r.Intn(2) == 0 {
			retain = append(retain, g)
		}
	}
	hash, err := w.Plan("fo_ann", retain)
	if err != nil {
		return err
	}
	if err := check("plan", w.State()); err != nil {
		return err
	}
	for i := 0; i < 30; i++ {
		var op string
		switch k := r.Intn(12); {
		case k < 5:
			op = "step"
			_ = w.Step()
		case k == 5:
			op = "fail+step"
			w.Fail()
			_ = w.Step()
		case k == 6:
			who := []string{"fo_ann", "fo_bob"}[r.Intn(2)] // the planner may not approve
			op = "approve factory by " + who
			_ = w.Approve("factory", who, hash)
		case k == 7:
			op = "approve tenant"
			_ = w.Approve("tenant", "to_cy", hash)
		case k == 8:
			op = "approve a stale hash"
			_ = w.Approve([]string{"factory", "tenant"}[r.Intn(2)], "fo_bob", "ph_stale")
		case k == 9:
			op = "abort"
			_ = w.Abort()
		case k == 10:
			op = "replan"
			if h, err := w.Plan("fo_ann", retain); err == nil {
				hash = h
			}
		default:
			op = "detach"
			_ = w.Detach()
		}
		if err := check(op, w.State()); err != nil {
			return err
		}
	}
	return nil
}

func subset(xs, of []string) []string {
	in := map[string]bool{}
	for _, x := range of {
		in[x] = true
	}
	var extra []string
	for _, x := range xs {
		if !in[x] {
			extra = append(extra, x)
		}
	}
	sort.Strings(extra)
	return extra
}

func past(phase string) bool {
	return phase != "" && phase != "Managed" && phase != "Planned"
}

func law17Handoff(r *rand.Rand, w HandoffWorld) error {
	return handoffRun(r, w, func(op string, s HandoffState) error {
		switch s.Phase {
		case "CutOver", "TenantOwned":
			if extra := subset(s.Access, s.Retained); len(extra) > 0 {
				return fmt.Errorf("after %s in %s the factory still holds unlisted paths %v (retained %v)", op, s.Phase, extra, s.Retained)
			}
		case "Detached":
			if len(s.Access) > 0 {
				return fmt.Errorf("after %s the detached cell still has factory paths %v", op, s.Access)
			}
		}
		if past(s.Phase) && (s.Approved["factory"] != s.Plan || s.Approved["tenant"] != s.Plan) {
			return fmt.Errorf("after %s the hand-off is in %s without both approvals of plan %s: %v", op, s.Phase, s.Plan, s.Approved)
		}
		if past(s.Phase) && s.Approvers["factory"] == s.Planner {
			return fmt.Errorf("after %s the hand-off is in %s with the planner %s as the factory approver", op, s.Phase, s.Planner)
		}
		if s.Phase == "Managed" && (s.Serving != "factoryCell" || !s.Wraps["factory"] || s.Wraps["tenant"] || strings.Join(subset(s.Paths, s.Access), ",") != "") {
			return fmt.Errorf("after %s, Managed isn't intact: %+v", op, s)
		}
		return nil
	})
}

func law18Handoff(r *rand.Rand, w HandoffWorld) error {
	return handoffRun(r, w, func(op string, s HandoffState) error {
		if !s.Wraps["factory"] && !s.Wraps["tenant"] {
			return fmt.Errorf("after %s in %s no custodian can unwrap the root key", op, s.Phase)
		}
		need := map[string]string{"factoryCell": "factory", "tenantCell": "tenant"}[s.Serving]
		if need == "" || !s.Wraps[need] {
			return fmt.Errorf("after %s in %s the serving cell %q can't unwrap its key (wraps %v)", op, s.Phase, s.Serving, s.Wraps)
		}
		if (s.Phase == "TenantOwned" || s.Phase == "Detached") && s.Wraps["factory"] {
			return fmt.Errorf("after %s in %s the factory can still unwrap the key", op, s.Phase)
		}
		return nil
	})
}
