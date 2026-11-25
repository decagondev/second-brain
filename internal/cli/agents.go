package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/workflow"
	"example.com/second-brain/internal/errs"
)

// agentsLint is sb agents lint [--root DIR] (sb-agents(7), Lint; SB-4403): every agent
// definition checked against the bundle in a source tree, each failure with its line.
func (e *env) agentsLint(args []string) int {
	const cmd = "sb agents lint"
	root, man := ".", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root", "--manifest":
			if i+1 >= len(args) {
				return e.fail(cmd, errs.New(errs.EUSAGE, "%s needs a value", args[i]))
			}
			if args[i] == "--root" {
				root = args[i+1]
			} else {
				man = args[i+1]
			}
			i++
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb agents lint [--root DIR] [--manifest FILE]"))
		}
	}
	defs, err := os.ReadFile(filepath.Join(root, "internal", "core", "ask", "agents.json"))
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "no agent definitions under %s: %v", root, err))
	}
	probs := ask.LintAgents(SourceBundle(root, defs))
	if man != "" { // SB-4601: the manifest's skills, with their lines
		src, err := os.ReadFile(man)
		if err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s: %v", man, err))
		}
		probs = append(probs, SkillProblems(man, src)...)
	}
	if e.g.JSON {
		ps := []string{}
		for _, p := range probs {
			ps = append(ps, p.String())
		}
		e.json(map[string]any{"problems": ps})
	} else {
		for _, p := range probs {
			fmt.Fprintln(e.out, p.String())
		}
	}
	if len(probs) > 0 {
		fmt.Fprintf(e.err, "sb agents lint: %d problem(s)\n", len(probs))
		return 1
	}
	if !e.g.JSON && !e.g.Quiet {
		fmt.Fprintln(e.out, "agents lint: every agent passes")
	}
	return 0
}

// SkillProblems are a tenant manifest's problems with its spaces' skills (SB-4601), each
// at its line: the manifest check's own rules for skills.
func SkillProblems(file string, src []byte) []ask.LintProblem {
	var out []ask.LintProblem
	_, ds := manifest.Check(src)
	for _, d := range ds {
		if d.Field == "syntax" || strings.Contains(d.Field, ".skills") {
			out = append(out, ask.LintProblem{File: file, Line: d.Line, Rule: "skill", Msg: d.Field + ": " + d.Msg})
		}
	}
	return out
}

// SourceBundle is the lint bundle of a source tree: its prompts and suites, the tools the
// tool bus declares, the template catalogue and every agent the code runs.
func SourceBundle(root string, defs []byte) ask.LintBundle {
	tools := map[string]bool{ask.NoteTool: true}
	for _, t := range workflow.Templates {
		tools[t.Tool] = true
	}
	tpls := map[string]string{}
	for n, t := range manifest.Templates {
		tpls[n] = t.Label
	}
	agents := append([]string{ask.AgentHandover, ask.AgentUpdate}, manifest.Agents...)
	agents = append(agents, ask.AgentResearch, ask.AgentPrep, ask.AgentIncident, ask.AgentVoC, ask.AgentFix, ask.AgentCheckin,
		ask.AgentCatchUp, ask.AgentLeaderWeek)
	return ask.LintBundle{Defs: defs, Tools: tools, Templates: tpls, Agents: agents,
		Prompt: func(name string) ([]byte, bool) {
			b, err := os.ReadFile(filepath.Join(root, "internal", "core", "prompts", "set", name+".md"))
			return b, err == nil
		},
		Suite: func(name string) bool {
			_, err := os.Stat(filepath.Join(root, "evals", "suites", name+".yaml"))
			return err == nil
		}}
}

// releaseNotes is sb release notes --from DIR --to DIR (sb-release(1); SB-4402): for each
// agent, its prompt's lines added and removed and its suite's result, before and after.
func (e *env) releaseNotes(args []string) int {
	const cmd = "sb release notes"
	var from, to string
	for i := 0; i+1 < len(args); i += 2 {
		switch args[i] {
		case "--from":
			from = args[i+1]
		case "--to":
			to = args[i+1]
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb release notes --from DIR --to DIR"))
		}
	}
	if from == "" || to == "" || len(args)%2 != 0 {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb release notes --from DIR --to DIR"))
	}
	notes, err := AgentNotes(from, to)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	if e.g.JSON {
		e.json(map[string]any{"agents": notes})
		return 0
	}
	for _, n := range notes {
		fmt.Fprintf(e.out, "%s (owner %s)%s\n", n.Name, n.Owner, map[string]string{"": "", "added": ": added", "removed": ": removed"}[n.Change])
		for _, l := range n.Prompt {
			fmt.Fprintf(e.out, "  %s\n", l)
		}
		fmt.Fprintf(e.out, "  eval %s: %s -> %s\n", n.Suite, n.Before, n.After)
	}
	return 0
}

// AgentNote is one agent's change between two releases.
type AgentNote struct {
	Name   string   `json:"name"`
	Owner  string   `json:"owner"`
	Change string   `json:"change,omitempty"` // added or removed
	Prompt []string `json:"prompt"`           // "+ line" and "- line"
	Suite  string   `json:"suite"`
	Before string   `json:"before"`
	After  string   `json:"after"`
}

// AgentNotes compares two agent bundles (agents.json, prompts/, evals/).
func AgentNotes(from, to string) ([]AgentNote, error) {
	load := func(dir string) (map[string]ask.AgentDef, error) {
		b, err := os.ReadFile(filepath.Join(dir, "agents.json"))
		if err != nil {
			return nil, err
		}
		defs, err := ask.ParseAgentDefs(b)
		if err != nil {
			return nil, err
		}
		out := map[string]ask.AgentDef{}
		for _, d := range defs {
			out[d.Name] = d
		}
		return out, nil
	}
	a, err := load(from)
	if err != nil {
		return nil, err
	}
	b, err := load(to)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for n := range a {
		names[n] = true
	}
	for n := range b {
		names[n] = true
	}
	var sorted []string
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	var out []AgentNote
	for _, n := range sorted {
		old, inOld := a[n]
		cur, inNew := b[n]
		note := AgentNote{Name: n, Owner: cur.Owner, Suite: cur.Suite, Prompt: []string{}}
		switch {
		case !inNew:
			note.Change, note.Owner, note.Suite = "removed", old.Owner, old.Suite
		case !inOld:
			note.Change = "added"
		}
		note.Prompt = lineDiff(promptLines(from, old.Prompt, inOld), promptLines(to, cur.Prompt, inNew))
		note.Before, note.After = suiteResult(from, note.Suite), suiteResult(to, note.Suite)
		out = append(out, note)
	}
	return out, nil
}

func promptLines(dir, name string, ok bool) []string {
	if !ok || name == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(dir, "prompts", name+".md"))
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func suiteResult(dir, suite string) string {
	b, err := os.ReadFile(filepath.Join(dir, "evals", suite+".json"))
	if err != nil {
		return "no results"
	}
	var r struct {
		PassRate float64 `json:"pass_rate"`
		Failures int     `json:"failures"`
	}
	if json.Unmarshal(b, &r) != nil {
		return "no results"
	}
	return fmt.Sprintf("%.0f%% passed, %d failures", r.PassRate*100, r.Failures)
}

// lineDiff is ask.LineDiff (SB-4402).
func lineDiff(a, b []string) []string { return ask.LineDiff(a, b) }
