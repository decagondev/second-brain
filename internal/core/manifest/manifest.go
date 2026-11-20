// Package manifest parses and validates tenant manifests (sb-tenant(5)). It does no
// I/O: callers pass the bytes and get back either a Tenant or positioned errors.
//
// The field table below is the single source for the checker and for the JSON
// Schema (schema.go); a test keeps the Tenant type's json tags in step with it.
package manifest

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Tenant is a valid manifest with defaults applied. Absent optional lists and maps are nil.
type Tenant struct {
	Tenant         string              `json:"tenant"`
	Channel        string              `json:"channel"`
	IdP            string              `json:"idp"`
	Region         string              `json:"region"`
	Connectors     map[string]string   `json:"connectors"`
	Agents         []string            `json:"agents"`
	Workflows      []string            `json:"workflows"`
	Approval       map[string]string   `json:"approval"`
	RetentionDays  int                 `json:"retention_days"`
	BudgetUSDMonth int                 `json:"budget_usd_month"`
	PilotGroup     string              `json:"pilot_group"`
	Exclude        map[string][]string `json:"exclude"`
	Groups         map[string][]string `json:"groups"`
	Spaces         map[string]Space    `json:"spaces"`
	Barriers       [][2]string         `json:"barriers"`
}

// Diag is one error, positioned at a key or a value (1-based line and column).
type Diag struct {
	Line  int    `json:"line"`
	Col   int    `json:"col"`
	Field string `json:"field"`
	Msg   string `json:"message"`
}

// Count is the summary line after the errors: "1 error", "8 errors".
func Count(n int) string {
	if n == 1 {
		return "1 error"
	}
	return fmt.Sprintf("%d errors", n)
}

// Format renders d in compiler format: file:line:col: field: message.
func (d Diag) Format(file string) string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", file, d.Line, d.Col, d.Field, d.Msg)
}

// The vocabulary of sb-tenant(5), shared by the checker and the schema.
var (
	Channels       = []string{"stable", "testing"}
	Regions        = []string{"eu", "us"}
	ConnectorKinds = []string{"docs", "chat", "tickets", "crm", "code", "hris", "accounting"}
	Agents         = []string{"ask", "draft-reply", "incident-summary", "spec-digest", "close-checklist", "runbook-answer"}
	// Workflows maps each known workflow to the connector kind it writes to.
	Workflows = map[string]string{"ticket-update": "tickets"}
)

const (
	MinRetentionDays     = 30
	MaxRetentionDays     = 3650
	DefaultRetentionDays = 365
	MinBudgetUSDMonth    = 1
	DefaultBudgetUSD     = 1000
	MaxWorkflows         = 1
	MaxGroups            = 200
	MaxGroupMembers      = 2000

	dnsLabelRe = `^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`
	idpRe      = `^(oidc://([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(/[^\s?#]*)?|identity-platform://[a-z][a-z0-9-]{4,28}[a-z0-9]/[A-Za-z0-9-]{1,63})$`
	emailRe    = `^[^@\s]+@[^@\s]+\.[^@\s]+$`
	vendorRe   = `^[a-z0-9-]+$`
	groupRe    = `^group:[A-Za-z0-9._-]+$`
)

var (
	dnsLabel = regexp.MustCompile(dnsLabelRe)
	idpURI   = regexp.MustCompile(idpRe)
	vendor   = regexp.MustCompile(vendorRe)
	group    = regexp.MustCompile(groupRe)
	email    = regexp.MustCompile(emailRe)
	integer  = regexp.MustCompile(`^-?[0-9]+$`)
)

type field struct {
	name     string
	required bool
	check    func(c *checker, n *node)
}

// fields is in sb-tenant(5) FIELDS order.
var fields = []field{
	{"tenant", true, func(c *checker, n *node) {
		c.t.Tenant = c.match(n, "tenant", dnsLabel, "must be a DNS label ([a-z0-9-], 3-40 chars)")
	}},
	{"channel", true, func(c *checker, n *node) { c.t.Channel = c.oneOf(n, "channel", Channels) }},
	{"idp", true, func(c *checker, n *node) {
		c.t.IdP = c.match(n, "idp", idpURI, "must be oidc://ISSUER-HOST[/path] or identity-platform://PROJECT/IDP_TENANT")
	}},
	{"region", true, func(c *checker, n *node) { c.t.Region = c.oneOf(n, "region", Regions) }},
	{"connectors", true, (*checker).connectors},
	{"agents", true, (*checker).agents},
	{"workflows", false, (*checker).workflows},
	{"approval", false, (*checker).approval},
	{"retention_days", false, func(c *checker, n *node) {
		c.t.RetentionDays = c.intIn(n, "retention_days", MinRetentionDays, MaxRetentionDays)
	}},
	{"budget_usd_month", false, func(c *checker, n *node) {
		c.t.BudgetUSDMonth = c.intIn(n, "budget_usd_month", MinBudgetUSDMonth, 0)
	}},
	{"pilot_group", false, func(c *checker, n *node) { c.t.PilotGroup = c.group(n, "pilot_group") }},
	{"exclude", false, (*checker).exclude},
	{"groups", false, (*checker).groups},
	{"spaces", false, (*checker).spaces},
	{"barriers", false, (*checker).barriers},
}

// Planned are top-level fields that are specified but not built yet, with their tickets.
// Refusing them by name keeps a company from believing a field protects what it doesn't.
// None today: spaces and barriers were here until SB-1001.
var Planned = map[string]string{}

const (
	msgVersion = "field not allowed; pick a channel (stable|testing) instead (ADR-0004)"
	msgSecret  = "field not allowed; secrets live in vault://tenants/<tenant>/connectors/<kind>"
	msgCustody = "field not allowed; custody changes only through a dual-approved hand-off, see sb-factory(7)"
)

// forbidden reports why a field name may never appear anywhere (sb-tenant(5) rules 4 and 6).
func forbidden(name string) (string, bool) {
	switch name {
	case "custody", "placement", "key_custodian":
		return msgCustody, true
	}
	for _, part := range strings.Split(name, "_") {
		switch part {
		case "version", "platform":
			return msgVersion, true
		case "secret", "password", "token", "key":
			return msgSecret, true
		}
	}
	return "", false
}

type checker struct {
	t     Tenant
	diags []Diag
	skip  map[*node]bool // keys already reported (forbidden or duplicate)
	seen  map[string]*node
	// For the cross-field rules: valid workflow items, approval keys, exclude keys.
	wfNodes, apKeys, exKeys []*node
	spaceNodes              map[string]spaceNodes
	channelOwner            map[string]string // chat channel -> the space it asks in
	barrierNodes            []*node
}

func (c *checker) add(n *node, field, format string, args ...any) {
	c.diags = append(c.diags, Diag{n.line, n.col, field, fmt.Sprintf(format, args...)})
}

// Check parses and validates a manifest. It returns the Tenant only if there are no errors.
func Check(src []byte) (Tenant, []Diag) {
	root, serr := parse(src)
	if serr != nil {
		return Tenant{}, []Diag{{serr.line, serr.col, "syntax", serr.msg}}
	}
	if root == nil {
		return Tenant{}, []Diag{{1, 1, "syntax", "the manifest is empty"}}
	}
	if root.kind != mapNode {
		return Tenant{}, []Diag{{root.line, root.col, "syntax", "the top level must be fields (`field: value`)"}}
	}
	c := &checker{skip: map[*node]bool{}, seen: map[string]*node{}}
	c.t.RetentionDays, c.t.BudgetUSDMonth = DefaultRetentionDays, DefaultBudgetUSD
	c.names(root, "")
	known := map[string]field{}
	for _, f := range fields {
		known[f.name] = f
	}
	for i, k := range root.keys {
		if c.skip[k] {
			continue
		}
		f, ok := known[k.value]
		if !ok {
			if t, later := Planned[k.value]; later {
				c.add(k, k.value, "not built yet (%s); a manifest that declares it is refused, never run without it (sb-space(5))", t)
				continue
			}
			c.add(k, k.value, "unknown field; see sb-tenant(5) FIELDS")
			continue
		}
		c.seen[k.value] = root.vals[i]
		f.check(c, root.vals[i])
	}
	for _, f := range fields {
		if _, ok := c.seen[f.name]; f.required && !ok {
			c.add(root, f.name, "required field is missing")
		}
	}
	c.crossChecks()
	sort.SliceStable(c.diags, func(i, j int) bool {
		a, b := c.diags[i], c.diags[j]
		return a.Line < b.Line || (a.Line == b.Line && a.Col < b.Col)
	})
	if len(c.diags) > 0 {
		return Tenant{}, c.diags
	}
	return c.t, nil
}

// names reports forbidden and duplicate field names at every depth.
func (c *checker) names(n *node, path string) {
	switch n.kind {
	case mapNode:
		first := map[string]*node{}
		for i, k := range n.keys {
			p := join(path, k.value)
			if msg, bad := forbidden(k.value); bad {
				c.add(k, p, "%s", msg)
				c.skip[k] = true
			} else if f, dup := first[k.value]; dup {
				c.add(k, p, "duplicate field; first set on line %d", f.line)
				c.skip[k] = true
			} else {
				first[k.value] = k
			}
			c.names(n.vals[i], p)
		}
	case seqNode:
		for i, it := range n.items {
			c.names(it, fmt.Sprintf("%s[%d]", path, i))
		}
	case scalarNode:
	}
}

func join(path, k string) string {
	if path == "" {
		return k
	}
	return path + "." + k
}

func (c *checker) scalar(n *node, field string) (string, bool) {
	if n.kind != scalarNode {
		c.add(n, field, "must be a single value")
		return "", false
	}
	if n.value == "" {
		c.add(n, field, "needs a value")
		return "", false
	}
	return n.value, true
}

func (c *checker) match(n *node, field string, re *regexp.Regexp, msg string) string {
	v, ok := c.scalar(n, field)
	if ok && !re.MatchString(v) {
		c.add(n, field, "%q %s", v, msg)
		return ""
	}
	return v
}

func (c *checker) oneOf(n *node, field string, allowed []string) string {
	v, ok := c.scalar(n, field)
	if ok && !contains(allowed, v) {
		c.add(n, field, "%q must be one of %s", v, strings.Join(allowed, ", "))
		return ""
	}
	return v
}

func (c *checker) group(n *node, field string) string {
	v, ok := c.scalar(n, field)
	if ok && !group.MatchString(v) {
		c.add(n, field, "%q must be a group (group:NAME), never a user", v)
		return ""
	}
	return v
}

// intIn parses a whole number in [lo, hi]; hi 0 means no upper bound.
func (c *checker) intIn(n *node, field string, lo, hi int) int {
	v, ok := c.scalar(n, field)
	if !ok {
		return 0
	}
	i, err := strconv.Atoi(v)
	if n.quoted || !integer.MatchString(v) || err != nil {
		c.add(n, field, "%q must be a whole number", v)
		return 0
	}
	switch {
	case hi > 0 && (i < lo || i > hi):
		c.add(n, field, "%d must be between %d and %d", i, lo, hi)
	case i < lo:
		c.add(n, field, "%d must be at least %d", i, lo)
	}
	return i
}

func (c *checker) mapOf(n *node, field, example string) bool {
	if n.kind != mapNode {
		c.add(n, field, "must be a map, e.g. %s", example)
		return false
	}
	return true
}

// list returns the scalar items of a list, reporting non-scalar or empty items.
func (c *checker) list(n *node, field, example string) ([]string, []*node, bool) {
	if n.kind != seqNode {
		c.add(n, field, "must be a list, e.g. %s", example)
		return nil, nil, false
	}
	var vs []string
	var ns []*node
	for i, it := range n.items {
		if v, ok := c.scalar(it, fmt.Sprintf("%s[%d]", field, i)); ok {
			vs, ns = append(vs, v), append(ns, it)
		}
	}
	return vs, ns, true
}

func (c *checker) connectors(n *node) {
	if !c.mapOf(n, "connectors", "{ docs: sharepoint }") {
		return
	}
	c.t.Connectors = map[string]string{}
	for i, k := range n.keys {
		if c.skip[k] {
			continue
		}
		p := "connectors." + k.value
		if !contains(ConnectorKinds, k.value) {
			c.add(k, p, "unknown connector kind; use %s", orList(ConnectorKinds))
			continue
		}
		c.t.Connectors[k.value] = c.match(n.vals[i], p, vendor, "must be a vendor name ([a-z0-9-])")
	}
	if len(n.keys) == 0 {
		c.add(n, "connectors", "needs at least one connector, e.g. { docs: sharepoint }")
	}
}

func (c *checker) agents(n *node) {
	vs, ns, ok := c.list(n, "agents", "[ask]")
	if !ok {
		return
	}
	at := map[string]bool{}
	for i, v := range vs {
		f := fmt.Sprintf("agents[%d]", i)
		switch {
		case !contains(Agents, v):
			c.add(ns[i], f, "%q is not an agent; use %s", v, orList(Agents))
		case at[v]:
			c.add(ns[i], f, "%q is listed twice", v)
		default:
			c.t.Agents = append(c.t.Agents, v)
		}
		at[v] = true
	}
	if !at["ask"] {
		c.add(n, "agents", `"ask" is required`)
	}
}

func (c *checker) workflows(n *node) {
	vs, ns, ok := c.list(n, "workflows", "[ticket-update]")
	if !ok {
		return
	}
	if len(vs) > MaxWorkflows {
		c.add(n, "workflows", "at most %d workflow in v1", MaxWorkflows)
	}
	for i, v := range vs {
		if _, known := Workflows[v]; !known {
			c.add(ns[i], fmt.Sprintf("workflows[%d]", i), "%q is not a known workflow; use %s", v, orList(keys(Workflows)))
			continue
		}
		c.t.Workflows = append(c.t.Workflows, v)
		c.wfNodes = append(c.wfNodes, ns[i])
	}
}

func (c *checker) approval(n *node) {
	if !c.mapOf(n, "approval", "{ ticket-update: group:it-leads }") {
		return
	}
	c.t.Approval = map[string]string{}
	for i, k := range n.keys {
		if c.skip[k] {
			continue
		}
		c.t.Approval[k.value] = c.group(n.vals[i], "approval."+k.value)
		c.apKeys = append(c.apKeys, k)
	}
}

func (c *checker) exclude(n *node) {
	if !c.mapOf(n, "exclude", `{ chat: ["#hr-*"] }`) {
		return
	}
	c.t.Exclude = map[string][]string{}
	for i, k := range n.keys {
		if c.skip[k] {
			continue
		}
		p := "exclude." + k.value
		if !contains(ConnectorKinds, k.value) {
			c.add(k, p, "unknown connector kind; use %s", orList(ConnectorKinds))
			continue
		}
		vs, _, ok := c.list(n.vals[i], p, `["#hr-*"]`)
		if ok {
			c.t.Exclude[k.value] = vs
		}
		c.exKeys = append(c.exKeys, k)
	}
}

// groups is the steward-managed group list for an identity-platform idp (SB-907).
func (c *checker) groups(n *node) {
	if !c.mapOf(n, "groups", `{ "group:finance": [ana@acme.example] }`) {
		return
	}
	c.t.Groups = map[string][]string{}
	if len(n.keys) > MaxGroups {
		c.add(n, "groups", "at most %d groups", MaxGroups)
	}
	members := 0
	for i, k := range n.keys {
		if c.skip[k] {
			continue
		}
		p := "groups." + k.value
		if !group.MatchString(k.value) {
			c.add(k, p, "%q must be a group (group:NAME)", k.value)
			continue
		}
		vs, ns, ok := c.list(n.vals[i], p, "[ana@acme.example]")
		if !ok {
			continue
		}
		at := map[string]bool{}
		var ok2 []string
		for j, v := range vs {
			f := fmt.Sprintf("%s[%d]", p, j)
			e := strings.ToLower(v)
			switch {
			case !email.MatchString(v):
				c.add(ns[j], f, "%q must be an email", v)
			case at[e]:
				c.add(ns[j], f, "%q is listed twice", v)
			default:
				ok2 = append(ok2, e)
			}
			at[e] = true
		}
		members += len(vs)
		c.t.Groups[k.value] = ok2
	}
	if members > MaxGroupMembers {
		c.add(n, "groups", "at most %d members in all, not %d", MaxGroupMembers, members)
	}
}

// IdentityPlatform splits an identity-platform://PROJECT/IDP_TENANT idp; ok is false for any other.
func IdentityPlatform(idp string) (project, idpTenant string, ok bool) {
	rest, found := strings.CutPrefix(idp, "identity-platform://")
	if !found {
		return "", "", false
	}
	project, idpTenant, ok = strings.Cut(rest, "/")
	return project, idpTenant, ok && project != "" && idpTenant != ""
}

// crossChecks are the rules that relate fields (sb-tenant(5) rule 3, and exclude kinds).
// A rule about connectors applies only when connectors is a map; otherwise that field
// already has its own error.
func (c *checker) crossChecks() {
	for i, w := range c.t.Workflows {
		n := c.wfNodes[i]
		f := fmt.Sprintf("workflows[%d]", indexOf(c.seen["workflows"].items, n))
		if _, ok := c.t.Approval[w]; !ok {
			c.add(n, f, "%q needs an approval rule; add approval: { %s: group:<name> }", w, w)
		}
		if kind := Workflows[w]; c.connectorKindAbsent(kind) {
			c.add(n, f, "%q needs connector kind %q in connectors", w, kind)
		}
	}
	for _, k := range c.apKeys {
		if wf := c.seen["workflows"]; wf != nil && wf.kind != seqNode {
			break // workflows already has its own error
		}
		if !contains(listValues(c.seen["workflows"]), k.value) {
			c.add(k, "approval."+k.value, "no workflow %q in workflows", k.value)
		}
	}
	if g := c.seen["groups"]; g != nil {
		if _, _, ip := IdentityPlatform(c.t.IdP); !ip && c.t.IdP != "" {
			c.add(g, "groups", "only with an identity-platform:// idp; an OIDC IdP's groups come from SCIM (sb-identity(7))")
		}
	}
	c.spaceChecks()
	for _, k := range c.exKeys {
		if c.connectorKindAbsent(k.value) {
			c.add(k, "exclude."+k.value, "no connector kind %q in connectors", k.value)
		}
	}
}

// connectorKindAbsent is true when connectors is a map and no entry names the kind, even
// an invalid one (an invalid vendor is already reported; don't report it twice).
func (c *checker) connectorKindAbsent(kind string) bool {
	n := c.seen["connectors"]
	if n == nil || n.kind != mapNode {
		return false
	}
	for _, k := range n.keys {
		if k.value == kind {
			return false
		}
	}
	return true
}

func listValues(n *node) []string {
	if n == nil || n.kind != seqNode {
		return nil
	}
	var out []string
	for _, it := range n.items {
		out = append(out, it.value)
	}
	return out
}

func indexOf(ns []*node, n *node) int {
	for i, x := range ns {
		if x == n {
			return i
		}
	}
	return -1
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// orList renders [a b c] as "a, b or c".
func orList(xs []string) string {
	if len(xs) == 1 {
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " or " + xs[len(xs)-1]
}
