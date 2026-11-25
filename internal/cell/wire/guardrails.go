package wire

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/report"
)

// Guardrails are the 12 weeks of guardrail counts (sb-observability(7), Guardrail
// insights; SB-4406), from the index, the traces and the audit log, counts only.
func (c *Cell) Guardrails(ctx context.Context, now time.Time) (report.Guardrails, error) {
	m := c.Manifest()
	chunks, err := c.spaceChunks(ctx, m)
	if err != nil {
		return report.Guardrails{}, err
	}
	from := now.AddDate(0, 0, -7*report.GuardrailWeeks)
	traces, err := c.Report.Traces(ctx, c.Config.Tenant, from, now)
	if err != nil {
		return report.Guardrails{}, err
	}
	var rows []audit.Row
	for seq := int64(1); ; {
		page, err := c.Audit.Store.Range(ctx, seq, 1000)
		if err != nil {
			return report.Guardrails{}, err
		}
		for _, r := range page {
			if r.Kind == "approval" || r.Kind == "decision" {
				rows = append(rows, r)
			}
		}
		if len(page) < 1000 {
			break
		}
		seq = page[len(page)-1].Seq + 1
	}
	return report.BuildGuardrails(now, chunks, traces, rows), nil
}

// GuardrailAlerts raises guardrail_flags_doubled once a week for each source whose flags
// doubled, keeping what it raised in state/guardrails/alerted.json.
func (c *Cell) GuardrailAlerts(ctx context.Context, now time.Time) error {
	if c.Alerter == nil {
		return nil
	}
	g, err := c.Guardrails(ctx, now)
	if err != nil || len(g.Doubled) == 0 {
		return err
	}
	path := filepath.Join(c.Config.State, "guardrails", "alerted.json")
	done := map[string]bool{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &done)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	week := g.Weeks[1].Week // the last full week
	changed := false
	for _, src := range g.Doubled {
		if done[src+"|"+week] {
			continue
		}
		if err := c.Alerter.Alert(ctx, canary.Alert{Severity: "sev3", Kind: "guardrail_flags_doubled", Tenant: c.Config.Tenant,
			Where: src, Ref: week}); err != nil {
			return err
		}
		done[src+"|"+week], changed = true, true
	}
	if !changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(done)
	return os.WriteFile(path, b, 0o600)
}

// GuardrailEvents are the guardrail events at or after since (sb-observability(7),
// Guardrail events; SB-5103), from the audit log, ids and counts only.
func (c *Cell) GuardrailEvents(ctx context.Context, since time.Time) ([]report.GuardrailEvent, error) {
	var rows []audit.Row
	for seq := int64(1); ; {
		page, err := c.Audit.Store.Range(ctx, seq, 1000)
		if err != nil {
			return nil, err
		}
		for _, r := range page {
			if r.Kind == "guardrail" || r.Kind == "approval" || r.Kind == "decision" {
				rows = append(rows, r)
			}
		}
		if len(page) < 1000 {
			break
		}
		seq = page[len(page)-1].Seq + 1
	}
	return report.GuardrailEvents(rows, since), nil
}

// guardEvent writes a suspect passage of a kept answer as a guardrail row (SB-5103). A row
// that can't be written is counted, never a reason to fail the answer already given.
func (c *Cell) guardEvent(ctx context.Context, e ask.GuardEvent) {
	space := e.Space
	if space == "" {
		space = "company"
	}
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: "sys_guard", Kind: "guardrail", Subject: "suspect:" + e.Cite, State: e.State,
		Reason: "agent=" + e.Agent, Space: space}); err != nil {
		c.Metrics.Add("sb_guardrail_write_failed_total", "Guardrail events that couldn't be written.", map[string]string{"tenant": c.Config.Tenant}, 1)
	}
}

// masked writes one way out's masked strings as a guardrail row (SB-5102, SB-5103).
func (c *Cell) masked(ctx context.Context, way string, n int) {
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: "sys_guard", Kind: "guardrail", Subject: "masked:" + way, State: "masked",
		Reason: "strings=" + strconv.Itoa(n)}); err != nil {
		c.Metrics.Add("sb_guardrail_write_failed_total", "Guardrail events that couldn't be written.", map[string]string{"tenant": c.Config.Tenant}, 1)
	}
}

// maskedOut is the API's masked responses (SB-5102).
func (c *Cell) maskedOut(r *http.Request, n int) { c.masked(r.Context(), "api", n) }
