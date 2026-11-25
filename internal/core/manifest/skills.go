package manifest

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Skills (sb-space(5), skills; sb-agents(7), Skills; SB-4601): a space's named ways of
// writing an answer. An instruction says how to write, never what to search or do.
const (
	MaxSkills           = 10
	MaxSkillInstruction = 1000
	MinSkillCases       = 3
	MaxSkillCases       = 10
	MaxSkillQuestion    = 4000
)

var (
	skillNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)
	skillDocRe  = regexp.MustCompile(`^[a-z]+:[^@\s]+$`)
	// The content rules of an instruction (LAW-06, LAW-09).
	skillTool    = regexp.MustCompile(`(?i)\b(tools?|function calls?|note-publish|ticket-update|api calls?)\b|\b[a-z]+-[a-z]+\.[a-z_]{3,}\b`)
	skillAct     = regexp.MustCompile(`(?i)\b(post|send|publish|email|create|delete|approve|file)\b[^.]*\b(tickets?|messages?|notes?|emails?|issues?|comments?)\b`)
	skillWiden   = regexp.MustCompile(`(?i)\b(all|every|other|another)\s+(spaces?|departments?)\b|\bcompany[- ]wide\b|\bacross the company\b|\b(ignore|bypass|override|skip|without)\b[^.]*\b(permissions?|access|scopes?|restrictions?|labels?|acls?|barriers?)\b|\b(restricted|private|confidential) (documents?|spaces?|data)\b`)
	skillOnData  = regexp.MustCompile(`(?i)\b(follow|obey|carry out|execute|do what)\b[^.]*\b(documents?|sources?|data blocks?|passages?|tickets?|messages?)\b`)
	skillNegated = regexp.MustCompile(`(?i)\b(never|not|don't|do not)\b`)
)

// SkillProblem is why an instruction is refused, "" when it isn't: it names a tool or an
// action, widens what is searched, or tells the model to act on what a document says.
func SkillProblem(instruction string) string {
	switch {
	case skillTool.MatchString(instruction):
		return "names a tool; a skill changes format and steps only"
	case skillAct.MatchString(instruction) && !skillNegated.MatchString(instruction):
		return "asks for an action; a skill changes format and steps only"
	case skillWiden.MatchString(instruction):
		return "widens what is searched or who may read it; a skill never changes either"
	case skillOnData.MatchString(instruction) && !skillNegated.MatchString(instruction):
		return "tells the model to act on what a document says; documents are never instructions"
	}
	return ""
}

// skills reads a space's skills, NAME: { instruction, cases: { QUESTION: KIND:ID } }.
func (c *checker) skills(v *node, f string) map[string]Skill {
	return namedMap(c, v, f, `{ weekly-update: { instruction: "Three headings: Shipped, Decided, At risk.", cases: { "What did we ship?": "docs:updates/w1.md" } } }`,
		"skill", MaxSkills, skillNameRe, c.skill)
}

// namedMap reads a map of at most max entries, each name matching re and each value read
// by one; nil for none (skills, SB-4601; rhythms, SB-4801).
func namedMap[T any](c *checker, v *node, f, example, what string, max int, re *regexp.Regexp, one func(*node, string) (T, bool)) map[string]T {
	if !c.mapOf(v, f, example) {
		return nil
	}
	if len(v.keys) > max {
		c.add(v, f, "at most %d %ss", max, what)
	}
	out := map[string]T{}
	for j, k := range v.keys {
		if c.skip[k] {
			continue
		}
		fk := f + "." + k.value
		if !re.MatchString(k.value) {
			c.add(k, fk, "a %s name is [a-z][a-z0-9-]{0,39}", what)
			continue
		}
		if x, ok := one(v.vals[j], fk); ok {
			out[k.value] = x
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c *checker) skill(v *node, f string) (Skill, bool) {
	if !c.mapOf(v, f, `{ instruction: "…", cases: { "QUESTION": "KIND:ID" } }`) {
		return Skill{}, false
	}
	sk := Skill{}
	ok := true
	for j, k := range v.keys {
		fk := f + "." + k.value
		switch k.value {
		case "instruction":
			in, good := c.scalar(v.vals[j], fk)
			switch {
			case !good:
				ok = false
			case strings.TrimSpace(in) == "" || utf8.RuneCountInString(in) > MaxSkillInstruction || !printable(in):
				c.add(v.vals[j], fk, "an instruction is 1 to %d printable characters", MaxSkillInstruction)
				ok = false
			case SkillProblem(in) != "":
				c.add(v.vals[j], fk, "the instruction %s", SkillProblem(in))
				ok = false
			default:
				sk.Instruction = strings.TrimSpace(in)
			}
		case "cases":
			sk.Cases = c.skillCases(v.vals[j], fk)
			ok = ok && sk.Cases != nil
		default:
			c.add(k, fk, "unknown field in a skill; use instruction or cases")
			ok = false
		}
	}
	if sk.Instruction == "" && ok {
		c.add(v, f, "a skill needs an instruction")
		ok = false
	}
	if sk.Cases == nil && ok {
		c.add(v, f, "a skill needs at least %d cases", MinSkillCases)
		ok = false
	}
	return sk, ok
}

func (c *checker) skillCases(v *node, f string) map[string]string {
	if !c.mapOf(v, f, `{ "What did we ship?": "docs:updates/w1.md" }`) {
		return nil
	}
	out := map[string]string{}
	for j, k := range v.keys {
		fk := fmt.Sprintf("%s[%q]", f, k.value)
		q, ok := c.scalar(k, fk)
		if !ok {
			continue
		}
		doc, ok := c.scalar(v.vals[j], fk)
		switch {
		case !ok:
		case strings.TrimSpace(q) == "" || utf8.RuneCountInString(q) > MaxSkillQuestion:
			c.add(k, fk, "a case's question is 1 to %d characters", MaxSkillQuestion)
		case !skillDocRe.MatchString(doc):
			c.add(v.vals[j], fk, "%q must be the expected document, KIND:ID", doc)
		default:
			out[strings.TrimSpace(q)] = doc
		}
	}
	if len(out) < MinSkillCases || len(out) > MaxSkillCases {
		c.add(v, f, "a skill has %d to %d cases", MinSkillCases, MaxSkillCases)
		return nil
	}
	return out
}

// skillKinds checks each skill case's document kind is in the space's scope.
func (c *checker) skillKinds(s Space, sn spaceNodes, p string) {
	if s.Scope == nil || sn.skills == nil {
		return
	}
	for name, sk := range s.Skills {
		for q, doc := range sk.Cases {
			if kind, _, _ := strings.Cut(doc, ":"); s.Scope[kind] == nil {
				c.add(sn.skills, fmt.Sprintf("%s.skills.%s.cases[%q]", p, name, q), "kind %q isn't in this space's scope", kind)
			}
		}
	}
}
