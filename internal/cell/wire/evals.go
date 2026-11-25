package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/canary"
)

// The tenant's cases from wrong citations (sb-evals(7); SB-2204): the evals job runs them on
// the release the cell runs and writes a hold; the updater reads it.

// HoldPath is where a cell keeps its hold.
func HoldPath(state string) string { return filepath.Join(state, "evals", "hold.json") }

// ReadHold reads the cell's hold; found is false when there is none. A hold that can't be
// read is an error, never no hold.
func ReadHold(state string) (ask.Hold, bool, error) {
	b, err := os.ReadFile(HoldPath(state))
	if errors.Is(err, os.ErrNotExist) {
		return ask.Hold{}, false, nil
	}
	if err != nil {
		return ask.Hold{}, false, err
	}
	var h ask.Hold
	if err := json.Unmarshal(b, &h); err != nil || h.Release == "" || h.Back == "" {
		return ask.Hold{}, false, fmt.Errorf("wire: the eval hold %s can't be read", HoldPath(state))
	}
	return h, true, nil
}

func writeHold(state string, h *ask.Hold) error {
	p := HoldPath(state)
	if h == nil {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(h)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// RunEvals runs the tenant's cases on the release the cell runs, once per release, and
// holds that release if it regresses one (sb-evals(7), Cases from wrong citations).
func (c *Cell) RunEvals(ctx context.Context) (*ask.Hold, error) {
	tenant := c.Config.Tenant
	cases, err := c.Traces.EvalCases(ctx, tenant)
	if err != nil {
		return nil, err
	}
	cases = ask.ActiveCases(cases)                             // a dismissed candidate never runs (SB-4401)
	if cases, err = c.withSkillCases(ctx, cases); err != nil { // SB-4601
		return nil, err
	}
	agent := c.Parts().Agent
	ran, smallFailed := 0, 0
	probe := func(ctx context.Context, ec ask.EvalCase) (bool, error) {
		s, err := c.ScopeAs(ctx, ec.By, ec.Space)
		if err != nil {
			return false, ask.ErrSkipCase // the steward has gone: the case can't run as them
		}
		resp, err := agent.ProbeAs(ctx, s, ec.Question, ec.Agent) // SB-4401: with the agent it came from
		if errors.Is(err, ask.ErrKilled) || errors.Is(err, ask.ErrBudget) {
			return false, ask.ErrSkipCase
		}
		if err != nil {
			return false, err
		}
		return ec.Passes(resp), nil
	}
	run := func(ec ask.EvalCase) (bool, error) {
		ok, err := probe(ask.WithTier(ctx, ask.TierMain), ec)
		if err != nil || !ok || agent.Small == nil {
			return ok, err
		}
		// SB-4405: a case that passes on the main model must pass on the small one too.
		ran++
		small, err := probe(ask.WithTier(ctx, ask.TierSmall), ec)
		if err == nil && !small {
			smallFailed++
		}
		return ok, nil
	}
	checks, err := agent.CaseChecks(ctx, cases)
	if err != nil {
		return nil, err
	}
	c.Metrics.Set("sb_eval_cases_to_check", "The tenant's eval cases whose expected document changed or is gone (sb-evals(7); SB-2904).",
		map[string]string{"tenant": tenant}, float64(len(checks)))
	updated, hold, err := ask.RunCases(cases, c.Release, run, checks)
	if err != nil {
		return nil, err
	}
	if ran > 0 {
		if err := writeSmallHeld(c.Config.State, smallFailed > 0); err != nil {
			return nil, err
		}
	}
	ids := map[string]bool{}
	for i, u := range updated {
		ids[u.ID] = true
		if u != cases[i] {
			if err := c.Traces.PutEvalCase(ctx, u); err != nil {
				return nil, err
			}
		}
	}
	if hold != nil {
		if err := writeHold(c.Config.State, hold); err != nil {
			return nil, err
		}
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev2", Kind: "tenant_eval_regressed", Tenant: tenant,
			Where: fmt.Sprintf("%d cases: %v", len(hold.Cases), hold.Cases), Ref: hold.Release})
		return hold, nil
	}
	// A hold whose regressed cases have all been deleted ends.
	old, found, err := ReadHold(c.Config.State)
	if err != nil || !found {
		return nil, err
	}
	for _, id := range old.Cases {
		if ids[id] {
			return &old, nil
		}
	}
	return nil, writeHold(c.Config.State, nil)
}

// evalHeld is whether a tenant eval hold holds this cell's release (SB-4004).
func (c *Cell) evalHeld() (bool, error) {
	_, found, err := ReadHold(c.Config.State)
	return found, err
}

// agentKilled is whether agent is killed now (SB-4004).
func (c *Cell) agentKilled(agent string) bool { return c.Kill.AgentKilled(agent)(context.Background()) }

// SmallHeldPath holds the small model tier (SB-4405): while it exists, every ask goes to
// the main model.
func SmallHeldPath(state string) string { return filepath.Join(state, "evals", "small-held") }

func writeSmallHeld(state string, held bool) error {
	p := SmallHeldPath(state)
	if !held {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte("held\n"), 0o600)
}

// smallHeld is whether the small tier is held; one that can't be read is held.
func (c *Cell) smallHeld() bool {
	_, err := os.Stat(SmallHeldPath(c.Config.State))
	return !errors.Is(err, os.ErrNotExist)
}

// withSkillCases is cases with every skill's cases of the manifest, as the tenant's own set
// keeps them (SB-4601): a skill case already kept keeps its results while its question and
// document stay the same, a new one is added, and one whose skill is gone is deleted. Each
// runs in its space as the space's first active steward; with none, it can't run.
func (c *Cell) withSkillCases(ctx context.Context, cases []ask.EvalCase) ([]ask.EvalCase, error) {
	m := c.Manifest()
	want := map[string]ask.EvalCase{}
	names := make([]string, 0, len(m.Spaces))
	for name := range m.Spaces {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sp := m.Spaces[name]
		if len(sp.Skills) == 0 {
			continue
		}
		key := ask.SpaceKey(name)
		by, err := c.firstSteward(ctx, m, key)
		if err != nil {
			return nil, err
		}
		for skill, sk := range sp.Skills {
			for q, doc := range sk.Cases {
				id := ask.SkillCaseID(key, skill, q)
				want[id] = ask.EvalCase{ID: id, Tenant: c.Config.Tenant, Space: key, Question: "/" + skill + " " + q, Expect: doc, By: by}
			}
		}
	}
	out := make([]ask.EvalCase, 0, len(cases)+len(want))
	for _, ec := range cases {
		if !ask.IsSkillCase(ec.ID) {
			out = append(out, ec)
			continue
		}
		w, ok := want[ec.ID]
		if !ok {
			if err := c.Traces.DeleteEvalCase(ctx, c.Config.Tenant, ec.ID); err != nil {
				return nil, err
			}
			continue
		}
		delete(want, ec.ID)
		if ec.Question != w.Question || ec.Expect != w.Expect {
			ec.LastRelease, ec.LastPassed, ec.PassedOn = "", false, "" // a changed case starts again
		}
		ec.Question, ec.Expect, ec.By, ec.Space = w.Question, w.Expect, w.By, w.Space
		out = append(out, ec)
	}
	ids := make([]string, 0, len(want))
	for id := range want {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ec := want[id]
		ec.At = time.Now().UTC()
		out = append(out, ec)
	}
	return out, nil
}
