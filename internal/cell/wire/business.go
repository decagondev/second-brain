package wire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/manifest"
)

// Run AI like a business (SB-5001 to SB-5003): the command centre's numbers, routing
// policies and personal limits, over the cell's traces and usage ledger.

// limitsClock keeps this cell's personal-limit lifts, one marker a person and month.
func (c *Cell) limitsClock() notifyClock { return notifyClock{filepath.Join(c.Config.State, "limits")} }

// liftKey is the marker of user's lift in month: no address or id in the file name.
func liftKey(user, month string) string {
	sum := sha256.Sum256([]byte(user))
	return "lift-" + hex.EncodeToString(sum[:8]) + "-" + month
}

// personSpent is each person's spend this month, from the usage ledger.
func (c *Cell) personSpent(ctx context.Context, month string) (map[string]float64, error) {
	pm, ok := any(c.Usage.Spend()).(budget.PersonMeter)
	if !ok {
		return map[string]float64{}, nil
	}
	return pm.SpentByPerson(ctx, c.Config.Tenant, month)
}

// personOver says user has reached the tenant's monthly limit, unless a steward lifted it
// for this month (SB-5003).
func (c *Cell) personOver(ctx context.Context, user string) (bool, error) {
	m := c.Manifest()
	if m.Limits == nil || user == "" {
		return false, nil
	}
	month := time.Now().UTC().Format("2006-01")
	if _, lifted, err := c.limitsClock().Last(ctx, liftKey(user, month)); err != nil || lifted {
		return false, err
	}
	spent, err := c.personSpent(ctx, month)
	if err != nil {
		return false, err
	}
	return spent[user] >= m.Limits.MonthlyUSDPerPerson, nil
}

// lift lifts the limit of the active member of space whose address is email, for this month,
// once audit has written its row.
func (c *Cell) lift(ctx context.Context, space, email string, audit func(user, month string) error) (string, string, error) {
	m := c.Manifest()
	sp, ok := m.Spaces[space]
	if !ok {
		return "", "", ask.ErrNotMember
	}
	ids, err := c.activeByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}
	month := time.Now().UTC().Format("2006-01")
	for _, id := range ids {
		s, err := c.ScopeAs(ctx, id, "company")
		if err != nil {
			continue
		}
		member := false
		for _, g := range s.Groups() {
			member = member || contains(sp.Members, "group:"+g)
		}
		if !member {
			continue
		}
		if err := audit(id, month); err != nil {
			return "", "", err
		}
		return id, month, c.limitsClock().Mark(ctx, liftKey(id, month), time.Now())
	}
	return "", "", ask.ErrNotMember
}

// routing is each space's routing policy, by key (SB-5002).
func routing(m manifest.Tenant) func(space string) string {
	return func(space string) string {
		name := strings.TrimPrefix(space, "spc_")
		if _, ok := m.Spaces[name]; !ok {
			return manifest.DefaultRouting
		}
		return m.RoutingOf(name)
	}
}

// suspectMode is a space's suspect mode by key (SB-5101): skip unless it, or its
// department, says monitor.
func suspectMode(m manifest.Tenant) func(space string) string {
	return func(space string) string {
		name := strings.TrimPrefix(space, "spc_")
		if _, ok := m.Spaces[name]; !ok {
			return ask.SuspectSkip
		}
		return m.SuspectOf(name)
	}
}

// aiUsage is the AI command centre: 12 weeks of traces counted, with each space's budget and
// this month's spend, each capped agent's spend and the personal limit's bands (SB-5001).
// aiClients counts the AI tools that reach the brain (SB-6006): live tokens by tool, and
// this month's asks from each tool's tokens.
func (c *Cell) aiClients(ctx context.Context) ([]ask.AIClient, error) {
	lt := c.tokenLister()
	if lt == nil {
		return nil, errors.New("wire: sessions can't be listed")
	}
	now := time.Now()
	tokens, err := lt.ListTokens(ctx, c.Config.Tenant)
	if err != nil {
		return nil, err
	}
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	traces, err := c.Report.Traces(ctx, c.Config.Tenant, from, now)
	if err != nil {
		return nil, err
	}
	return ask.BuildAIClients(tokens, traces, c.Config.Tenant, now), nil
}

func (c *Cell) aiUsage(ctx context.Context) (ask.AIUsage, error) {
	now := time.Now()
	from := ask.WeekStart(now).AddDate(0, 0, -7*(ask.UsageWeeks-1))
	traces, err := c.Report.Traces(ctx, c.Config.Tenant, from, now)
	if err != nil {
		return ask.AIUsage{}, err
	}
	u := ask.BuildAIUsage(traces, now)
	m := c.Manifest()
	month := now.UTC().Format("2006-01")
	spent, err := c.Usage.Spend().SpentBySpace(ctx, c.Config.Tenant, month)
	if err != nil {
		return ask.AIUsage{}, err
	}
	for _, n := range sortedSpaces(m) {
		if b := m.Spaces[n].BudgetUSDMonth; b > 0 {
			u.Budgets = append(u.Budgets, ask.SpaceBudget{Space: n, MonthlyUSD: b, SpentUSD: spent[ask.SpaceKey(n)]})
		}
	}
	caps, err := c.monthSpend(ctx, m, now, nil)
	if err != nil {
		return ask.AIUsage{}, err
	}
	for _, s := range caps {
		if s.MonthlyUSD != nil {
			u.Caps = append(u.Caps, s)
		}
	}
	sort.Slice(u.Caps, func(i, j int) bool { return u.Caps[i].Agent < u.Caps[j].Agent })
	if m.Limits != nil {
		people, err := c.personSpent(ctx, month)
		if err != nil {
			return ask.AIUsage{}, err
		}
		u.Limit = ask.Bands(people, m.Limits.MonthlyUSDPerPerson)
	}
	return u, nil
}

// nearLimit are the people at 80% or more of the personal limit this month (SB-5003).
func (c *Cell) nearLimit(ctx context.Context, now time.Time) ([]string, error) {
	m := c.Manifest()
	if m.Limits == nil {
		return nil, nil
	}
	spent, err := c.personSpent(ctx, now.UTC().Format("2006-01"))
	if err != nil {
		return nil, err
	}
	var out []string
	for u, usd := range spent {
		if usd >= 0.8*m.Limits.MonthlyUSDPerPerson {
			out = append(out, u)
		}
	}
	sort.Strings(out)
	return out, nil
}
