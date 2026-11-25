package wire

import (
	"context"
	"fmt"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/tenancy"
)

// Every agent accountable (sb-agents(7), Value, Pause here, Spend caps; SB-4701 to SB-4703):
// value counted from the queues, traces and feedback; a pause is a kill switch scoped to a
// space; a cap is the month's traces against the manifest's monthly_usd.

// ownerUsers are the active users who answer for agent now: its owner's email, or its owner
// group's active members.
func (c *Cell) ownerUsers(ctx context.Context, agent string) ([]string, error) {
	owner := strings.TrimSpace(ask.AgentOwner(agent))
	name, isGroup := strings.CutPrefix(owner, "group:")
	if owner == "" || c.Directory == nil {
		return nil, nil
	}
	if !isGroup {
		ids, err := c.activeByEmail(ctx, owner)
		if err != nil || len(ids) != 1 {
			return nil, err // unknown, or two people: no one (LAW-12)
		}
		return ids, nil
	}
	ids, err := c.Directory.Members(ctx, c.Config.Tenant, name)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, id := range ids {
		if u, found, err := c.Directory.User(ctx, c.Config.Tenant, id); err == nil && found && u.Active {
			out = append(out, id)
		}
	}
	return out, nil
}

// monthStart is the first instant of now's month, UTC.
func monthStart(now time.Time) time.Time {
	n := now.UTC()
	return time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// monthSpend is each agent's spend this month, with the manifest's caps.
func (c *Cell) monthSpend(ctx context.Context, m manifest.Tenant, now time.Time, only []string) ([]ask.AgentSpend, error) {
	traces, err := c.Report.Traces(ctx, c.Config.Tenant, monthStart(now), now)
	if err != nil {
		return nil, err
	}
	return ask.MonthSpend(traces, now.UTC().Format("2006-01"), m.AgentCaps, only), nil
}

// agentCapped says agent is at its monthly cap; spend that can't be read is an error, which
// pauses its runs (SB-4703).
func (c *Cell) agentCapped(ctx context.Context, agent string) (bool, error) {
	m := c.Manifest()
	if _, ok := m.AgentCaps[agent]; !ok {
		return false, nil
	}
	sp, err := c.monthSpend(ctx, m, time.Now(), []string{agent})
	if err != nil {
		return true, err
	}
	return len(sp) == 1 && sp[0].Capped, nil
}

// agentValue is the agents' value over 12 weeks, for only those agents, or all when nil.
func (c *Cell) agentValue(ctx context.Context, only []string) (ask.ValueReport, error) {
	now := time.Now()
	from := now.Add(-ask.ValueWeeks * 7 * 24 * time.Hour)
	qs, err := c.Traces.StewardQuestionsBetween(ctx, c.Config.Tenant, from, now)
	if err != nil {
		return ask.ValueReport{}, err
	}
	traces, err := c.Report.Traces(ctx, c.Config.Tenant, from, now)
	if err != nil {
		return ask.ValueReport{}, err
	}
	fb, err := c.Traces.FeedbackBetween(ctx, c.Config.Tenant, from, now)
	if err != nil {
		return ask.ValueReport{}, err
	}
	rows := ask.AgentValue(qs, traces, fb, from, only)
	for i := range rows {
		rows[i].Paused = c.Kill.Paused(rows[i].Agent, rows[i].Space)
	}
	spend, err := c.monthSpend(ctx, c.Manifest(), now, only)
	if err != nil {
		return ask.ValueReport{}, err
	}
	return ask.ValueReport{Weeks: ask.ValueWeeks, From: from.UTC().Format("2006-01-02"), Rows: rows, Spend: spend}, nil
}

// ownerValue is the value of the agents s owns.
func (c *Cell) ownerValue(ctx context.Context, s tenancy.Scope) (ask.ValueReport, error) {
	owned, err := c.Parts().Agent.OwnedAgents(ctx, s)
	if err != nil {
		return ask.ValueReport{}, err
	}
	if owned == nil {
		owned = []string{}
	}
	return c.agentValue(ctx, owned)
}

// agentAlerts are what to tell agents' owners now: a doubled discard rate last week, and
// 80% and 100% of a monthly cap (SB-4701, SB-4703).
func (c *Cell) agentAlerts(ctx context.Context, now time.Time) ([]notify.AgentAlert, error) {
	m := c.Manifest()
	zone, _ := manifest.DigestZone(m.Digest)
	var out []notify.AgentAlert
	qs, err := c.Traces.StewardQuestionsBetween(ctx, c.Config.Tenant, now.AddDate(0, 0, -21), now)
	if err != nil {
		return nil, err
	}
	for _, d := range ask.DoubledDiscards(qs, now, zone) {
		to, err := c.ownerUsers(ctx, d.Agent)
		if err != nil {
			return nil, err
		}
		out = append(out, notify.AgentAlert{Key: "discards-" + d.Agent + "-" + d.Week, To: to,
			Text: fmt.Sprintf("Twice as many drafts by %s were discarded last week: %d of %d", d.Agent, d.Discarded, d.Drafts)})
	}
	if len(m.AgentCaps) == 0 {
		return out, nil
	}
	spend, err := c.monthSpend(ctx, m, now, nil)
	if err != nil {
		return nil, err
	}
	for _, s := range spend {
		if s.MonthlyUSD == nil || *s.MonthlyUSD <= 0 || s.SpentUSD < 0.8**s.MonthlyUSD {
			continue
		}
		to, err := c.ownerUsers(ctx, s.Agent)
		if err != nil {
			return nil, err
		}
		if s.Capped {
			out = append(out, notify.AgentAlert{Key: "cap100-" + s.Agent + "-" + s.Month, To: to,
				Text: s.Agent + " has spent its monthly budget: its drafts and routines pause until next month"})
			continue
		}
		out = append(out, notify.AgentAlert{Key: "cap80-" + s.Agent + "-" + s.Month, To: to,
			Text: fmt.Sprintf("%s has used %d%% of its monthly budget", s.Agent, int(100*s.SpentUSD / *s.MonthlyUSD))})
	}
	return out, nil
}

// pauseAgent pauses (on) or resumes agent in space name for s, a steward there or its
// owner: the kill switch agent:AGENT@NAME, audited with the reason, and the owner told.
func (c *Cell) pauseAgent(ctx context.Context, s tenancy.Scope, name, agent, reason string, on bool) error {
	if err := c.Parts().Stewards.MayPause(ctx, s, name, agent); err != nil {
		return err
	}
	reason = strings.Join(strings.Fields(reason), " ")
	if !on && reason == "" {
		reason = "resumed"
	}
	if reason == "" || len([]rune(reason)) > ask.MaxPauseReason {
		return fmt.Errorf("%w: a reason is 1 to %d characters", ask.ErrInvalid, ask.MaxPauseReason)
	}
	ctl := kill.Control{Store: c.Kill.Store, Audit: c.Audit, Now: time.Now}
	change := ctl.Unkill
	if on {
		change = ctl.Kill
	}
	if err := change(ctx, c.Config.Tenant, kill.PauseScope(agent, name), reason, s.User()); err != nil {
		return fmt.Errorf("%w: %v", ask.ErrUnavailable, err)
	}
	if err := c.Kill.Refresh(ctx); err != nil {
		return fmt.Errorf("%w: %v", ask.ErrUnavailable, err)
	}
	if !on {
		return nil
	}
	to, err := c.ownerUsers(ctx, agent)
	if err != nil {
		return nil //nolint:nilerr // paused; the owner can't be found to tell
	}
	for _, id := range to {
		_ = c.tell(ctx, id, notify.KindAgent, agent+" was paused in "+name+": "+reason) // best effort: the pause stands
	}
	return nil
}
