package handoff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/ports"
)

// The controller's phases and transitions are the model's: read Handoff.tla and compare.
func TestStatesMatchTheModel(t *testing.T) {
	src, err := os.ReadFile("../../../proofs/tla/Handoff.tla")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`Phases == \{([^}]*)\}`).FindSubmatch(src)
	if m == nil {
		t.Fatal("no Phases in Handoff.tla")
	}
	var model []string
	for _, p := range strings.Split(string(m[1]), ",") {
		model = append(model, strings.Trim(strings.TrimSpace(p), `"`))
	}
	ours := []string{string(Managed), string(Planned), string(Provisioned), string(KeyDual), string(Moved), string(Verified), string(CutOver), string(TenantOwned), string(Detached)}
	sort.Strings(model)
	sort.Strings(ours)
	if fmt.Sprint(model) != fmt.Sprint(ours) {
		t.Fatalf("phases: model %v, controller %v", model, ours)
	}
	// Each action's guard (phase = "X") and effect (phase' = "Y").
	act := regexp.MustCompile(`(?ms)^(\w+) ==\s*\n\s*/\\ phase = "(\w+)"(.*?)phase' = "(\w+)"`)
	found := map[string][2]Phase{}
	for _, a := range act.FindAllSubmatch(src, -1) {
		found[string(a[1])] = [2]Phase{Phase(a[2]), Phase(a[4])}
	}
	delete(found, "Init") // the initial state, not an action (the regexp runs on into MakePlan)
	for name, tr := range Transitions {
		if name == "MakePlan" {
			continue // its guard is a set: Managed or Planned
		}
		if found[name] != tr {
			t.Errorf("%s: model %v, controller %v", name, found[name], tr)
		}
	}
	if len(found) != len(Transitions)-1 {
		t.Errorf("the model has %d single-phase actions, the controller %d: %v", len(found), len(Transitions)-1, found)
	}
	inflight := regexp.MustCompile(`InFlight == \{([^}]*)\}`).FindSubmatch(src)
	if inflight == nil || strings.Count(string(inflight[1]), `"`)/2 != len(InFlight) {
		t.Fatalf("InFlight differs from the model: %s", inflight)
	}
}

type xorKeys struct{}

func (xorKeys) EnsureRootKey(context.Context, ports.CellResource) (string, error) { return "", nil }
func (xorKeys) mask(id string, b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[i] ^ id[i%len(id)]
	}
	return out
}
func (x xorKeys) Wrap(_ context.Context, id string, k []byte) ([]byte, error) {
	return append(x.mask(id, k), id...), nil
}
func (x xorKeys) Unwrap(_ context.Context, id string, w []byte) ([]byte, error) {
	if !bytes.HasSuffix(w, []byte(id)) {
		return nil, errors.New("wrong key")
	}
	return x.mask(id, w[:len(w)-len(id)]), nil
}

type access struct{ m map[string]bool }

func (a *access) Paths(context.Context, string) ([]string, error) {
	var out []string
	for k := range a.m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}
func (a *access) Revoke(_ context.Context, _, p string) error { delete(a.m, p); return nil }

type target struct{ calls []string }

func (t *target) Provision(context.Context, string, Plan) error {
	t.calls = append(t.calls, "provision")
	return nil
}
func (t *target) Teardown(context.Context, string, Plan) error {
	t.calls = append(t.calls, "teardown")
	return nil
}
func (t *target) Move(context.Context, string, Plan) error {
	t.calls = append(t.calls, "move")
	return nil
}
func (t *target) Verify(context.Context, string, Plan) error {
	t.calls = append(t.calls, "verify")
	return nil
}
func (t *target) Cutover(context.Context, string, Plan) error {
	t.calls = append(t.calls, "cutover")
	return nil
}

type custody struct{ got []string }

func (c *custody) SetCustody(_ context.Context, _, cu, pl, kc string) error {
	c.got = []string{cu, pl, kc}
	return nil
}

type rows struct{ ev []audit.Event }

func (r *rows) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	r.ev = append(r.ev, e)
	return audit.Row{}, nil
}

func setup(t *testing.T) (*Controller, *access, *target, *custody, *rows) {
	t.Helper()
	ctx := context.Background()
	f := keywrap.Custodian{Name: "factory", Keys: xorKeys{}, KeyID: "factory-root"}
	tn := keywrap.Custodian{Name: "tenant", Keys: xorKeys{}, KeyID: "tenant-root-1"}
	st := &keywrap.MemoryStore{}
	if err := keywrap.Seed(ctx, "gamma-parts", st, f, keywrap.PurposeAuditSigning); err != nil {
		t.Fatal(err)
	}
	a := &access{m: map[string]bool{"deploy-role": true, "break-glass": true, "publish-releases": true, "platform-kill": true}}
	tg, cu, r := &target{}, &custody{}, &rows{}
	return &Controller{Store: &MemoryStore{}, Access: a, Target: tg, Custody: cu, Audit: r, Wraps: st,
		Factory: func(string) keywrap.Custodian { return f }, Tenant: func(string, Plan) keywrap.Custodian { return tn }}, a, tg, cu, r
}

const to = cell.TenantAccountPrefix + "gcp/gamma-prod-7731"

func TestAFullHandoff(t *testing.T) {
	ctx := context.Background()
	c, a, tg, cu, r := setup(t)
	h, err := c.Plan(ctx, "gamma-parts", "fo_ann", to, "eu", []string{"platform-kill"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Execute(ctx, "gamma-parts", "fo_ann", h); !errors.Is(err, ErrApproval) {
		t.Fatalf("execute without approvals: %v", err)
	}
	if err := c.Approve(ctx, "gamma-parts", "factory", "fo_ann", h); !errors.Is(err, ErrApproval) {
		t.Fatalf("the planner approved: %v", err)
	}
	if err := c.Approve(ctx, "gamma-parts", "tenant", "to_cy", "ph_other"); !errors.Is(err, ErrApproval) {
		t.Fatalf("a stale hash was approved: %v", err)
	}
	_ = c.Approve(ctx, "gamma-parts", "factory", "fo_bob", h)
	_ = c.Approve(ctx, "gamma-parts", "tenant", "to_cy", h)
	ph, err := c.Execute(ctx, "gamma-parts", "fo_ann", h)
	if err != nil || ph != TenantOwned {
		t.Fatalf("execute: %s %v", ph, err)
	}
	if strings.Join(tg.calls, ",") != "provision,move,verify,cutover" {
		t.Fatalf("target calls: %v", tg.calls)
	}
	left, _ := a.Paths(ctx, "gamma-parts")
	if strings.Join(left, ",") != "platform-kill,publish-releases" {
		t.Fatalf("paths left: %v", left)
	}
	if strings.Join(cu.got, ",") != cell.CustodyTenant+","+to+","+cell.KeyCustodianTenant {
		t.Fatalf("custody: %v", cu.got)
	}
	inv, _ := keywrap.Inventory(ctx, "gamma-parts", c.Wraps, map[string]ports.Keys{"factory": xorKeys{}, "tenant": xorKeys{}})
	if fmt.Sprint(inv[0].Working) != "[tenant]" {
		t.Fatalf("wraps: %+v", inv)
	}
	var states []string
	for _, e := range r.ev {
		if e.Kind != "custody" {
			t.Fatalf("a %s row", e.Kind)
		}
		states = append(states, e.State)
	}
	want := "HandoffPlanned,Approved,Approved,Provisioned,KeyDual,Moved,Verified,CutOver,TenantOwned"
	if strings.Join(states, ",") != want {
		t.Fatalf("audit rows: %v", states)
	}
}

func TestAbortBeforeCutoverOnly(t *testing.T) {
	ctx := context.Background()
	c, a, tg, _, _ := setup(t)
	h, _ := c.Plan(ctx, "gamma-parts", "fo_ann", to, "eu", nil)
	_ = c.Approve(ctx, "gamma-parts", "factory", "fo_bob", h)
	_ = c.Approve(ctx, "gamma-parts", "tenant", "to_cy", h)
	for i := 0; i < 2; i++ { // Provisioned, KeyDual
		if _, err := c.Step(ctx, "gamma-parts", "fo_ann"); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Abort(ctx, "gamma-parts", "fo_ann", "tenant asked"); err != nil {
		t.Fatal(err)
	}
	s, _ := c.State(ctx, "gamma-parts")
	ws, _ := c.Wraps.List(ctx, "gamma-parts")
	paths, _ := a.Paths(ctx, "gamma-parts")
	if s.Phase != Managed || s.Serving != "factoryCell" || len(ws) != 1 || ws[0].Custodian != "factory" || len(paths) != 4 || tg.calls[len(tg.calls)-1] != "teardown" {
		t.Fatalf("after abort: %+v %+v %v %v", s, ws, paths, tg.calls)
	}
	h, _ = c.Plan(ctx, "gamma-parts", "fo_ann", to, "eu", nil)
	_ = c.Approve(ctx, "gamma-parts", "factory", "fo_bob", h)
	_ = c.Approve(ctx, "gamma-parts", "tenant", "to_cy", h)
	if _, err := c.Execute(ctx, "gamma-parts", "fo_ann", h); err != nil {
		t.Fatal(err)
	}
	if err := c.Abort(ctx, "gamma-parts", "fo_ann", "too late"); !errors.Is(err, ErrPhase) {
		t.Fatalf("abort after cutover: %v", err)
	}
	if err := c.Detach(ctx, "gamma-parts", "to_cy"); err != nil {
		t.Fatal(err)
	}
	if paths, _ := a.Paths(ctx, "gamma-parts"); len(paths) != 0 {
		t.Fatalf("detached with paths %v", paths)
	}
}

func TestPlanRules(t *testing.T) {
	ctx := context.Background()
	c, _, _, _, _ := setup(t)
	for _, bad := range []struct {
		target string
		retain []string
	}{{"gcp://x", nil}, {to, []string{"read-content"}}} {
		if _, err := c.Plan(ctx, "gamma-parts", "fo_ann", bad.target, "eu", bad.retain); !errors.Is(err, ErrPlan) {
			t.Errorf("%v: %v", bad, err)
		}
	}
	h1, _ := c.Plan(ctx, "gamma-parts", "fo_ann", to, "eu", nil)
	_ = c.Approve(ctx, "gamma-parts", "tenant", "to_cy", h1)
	h2, _ := c.Plan(ctx, "gamma-parts", "fo_ann", to, "eu", nil)
	if h1 == h2 {
		t.Fatal("a new plan kept the old hash")
	}
	s, _ := c.State(ctx, "gamma-parts")
	if len(s.Approvals) != 0 {
		t.Fatal("a new plan kept the old approvals")
	}
}
