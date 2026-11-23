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

var spaceFields = []string{"template", "members", "stewards", "label", "scope", "agents", "retention_days", "channels", "fields", "start_here", "review_after_days", "glossary"}

var fieldName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var channelName = regexp.MustCompile(`^#[a-z0-9][a-z0-9_-]{0,79}$`)

type spaceNodes struct {
	members, scope, agents, retention, fields *node
	set                                       map[string]bool
	start                                     []*node
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
			s.Label = c.oneOf(v, f, Labels)
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
		case "channels":
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
		case "start_here":
			c.startHere(v, f, &s, &sn)
		case "glossary":
			s.Glossary = c.glossary(v, f)
		case "review_after_days":
			if s.ReviewAfterDays = c.intIn(v, f, MinRetentionDays, MaxRetentionDays); s.ReviewAfterDays == DefaultReviewAfterDays {
				s.ReviewAfterDays = 0 // one form for the default, so Format reads back the same
			}
		default:
			c.add(k, f, "unknown field in a space; use %s", orList(spaceFields))
		}
	}
	c.startKinds(s, sn, p)
	for _, req := range []string{"members", "scope"} {
		if !sn.set[req] {
			c.add(n, p+"."+req, "required field is missing")
		}
	}
	c.t.Spaces[name] = s
	c.spaceNodes[name] = sn
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
	for _, name := range names {
		s, sn, p := c.t.Spaces[name], c.spaceNodes[name], "spaces."+name
		tpl := Templates[s.Template]
		if s.Label == "" {
			s.Label = tpl.Label
		}
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
			s.Agents = []string{}
			for _, a := range tpl.Agents {
				if contains(c.t.Agents, a) {
					s.Agents = append(s.Agents, a)
				}
			}
		}
		if sn.retention != nil {
			if s.RetentionDays > c.t.RetentionDays && c.t.RetentionDays > 0 {
				c.add(sn.retention, p+".retention_days", "%d is above the tenant's retention_days (%d); a space can keep less, never more", s.RetentionDays, c.t.RetentionDays)
			}
		} else {
			s.RetentionDays = c.t.RetentionDays
			if tpl.RetentionDays > 0 && tpl.RetentionDays < s.RetentionDays {
				s.RetentionDays = tpl.RetentionDays
			}
		}
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
