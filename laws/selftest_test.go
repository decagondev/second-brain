package laws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/core/export"
	"example.com/second-brain/internal/core/verify"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/policy/cedar"
	"example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/attest"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/runtime"
	"example.com/second-brain/internal/core/schedule"
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

// LAW-09's check catches a pipeline that indexes raw text, unstripped and unflagged.
func TestSelf_LAW09MutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law09(r, rawText{}) }) {
		t.Error("LAW-09: a pipeline without a sanitiser was not caught")
	}
}

type rawText struct{}

func (rawText) Ingest(text string) ([]SanitisedChunk, error) {
	return []SanitisedChunk{{Text: text}}, nil
}
func (rawText) Wrap(text, cite string, flags []string) string {
	return "<<data cite=\"" + cite + "\">>\n" + text + "\n<</data>>"
}

// LAW-01's catch-up check catches a pipeline that ignores permission-only changes.
func TestSelf_LAW01PrefilterMutationIsCaught(t *testing.T) {
	mutant := func() PrefilterWorld {
		w := NewPrefilterWorld().(*prefilterWorld)
		w.p.Connector = dropACLOnly{w.c}
		return w
	}
	if !violates(runs, func(r *rand.Rand) error { return law01Prefilter(r, mutant()) }) {
		t.Error("LAW-01 (prefilter): ignoring permission-only changes was not caught")
	}
}

type dropACLOnly struct{ ports.Connector }

func (d dropACLOnly) Changes(ctx context.Context, c ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	set, next, err := d.Connector.Changes(ctx, c)
	var out ports.ChangeSet
	for _, ch := range set {
		if !ch.ACLOnly {
			out = append(out, ch)
		}
	}
	return out, next, err
}

// The retrieval-level laws catch mutations of the real ask pipeline, not only of the reference.
func TestSelf_RealAskMutationsAreCaught(t *testing.T) {
	cases := []struct {
		name string
		sys  askSystem
		law  string
	}{
		{"cell indexes other tenants' documents", askSystem{indexOthers: true}, "LAW-02"},
		{"unknown live answer treated as allow", askSystem{unknownAsAllow: true}, "LAW-03"},
	}
	for _, c := range cases {
		if !violates(runs, run(allLaws[c.law], c.sys)) {
			t.Errorf("%s: mutation %q was not caught", c.law, c.name)
		}
	}
}

// LAW-14's check catches a composer that trusts the model's labels.
func TestSelf_LAW14MutationIsCaught(t *testing.T) {
	trusting := func(raw string, labels map[string]string) ([]ComposedClaim, error) {
		var parsed struct {
			Claims []struct {
				Text  string   `json:"text"`
				Cites []string `json:"cites"`
			} `json:"claims"`
		}
		if json.Unmarshal([]byte(raw), &parsed) != nil {
			return []ComposedClaim{{Text: raw, Uncited: true}}, nil
		}
		var out []ComposedClaim
		for _, c := range parsed.Claims {
			cc := ComposedClaim{Text: c.Text}
			for _, l := range c.Cites {
				if cite, ok := labels[l]; ok {
					cc.Cites = append(cc.Cites, cite)
				} else {
					cc.Cites = append(cc.Cites, l) // passes made-up citations through
				}
			}
			out = append(out, cc)
		}
		return out, nil
	}
	if !violates(runs, func(r *rand.Rand) error { return law14(r, trusting) }) {
		t.Error("LAW-14: a composer that trusts the model's citations was not caught")
	}
}

// LAW-07's prompt check catches a provider called without the gateway in front.
func TestSelf_LAW07PromptMutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law07Prompt(r, newGatewayWorld(true)) }) {
		t.Error("LAW-07: a model call without the gateway's secret check was not caught")
	}
}

// LAW-14's end-to-end check catches an ask path that cites refused candidates.
func TestSelf_LAW14AskMutationIsCaught(t *testing.T) {
	f := func(r *rand.Rand) error {
		w := Gen(r)
		return law14Ask(w, askSystem{w: w, citeCandidates: true})
	}
	if !violates(runs, f) {
		t.Error("LAW-14: an ask path citing refused candidates was not caught")
	}
}

// LAW-01's chat check catches a chat app that maps the Slack user to the wrong person.
func TestSelf_LAW01ChatMutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { w := Gen(r); return law01Chat(w, chatSystem{wrongPerson: true}.ask) }) {
		t.Error("LAW-01: a chat app answering as the wrong person was not caught")
	}
}

// LAW-11's tamper check catches a verifier that only looks for anchors.
func TestSelf_LAW11MutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law11(r, newChainWorld(true)) }) {
		t.Error("LAW-11: a verifier that doesn't recompute the chain was not caught")
	}
}

// LAW-10's timed check catches a checker that never reloads the kill list.
func TestSelf_LAW10MutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law10(r, newKillWorld(true)) }) {
		t.Error("LAW-10: a checker with a stale kill list was not caught")
	}
}

// LAW-04 watches the real audit path: a store that drops tool_call rows is caught.
func TestSelf_LAW04BusMutationIsCaught(t *testing.T) {
	f := func(r *rand.Rand) error { w := Gen(r); return law04(w, askSystem{w: w, dropAudit: true}, r) }
	if !violates(runs, f) {
		t.Error("LAW-04: a tool bus whose audit rows are never stored was not caught")
	}
}

// LAW-15's check catches approvals that don't bind the hash the approver saw.
func TestSelf_LAW15MutationIsCaught(t *testing.T) {
	f := func(r *rand.Rand) error { w := newApprovalWorld(); w.ignoreEcho = true; return law15(r, w) }
	if !violates(runs, f) {
		t.Error("LAW-15: approving without checking the echoed hash was not caught")
	}
}

// LAW-07's audit check catches a log that doesn't redact.
func TestSelf_LAW07AuditMutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law07Audit(r, newAuditSecretWorld(true)) }) {
		t.Error("LAW-07: an audit log without redaction was not caught")
	}
}

// LAW-06's check catches a cell that verifies the signature but not the gate.
func TestSelf_LAW06MutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law06(r, newReleaseWorld(true)) }) {
		t.Error("LAW-06: a cell that ignores the gate record was not caught")
	}
}

// LAW-08's check catches a controller that promotes past the guard.
func TestSelf_LAW08MutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law08(r, newRolloutWorld(stableTenants, testingTenants, true)) }) {
		t.Error("LAW-08: promoting without the promote guard was not caught")
	}
}

// LAW-16's drift check catches a detector that only compares attributes.
func TestSelf_LAW16DriftMutationIsCaught(t *testing.T) {
	if !violates(runs, func(r *rand.Rand) error { return law16Drift(r, detectDrift(true)) }) {
		t.Error("LAW-16: a drift detector that misses added and removed resources was not caught")
	}
}

// LAW-01's CRM check catches a connector that reads a team-level role as view-all.
func TestSelf_LAW01CRMMutationIsCaught(t *testing.T) {
	mutant := func() CRMWorld { return teamAsAll{NewCRMWorld()} }
	if !violates(runs, func(r *rand.Rand) error { return law01CRM(r, mutant()) }) {
		t.Error("LAW-01 (CRM): reading a team role as view-all was not caught")
	}
}

type teamAsAll struct{ CRMWorld }

func (m teamAsAll) AddUser(email, level string, teams ...string) {
	if level == "team" {
		level = "all"
	}
	m.CRMWorld.AddUser(email, level, teams...)
}

func (m teamAsAll) SetLevel(email, level string) {
	if level == "team" {
		level = "all"
	}
	m.CRMWorld.SetLevel(email, level)
}

// LAW-13's retention check catches a job that ages the index with the derived stores, and
// one that keeps the record exactly retention_days old out of the sweep's keep set.
func TestSelf_LAW13RetentionMutationsAreCaught(t *testing.T) {
	ages := func() RetentionWorld {
		return newRetentionWorld(func(w *retentionWorld) []retention.Store { return []retention.Store{indexAger{w.in}} })
	}
	if !violates(runs, func(r *rand.Rand) error { return law13Retention(r, ages()) }) {
		t.Error("LAW-13 (retention): a job that expires the index was not caught")
	}
	strict := func() RetentionWorld {
		return newRetentionWorld(func(w *retentionWorld) []retention.Store { return []retention.Store{offByOne{w}} })
	}
	if !violates(runs, func(r *rand.Rand) error { return law13Retention(r, strict()) }) {
		t.Error("LAW-13 (retention): deleting records exactly retention_days old was not caught")
	}
}

type indexAger struct{ w *ingestWorld }

func (indexAger) Name() string { return "index" }
func (a indexAger) Expire(context.Context, string, time.Time) (int, error) {
	return 1, a.w.Drop()
}
func (indexAger) Erase(context.Context, string, string) (int, error) { return 0, nil }

// offByOne deletes traces at the cutoff too.
type offByOne struct{ w *retentionWorld }

func (offByOne) Name() string { return "traces-strict" }
func (o offByOne) Expire(ctx context.Context, tenant string, before time.Time) (int, error) {
	n := 0
	for _, s := range o.w.traces.Retention() {
		k, err := s.Expire(ctx, tenant, before.Add(time.Nanosecond))
		if err != nil {
			return n, err
		}
		n += k
	}
	return n, nil
}
func (offByOne) Erase(context.Context, string, string) (int, error) { return 0, nil }

// LAW-11's restore check catches a restore that trusts the first copy without checking it.
func TestSelf_LAW11RestoreMutationIsCaught(t *testing.T) {
	if !violates(60, func(r *rand.Rand) error { return law11Restore(r, newDRWorld(t.TempDir(), true)) }) {
		t.Error("LAW-11 (restore): restoring an unchecked copy was not caught")
	}
}

// refHandoff is a reference hand-off controller written from Handoff.tla, so the LAW-17
// and LAW-18 checks can be shown to pass on a correct controller and to catch broken ones
// before SB-705 exists.
type refHandoff struct {
	s       HandoffState
	failing bool
	bug     string // "", "destroy-early", "keep-paths", "planner-approves", "abort-keeps-wrap"
}

func newRefHandoff(bug string) *refHandoff {
	paths := []string{"deploy-role", "break-glass-1", "break-glass-2", "iam-1", "iam-2", "publish-releases", "platform-kill", "support-metrics"}
	return &refHandoff{bug: bug, s: HandoffState{Phase: "Managed", Serving: "factoryCell", Wraps: map[string]bool{"factory": true},
		Access: append([]string(nil), paths...), Paths: paths, Approved: map[string]string{}, Approvers: map[string]string{}}}
}

func (h *refHandoff) Plan(planner string, retain []string) (string, error) {
	if h.s.Phase != "Managed" && h.s.Phase != "Planned" {
		return "", fmt.Errorf("EINVAL: a hand-off is running")
	}
	h.s.Phase, h.s.Planner, h.s.Retained = "Planned", planner, append([]string(nil), retain...)
	h.s.Plan = fmt.Sprintf("ph_%s_%d", strings.Join(retain, "+"), len(h.s.Approved)+len(h.s.Plan))
	h.s.Approved, h.s.Approvers = map[string]string{}, map[string]string{}
	return h.s.Plan, nil
}

func (h *refHandoff) Approve(side, who, hash string) error {
	if h.s.Phase != "Planned" || hash != h.s.Plan || (side == "factory" && who == h.s.Planner && h.bug != "planner-approves") {
		return fmt.Errorf("ENOAPPROVAL")
	}
	h.s.Approved[side], h.s.Approvers[side] = hash, who
	return nil
}

func (h *refHandoff) Fail() { h.failing = true }

func (h *refHandoff) Step() error {
	if h.failing {
		h.failing = false
		return fmt.Errorf("check failed")
	}
	switch h.s.Phase {
	case "Planned":
		if h.s.Approved["factory"] != h.s.Plan || h.s.Approved["tenant"] != h.s.Plan {
			return fmt.Errorf("ENOAPPROVAL")
		}
		h.s.Phase = "Provisioned"
	case "Provisioned":
		h.s.Wraps["tenant"], h.s.Phase = true, "KeyDual"
		if h.bug == "destroy-early" {
			h.s.Wraps["factory"] = false
		}
	case "KeyDual":
		h.s.Phase = "Moved"
	case "Moved":
		h.s.Phase = "Verified"
	case "Verified":
		h.s.Serving, h.s.Phase = "tenantCell", "CutOver"
		if h.bug != "keep-paths" {
			h.s.Access = append([]string(nil), h.s.Retained...)
		}
	case "CutOver":
		h.s.Wraps["factory"], h.s.Phase = false, "TenantOwned"
	}
	return nil
}

func (h *refHandoff) Abort() error {
	switch h.s.Phase {
	case "Planned", "Provisioned", "KeyDual", "Moved", "Verified":
		h.s.Phase, h.s.Plan = "Managed", ""
		h.s.Approved, h.s.Approvers = map[string]string{}, map[string]string{}
		h.s.Wraps = map[string]bool{"factory": true, "tenant": h.bug == "abort-keeps-wrap" && h.s.Wraps["tenant"]}
		return nil
	}
	return fmt.Errorf("EINVAL")
}

func (h *refHandoff) Detach() error {
	if h.s.Phase != "TenantOwned" {
		return fmt.Errorf("EINVAL")
	}
	h.s.Phase, h.s.Access = "Detached", nil
	return nil
}

func (h *refHandoff) State() HandoffState {
	c := h.s
	c.Wraps = map[string]bool{}
	for k, v := range h.s.Wraps {
		if v {
			c.Wraps[k] = true
		}
	}
	c.Access = append([]string(nil), h.s.Access...)
	return c
}

func TestSelf_HandoffReferencePassesLAW17AndLAW18(t *testing.T) {
	for seed := int64(1); seed <= runs; seed++ {
		if err := law17Handoff(rand.New(rand.NewSource(seed)), newRefHandoff("")); err != nil {
			t.Fatalf("seed %d: LAW-17 on the reference: %v", seed, err)
		}
		if err := law18Handoff(rand.New(rand.NewSource(seed)), newRefHandoff("")); err != nil {
			t.Fatalf("seed %d: LAW-18 on the reference: %v", seed, err)
		}
	}
}

func TestSelf_HandoffMutationsAreCaught(t *testing.T) {
	for bug, law := range map[string]func(*rand.Rand, HandoffWorld) error{
		"destroy-early": law18Handoff, "keep-paths": law17Handoff, "planner-approves": law17Handoff, "abort-keeps-wrap": law17Handoff,
	} {
		if !violates(runs, func(r *rand.Rand) error { return law(r, newRefHandoff(bug)) }) {
			t.Errorf("hand-off bug %q was not caught", bug)
		}
	}
}

// LAW-18's wrap check catches a destroy that doesn't verify the other custodian first, and a
// drop that removes the last wrap.
func TestSelf_LAW18WrapMutationsAreCaught(t *testing.T) {
	for name, mk := range map[string]func(string) WrapWorld{
		"unchecked destroy":  func(root string) WrapWorld { return uncheckedDestroy{NewWrapWorld(root).(*wrapWorld)} },
		"drop the last wrap": func(root string) WrapWorld { return dropAll{NewWrapWorld(root).(*wrapWorld)} },
	} {
		if !violates(runs, func(r *rand.Rand) error { return law18Wraps(r, mk(t.TempDir())) }) {
			t.Errorf("LAW-18 (wraps): %s was not caught", name)
		}
	}
}

type uncheckedDestroy struct{ *wrapWorld }

func (u uncheckedDestroy) Destroy(gone string) error {
	ws, _ := u.st.List(context.Background(), wrapTenant)
	for _, x := range ws {
		if x.Custodian == gone {
			_ = u.st.Delete(context.Background(), wrapTenant, x.Purpose, gone)
		}
	}
	return nil
}

type dropAll struct{ *wrapWorld }

func (d dropAll) Drop(c string) error {
	ws, _ := d.st.List(context.Background(), wrapTenant)
	for _, x := range ws {
		if x.Custodian == c {
			_ = d.st.Delete(context.Background(), wrapTenant, x.Purpose, c)
		}
	}
	return nil
}

// The hand-off law tests reach every phase on the real controller, so their checks of
// CutOver, TenantOwned and Detached aren't vacuous.
func TestSelf_HandoffRunsReachEveryPhase(t *testing.T) {
	seen := map[string]bool{}
	for seed := int64(1); seed <= runs; seed++ {
		_ = handoffRun(rand.New(rand.NewSource(seed)), NewHandoffWorld(t.TempDir()), func(_ string, s HandoffState) error {
			seen[s.Phase] = true
			return nil
		})
	}
	for _, p := range []string{"Managed", "Planned", "Provisioned", "KeyDual", "Moved", "Verified", "CutOver", "TenantOwned", "Detached"} {
		if !seen[p] {
			t.Errorf("no run reached %s", p)
		}
	}
}

// LAW-17's access check catches a source that doesn't read the key's policy.
func TestSelf_LAW17AccessMutationIsCaught(t *testing.T) {
	mutant := func() (AccessWorld, func()) {
		w, done := NewAccessWorld()
		aw := w.(*accessWorld)
		aw.sources[0] = skipKey{aw.sources[0]}
		return aw, done
	}
	if !violates(runs, func(r *rand.Rand) error { w, done := mutant(); defer done(); return law17Access(r, w) }) {
		t.Error("LAW-17 (access): ignoring the key's IAM policy was not caught")
	}
}

type skipKey struct{ access.Source }

func (s skipKey) Paths(ctx context.Context, tenant string) ([]access.Path, error) {
	ps, err := s.Source.Paths(ctx, tenant)
	var out []access.Path
	for _, p := range ps {
		if !strings.HasPrefix(p.ID, "cell-key:") {
			out = append(out, p)
		}
	}
	return out, err
}

// LAW-08's check catches a promote guard that ignores cells in a tenant's account (SB-707).
func TestSelf_LAW08TenantCustodyMutationIsCaught(t *testing.T) {
	mutant := func() RolloutWorld {
		w := NewTenantCustodyRolloutWorld(stableTenants, testingTenants).(*rolloutWorld)
		w.skipTenantCustody = true
		return w
	}
	if !violates(runs, func(r *rand.Rand) error { return law08(r, mutant()) }) {
		t.Error("LAW-08: a guard that skips tenant-custody cells was not caught")
	}
}

// LAW-11's export check catches a verifier that only checks file digests.
func TestSelf_LAW11ExportMutationIsCaught(t *testing.T) {
	mutant := func(root string) ExportWorld {
		return &exportWorld{root: root, verify: func(ctx context.Context, dir string) (export.Report, error) {
			raw, err := os.ReadFile(filepath.Join(dir, "MANIFEST.json"))
			if err != nil {
				return export.Report{}, err
			}
			var man struct{ Files map[string]string }
			_ = json.Unmarshal(raw, &man)
			for name, want := range man.Files {
				b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
				s := sha256.Sum256(b)
				if err != nil || hex.EncodeToString(s[:]) != want {
					return export.Report{}, export.ErrFiles
				}
			}
			return export.Report{}, nil
		}}
	}
	if !violates(runs, func(r *rand.Rand) error { return law11Export(r, mutant(t.TempDir())) }) {
		t.Error("LAW-11 (export): a digest-only verifier was not caught")
	}
}

// LAW-16's planned-field check catches a parser that quietly drops a spaces block.
func TestSelf_LAW16PlannedMutationIsCaught(t *testing.T) {
	drops := func(src []byte) (manifest.Tenant, []manifest.Diag) {
		var keep []string
		skipping := false
		for _, l := range strings.SplitAfter(string(src), "\n") {
			if strings.HasPrefix(l, "spaces:") || strings.HasPrefix(l, "barriers:") {
				skipping = true
				continue
			}
			if skipping && (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "-")) {
				continue
			}
			skipping = false
			keep = append(keep, l)
		}
		return manifest.Check([]byte(strings.Join(keep, "")))
	}
	if !violates(runs, func(r *rand.Rand) error { return law16Planned(r, drops) }) {
		t.Error("LAW-16 (planned fields): a parser that drops spaces was not caught")
	}
}

// LAW-06's runtime check catches an updater that deploys without admitting the release.
func TestSelf_LAW06RuntimeMutationIsCaught(t *testing.T) {
	if !violates(40, func(r *rand.Rand) error {
		w, done := NewRuntimeWorld()
		defer done()
		return law06Runtime(r, skipAdmit{w.(*runtimeWorld)})
	}) {
		t.Error("LAW-06 (runtime): an updater that skips admission was not caught")
	}
}

type skipAdmit struct{ *runtimeWorld }

func (s skipAdmit) Update(target string) error {
	m, err := s.rw.manifest(target)
	if err != nil {
		return err
	}
	u := s.updater(target)
	for _, svc := range runtime.Services {
		img := u.Image(svc, m.Code[svc.Image])
		if err := s.cr.Attest(context.Background(), img); err != nil {
			return err
		}
		if err := s.cr.Deploy(context.Background(), svc, img, target); err != nil {
			return err
		}
	}
	return nil
}

// LAW-02's perimeter check catches cells with no perimeter and a cross-cell IAM mistake.
func TestSelf_LAW02PerimeterMutationIsCaught(t *testing.T) {
	saved := NewPerimeterWorld
	defer func() { NewPerimeterWorld = saved }()
	NewPerimeterWorld = func(bool) (PerimeterWorld, func(), error) {
		w, done, err := saved(true)
		if err == nil {
			pw := w.(*perimeterWorld)
			pw.f.Perimeter(rtProjectB, "serviceAccount:"+gcp.ServiceAccount(rtProjectB, "api"), "serviceAccount:"+gcp.ServiceAccount(rtProjectA, "api"))
		}
		return w, done, err
	}
	if !violates(40, func(r *rand.Rand) error { return law02Perimeter(r, true) }) {
		t.Error("LAW-02 (perimeter): a perimeter that lets the other cell in was not caught")
	}
}

// The missed-run checks catch a watchdog that treats an unreadable record as fine, and one
// that alerts law-carrying jobs below Sev-1.
func TestSelf_WatchMutationsAreCaught(t *testing.T) {
	lenient := func(ctx context.Context, runs schedule.Runs, now time.Time) []schedule.Alert {
		var out []schedule.Alert
		for _, a := range schedule.Watch(ctx, runs, now, nil) {
			if _, _, err := runs.LastFinish(ctx, a.Job); err == nil {
				out = append(out, a)
			}
		}
		return out
	}
	quiet := func(ctx context.Context, runs schedule.Runs, now time.Time) []schedule.Alert {
		out := schedule.Watch(ctx, runs, now, nil)
		for i := range out {
			out[i].Severity = "sev2"
		}
		return out
	}
	for name, w := range map[string]WatchFunc{"unknown is fine": lenient, "all Sev-2": quiet} {
		if !violates(runs, func(r *rand.Rand) error { return lawWatched("LAW-11")(r, w) }) {
			t.Errorf("watchdog bug %q was not caught", name)
		}
	}
}

// LAW-06's attestation check catches an attestor that reports the release it meant to run.
func TestSelf_LAW06AttestationMutationIsCaught(t *testing.T) {
	if !violates(40, func(r *rand.Rand) error {
		w, done := NewRuntimeWorld()
		defer done()
		rw := w.(*runtimeWorld)
		rw.lyingAttestor = lying{rw}
		return law06Attestation(r, rw)
	}) {
		t.Error("LAW-06 (attestation): an attestor that reports intent, not what runs, was not caught")
	}
}

type lying struct{ w *runtimeWorld }

func (l lying) Attest(ctx context.Context) (attest.Signed, error) {
	api, rel, err := l.w.cr.Running(ctx, runtime.Services[0])
	if err != nil || api == "" {
		return attest.Signed{}, fmt.Errorf("nothing runs")
	}
	m, err := l.w.rw.manifest(rel)
	if err != nil {
		return attest.Signed{}, err
	}
	r := attest.Report{Tenant: "gamma-parts", Release: rel, Digests: map[string]string{}, Seq: time.Now().UnixNano(), At: time.Now()}
	for _, s := range runtime.Services {
		r.Digests[s.Image] = m.Code[s.Image]
	}
	return attest.Sign(l.w.attestKey, r)
}

// LAW-17's verify check catches a verify that passes when the enumeration failed.
func TestSelf_LAW17VerifyMutationIsCaught(t *testing.T) {
	lenient := func(ctx context.Context, tg verify.Target, o verify.Options) verify.Report {
		rep := verify.Run(ctx, tg, o)
		for i := range rep.Results {
			if rep.Results[i].Status == "unknown" {
				rep.Results[i].Status = "pass"
			}
		}
		return rep
	}
	if !violates(runs, func(r *rand.Rand) error { return law17Verify(r, lenient) }) {
		t.Error("LAW-17 (verify): unknown counted as a pass was not caught")
	}
}
