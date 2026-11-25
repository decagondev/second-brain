package manifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"example.com/second-brain/internal/core/route"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Space is one department space after its template's defaults are filled in
// (sb-space(5); SB-1001). Every field is set: nothing is left for the cell to guess.
type Space struct {
	Template      string              `json:"template"`
	Members       []string            `json:"members"`
	Stewards      []string            `json:"stewards"`
	Label         string              `json:"label"`
	Scope         map[string][]string `json:"scope"`
	Agents        []string            `json:"agents"`
	RetentionDays int                 `json:"retention_days"`
	Channels      []string            `json:"channels"`
	Fields        map[string][]string `json:"fields"`
	// StartHere is a new member's reading list: KIND:ID or KIND:/Folder/ (SB-1706).
	StartHere []string `json:"start_here"`
	// ReviewAfterDays: a relied-on document older than this goes on the stewards' review
	// list (SB-1709). 0 means the default.
	ReviewAfterDays int `json:"review_after_days"`
	// Glossary is the space's terms and their meanings (SB-2203); nil for none.
	Glossary map[string]string `json:"glossary,omitempty"`
	// Parent is the department space this team space sits under, "" for none (SB-3901).
	Parent string `json:"parent,omitempty"`
	// Brief is the space's one scheduled agent (SB-4005), nil for none.
	Brief *Brief `json:"brief,omitempty"`
	// Updates drafts a weekly status update for the stewards each Monday (SB-4102).
	Updates bool `json:"updates,omitempty"`
	// LeaderWeek false turns off the leader's week for a space with teams (SB-4503); nil
	// is on.
	LeaderWeek *bool `json:"leader_week,omitempty"`
	// Requests are the categories a member can send a request in, nil for none (SB-4104).
	Requests *Requests `json:"requests,omitempty"`
	// Model is "main" to pin every ask in the space to the main model (SB-4405).
	Model string `json:"model,omitempty"`
	// BudgetUSDMonth is the space's monthly model budget, 0 for none (SB-4105): at
	// BudgetWarnAt its stewards are told, and at it its scheduled runs pause.
	BudgetUSDMonth int `json:"budget_usd_month,omitempty"`
	// IncidentLabels are the ticket labels that draft an incident brief for the stewards
	// (SB-4204); nil is DefaultIncidentLabels.
	IncidentLabels []string `json:"incident_labels,omitempty"`
	// Skills are the space's named ways of writing an answer (SB-4601), by name; nil for none.
	Skills map[string]Skill `json:"skills,omitempty"`
	// Rhythms are the space's own playbooks (SB-4801), by name; nil for none beyond the
	// built-in ones.
	Rhythms map[string]Rhythm `json:"rhythms,omitempty"`
	// Onboarding false turns off the onboarding rhythm for the space (SB-4802); nil is on.
	Onboarding *bool `json:"onboarding,omitempty"`
}

// Skill is one way of writing an answer: an instruction on format and steps only, and the
// eval cases that hold it, question to the expected document (KIND:ID).
type Skill struct {
	Instruction string            `json:"instruction"`
	Cases       map[string]string `json:"cases"`
}

// DefaultIncidentLabels are incident_labels when a space leaves it out (SB-4204).
var DefaultIncidentLabels = []string{"sev-1"}

// MaxIncidentLabels is the most incident_labels a space lists.
const MaxIncidentLabels = 10

var incidentLabelRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.:/-]{0,39}$`)

// Incidents are the space's incident labels, its own or the default.
func (s Space) Incidents() []string {
	if s.IncidentLabels == nil {
		return DefaultIncidentLabels
	}
	return s.IncidentLabels
}

// Requests are a space's request categories and how many working days one is due in.
type Requests struct {
	Categories []string `json:"categories"`
	DueDays    int      `json:"due_days"`
}

// Request limits (SB-4104).
const (
	MaxRequestCategories = 10
	MaxCategory          = 40
	DefaultDueDays       = 5
	MaxDueDays           = 30
)

var categoryRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 &/:'-]{0,39}$`)

// Brief runs Agent once a week, on Weekly (monday to sunday), as the space's first active
// steward; its answer waits in the steward queue as a draft (sb-space(5), brief).
type Brief struct {
	Agent  string `json:"agent"`
	Weekly string `json:"weekly"`
}

// Weekdays are the days a brief can run on, lower case.
var Weekdays = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

// Weekday is a brief's day as a time.Weekday.
func (b Brief) Weekday() time.Weekday {
	for i, d := range Weekdays {
		if d == b.Weekly {
			return time.Weekday((i + 1) % 7)
		}
	}
	return time.Monday
}

// MaxGlossary is the most terms a space keeps, and MaxTerm and MaxMeaning their lengths.
const (
	MaxGlossary = 200
	MaxTerm     = 40
	MaxMeaning  = 200
)

// DefaultReviewAfterDays is review_after_days when a space leaves it out.
const DefaultReviewAfterDays = 365

// MaxStartHere is the longest reading list.
const MaxStartHere = 10

var startItem = regexp.MustCompile(`^([a-z]+):(/[^\s]*/|[^/@\s][^@\s]*)$`)

// Template is a department template's defaults (sb-departments(7), The catalogue).
// RetentionDays 0 means the tenant's.
type Template struct {
	Name          string   `json:"name"`
	Label         string   `json:"label"`
	Sources       []string `json:"sources"`
	Agents        []string `json:"agents"`
	RetentionDays int      `json:"retention_days"`
	Suite         string   `json:"suite"`
}

// Catalogue is the department template catalogue as shipped: data, never code (SB-1101).
// Its hash is pinned in every release bundle (sb-release(7)).
//
//go:embed catalogue.json
var Catalogue []byte

// Templates is the catalogue by name, read from Catalogue when the package loads. A
// catalogue that doesn't parse, or breaks a rule, stops the binary: no cell runs on one.
var Templates = mustCatalogue(Catalogue)

// ParseCatalogue reads a catalogue and checks it: unique names, a known label, agents
// that include ask, a retention of 0 or 30 and over, and a custom template.
func ParseCatalogue(b []byte) (map[string]Template, error) {
	var c struct {
		Version   int        `json:"version"`
		About     string     `json:"about"`
		Templates []Template `json:"templates"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("catalogue: %w", err)
	}
	if c.Version != 1 {
		return nil, fmt.Errorf("catalogue: version %d isn't 1", c.Version)
	}
	out := map[string]Template{}
	for _, t := range c.Templates {
		switch {
		case !spaceName.MatchString(t.Name) || t.Name == "company":
			return nil, fmt.Errorf("catalogue: bad template name %q", t.Name)
		case out[t.Name].Name != "":
			return nil, fmt.Errorf("catalogue: %s twice", t.Name)
		case !contains(Labels, t.Label):
			return nil, fmt.Errorf("catalogue: %s has label %q", t.Name, t.Label)
		case !contains(t.Agents, "ask"):
			return nil, fmt.Errorf("catalogue: %s has no ask agent", t.Name)
		case t.RetentionDays != 0 && t.RetentionDays < MinSpaceRetention:
			return nil, fmt.Errorf("catalogue: %s keeps less than %d days", t.Name, MinSpaceRetention)
		}
		out[t.Name] = t
	}
	if _, ok := out["custom"]; !ok {
		return nil, fmt.Errorf("catalogue: no custom template")
	}
	return out, nil
}

func mustCatalogue(b []byte) map[string]Template {
	t, err := ParseCatalogue(b)
	if err != nil {
		panic(err) // the catalogue is part of the release; a bad one is a build error
	}
	return t
}

// Labels are the four labels, least to most sensitive (sb-spaces(7)).
var Labels = []string{"public", "internal", "confidential", "restricted"}

const (
	MaxSpaces         = 50
	MaxBarriers       = 20
	MinSpaceRetention = 30
	spaceNameRe       = `^[a-z][a-z0-9-]{1,30}$`
)

var spaceName = regexp.MustCompile(spaceNameRe)

var spaceFields = []string{"template", "members", "stewards", "label", "scope", "agents", "retention_days", "channels", "fields", "start_here", "review_after_days", "glossary", "parent", "brief", "updates", "leader_week", "requests", "budget_usd_month", "incident_labels", "model", "skills", "rhythms", "onboarding"}

var fieldName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var channelName = regexp.MustCompile(`^#[a-z0-9][a-z0-9_-]{0,79}$`)

type spaceNodes struct {
	members, scope, agents, retention, fields, label, parent, brief, updates, budget, skills *node
	set                                                                                      map[string]bool
	start                                                                                    []*node
}

func (c *checker) spaces(n *node) {
	if !c.mapOf(n, "spaces", "{ finance: { members: [group:finance], scope: { docs: [\"/Finance/**\"] } } }") {
		return
	}
	c.t.Spaces = map[string]Space{}
	if len(n.keys) > MaxSpaces {
		c.add(n, "spaces", "at most %d spaces", MaxSpaces)
	}
	c.spaceNodes = map[string]spaceNodes{}
	for i, k := range n.keys {
		if c.skip[k] {
			continue
		}
		p := "spaces." + k.value
		switch {
		case k.value == "company":
			c.add(k, p, `"company" is reserved for company-wide asks`)
			continue
		case !spaceName.MatchString(k.value):
			c.add(k, p, "a space name is [a-z][a-z0-9-]{1,30}")
			continue
		}
		c.space(k.value, n.vals[i], p)
	}
}

func (c *checker) space(name string, n *node, p string) {
	if !c.mapOf(n, p, "{ members: [group:finance], scope: { docs: [\"/Finance/**\"] } }") {
		return
	}
	s := Space{Template: "custom", Stewards: []string{}, Channels: []string{}, Fields: map[string][]string{}}
	sn := spaceNodes{set: map[string]bool{}}
	for i, k := range n.keys {
		if c.skip[k] {
			continue
		}
		f, v := p+"."+k.value, n.vals[i]
		sn.set[k.value] = true
		switch k.value {
		case "template":
			if t, ok := c.scalar(v, f); ok {
				if _, known := Templates[t]; !known {
					c.add(v, f, "%q is not a template; use %s", t, orList(keys(Templates)))
				} else {
					s.Template = t
				}
			}
		case "members":
			s.Members, sn.members = c.groupList(v, f), v
		case "stewards":
			s.Stewards = c.groupList(v, f)
		case "label":
			s.Label, sn.label = c.oneOf(v, f, Labels), v
		case "parent":
			if pn, ok := c.scalar(v, f); ok {
				s.Parent, sn.parent = pn, v
			}
		case "scope":
			s.Scope, sn.scope = c.scope(v, f), v
		case "agents":
			sn.agents = v
			vs, ns, ok := c.list(v, f, "[ask]")
			if ok {
				s.Agents = []string{}
				for j, a := range vs {
					if !contains(s.Agents, a) {
						s.Agents = append(s.Agents, a)
					} else {
						c.add(ns[j], fmt.Sprintf("%s[%d]", f, j), "%q is listed twice", a)
					}
				}
			}
		case "retention_days":
			sn.retention = v
			s.RetentionDays = c.intIn(v, f, MinSpaceRetention, 0)
		case "fields":
			sn.fields = v
			c.spaceFields(v, f, &s)
		case "channels":
			c.spaceChannels(name, v, f, &s)
		case "start_here":
			c.startHere(v, f, &s, &sn)
		case "glossary":
			s.Glossary = c.glossary(v, f)
		case "brief":
			s.Brief, sn.brief = c.brief(v, f), v
		case "updates":
			if b, ok := c.scalar(v, f); ok && b != "true" && b != "false" {
				c.add(v, f, "must be true or false")
			} else {
				s.Updates, sn.updates = b == "true", v
			}
		case "leader_week": // SB-4503
			s.LeaderWeek = c.boolOf(v, f)
		case "requests":
			s.Requests = c.requests(v, f)
		case "model": // SB-4405
			if m, ok := c.scalar(v, f); ok && m != "main" {
				c.add(v, f, "must be main; leave it out to let short asks use the small model")
			} else {
				s.Model = m
			}
		case "budget_usd_month":
			s.BudgetUSDMonth, sn.budget = c.intIn(v, f, MinBudgetUSDMonth, 0), v
		case "incident_labels":
			s.IncidentLabels = c.incidentLabels(v, f)
		case "skills": // SB-4601
			s.Skills, sn.skills = c.skills(v, f), v
		case "rhythms", "onboarding": // SB-4801, SB-4802
			c.rhythmField(k.value, v, f, &s)
		case "review_after_days":
			if s.ReviewAfterDays = c.intIn(v, f, MinRetentionDays, MaxRetentionDays); s.ReviewAfterDays == DefaultReviewAfterDays {
				s.ReviewAfterDays = 0 // one form for the default, so Format reads back the same
			}
		default:
			c.add(k, f, "unknown field in a space; use %s", orList(spaceFields))
		}
	}
	c.startKinds(s, sn, p)
	c.skillKinds(s, sn, p)
	for _, req := range []string{"members", "scope"} {
		if !sn.set[req] {
			c.add(n, p+"."+req, "required field is missing")
		}
	}
	c.t.Spaces[name] = s
	c.spaceNodes[name] = sn
}

// spaceFields reads a space's fields, kind → sensitive field names.
func (c *checker) spaceFields(v *node, f string, s *Space) {
	if c.mapOf(v, f, "{ hris: [salary_band] }") {
		for j, kk := range v.keys {
			if c.skip[kk] {
				continue
			}
			fk := f + "." + kk.value
			vs, ns, ok := c.list(v.vals[j], fk, "[salary_band]")
			if !ok {
				continue
			}
			for m, name := range vs {
				if !fieldName.MatchString(name) {
					c.add(ns[m], fmt.Sprintf("%s[%d]", fk, m), "%q isn't a field name ([a-z][a-z0-9_]*)", name)
					continue
				}
				s.Fields[kk.value] = append(s.Fields[kk.value], name)
			}
		}
	}
}

// spaceChannels reads a space's chat channels, each in one space only.
func (c *checker) spaceChannels(name string, v *node, f string, s *Space) {
	vs, ns, ok := c.list(v, f, `["#finance-ask"]`)
	if ok {
		for j, ch := range vs {
			fj := fmt.Sprintf("%s[%d]", f, j)
			switch {
			case !channelName.MatchString(ch):
				c.add(ns[j], fj, "%q must be a channel, #name", ch)
			case contains(s.Channels, ch):
				c.add(ns[j], fj, "%q is listed twice", ch)
			default:
				if other, taken := c.channelOwner[ch]; taken {
					c.add(ns[j], fj, "%q already asks in space %q; a channel belongs to one space", ch, other)
					continue
				}
				if c.channelOwner == nil {
					c.channelOwner = map[string]string{}
				}
				c.channelOwner[ch] = name
				s.Channels = append(s.Channels, ch)
			}
		}
	}
}

// groupList is a list of distinct group:NAME values.
func (c *checker) groupList(n *node, field string) []string {
	vs, ns, ok := c.list(n, field, "[group:finance]")
	if !ok {
		return nil
	}
	out := []string{}
	for i, v := range vs {
		f := fmt.Sprintf("%s[%d]", field, i)
		switch {
		case !group.MatchString(v):
			c.add(ns[i], f, "%q must be a group (group:NAME), never a user", v)
		case contains(out, v):
			c.add(ns[i], f, "%q is listed twice", v)
		default:
			out = append(out, v)
		}
	}
	if len(vs) == 0 {
		c.add(n, field, "needs at least one group")
	}
	return out
}

func (c *checker) scope(n *node, field string) map[string][]string {
	if !c.mapOf(n, field, `{ docs: ["/Finance/**"] }`) {
		return nil
	}
	out := map[string][]string{}
	for i, k := range n.keys {
		if c.skip[k] {
			continue
		}
		f := field + "." + k.value
		if !contains(ConnectorKinds, k.value) {
			c.add(k, f, "unknown connector kind; use %s", orList(ConnectorKinds))
			continue
		}
		vs, _, ok := c.list(n.vals[i], f, `["/Finance/**"]`)
		if !ok {
			continue
		}
		if len(vs) == 0 {
			c.add(n.vals[i], f, "needs at least one pattern")
			continue
		}
		out[k.value] = vs
	}
	if len(n.keys) == 0 {
		c.add(n, field, "needs at least one connector kind")
	}
	return out
}

func (c *checker) barriers(n *node) {
	if n.kind != seqNode {
		c.add(n, "barriers", "must be a list of group pairs, e.g. [[group:deal-team, group:everyone]]")
		return
	}
	if len(n.items) > MaxBarriers {
		c.add(n, "barriers", "at most %d barriers", MaxBarriers)
	}
	c.t.Barriers = [][2]string{}
	seen := map[[2]string]int{}
	for i, it := range n.items {
		f := fmt.Sprintf("barriers[%d]", i)
		vs, ns, ok := c.list(it, f, "[group:deal-team, group:everyone]")
		if !ok {
			continue
		}
		if len(vs) != 2 {
			c.add(it, f, "a barrier is exactly two groups")
			continue
		}
		bad := false
		for j, v := range vs {
			if !group.MatchString(v) {
				c.add(ns[j], fmt.Sprintf("%s[%d]", f, j), "%q must be a group (group:NAME)", v)
				bad = true
			}
		}
		if bad {
			continue
		}
		if vs[0] == vs[1] {
			c.add(it, f, "a group can't be walled off from itself")
			continue
		}
		pair := [2]string{vs[0], vs[1]}
		if pair[1] < pair[0] {
			pair[0], pair[1] = pair[1], pair[0]
		}
		if first, dup := seen[pair]; dup {
			c.add(it, f, "the same barrier as barriers[%d]", first)
			continue
		}
		seen[pair] = i
		c.t.Barriers = append(c.t.Barriers, pair)
		c.barrierNodes = append(c.barrierNodes, it)
	}
}

// spaceChecks are sb-space(5)'s rules that relate a space to the rest of the manifest,
// and fill in each space's defaults from its template.
func (c *checker) spaceChecks() {
	names := make([]string, 0, len(c.t.Spaces))
	for name := range c.t.Spaces {
		names = append(names, name)
	}
	sort.Strings(names)
	c.treeChecks(names)
	// Departments before their teams, so a team inherits from a parent already filled in.
	sort.SliceStable(names, func(i, j int) bool { return c.t.Spaces[names[i]].Parent == "" && c.t.Spaces[names[j]].Parent != "" })
	for _, name := range names {
		s, sn, p := c.t.Spaces[name], c.spaceNodes[name], "spaces."+name
		par, hasParent := c.t.Spaces[s.Parent]
		hasParent = hasParent && s.Parent != ""
		if hasParent && !sn.set["template"] {
			s.Template = par.Template // a team under Finance is a Finance team (SB-3902)
		}
		tpl := Templates[s.Template]
		c.spaceLabel(&s, sn, p, tpl, par, hasParent)
		if s.Label == "restricted" && len(s.Stewards) == 0 && c.seen["spaces"] != nil {
			c.add(c.spaceNode(name), p, "a restricted space needs stewards; someone has to own who can read it")
		}
		if sn.scope != nil && sn.scope.kind == mapNode {
			for _, k := range sn.scope.keys {
				if contains(ConnectorKinds, k.value) && c.connectorKindAbsent(k.value) {
					c.add(k, p+".scope."+k.value, "no connector kind %q in connectors", k.value)
				}
			}
		}
		if sn.fields != nil && sn.fields.kind == mapNode {
			for _, k := range sn.fields.keys {
				if _, inScope := s.Scope[k.value]; !inScope {
					c.add(k, p+".fields."+k.value, "kind %q isn't in this space's scope", k.value)
				}
			}
		}
		c.spaceAgents(&s, sn, p, tpl, par, hasParent)
		c.spaceBrief(s, sn, p)
		c.spaceRetention(&s, sn, p, tpl, par, hasParent)
		c.spaceBudget(s, sn, p, par, hasParent)
		c.t.Spaces[name] = s
	}
	c.reachChecks(names)
	for i, pair := range c.t.Barriers {
		for _, name := range names {
			if m := c.t.Spaces[name].Members; contains(m, pair[0]) && contains(m, pair[1]) {
				c.add(c.barrierNodes[i], fmt.Sprintf("barriers[%d]", indexOf(c.seen["barriers"].items, c.barrierNodes[i])),
					"space %q has both %s and %s as members, so it could never answer either", name, pair[0], pair[1])
			}
		}
	}
}

// spaceLabel fills in a space's label, from its template or, for a team, its department,
// and refuses a team looser than its department (SB-3901).
func (c *checker) spaceLabel(s *Space, sn spaceNodes, p string, tpl Template, par Space, hasParent bool) {
	if s.Label == "" {
		s.Label = tpl.Label
		if hasParent && !sn.set["template"] {
			s.Label = par.Label // the department's label, not its template's default
		} else if hasParent && labelRank(par.Label) > labelRank(s.Label) {
			s.Label = par.Label
		}
	} else if hasParent && labelRank(s.Label) < labelRank(par.Label) {
		c.add(sn.label, p+".label", "%s is looser than its parent %s's %s; a team can be stricter than its department, never looser (LAW-24)", s.Label, s.Parent, par.Label)
	}
}

// spaceAgents checks a space's agents, or fills them in from its template or, for a
// team, its department's that its template allows (SB-3902).
func (c *checker) spaceAgents(s *Space, sn spaceNodes, p string, tpl Template, par Space, hasParent bool) {
	if sn.agents != nil {
		if c.seen["agents"] != nil && c.seen["agents"].kind == seqNode {
			for i, a := range s.Agents {
				switch {
				case !contains(c.t.Agents, a):
					c.add(sn.agents.items[i], fmt.Sprintf("%s.agents[%d]", p, i), "%q is not in the manifest's agents", a)
				case a != "ask" && a != "draft-reply" && !contains(tpl.Agents, a):
					c.add(sn.agents.items[i], fmt.Sprintf("%s.agents[%d]", p, i), "%q is another department's agent; it runs only in spaces of its own template (sb-agents(7))", a)
				}
			}
		}
	} else {
		from := tpl.Agents
		if hasParent {
			from = par.Agents // settings flow down (SB-3902)
		}
		s.Agents = []string{}
		for _, a := range from {
			if contains(c.t.Agents, a) && (a == "ask" || a == "draft-reply" || contains(tpl.Agents, a)) {
				s.Agents = append(s.Agents, a)
			}
		}
	}
}

// spaceRetention checks a space's retention against the tenant's and, for a team, its
// department's, or fills it in as the least of them (SB-3901).
func (c *checker) spaceRetention(s *Space, sn spaceNodes, p string, tpl Template, par Space, hasParent bool) {
	if sn.retention != nil {
		switch {
		case s.RetentionDays > c.t.RetentionDays && c.t.RetentionDays > 0:
			c.add(sn.retention, p+".retention_days", "%d is above the tenant's retention_days (%d); a space can keep less, never more", s.RetentionDays, c.t.RetentionDays)
		case hasParent && par.RetentionDays > 0 && s.RetentionDays > par.RetentionDays:
			c.add(sn.retention, p+".retention_days", "%d is above its parent %s's retention_days (%d); a team can keep less, never more (LAW-24)", s.RetentionDays, s.Parent, par.RetentionDays)
		}
	} else {
		s.RetentionDays = c.t.RetentionDays
		if tpl.RetentionDays > 0 && tpl.RetentionDays < s.RetentionDays {
			s.RetentionDays = tpl.RetentionDays
		}
		if hasParent && par.RetentionDays > 0 && par.RetentionDays < s.RetentionDays {
			s.RetentionDays = par.RetentionDays
		}
	}
}

func (c *checker) spaceNode(name string) *node {
	n := c.seen["spaces"]
	for _, k := range n.keys {
		if k.value == name {
			return k
		}
	}
	return n
}

// SpaceAttrs renders a space for the plan, in sb-tenant(1)'s order.
func SpaceAttrs(s Space) map[string]string {
	scope := make([]string, 0, len(s.Scope))
	for _, k := range sorted(s.Scope) {
		scope = append(scope, k+" "+strings.Join(s.Scope[k], ","))
	}
	a := map[string]string{
		"template": s.Template, "label": s.Label, "members": strings.Join(s.Members, ","), "stewards": strings.Join(s.Stewards, ","),
		"agents": strings.Join(s.Agents, ","), "retention_days": strconv.Itoa(s.RetentionDays), "scope": strings.Join(scope, "; "),
	}
	if len(s.Channels) > 0 {
		a["channels"] = strings.Join(s.Channels, ",")
	}
	if len(s.StartHere) > 0 {
		a["start_here"] = strings.Join(s.StartHere, ",")
	}
	if len(s.Glossary) > 0 {
		var gs []string
		for _, k := range sorted(s.Glossary) {
			gs = append(gs, k+": "+s.Glossary[k])
		}
		a["glossary"] = strings.Join(gs, "; ")
	}
	if s.Parent != "" {
		a["parent"] = s.Parent
	}
	if s.ReviewAfterDays != 0 && s.ReviewAfterDays != DefaultReviewAfterDays {
		a["review_after_days"] = strconv.Itoa(s.ReviewAfterDays)
	}
	if len(s.Fields) > 0 {
		var fs []string
		for _, k := range sorted(s.Fields) {
			fs = append(fs, k+" "+strings.Join(s.Fields[k], ","))
		}
		a["fields"] = strings.Join(fs, "; ")
	}
	return a
}

// reachChecks refuses a scope that reaches into a restricted space's scope for people who
// aren't that space's members (sb-space(5) rule 13; found by the red team, PT-0003): a
// space holding a restricted space's documents makes its own members holders of them.
// "Reaches" is pattern containment either way: one pattern matches the other as text.
func (c *checker) reachChecks(names []string) {
	for _, a := range names {
		sa := c.t.Spaces[a]
		for _, b := range names {
			sb := c.t.Spaces[b]
			if a == b || sb.Label != "restricted" || subset(sa.Members, sb.Members) {
				continue
			}
			for kind, pats := range sa.Scope {
				for _, pa := range pats {
					for _, pb := range sb.Scope[kind] {
						if route.Match(pa, pb) || route.Match(pb, pa) {
							c.add(c.spaceNode(a), "spaces."+a+".scope."+kind,
								"%q reaches restricted space %s's %q, so %s's members who aren't %s's would read it; narrow the pattern or make the members match",
								pa, b, pb, a, b)
						}
					}
				}
			}
		}
	}
}

func subset(a, b []string) bool {
	for _, x := range a {
		if !contains(b, x) {
			return false
		}
	}
	return true
}

// brief reads a space's brief, { agent: NAME, weekly: DAY } (SB-4005).
func (c *checker) brief(v *node, f string) *Brief {
	if !c.mapOf(v, f, `{ agent: incident-summary, weekly: monday }`) {
		return nil
	}
	b := &Brief{}
	for j, k := range v.keys {
		fk := f + "." + k.value
		val, ok := c.scalar(v.vals[j], fk)
		switch {
		case !ok:
		case k.value == "agent":
			b.Agent = val
		case k.value == "weekly":
			if !contains(Weekdays, val) {
				c.add(v.vals[j], fk, "%q is not a day; use %s", val, orList(Weekdays))
			}
			b.Weekly = val
		default:
			c.add(k, fk, "unknown field in a brief; use agent or weekly")
		}
	}
	if b.Agent == "" || b.Weekly == "" {
		c.add(v, f, "a brief needs an agent and a weekly day")
		return nil
	}
	return b
}

// spaceBrief checks a brief against the space's agents and stewards, once both are known.
func (c *checker) spaceBrief(s Space, sn spaceNodes, p string) {
	switch b := s.Brief; {
	case b == nil:
	case b.Agent == "draft-reply":
		c.add(sn.brief, p+".brief.agent", "draft-reply drafts replies to a ticket; a brief runs an agent that answers")
	case b.Agent == "voc" && s.Template != "product" && s.Template != "support":
		c.add(sn.brief, p+".brief.agent", "voc runs only in a product or support space (SB-4205)")
	case b.Agent != "ask" && b.Agent != "voc" && !contains(s.Agents, b.Agent):
		c.add(sn.brief, p+".brief.agent", "%q isn't one of this space's agents", b.Agent)
	case len(s.Stewards) == 0:
		c.add(sn.brief, p+".brief", "a brief needs stewards; its draft waits for them")
	}
	if s.Updates && len(s.Stewards) == 0 {
		c.add(sn.updates, p+".updates", "updates need stewards; each draft waits for them")
	}
}

// requests reads a space's requests, { categories: [Access, Laptop], due_days: 5 } (SB-4104).
// incidentLabels checks incident_labels: 1 to MaxIncidentLabels labels, none twice.
func (c *checker) incidentLabels(v *node, f string) []string {
	vs, ns, ok := c.list(v, f, "[sev-1, sev-2]")
	if !ok {
		return nil
	}
	if len(vs) == 0 || len(vs) > MaxIncidentLabels {
		c.add(v, f, "1 to %d labels", MaxIncidentLabels)
	}
	out := []string{}
	for m, l := range vs {
		fm := fmt.Sprintf("%s[%d]", f, m)
		switch {
		case !incidentLabelRe.MatchString(l):
			c.add(ns[m], fm, "%q must be 1 to 40 letters, digits, spaces or _ . : / -", l)
		case containsFold(out, l):
			c.add(ns[m], fm, "%q is listed twice", l)
		default:
			out = append(out, strings.ToLower(l))
		}
	}
	return out
}

func (c *checker) requests(v *node, f string) *Requests {
	if !c.mapOf(v, f, `{ categories: [Access, Laptop], due_days: 5 }`) {
		return nil
	}
	r := &Requests{DueDays: DefaultDueDays}
	for j, k := range v.keys {
		fk := f + "." + k.value
		switch k.value {
		case "categories":
			vs, ns, ok := c.list(v.vals[j], fk, "[Access, Laptop]")
			if !ok {
				continue
			}
			if len(vs) > MaxRequestCategories {
				c.add(v.vals[j], fk, "at most %d categories", MaxRequestCategories)
			}
			for m, cat := range vs {
				fm := fmt.Sprintf("%s[%d]", fk, m)
				switch {
				case !categoryRe.MatchString(cat):
					c.add(ns[m], fm, "%q must be 1 to %d letters, digits, spaces or & / : ' -", cat, MaxCategory)
				case containsFold(r.Categories, cat):
					c.add(ns[m], fm, "%q is listed twice", cat)
				default:
					r.Categories = append(r.Categories, cat)
				}
			}
		case "due_days":
			r.DueDays = c.intIn(v.vals[j], fk, 1, MaxDueDays)
		default:
			c.add(k, fk, "unknown field in requests; use categories or due_days")
		}
	}
	if len(r.Categories) == 0 {
		c.add(v, f, "requests need at least one category")
		return nil
	}
	return r
}

func containsFold(xs []string, s string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

// spaceBudget checks a space's budget against the tenant's and its department's: a team
// spends at most what its department may (SB-4105).
func (c *checker) spaceBudget(s Space, sn spaceNodes, p string, par Space, hasParent bool) {
	switch {
	case sn.budget == nil || s.BudgetUSDMonth == 0:
	case c.t.BudgetUSDMonth > 0 && s.BudgetUSDMonth > c.t.BudgetUSDMonth:
		c.add(sn.budget, p+".budget_usd_month", "%d is above the tenant's budget_usd_month (%d)", s.BudgetUSDMonth, c.t.BudgetUSDMonth)
	case hasParent && par.BudgetUSDMonth > 0 && s.BudgetUSDMonth > par.BudgetUSDMonth:
		c.add(sn.budget, p+".budget_usd_month", "%d is above its parent %s's budget_usd_month (%d)", s.BudgetUSDMonth, s.Parent, par.BudgetUSDMonth)
	}
}

// glossary reads a space's glossary, TERM: MEANING (SB-2203).
func (c *checker) glossary(v *node, f string) map[string]string {
	if !c.mapOf(v, f, `{ "PO": "purchase order" }`) {
		return nil
	}
	if len(v.keys) > MaxGlossary {
		c.add(v, f, "at most %d terms", MaxGlossary)
	}
	out := map[string]string{}
	seen := map[string]bool{}
	for j, k := range v.keys {
		if c.skip[k] {
			continue
		}
		fk := f + "." + k.value
		term, ok := c.scalar(k, fk)
		if !ok {
			continue
		}
		means, ok := c.scalar(v.vals[j], fk)
		switch {
		case !ok:
		case utf8.RuneCountInString(term) > MaxTerm || strings.TrimSpace(term) != term:
			c.add(k, fk, "a term is 1 to %d characters, with no space around it", MaxTerm)
		case utf8.RuneCountInString(means) > MaxMeaning:
			c.add(v.vals[j], fk, "a meaning is at most %d characters", MaxMeaning)
		case !printable(term) || !printable(means):
			c.add(k, fk, "a term and its meaning are printable characters only")
		case seen[strings.ToLower(term)]:
			c.add(k, fk, "%q is listed twice (ignoring case)", term)
		default:
			seen[strings.ToLower(term)] = true
			out[term] = means
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func printable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// startHere reads a space's start_here list (SB-1706).
func (c *checker) startHere(v *node, f string, s *Space, sn *spaceNodes) {
	vs, ns, ok := c.list(v, f, `["docs:onboarding/welcome.md", "docs:/Support/Start/"]`)
	if !ok {
		return
	}
	if len(vs) > MaxStartHere {
		c.add(v, f, "at most %d items", MaxStartHere)
	}
	for j, it := range vs {
		fj := fmt.Sprintf("%s[%d]", f, j)
		m := startItem.FindStringSubmatch(it)
		switch {
		case m == nil:
			c.add(ns[j], fj, "%q must be a document KIND:ID or a folder KIND:/Folder/", it)
		case contains(s.StartHere, it):
			c.add(ns[j], fj, "%q is listed twice", it)
		default:
			s.StartHere = append(s.StartHere, it)
			sn.start = append(sn.start, ns[j])
		}
	}
}

// startKinds checks each start_here item's kind is in the space's scope.
func (c *checker) startKinds(s Space, sn spaceNodes, p string) {
	for j, it := range s.StartHere {
		if kind, _, _ := strings.Cut(it, ":"); s.Scope != nil && s.Scope[kind] == nil {
			c.add(sn.start[j], fmt.Sprintf("%s.start_here[%d]", p, j), "kind %q isn't in this space's scope", kind)
		}
	}
}

// labelRank is a label's place in Labels, least sensitive first; -1 for none.
func labelRank(l string) int {
	for i, x := range Labels {
		if x == l {
			return i
		}
	}
	return -1
}

// treeChecks are the rules of the tree (sb-space(5) rule 17; SB-3901): a parent is
// another space in the manifest, and a parent has no parent of its own, so the tree is
// at most company, department, team and can hold no cycle.
func (c *checker) treeChecks(names []string) {
	for _, name := range names {
		s, sn := c.t.Spaces[name], c.spaceNodes[name]
		if s.Parent == "" {
			continue
		}
		f := "spaces." + name + ".parent"
		par, ok := c.t.Spaces[s.Parent]
		switch {
		case s.Parent == name:
			c.add(sn.parent, f, "a space can't be its own parent")
		case !ok:
			c.add(sn.parent, f, "%q is not a space in this manifest", s.Parent)
		case par.Parent != "":
			c.add(sn.parent, f, "%s is itself under %s; the tree is at most company, department and team", s.Parent, par.Parent)
		default:
			continue
		}
		s.Parent = "" // a bad parent is reported once and not inherited from
		c.t.Spaces[name] = s
	}
}

// Depth is a space's level in the tree (SB-3901): 1 for the company (""), 2 for a space
// with no parent, 3 for a team under one.
func (t Tenant) Depth(name string) int {
	switch s, ok := t.Spaces[name]; {
	case name == "" || name == "company" || !ok:
		return 1
	case s.Parent == "":
		return 2
	default:
		return 3
	}
}

// Children are the spaces whose parent is name, sorted (SB-3901).
func (t Tenant) Children(name string) []string {
	var out []string
	for n, s := range t.Spaces {
		if s.Parent == name && name != "" {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// Branch is name and its children, the space first (SB-3903).
func (t Tenant) Branch(name string) []string {
	if _, ok := t.Spaces[name]; !ok {
		return nil
	}
	return append([]string{name}, t.Children(name)...)
}

// Up is the chain from name to its department, name first (SB-3902, SB-3904).
func (t Tenant) Up(name string) []string {
	s, ok := t.Spaces[name]
	if !ok {
		return nil
	}
	if s.Parent == "" {
		return []string{name}
	}
	return []string{name, s.Parent}
}

// Effective is a space with what flows down from its parent filled in (SB-3902): a team
// with no glossary, start-here list or review interval of its own takes its department's.
// Glossaries and skills merge, the team's own winning; start-here lists join, the department's
// first, at most MaxStartHere. Label, retention and agents are settled by Check already.
func (t Tenant) Effective(name string) Space {
	s := t.Spaces[name]
	par, ok := t.Spaces[s.Parent]
	if s.Parent == "" || !ok {
		return s
	}
	if len(par.Glossary) > 0 {
		g := map[string]string{}
		own := map[string]bool{}
		for k := range s.Glossary {
			own[strings.ToLower(k)] = true
		}
		for k, v := range par.Glossary {
			if !own[strings.ToLower(k)] {
				g[k] = v
			}
		}
		for k, v := range s.Glossary {
			g[k] = v
		}
		s.Glossary = g
	}
	if len(par.Skills) > 0 { // SB-4601: the team's own skill of a name wins
		sk := map[string]Skill{}
		for k, v := range par.Skills {
			sk[k] = v
		}
		for k, v := range s.Skills {
			sk[k] = v
		}
		s.Skills = sk
	}
	if len(par.Rhythms) > 0 { // SB-4801: the team's own rhythm of a name wins
		rs := map[string]Rhythm{}
		for k, v := range par.Rhythms {
			rs[k] = v
		}
		for k, v := range s.Rhythms {
			rs[k] = v
		}
		s.Rhythms = rs
	}
	if s.Onboarding == nil {
		s.Onboarding = par.Onboarding
	}
	if len(par.StartHere) > 0 {
		joined := append([]string{}, par.StartHere...)
		for _, it := range s.StartHere {
			if !contains(joined, it) {
				joined = append(joined, it)
			}
		}
		if len(joined) > MaxStartHere {
			joined = joined[:MaxStartHere]
		}
		s.StartHere = joined
	}
	if s.ReviewAfterDays == 0 {
		s.ReviewAfterDays = par.ReviewAfterDays
	}
	return s
}

// boolOf is a true or false field, nil when it is neither.
func (c *checker) boolOf(n *node, field string) *bool {
	b, ok := c.scalar(n, field)
	if ok && b != "true" && b != "false" {
		c.add(n, field, "must be true or false")
	}
	if !ok || (b != "true" && b != "false") {
		return nil
	}
	on := b == "true"
	return &on
}

// LeaderWeek says whether space name gets the leader's week (SB-4503): it has teams and
// stewards, and doesn't set leader_week: false.
func (t Tenant) LeaderWeek(name string) bool {
	sp, ok := t.Spaces[name]
	return ok && len(sp.Stewards) > 0 && len(t.Children(name)) > 0 && (sp.LeaderWeek == nil || *sp.LeaderWeek)
}
