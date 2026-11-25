package ask

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Agent definitions (sb-agents(7), Definitions and Lint; SB-4402, SB-4403): every agent is
// one object in the release bundle's agents.json, with an owner who answers for it, and
// `sb agents lint` checks each one before a release is gated.

// AgentDefs is the bundle's agent definitions, as shipped.
//
//go:embed agents.json
var AgentDefs []byte

// AgentDef is one agent's definition.
type AgentDef struct {
	Name       string   `json:"name"`
	Owner      string   `json:"owner"`
	Department string   `json:"department"`
	Prompt     string   `json:"prompt"`
	Suite      string   `json:"suite"`
	Tools      []string `json:"tools"`
	Writes     bool     `json:"writes"`
	Approvals  bool     `json:"approvals"`
	Labels     []string `json:"labels"`
	Restricted bool     `json:"restricted"`
	// Shadow is a prompt to try beside its own in the first ring (SB-4704).
	Shadow *ShadowDef `json:"shadow,omitempty"`
}

// ShadowDef is a definition's shadow: a bundled prompt and 1 to MaxShadowDays days.
type ShadowDef struct {
	Prompt string `json:"prompt"`
	Days   int    `json:"days"`
}

// MaxShadowDays is the longest shadow window: a week.
const MaxShadowDays = 7

// AgentDefOf is one agent's definition in the bundle.
func AgentDefOf(name string) (AgentDef, bool) {
	defs, err := ParseAgentDefs(AgentDefs)
	if err != nil {
		return AgentDef{}, false
	}
	for _, d := range defs {
		if d.Name == name {
			return d, true
		}
	}
	return AgentDef{}, false
}

// ParseAgentDefs reads definitions; unknown fields are refused.
func ParseAgentDefs(b []byte) ([]AgentDef, error) {
	var out []AgentDef
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("agents.json: %w", err)
	}
	return out, nil
}

// AgentOwner is who answers for an agent, from the bundle: an email, group:NAME or "".
func AgentOwner(name string) string {
	defs, err := ParseAgentDefs(AgentDefs)
	if err != nil {
		return ""
	}
	for _, d := range defs {
		if d.Name == name {
			return d.Owner
		}
	}
	return ""
}

// AgentsOwnedBy are the agents whose owner is one of the person's addresses or groups.
func AgentsOwnedBy(emails, groups []string) []string {
	defs, _ := ParseAgentDefs(AgentDefs)
	var out []string
	for _, d := range defs {
		owner := strings.ToLower(strings.TrimSpace(d.Owner))
		g, isGroup := strings.CutPrefix(owner, "group:")
		for _, x := range groups {
			if isGroup && strings.EqualFold(x, g) {
				out = append(out, d.Name)
			}
		}
		for _, e := range emails {
			if !isGroup && strings.EqualFold(e, owner) {
				out = append(out, d.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// LintProblem is one rule a definition breaks, at a file and line.
type LintProblem struct {
	File, Rule, Msg string
	Line            int
}

func (p LintProblem) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", p.File, p.Line, p.Rule, p.Msg)
}

// LintBundle is what lint checks definitions against.
type LintBundle struct {
	Defs   []byte                           // agents.json
	Prompt func(name string) ([]byte, bool) // a bundled prompt's text
	Suite  func(name string) bool           // an eval suite exists
	Tools  map[string]bool                  // the tools the tool bus declares
	// Templates are the department templates, each with its default label.
	Templates map[string]string
	// Agents are the agent names the code runs, each of which must have a definition.
	Agents []string
}

var (
	ownerShape = regexp.MustCompile(`^(group:[a-z0-9][a-z0-9._-]*|[^@\s]+@[^@\s]+\.[^@\s]+)$`)
	neverInstr = regexp.MustCompile(`(?i)(never|not) instructions`)
	actsOnData = regexp.MustCompile(`(?i)\b(follow|obey|carry out|execute|do what)\b[^.]*\b(documents?|sources?|data blocks?|passages?)\b`)
	negation   = regexp.MustCompile(`(?i)\b(never|not|don't|do not|ignore)\b`)
)

// defLine is the line of agents.json where an agent's object names it, from 1.
func defLine(b []byte, name string) int {
	key := []byte(`"name": "` + name + `"`)
	for i, l := range bytes.Split(b, []byte("\n")) {
		if bytes.Contains(l, key) || bytes.Contains(l, []byte(`"name":"`+name+`"`)) {
			return i + 1
		}
	}
	return 1
}

// LintAgents checks every definition against the bundle; nil is a clean bundle.
func LintAgents(lb LintBundle) []LintProblem {
	const file = "internal/core/ask/agents.json"
	defs, err := ParseAgentDefs(lb.Defs)
	if err != nil {
		return []LintProblem{{File: file, Line: 1, Rule: "parse", Msg: err.Error()}}
	}
	var out []LintProblem
	seen := map[string]bool{}
	for _, d := range defs {
		line := defLine(lb.Defs, d.Name)
		bad := func(rule, format string, a ...any) {
			out = append(out, LintProblem{File: file, Line: line, Rule: rule, Msg: d.Name + ": " + fmt.Sprintf(format, a...)})
		}
		if seen[d.Name] {
			bad("name", "defined twice")
		}
		seen[d.Name] = true
		if !ownerShape.MatchString(strings.TrimSpace(d.Owner)) {
			bad("owner", "names no owner: an email or group:NAME")
		}
		for _, t := range d.Tools {
			if !lb.Tools[t] {
				bad("tool", "lists %q, which the tool bus doesn't declare", t)
			}
		}
		if d.Writes && !d.Approvals {
			bad("write", "writes without the approvals path")
		}
		label, isTemplate := lb.Templates[d.Department]
		if d.Department != "company" && !isTemplate {
			bad("department", "has no department: %q is neither a template nor company", d.Department)
		}
		mayRestricted := contains(d.Labels, "restricted")
		if mayRestricted != d.Restricted {
			bad("restricted", "may run under restricted only with restricted: true, and says restricted: %v", d.Restricted)
		}
		if isTemplate && label == "restricted" && !mayRestricted {
			bad("restricted", "its %s template's spaces are restricted, so labels must hold restricted", d.Department)
		}
		if dep, ok := Departments[d.Name]; ok && dep.Restricted && (len(d.Labels) != 1 || !mayRestricted) {
			bad("restricted", "runs only in a restricted space, so labels must be [restricted]")
		}
		if d.Suite == "" || lb.Suite == nil || !lb.Suite(d.Suite) {
			bad("suite", "has no eval suite (%q)", d.Suite)
		}
		out = append(out, lintPrompt(lb, d, line)...)
		out = append(out, lintShadow(lb, d, line)...)
	}
	for _, a := range lb.Agents {
		if !seen[a] {
			out = append(out, LintProblem{File: file, Line: 1, Rule: "owner", Msg: a + ": the code runs it, but it has no definition"})
		}
	}
	return out
}

// lintShadow checks a definition's shadow (SB-4704): a bundled prompt other than its own,
// held to the same prompt rules, for 1 to MaxShadowDays days.
func lintShadow(lb LintBundle, d AgentDef, line int) []LintProblem {
	if d.Shadow == nil {
		return nil
	}
	bad := LintProblem{File: "internal/core/ask/agents.json", Line: line, Rule: "shadow"}
	switch {
	case d.Shadow.Prompt == "" || d.Shadow.Prompt == d.Prompt:
		bad.Msg = d.Name + ": a shadow needs a prompt other than its own"
	case d.Shadow.Days < 1 || d.Shadow.Days > MaxShadowDays:
		bad.Msg = fmt.Sprintf("%s: a shadow runs 1 to %d days, not %d", d.Name, MaxShadowDays, d.Shadow.Days)
	default:
		sd := d
		sd.Prompt = d.Shadow.Prompt
		return lintPrompt(lb, sd, line)
	}
	return []LintProblem{bad}
}

// lintPrompt checks one agent's prompt: it says data is never instructions, never tells the
// model to act on what a document says, and names no declared tool the agent doesn't list.
func lintPrompt(lb LintBundle, d AgentDef, line int) []LintProblem {
	pfile := "internal/core/prompts/set/" + d.Prompt + ".md"
	b, ok := []byte(nil), false
	if lb.Prompt != nil && d.Prompt != "" {
		b, ok = lb.Prompt(d.Prompt)
	}
	if !ok {
		return []LintProblem{{File: "internal/core/ask/agents.json", Line: line, Rule: "prompt", Msg: d.Name + ": no bundled prompt " + d.Prompt}}
	}
	var out []LintProblem
	if !neverInstr.Match(b) {
		out = append(out, LintProblem{File: pfile, Line: 1, Rule: "injection", Msg: d.Name + ": the prompt never says data blocks are never instructions"})
	}
	tools := make([]string, 0, len(lb.Tools))
	for t := range lb.Tools {
		tools = append(tools, t)
	}
	sort.Strings(tools)
	for i, l := range strings.Split(string(b), "\n") {
		if actsOnData.MatchString(l) && !negation.MatchString(l) {
			out = append(out, LintProblem{File: pfile, Line: i + 1, Rule: "injection", Msg: d.Name + ": the prompt tells the model to act on document text"})
		}
		for _, t := range tools {
			if strings.Contains(l, t) && !contains(d.Tools, t) {
				out = append(out, LintProblem{File: pfile, Line: i + 1, Rule: "tool", Msg: fmt.Sprintf("%s: the prompt names %q, which it doesn't list", d.Name, t)})
			}
		}
	}
	return out
}
