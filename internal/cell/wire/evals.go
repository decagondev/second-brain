package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	agent := c.Parts().Agent
	run := func(ec ask.EvalCase) (bool, error) {
		s, err := c.ScopeAs(ctx, ec.By, ec.Space)
		if err != nil {
			return false, ask.ErrSkipCase // the steward has gone: the case can't run as them
		}
		resp, err := agent.Probe(ctx, s, ec.Question)
		if errors.Is(err, ask.ErrKilled) || errors.Is(err, ask.ErrBudget) {
			return false, ask.ErrSkipCase
		}
		if err != nil {
			return false, err
		}
		return ec.Passes(resp), nil
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
