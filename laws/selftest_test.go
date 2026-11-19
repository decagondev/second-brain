package laws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/policy/cedar"
	"example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/obs"
)

// The law checks are themselves tested: a correct reference system must pass
// every law, and each deliberately broken system must be caught. This is the
// permanent mutation check for the guardrails (docs/05-living-documents.md, DoD).

// reference is the simplest correct system: prefilter in own tenant, live check,
// keep only Allow; audit before every call; approvals for consequential calls; Ran only with a receipt.
type reference struct {
	prefilterOnly, crossTenant, unknownAsAllow bool // ask mutations
	execFirst, ignoreApproval, claimRan        bool // tool mutations
}

func (s reference) Ask(w *World, userID, _ string) (Answer, error) {
	u := w.user(userID)
	var a Answer
	for _, d := range w.Docs {
		if (d.Tenant != u.Tenant && !s.crossTenant) || !w.PrefilterMatch(userID, d.ID) && !s.crossTenant {
			continue
		}
		v := ports.VerdictAllow
		if !s.prefilterOnly {
			v = w.ACL(userID, d.ID)
		}
		if v == ports.VerdictAllow || (s.unknownAsAllow && v == ports.VerdictUnknown) {
			a.Context = append(a.Context, d.ID)
			a.Citations = append(a.Citations, Citation{Doc: d.ID, Version: d.Version})
		}
	}
	return a, nil
}

func (s reference) Call(w *World, _ string, tc ToolCall) (Outcome, error) {
	if tc.Consequential && !w.Approvals[tc.Approval] && !s.ignoreApproval {
		return Outcome{State: "Refused"}, nil
	}
	var rc string
	if s.execFirst {
		rc = w.Act(tc.ID)
		w.Audit(tc.ID, tc.Approval)
	} else {
		w.Audit(tc.ID, tc.Approval)
		rc = w.Act(tc.ID)
	}
	if rc == "" && !s.claimRan {
		return Outcome{State: "DidNotRun"}, nil
	}
	if rc == "" {
		rc = "rcpt_claimed"
	}
	return Outcome{State: "Ran", Receipt: rc}, nil
}

type lawFn func(w *World, sys System, r *rand.Rand) error

var allLaws = map[string]lawFn{
	"LAW-01": func(w *World, s System, _ *rand.Rand) error { return law01(w, s) },
	"LAW-02": func(w *World, s System, _ *rand.Rand) error { return law02(w, s) },
	"LAW-03": func(w *World, s System, _ *rand.Rand) error { return law03(w, s) },
	"LAW-04": law04,
	"LAW-05": law05,
}

func run(law lawFn, sys System) func(r *rand.Rand) error {
	return func(r *rand.Rand) error { w := Gen(r); return law(w, sys, r) }
}

func TestSelf_ReferencePassesEveryLaw(t *testing.T) {
	for id, law := range allLaws {
		Check(t, id+" (reference)", runs, run(law, reference{}))
	}
}

func TestSelf_EachMutationIsCaught(t *testing.T) {
	cases := []struct {
		name string
		sys  reference
		law  string
	}{
		{"prefilter only, no live check", reference{prefilterOnly: true}, "LAW-01"},
		{"ignores tenant boundary", reference{crossTenant: true}, "LAW-02"},
		{"treats unknown as allow", reference{unknownAsAllow: true}, "LAW-03"},
		{"executes before auditing", reference{execFirst: true}, "LAW-04"},
		{"ignores approvals", reference{ignoreApproval: true}, "LAW-04"},
		{"claims Ran without a receipt", reference{claimRan: true}, "LAW-05"},
	}
	for _, c := range cases {
		if !violates(runs, run(allLaws[c.law], c.sys)) {
			t.Errorf("%s: mutation %q was not caught", c.law, c.name)
		}
	}
}

// The generator must actually produce the hard cases, or the laws prove little.
func TestSelf_GeneratorCoversHardCases(t *testing.T) {
	var stale, unknown, missing, timeout, noReceipt, multiTenant, public int
	for seed := int64(1); seed <= runs; seed++ {
		w := Gen(rand.New(rand.NewSource(seed)))
		tenants := map[string]bool{}
		for _, d := range w.Docs {
			tenants[d.Tenant] = true
			if d.Missing {
				missing++
			}
			if d.Public {
				public++
			}
			for _, u := range w.Users {
				if u.Tenant != d.Tenant {
					continue
				}
				v, known := d.Live[u.ID]
				if !known {
					unknown++
				}
				if w.PrefilterMatch(u.ID, d.ID) && v == ports.VerdictDeny {
					stale++
				}
			}
		}
		if len(tenants) > 1 {
			multiTenant++
		}
		if w.Fault == FaultTimeout {
			timeout++
		}
		if w.NoReceipt {
			noReceipt++
		}
	}
	for name, n := range map[string]int{"stale prefilter": stale, "unknown verdict": unknown, "missing doc": missing,
		"timeout fault": timeout, "no receipt": noReceipt, "multi-tenant world": multiTenant, "public doc": public} {
		if n == 0 {
			t.Errorf("generator never produced: %s", name)
		}
	}
}

// LAW-16's check catches checkers that drop, invent or let through fields.
func TestSelf_LAW16MutationsAreCaught(t *testing.T) {
	ok := manifest.Check
	cases := map[string]ManifestChecker{
		"ignores unknown and forbidden fields": func(src []byte) (manifest.Tenant, []manifest.Diag) {
			var kept []string
			for _, l := range strings.SplitAfter(string(src), "\n") {
				if !regexp.MustCompile(`^(version|platform|custody|placement|key_custodian|api_token|secret|password|token|key|colour):`).MatchString(l) {
					kept = append(kept, l)
				}
			}
			return ok([]byte(strings.Join(kept, "")))
		},
		"drops exclude": func(src []byte) (manifest.Tenant, []manifest.Diag) {
			t, ds := ok(src)
			t.Exclude = nil
			return t, ds
		},
		"invents a default workflow": func(src []byte) (manifest.Tenant, []manifest.Diag) {
			t, ds := ok(src)
			if t.Workflows == nil && len(ds) == 0 {
				t.Workflows = []string{}
			}
			return t, ds
		},
	}
	for name, c := range cases {
		if !violates(runs, func(r *rand.Rand) error { return law16(r, c) }) {
			t.Errorf("LAW-16: mutation %q was not caught", name)
		}
	}
}

// LAW-16's plan check catches planners that forget removals or changes.
func TestSelf_LAW16PlanMutationsAreCaught(t *testing.T) {
	drop := func(op string) Planner {
		return func(m manifest.Tenant, applied []ports.Resource, found bool) (plan.Plan, error) {
			p, err := plan.Make(m, applied, found)
			var kept []plan.Change
			for _, c := range p.Changes {
				if c.Op != op {
					kept = append(kept, c)
				}
			}
			p.Changes = kept
			return p, err
		}
	}
	for _, op := range []string{plan.OpRemove, plan.OpChange} {
		if !violates(runs, func(r *rand.Rand) error { return law16Plan(r, drop(op)) }) {
			t.Errorf("LAW-16: planner that drops %q changes was not caught", op)
		}
	}
}

// LAW-02's cell check catches templates that share a resource or lose the region pin.
func TestSelf_LAW02CellMutationsAreCaught(t *testing.T) {
	shared := map[string]string{} // one "database" id for every tenant in an account
	cases := map[string]func(root string) CellStamper{
		"shares the database": func(root string) CellStamper {
			real := NewCellStamper(root)
			return func(acct string, m manifest.Tenant) ([]Stamped, error) {
				rs, err := real(acct, m)
				for i := range rs {
					if rs[i].Kind == "database" {
						if _, ok := shared[root+acct]; !ok {
							shared[root+acct] = rs[i].ID
						}
						rs[i].ID = shared[root+acct]
					}
				}
				return rs, err
			}
		},
		"drops the region pin": func(root string) CellStamper {
			real := NewCellStamper(root)
			return func(acct string, m manifest.Tenant) ([]Stamped, error) {
				rs, err := real(acct, m)
				for i := range rs {
					rs[i].Region = ""
				}
				return rs, err
			}
		},
		"forgets the queue": func(root string) CellStamper {
			real := NewCellStamper(root)
			return func(acct string, m manifest.Tenant) ([]Stamped, error) {
				rs, err := real(acct, m)
				var kept []Stamped
				for _, x := range rs {
					if x.Kind != "queue" {
						kept = append(kept, x)
					}
				}
				return kept, err
			}
		},
	}
	for name, mk := range cases {
		if !violates(20, func(r *rand.Rand) error { return law02Cells(r, mk(t.TempDir())) }) {
			t.Errorf("LAW-02 (cells): mutation %q was not caught", name)
		}
	}
}

// LAW-16's apply check catches an applier that skips removals.
func TestSelf_LAW16ApplyMutationIsCaught(t *testing.T) {
	skipRemovals := func(root string, m manifest.Tenant) (plan.Plan, error) {
		reg := localfs.Dir{Root: root}
		applied, found, err := reg.Applied(context.Background(), m.Tenant)
		if err != nil {
			return plan.Plan{}, err
		}
		p, err := plan.Make(m, applied, found)
		if err != nil {
			return p, err
		}
		var kept []plan.Change
		for _, c := range p.Changes {
			if c.Op != plan.OpRemove {
				kept = append(kept, c)
			}
		}
		p.Changes = kept
		return p, reg.Put(context.Background(), m.Tenant, plan.Apply(applied, p))
	}
	if !violates(40, func(r *rand.Rand) error { return law16Apply(r, t.TempDir(), skipRemovals) }) {
		t.Error("LAW-16: an applier that skips removals was not caught")
	}
}

// LAW-02's request check catches a router that trusts the token over the host.
func TestSelf_LAW02RouteMutationIsCaught(t *testing.T) {
	trustToken := func(cells []ports.CellRecord, req Request) (Routed, error) {
		if !req.TokenValid {
			return Routed{Code: "EAUTH"}, nil
		}
		for _, c := range cells {
			if c.Tenant == req.TokenTenant {
				return Routed{Cell: c.Tenant}, nil
			}
		}
		return Routed{Code: "EPERM"}, nil
	}
	if !violates(runs, func(r *rand.Rand) error { return law02Requests(r, trustToken) }) {
		t.Error("LAW-02 (requests): a router that routes by token tenant was not caught")
	}
}

// LAW-02's sign-in check catches a cell that ignores the tenant claim.
func TestSelf_LAW02TokenMutationIsCaught(t *testing.T) {
	real := NewTokenCheck()
	ignoresTenant := func(s TokenSpec) (string, error) {
		claim := s.Tenant
		s.Tenant = s.PresentAt
		got, err := real(s)
		if got != "" && claim != s.PresentAt {
			return s.PresentAt, err
		}
		return got, err
	}
	if !violates(100, func(r *rand.Rand) error { return law02Tokens(r, ignoresTenant) }) {
		t.Error("LAW-02 (sign-in): a cell that ignores the tenant claim was not caught")
	}
}

// LAW-12's check catches a cell that never re-validates sessions when the fan-out is lost.
func TestSelf_LAW12MutationIsCaught(t *testing.T) {
	noRevalidation := func(bool) DeprovSystem {
		d := NewDeprovSystem(true).(*deprov)
		d.cell.Directory = alwaysActive{}
		return d
	}
	if !violates(runs, func(r *rand.Rand) error { return law12(r, noRevalidation) }) {
		t.Error("LAW-12: a cell that ignores the directory after a lost fan-out was not caught")
	}
}

type alwaysActive struct{}

func (alwaysActive) Active(context.Context, string, string) (bool, bool, error) {
	return true, true, nil
}

// LAW-02's scope check catches a directory that searches every tenant.
func TestSelf_LAW02ScopeMutationIsCaught(t *testing.T) {
	leaky := func(cell string, seed []Seeded) ScopedCell {
		s := NewScopedCell(cell, seed).(*scoped)
		return leakyScoped{s, seed}
	}
	if !violates(runs, func(r *rand.Rand) error { return law02Scope(r, leaky) }) {
		t.Error("LAW-02 (scope): a lookup that ignores the scope's tenant was not caught")
	}
}

type leakyScoped struct {
	*scoped
	seed []Seeded
}

func (l leakyScoped) Person(id string) (bool, error) {
	for _, s := range l.seed {
		if s.ID == id {
			return true, nil
		}
	}
	return false, nil
}

// LAW-03's policy check catches a caller that tests "not denied" instead of "allowed".
func TestSelf_LAW03PolicyMutationIsCaught(t *testing.T) {
	notDenied := func(c PolicyCase) (bool, error) {
		s, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_1", nil)
		if c.NoScope {
			s = tenancy.Scope{}
		}
		res := policy.Entity{Type: "Chunk", ID: "c1"}
		if c.OtherTenant {
			res.Tenant = "beta-freight"
		}
		return policy.Kernel{Engine: faulty{c}, Recorder: okRecorder{}}.Authorise(context.Background(), s, "read", res, nil).Decision != policy.Deny, nil //nolint:decisions // the mutation under test
	}
	if !violates(100, func(r *rand.Rand) error { return law03Policy(r, notDenied) }) {
		t.Error("LAW-03 (policy): a caller that treats Unknown as allow was not caught")
	}
}

// LAW-02's policy check catches a permit without the tenant condition.
func TestSelf_LAW02PolicyMutationIsCaught(t *testing.T) {
	bad, err := cedar.Load(map[string][]byte{"bad.cedar": []byte(`permit (principal, action == Action::"read", resource is Chunk);`)})
	if err != nil {
		t.Fatal(err)
	}
	if !violates(runs, func(r *rand.Rand) error { return law02Policies(r, bad) }) {
		t.Error("LAW-02 (policies): a permit with no tenant condition was not caught")
	}
}

// LAW-04's decision check catches a kernel that records after deciding but ignores a failed write.
func TestSelf_LAW04DecisionMutationIsCaught(t *testing.T) {
	ignoresFailure := func(c DecisionCase) (DecisionOutcome, error) {
		c2 := c
		c2.RecorderFails = false
		out, err := NewDecisionKernel()(c2)
		if c.RecorderFails {
			out.RecordedBeforeExit = false
		}
		return out, err
	}
	if !violates(runs, func(r *rand.Rand) error { return law04Decisions(r, ignoresFailure) }) {
		t.Error("LAW-04 (decisions): a kernel that allows when the audit write fails was not caught")
	}
}

// LAW-18's check catches a cell recorded without a key custodian, or with a dead root key.
func TestSelf_LAW18MutationsAreCaught(t *testing.T) {
	for name, broke := range map[string]func(CustodyState) CustodyState{
		"no custodian":  func(s CustodyState) CustodyState { s.KeyCustodian = ""; return s },
		"dead root key": func(s CustodyState) CustodyState { s.KeyWorks = false; return s },
	} {
		real := NewCustodyCheck()
		mutant := func(root string, ts []manifest.Tenant) (map[string]CustodyState, error) {
			got, err := real(root, ts)
			for k, v := range got {
				got[k] = broke(v)
			}
			return got, err
		}
		if !violates(5, func(r *rand.Rand) error {
			root := t.TempDir()
			_ = os.MkdirAll(filepath.Join(root, "tenants"), 0o755)
			return law18(r, root, mutant)
		}) {
			t.Errorf("LAW-18: mutation %q was not caught", name)
		}
	}
}

// The connector laws catch a connector that allows on error, and one that acts twice.
func TestSelf_ConnectorMutationsAreCaught(t *testing.T) {
	if !violates(40, func(r *rand.Rand) error { return law03Connector(r, allowOnFault{NewConnectorWorld()}) }) {
		t.Error("LAW-03 (connector): allow-on-fault was not caught")
	}
	if !violates(runs, func(r *rand.Rand) error { return law05Connector(r, &actsTwice{ConnectorWorld: NewConnectorWorld()}) }) {
		t.Error("LAW-05 (connector): a non-idempotent Act was not caught")
	}
}

type allowOnFault struct{ ConnectorWorld }

func (a allowOnFault) ACL(access []bool, fault string) ([]string, error) {
	got, err := a.ConnectorWorld.ACL(access, fault)
	if fault != "" {
		for i := range got {
			got[i] = "allow"
		}
	}
	return got, err
}

type actsTwice struct {
	ConnectorWorld
	n int
}

func (a *actsTwice) Act(key string, issues bool) (string, error) {
	a.n++
	return a.ConnectorWorld.Act(fmt.Sprintf("%s#%d", key, a.n), issues)
}

// LAW-07's check catches a logger that doesn't redact, and one that redacts by dropping
// every line (the check requires output).
func TestSelf_LAW07MutationsAreCaught(t *testing.T) {
	plain := func(root string) SecretWorld {
		w := NewSecretWorld(root).(*secretWorld)
		w.red = obs.NewRedactor() // the logger never learns the secrets
		return w
	}
	if !violates(runs, func(r *rand.Rand) error { return law07(r, plain(t.TempDir())) }) {
		t.Error("LAW-07: a logger that doesn't know the secrets was not caught")
	}
	if !violates(runs, func(r *rand.Rand) error { return law07(r, silent{NewSecretWorld(t.TempDir())}) }) {
		t.Error("LAW-07: a logger that writes nothing was not caught")
	}
}

type silent struct{ SecretWorld }

func (silent) Log([]LogEvent) (string, error) { return "", nil }

// LAW-13's check catches a pipeline whose chunk ids aren't derived (random per run).
func TestSelf_LAW13MutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law13(r, nondeterministic{NewIngestWorld().(*ingestWorld), r}) }) {
		t.Error("LAW-13: an index that differs after rebuild was not caught")
	}
}

type nondeterministic struct {
	*ingestWorld
	r *rand.Rand
}

func (n nondeterministic) Snapshot() ([]string, error) {
	got, err := n.ingestWorld.Snapshot()
	if len(got) > 0 && n.r.Intn(2) == 0 {
		got[0] += "#" + fmt.Sprint(n.r.Int())
	}
	return got, err
}
