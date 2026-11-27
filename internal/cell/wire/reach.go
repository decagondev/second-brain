package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/costs"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/tenancy"
)

// How far each agent reaches (SB-5406) and the sensitive map (SB-5407): replayed and counted
// now, with each week's numbers kept under state/ so a week can be compared with the last.

// AgentReach is agent name's reach now: every space whose manifest enables it, read as each
// active steward of that space (whom its drafts, triggers and rhythms run as).
func (c *Cell) AgentReach(ctx context.Context, name string) (ask.AgentReach, error) {
	if _, ok := ask.AgentDefOf(name); !ok {
		return ask.AgentReach{}, fmt.Errorf("%w: no agent %q in the bundle", ask.ErrInvalid, name)
	}
	m := c.Manifest()
	now := time.Now()
	week := ask.WeekStart(now).Format(time.DateOnly)
	out := ask.AgentReach{Agent: name, Week: week, Spaces: []ask.SpaceReach{}}
	names := make([]string, 0, len(m.Spaces))
	for n := range m.Spaces {
		names = append(names, n)
	}
	sort.Strings(names)
	a := c.Parts().Agent
	for _, n := range names {
		if !contains(m.Effective(n).Agents, name) {
			continue
		}
		st, _, _ := stewardFacts(m)("spc_" + n)
		var scopes []tenancy.Scope
		for _, id := range c.principalUsers(ctx, st) {
			s, err := c.ScopeAs(ctx, id, "spc_"+n)
			if err != nil {
				continue // not active now: reaches nothing
			}
			scopes = append(scopes, s)
		}
		r, err := a.ReachIn(ctx, "spc_"+n, scopes)
		if err != nil {
			return out, err
		}
		out.Spaces = append(out.Spaces, r)
	}
	out.Total()
	kept, err := c.weekly("reach", name)
	if err != nil {
		return out, err
	}
	if last, ok := kept[ask.WeekStart(now).AddDate(0, 0, -7).Format(time.DateOnly)]; ok {
		n := last
		out.LastWeek = &n
		out.Grew = ask.Grown(last, out.Documents)
	}
	kept[week] = out.Documents
	return out, c.keepWeekly("reach", name, kept)
}

// principalUsers are the active people stewards (user:ID and group:NAME) name.
func (c *Cell) principalUsers(ctx context.Context, principals []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range principals {
		if id, ok := strings.CutPrefix(p, "user:"); ok && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range c.groupUsers(ctx, principals) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// reachAlerts tell an agent's owners once a week when its reach grew by more than a tenth.
func (c *Cell) reachAlerts(ctx context.Context) ([]notify.AgentAlert, error) {
	defs, err := ask.ParseAgentDefs(ask.AgentDefs)
	if err != nil {
		return nil, err
	}
	var out []notify.AgentAlert
	for _, d := range defs {
		if d.Owner == "" {
			continue
		}
		r, err := c.AgentReach(ctx, d.Name)
		if err != nil {
			return nil, err
		}
		if !r.Grew || r.LastWeek == nil {
			continue
		}
		to, err := c.ownerUsers(ctx, d.Name)
		if err != nil {
			return nil, err
		}
		out = append(out, notify.AgentAlert{Key: "reach-" + d.Name + "-" + r.Week, To: to,
			Text: fmt.Sprintf("%s can now read %d documents, up from %d last week (%d labelled confidential or above). See its reach in the agent registry", d.Name, r.Documents, *r.LastWeek, r.Labelled)})
	}
	return out, nil
}

// SensitiveMap is every space's sensitive counts this week (SB-5407), with last week's.
func (c *Cell) SensitiveMap(ctx context.Context) ([]ask.SensitiveCounts, error) {
	m := c.Manifest()
	names := make([]string, 0, len(m.Spaces))
	for n := range m.Spaces {
		names = append(names, n)
	}
	sort.Strings(names)
	week := ask.WeekStart(time.Now()).Format(time.DateOnly)
	out := []ask.SensitiveCounts{}
	for _, n := range names {
		sc, err := c.Parts().Agent.SensitiveIn(ctx, "spc_"+n, week)
		if err != nil {
			return nil, err
		}
		if lw, ok := c.lastSensitive(n, ask.WeekStart(time.Now()).AddDate(0, 0, -7).Format(time.DateOnly)); ok {
			sc.LastWeek = &lw
		}
		if err := c.keepSensitive(sc); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, nil
}

func (c *Cell) sensitivePath(space string) string {
	return filepath.Join(c.Config.State, "sensitive", space+".json")
}

func (c *Cell) lastSensitive(space, week string) (ask.SensitiveCounts, bool) {
	var all map[string]ask.SensitiveCounts
	b, err := os.ReadFile(c.sensitivePath(space))
	if err != nil || json.Unmarshal(b, &all) != nil {
		return ask.SensitiveCounts{}, false
	}
	sc, ok := all[week]
	return sc, ok
}

func (c *Cell) keepSensitive(sc ask.SensitiveCounts) error {
	all := map[string]ask.SensitiveCounts{}
	if b, err := os.ReadFile(c.sensitivePath(sc.Space)); err == nil {
		_ = json.Unmarshal(b, &all)
	}
	sc.LastWeek = nil
	all[sc.Week] = sc
	return writeJSON(c.sensitivePath(sc.Space), all)
}

// weekly is what one kind keeps for one name, by week.
func (c *Cell) weekly(kind, name string) (map[string]int, error) {
	out := map[string]int{}
	b, err := os.ReadFile(filepath.Join(c.Config.State, kind, name+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Cell) keepWeekly(kind, name string, v map[string]int) error {
	return writeJSON(filepath.Join(c.Config.State, kind, name+".json"), v)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", b, 0o600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// sensitiveLines are the admin's week lines for the sensitive map: spaces with documents
// shared too widely or held, counts only.
func (c *Cell) sensitiveLines(ctx context.Context) ([]ask.AdminLine, error) {
	sm, err := c.SensitiveMap(ctx)
	if err != nil {
		return nil, err
	}
	var out []ask.AdminLine
	for _, s := range sm {
		if s.Wide == 0 && s.Held == 0 {
			continue
		}
		out = append(out, ask.AdminLine{Section: "Sensitive map", Text: fmt.Sprintf("%s: %d shared too widely, %d held, %d with personal data", s.Space, s.Wide, s.Held, s.Personal), Link: "/v1/admin/sensitive"})
	}
	return out, nil
}

// SavedIn is what month's answers saved (SB-5606), from their traces.
func (c *Cell) SavedIn(ctx context.Context, month string) (costs.Saved, error) {
	from, to, err := costs.ParseMonth(month)
	if err != nil {
		return costs.Saved{}, err
	}
	ts, err := c.Report.Traces(ctx, c.Config.Tenant, from, to)
	if err != nil {
		return costs.Saved{}, err
	}
	var each []map[string]float64
	for _, t := range ts {
		if len(t.Saved) > 0 {
			each = append(each, t.Saved)
		}
	}
	return costs.AddSaved(month, each), nil
}
