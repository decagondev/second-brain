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
	root := "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				return e.fail(cmd, errs.New(errs.EUSAGE, "--root needs a directory"))
			}
			root = args[i+1]
			i++
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb agents lint [--root DIR]"))
		}
	}
	defs, err := os.ReadFile(filepath.Join(root, "internal", "core", "ask", "agents.json"))
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "no agent definitions under %s: %v", root, err))
	}
	probs := ask.LintAgents(SourceBundle(root, defs))
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
	agents = append(agents, ask.AgentResearch, ask.AgentPrep, ask.AgentIncident, ask.AgentVoC, ask.AgentFix, ask.AgentCheckin)
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

// lineDiff is the lines removed ("- ") and added ("+ ") from a to b, by their longest
// common subsequence, in order.
func lineDiff(a, b []string) []string {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	out := []string{}
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			i, j = i+1, j+1
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			out = append(out, "+ "+b[j])
			j++
		default:
			out = append(out, "- "+a[i])
			i++
		}
	}
	return out
}
