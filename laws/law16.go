package laws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/placement/local"
	"example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/apply"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/plan"
	"example.com/second-brain/internal/core/ports"
)

// LAW-16, manifest side (SB-101): what the manifest says is exactly what is applied.
// A valid manifest, written in any allowed style, parses to exactly the tenant it
// describes: nothing dropped, nothing invented. A manifest with a field outside
// sb-tenant(5) is never applied, so nothing can enter a cell beside the manifest.
// The plan side (SB-102): the plan is exactly the diff between manifests.
// The apply side (SB-104): after apply, the recorded state is exactly the manifest.
// The live side (drift detection, SB-510) is checked when it is built.

// ManifestChecker is the system under test: bytes in, tenant or errors out.
type ManifestChecker func(src []byte) (manifest.Tenant, []manifest.Diag)

func law16(r *rand.Rand, check ManifestChecker) error {
	want := genTenant(r)
	entries := renderTenant(r, want)
	src := strings.Join(entries, "")
	got, ds := check([]byte(src))
	if len(ds) > 0 {
		return fmt.Errorf("valid manifest rejected: %s\n%s", ds[0].Format("gen.yaml"), src)
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("manifest says %+v but %+v would be applied\n%s", want, got, src)
	}

	// Add one field the manifest may not carry, between two fields.
	bad := []string{"version", "platform", "custody", "placement", "key_custodian",
		"api_token", "secret", "password", "token", "key", "colour"}[r.Intn(11)]
	at := r.Intn(len(entries) + 1)
	line := 1
	for _, e := range entries[:at] {
		line += strings.Count(e, "\n")
	}
	mutated := strings.Join(entries[:at], "") + bad + ": x\n" + strings.Join(entries[at:], "")
	got, ds = check([]byte(mutated))
	if !reflect.DeepEqual(got, manifest.Tenant{}) {
		return fmt.Errorf("manifest with %q was applied:\n%s", bad, mutated)
	}
	for _, d := range ds {
		if d.Line == line && d.Col == 1 && d.Field == bad {
			return nil
		}
	}
	return fmt.Errorf("field %q on line %d not reported at its key; got %v\n%s", bad, line, ds, mutated)
}

func genTenant(r *rand.Rand) manifest.Tenant {
	pick := func(xs []string) string { return xs[r.Intn(len(xs))] }
	t := manifest.Tenant{
		Tenant:         pick([]string{"acme-logistics", "beta-freight", "gamma-parts"}),
		Channel:        pick(manifest.Channels),
		IdP:            pick([]string{"oidc://login.acme.example", "oidc://id.beta.example/realms/main", "oidc://sso.gamma.example"}),
		Region:         pick(manifest.Regions),
		Connectors:     map[string]string{},
		Agents:         []string{"ask"},
		RetentionDays:  manifest.DefaultRetentionDays,
		BudgetUSDMonth: manifest.DefaultBudgetUSD,
	}
	vendors := map[string][]string{"docs": {"sharepoint", "confluence"}, "chat": {"slack", "teams"},
		"tickets": {"jira", "zendesk"}, "crm": {"salesforce", "hubspot"}, "code": {"github"}}
	for _, k := range manifest.ConnectorKinds {
		if r.Intn(2) == 0 || len(t.Connectors) == 0 && k == "crm" {
			t.Connectors[k] = pick(vendors[k])
		}
	}
	if r.Intn(2) == 0 {
		t.Agents = append(t.Agents, "draft-reply")
		if r.Intn(2) == 0 {
			t.Agents[0], t.Agents[1] = t.Agents[1], t.Agents[0]
		}
	}
	if r.Intn(2) == 0 {
		t.Connectors["tickets"] = pick(vendors["tickets"])
		t.Workflows = []string{"ticket-update"}
		t.Approval = map[string]string{"ticket-update": pick([]string{"group:it-leads", "group:ops.managers"})}
	}
	if r.Intn(2) == 0 {
		t.RetentionDays = manifest.MinRetentionDays + r.Intn(manifest.MaxRetentionDays-manifest.MinRetentionDays+1)
	}
	if r.Intn(2) == 0 {
		t.BudgetUSDMonth = 1 + r.Intn(100000)
	}
	if r.Intn(2) == 0 {
		t.PilotGroup = "group:sb-pilot"
	}
	if r.Intn(2) == 0 {
		t.Exclude = map[string][]string{}
		for k := range t.Connectors {
			if r.Intn(2) == 0 {
				t.Exclude[k] = []string{"#hr-*", "/finance/**", "#legal-*"}[:1+r.Intn(3)]
			}
		}
		if len(t.Exclude) == 0 {
			t.Exclude = nil
		}
	}
	return t
}

// renderTenant writes t as top-level entries (each ending in a newline), in random
// order and style: flow or block collections, plain or quoted scalars, comments.
func renderTenant(r *rand.Rand, t manifest.Tenant) []string {
	sc := func(v string) string {
		switch r.Intn(3) {
		case 0:
			return `"` + v + `"`
		case 1:
			return "'" + v + "'"
		}
		return v
	}
	comment := func() string {
		if r.Intn(4) == 0 {
			return "  # note"
		}
		return ""
	}
	scalar := func(k, v string) string { return k + ": " + sc(v) + comment() + "\n" }
	list := func(k string, vs []string, quote bool) string {
		q := func(v string) string {
			if quote {
				return `"` + v + `"`
			}
			return sc(v)
		}
		if r.Intn(2) == 0 {
			var parts []string
			for _, v := range vs {
				parts = append(parts, q(v))
			}
			return k + ": [" + strings.Join(parts, ", ") + "]" + comment() + "\n"
		}
		indent := []string{"", "  "}[r.Intn(2)]
		s := k + ":" + comment() + "\n"
		for _, v := range vs {
			s += indent + "- " + q(v) + "\n"
		}
		return s
	}
	dict := func(k string, m map[string]string) string {
		ks := sortedKeys(m)
		if r.Intn(2) == 0 {
			var parts []string
			for _, x := range ks {
				parts = append(parts, x+": "+m[x])
			}
			return k + ": { " + strings.Join(parts, ", ") + " }" + comment() + "\n"
		}
		s := k + ":\n"
		for _, x := range ks {
			s += "  " + x + ": " + sc(m[x]) + comment() + "\n"
		}
		return s
	}
	es := []string{
		scalar("tenant", t.Tenant), scalar("channel", t.Channel), scalar("idp", t.IdP),
		scalar("region", t.Region), dict("connectors", t.Connectors), list("agents", t.Agents, false),
	}
	if t.Workflows != nil {
		es = append(es, list("workflows", t.Workflows, false), dict("approval", t.Approval))
	}
	if t.RetentionDays != manifest.DefaultRetentionDays || r.Intn(2) == 0 {
		es = append(es, fmt.Sprintf("retention_days: %d\n", t.RetentionDays))
	}
	if t.BudgetUSDMonth != manifest.DefaultBudgetUSD || r.Intn(2) == 0 {
		es = append(es, fmt.Sprintf("budget_usd_month: %d\n", t.BudgetUSDMonth))
	}
	if t.PilotGroup != "" {
		es = append(es, scalar("pilot_group", t.PilotGroup))
	}
	if t.Exclude != nil {
		s := "exclude:\n"
		for _, k := range sortedKeys(t.Exclude) {
			for _, l := range strings.SplitAfter(list(k, t.Exclude[k], true), "\n") {
				if l != "" {
					s += "  " + l
				}
			}
		}
		es = append(es, s)
	}
	r.Shuffle(len(es), func(i, j int) { es[i], es[j] = es[j], es[i] })
	return es
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// Planner is the system under test for the plan side of LAW-16 (SB-102).
type Planner func(t manifest.Tenant, applied []ports.Resource, found bool) (plan.Plan, error)

// law16Plan: the plan is exactly the manifest diff. Enacting the plan on a cell built
// from one manifest yields the cell of the next; a cell that already matches plans nothing.
func law16Plan(r *rand.Rand, mk Planner) error {
	before := genTenant(r)
	after := genTenant(r)
	after.Tenant, after.Region = before.Tenant, before.Region // immutable fields stay
	p, err := mk(after, plan.Desired(before), true)
	if err != nil {
		return fmt.Errorf("plan failed: %v", err)
	}
	if got := plan.Apply(plan.Desired(before), p); !reflect.DeepEqual(got, plan.Desired(after)) {
		return fmt.Errorf("enacting the plan gives %v, but the manifest says %v", got, plan.Desired(after))
	}
	if p, err := mk(after, plan.Desired(after), true); err != nil || len(p.Changes) != 0 {
		return fmt.Errorf("a cell that matches its manifest plans %v (err %v)", p.Changes, err)
	}
	if p, err := mk(after, nil, false); err != nil || p.Add != len(plan.Desired(after)) || p.Change+p.Remove != 0 {
		return fmt.Errorf("a new cell plans %+v (err %v)", p, err)
	}
	return nil
}

// Applier enacts a manifest against a control plane rooted at a directory (SB-104).
type Applier func(root string, t manifest.Tenant) (plan.Plan, error)

// NewApplier is the system under test: the real apply engine on the local control plane.
var NewApplier = func() Applier {
	return func(root string, t manifest.Tenant) (plan.Plan, error) {
		reg := localfs.Dir{Root: root}
		acct := local.Account{Root: filepath.Join(root, "accounts"), Name: cell.PlacementFactoryAccount}
		// Every workflow's evals have passed here: LAW-16 is about apply matching the
		// manifest, and the eval gate has its own tests (SB-410).
		d := apply.Deps{Registry: reg, Writer: reg, Journal: reg, Cells: reg, BaseDomain: "sb.localhost", Placement: acct, Keys: acct,
			Evals: passedEvals{}, Now: time.Now}
		return apply.Run(context.Background(), d, t, nil)
	}
}

// law16Apply: after apply, the cell's recorded state is exactly the manifest, and
// applying the same manifest again changes nothing.
func law16Apply(r *rand.Rand, root string, run Applier) error {
	if err := os.MkdirAll(filepath.Join(root, "tenants"), 0o755); err != nil {
		return err
	}
	before := genTenant(r)
	after := genTenant(r)
	after.Tenant, after.Region = before.Tenant, before.Region
	for _, t := range []manifest.Tenant{before, after} {
		if _, err := run(root, t); err != nil {
			return fmt.Errorf("apply failed: %v", err)
		}
	}
	got, found, err := localfs.Dir{Root: root}.Applied(context.Background(), after.Tenant)
	if err != nil || !found || !reflect.DeepEqual(got, plan.Desired(after)) {
		return fmt.Errorf("after apply the registry holds %v (found %v, err %v), but the manifest says %v", got, found, err, plan.Desired(after))
	}
	if p, err := run(root, after); err != nil || len(p.Changes) != 0 {
		return fmt.Errorf("re-applying the same manifest changed %v (err %v)", p.Changes, err)
	}
	return nil
}

// passedEvals says every workflow's evals passed an hour ago.
type passedEvals struct{}

func (passedEvals) EvalRecord(context.Context, string, string) (ports.EvalRecord, bool, error) {
	return ports.EvalRecord{Passed: true, At: time.Now().Add(-time.Hour), Suites: []string{"tenant/law"}, Model: "law"}, true, nil
}

// DriftDetector compares live resources with applied ones and names each difference as
// "kind/name/field" (field is "" for a resource that is present on one side only).
type DriftDetector func(applied, live []ports.Resource) []string

// NewDriftDetector builds the system under test (SB-510).
var NewDriftDetector = func() DriftDetector {
	return func([]ports.Resource, []ports.Resource) []string { return []string{"pending"} }
}

// law16Drift: every live change to applied state is reported, by resource and field, and
// nothing else is (SB-510).
func law16Drift(r *rand.Rand, detect DriftDetector) error {
	if got := detect(nil, nil); len(got) == 1 && got[0] == "pending" {
		return fmt.Errorf("%w: drift detection arrives with SB-510", ErrPending)
	}
	kinds := []string{"connector", "agent", "setting", "workflow"}
	var applied []ports.Resource
	for i := 0; i < 2+r.Intn(5); i++ {
		applied = append(applied, ports.Resource{Kind: kinds[r.Intn(len(kinds))], Name: fmt.Sprintf("r%d", i),
			Attrs: map[string]string{"a": fmt.Sprint(r.Intn(3)), "b": "x"}})
	}
	live := make([]ports.Resource, 0, len(applied))
	want := map[string]bool{}
	for _, res := range applied {
		c := ports.Resource{Kind: res.Kind, Name: res.Name, Attrs: map[string]string{}}
		for k, v := range res.Attrs {
			c.Attrs[k] = v
		}
		switch r.Intn(5) {
		case 0:
			c.Attrs["a"] = c.Attrs["a"] + "!"
			want[c.Kind+"/"+c.Name+"/a"] = true
		case 1:
			delete(c.Attrs, "b")
			want[c.Kind+"/"+c.Name+"/b"] = true
		case 2:
			want[c.Kind+"/"+c.Name+"/"] = true
			continue // gone live
		}
		live = append(live, c)
	}
	if r.Intn(2) == 0 {
		live = append(live, ports.Resource{Kind: "agent", Name: "rogue", Attrs: map[string]string{}})
		want["agent/rogue/"] = true
	}
	got := map[string]bool{}
	for _, g := range detect(applied, live) {
		got[g] = true
	}
	for k := range want {
		if !got[k] {
			return fmt.Errorf("drift %s was not reported (got %v)", k, got)
		}
	}
	for k := range got {
		if !want[k] {
			return fmt.Errorf("reported drift %s that isn't there", k)
		}
	}
	return nil
}

// LAW-16 for spaces (SB-1001, which replaced SB-801's refusal): every space and barrier the
// manifest declares is in the checked tenant and in the plan, exactly as written, and no
// other exists. A parser that drops or invents a space would let a company believe its
// People brain was restricted when nothing said so.
func law16Spaces(r *rand.Rand, check ManifestChecker) error {
	t := genTenant(r)
	entries := renderTenant(r, t)
	if t.Connectors["docs"] == "" {
		return nil // spaces here scope docs; another run covers it
	}
	type decl struct {
		name, template string
		members        []string
		stewards       []string
		docs           string
	}
	templates := []string{"engineering", "people", "finance", "support", "custom", "legal"}
	var decls []decl
	var b strings.Builder
	b.WriteString("spaces:\n")
	for i, n := 0, 1+r.Intn(3); i < n; i++ {
		d := decl{name: fmt.Sprintf("space-%d", i), template: templates[r.Intn(len(templates))],
			members: []string{fmt.Sprintf("group:m%d", i)}, docs: fmt.Sprintf("/Dept%d/**", i)}
		if r.Intn(2) == 0 {
			d.members = append(d.members, "group:shared")
		}
		if manifest.Templates[d.template].Label == "restricted" || r.Intn(2) == 0 {
			d.stewards = []string{fmt.Sprintf("group:s%d", i)}
		}
		decls = append(decls, d)
		fmt.Fprintf(&b, "  %s:\n    template: %s\n    members: [%s]\n", d.name, d.template, strings.Join(d.members, ", "))
		if len(d.stewards) > 0 {
			fmt.Fprintf(&b, "    stewards: [%s]\n", strings.Join(d.stewards, ", "))
		}
		fmt.Fprintf(&b, "    scope:\n      docs: [%q]\n", d.docs)
	}
	barrier := r.Intn(2) == 0
	if barrier {
		b.WriteString("barriers:\n  - [group:deal-team-orion, group:everyone]\n")
	}
	at := r.Intn(len(entries) + 1)
	src := strings.Join(entries[:at], "") + b.String() + strings.Join(entries[at:], "")
	got, ds := check([]byte(src))
	if len(ds) > 0 {
		return fmt.Errorf("valid spaces rejected: %s\n%s", ds[0].Format("gen.yaml"), src)
	}
	if len(got.Spaces) != len(decls) {
		return fmt.Errorf("manifest declares %d spaces but %d would be applied\n%s", len(decls), len(got.Spaces), src)
	}
	for _, d := range decls {
		s, ok := got.Spaces[d.name]
		switch {
		case !ok:
			return fmt.Errorf("space %s was dropped\n%s", d.name, src)
		case !reflect.DeepEqual(s.Members, d.members) || len(s.Stewards) != len(d.stewards) || s.Template != d.template:
			return fmt.Errorf("space %s is %+v, not as declared\n%s", d.name, s, src)
		case !reflect.DeepEqual(s.Scope, map[string][]string{"docs": {d.docs}}):
			return fmt.Errorf("space %s scope is %v, not docs %s", d.name, s.Scope, d.docs)
		}
	}
	if barrier != (len(got.Barriers) == 1) {
		return fmt.Errorf("barriers %v, declared %v\n%s", got.Barriers, barrier, src)
	}
	planned := 0
	for _, res := range plan.Desired(got) {
		switch res.Kind {
		case plan.KindSpace:
			if _, ok := got.Spaces[res.Name]; !ok {
				return fmt.Errorf("the plan has space %s, which the manifest doesn't declare", res.Name)
			}
			planned++
		case plan.KindBarrier:
			planned++
		}
	}
	if want := len(decls) + len(got.Barriers); planned != want {
		return fmt.Errorf("the plan has %d spaces and barriers, the manifest %d", planned, want)
	}
	return nil
}
