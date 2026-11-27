package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/inventory"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proposals"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/errs"
)

// Learn what works (SB-5005 to SB-5007): checks that come back four weeks after a change,
// the admin's week, and the inventory export.

// checkLookback is how far past due a change is still checked: a cell that was off for a
// while catches up, and older history is never checked.
const checkLookback = 30 * 24 * time.Hour

// learnEvery is how often the checks and the admin's week are counted.
const learnEvery = time.Hour

// operatorGroup is the group whose members administer the cell (sb-admin(3), Roles).
const operatorGroup = "sb-operators"

func (c *Cell) checksDir() string { return filepath.Join(c.Config.State, "checks") }

func auditLink(seq int64) string { return "/v1/admin/audit?since=" + strconv.FormatInt(seq, 10) }

// changes are the changes that shipped in [from, to): steward proposals approved, policies
// marked, fixes applied (a newer version of the fixed document) and agents' new releases.
func (c *Cell) changes(ctx context.Context, rows []audit.Row, traces []ask.Trace, from, to time.Time) []ask.Change {
	var out []ask.Change
	in := func(t time.Time) bool { return !t.Before(from) && t.Before(to) }
	for _, r := range rows {
		at, err := time.Parse(time.RFC3339Nano, r.At)
		if err != nil {
			continue
		}
		space := strings.TrimPrefix(r.Space, "spc_")
		switch {
		case r.Kind == "config" && r.State == proposals.Approved && strings.HasPrefix(r.Space, "spc_") && in(at):
			kind := ask.ChangeSpace
			if p, ok, err := c.Proposals.Get(ctx, r.Subject); err == nil && ok {
				kind = ask.ProposalChange(p.Diff)
			}
			out = append(out, ask.Change{Kind: kind, Ref: r.Subject, Space: space, At: at, By: r.Actor, Source: auditLink(r.Seq)})
		case r.Kind == "config" && r.State == "policy_marked" && in(at):
			out = append(out, ask.Change{Kind: ask.ChangePolicy, Ref: strings.TrimPrefix(r.Subject, "policy:"), Space: space, At: at, By: r.Actor, Source: auditLink(r.Seq)})
		case r.Kind == "access" && r.State == "fix_suggested":
			if at, ok := c.fixApplied(ctx, r.Subject); ok && in(at) {
				out = append(out, ask.Change{Kind: ask.ChangeFix, Ref: r.Subject, Space: space, At: at, Source: auditLink(r.Seq)})
			}
		}
	}
	return append(out, ask.ReleaseChanges(traces, from, to)...)
}

// fixApplied is when the document of fix suggestion id first had a newer version.
func (c *Cell) fixApplied(ctx context.Context, id string) (time.Time, bool) {
	rs, ok := c.Traces.(ask.ResearchStore)
	lk, ok2 := c.Index.(ports.DocLookup)
	if !ok || !ok2 {
		return time.Time{}, false
	}
	r, found, err := rs.GetResearch(ctx, c.Config.Tenant, id)
	if err != nil || !found || len(r.Documents) != 1 {
		return time.Time{}, false
	}
	doc, ver, _ := strings.Cut(r.Documents[0], "@")
	kind, sid, _ := strings.Cut(doc, ":")
	chunks, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: sid}}})
	if err != nil {
		return time.Time{}, false
	}
	var at time.Time
	for _, ch := range chunks {
		if _, v, _ := strings.Cut(ch.Cite, "@"); v != ver && ch.Seen.After(r.At) && (at.IsZero() || ch.Seen.Before(at)) {
			at = ch.Seen
		}
	}
	return at, !at.IsZero()
}

// Checks are the kept check results, newest first; space "" is every space's.
func (c *Cell) Checks(space string) ([]ask.Check, error) {
	es, err := os.ReadDir(c.checksDir())
	if errors.Is(err, os.ErrNotExist) {
		return []ask.Check{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []ask.Check{}
	for _, e := range es {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(c.checksDir(), e.Name()))
		if err != nil {
			return nil, err
		}
		var ch ask.Check
		if err := json.Unmarshal(b, &ch); err != nil {
			return nil, err
		}
		if space == "" || ch.Change.Space == space {
			out = append(out, ch)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Made.After(out[j].Made) || out[i].Made.Equal(out[j].Made) && out[i].ID < out[j].ID
	})
	return out, nil
}

// runChecks makes each check now due and not yet made, keeps it, and says whom to tell: the
// space's stewards, or the agent's owner for an agent's release.
func (c *Cell) runChecks(ctx context.Context, now time.Time) ([]notify.AgentAlert, error) {
	from := now.Add(-ask.CheckAfter - checkLookback)
	readFrom := from.Add(-ask.CheckAfter)
	traces, err := c.Report.Traces(ctx, c.Config.Tenant, readFrom, now)
	if err != nil {
		return nil, err
	}
	rows, err := c.Audit.Store.Range(ctx, 1, 1<<30)
	if err != nil {
		return nil, err
	}
	chs := c.changes(ctx, rows, traces, from, now.Add(-ask.CheckAfter))
	qs, err := c.Traces.StewardQuestionsBetween(ctx, c.Config.Tenant, readFrom, now)
	if err != nil {
		return nil, err
	}
	fb, err := c.Traces.FeedbackBetween(ctx, c.Config.Tenant, readFrom, now)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.checksDir(), 0o700); err != nil {
		return nil, err
	}
	for _, ch := range chs {
		path := filepath.Join(c.checksDir(), ch.Key()+".json")
		if _, err := os.Stat(path); err == nil {
			continue
		}
		b, err := json.Marshal(ask.RunCheck(ch, qs, traces, fb, now))
		if err != nil {
			return nil, err
		}
		if err := writeAtomic(path, b); err != nil {
			return nil, err
		}
	}
	return c.checkAlerts(ctx, now)
}

// checkAlerts tell each check made in the lookback; the notify job tells each once.
func (c *Cell) checkAlerts(ctx context.Context, now time.Time) ([]notify.AgentAlert, error) {
	all, err := c.Checks("")
	if err != nil {
		return nil, err
	}
	m := c.Manifest()
	var out []notify.AgentAlert
	for _, res := range all {
		if res.Made.Before(now.Add(-checkLookback)) {
			continue
		}
		var to []string
		if res.Change.Agent != "" {
			to, err = c.ownerUsers(ctx, res.Change.Agent)
		} else {
			to = c.groupUsers(ctx, m.Spaces[res.Change.Space].Stewards)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, notify.AgentAlert{Kind: notify.KindCheck, Key: "check-" + res.ID, To: to, Text: res.Text, Agent: res.Change.Agent, Space: res.Change.Space})
	}
	return out, nil
}

// groupUsers are the active members of the group: principals, each once.
func (c *Cell) groupUsers(ctx context.Context, principals []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range principals {
		name, ok := strings.CutPrefix(p, "group:")
		if !ok || c.Directory == nil {
			continue
		}
		ids, err := c.Directory.Members(ctx, c.Config.Tenant, name)
		if err != nil {
			continue // a group the directory doesn't have (yet) has no one to tell (LAW-12)
		}
		for _, id := range ids {
			if u, found, err := c.Directory.User(ctx, c.Config.Tenant, id); err == nil && found && u.Active && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	sort.Strings(out)
	return out
}

// adminWeek is the Monday digest for the operators (SB-5006): the last week's changes,
// caps and limits hit, pauses, guardrail trends, access reviews due, checks back and the
// top spaces by spend, every line with its source record. Counts and ids only.
func (c *Cell) adminWeek(ctx context.Context, now time.Time) ([]notify.AgentAlert, error) {
	m := c.Manifest()
	zone, _ := manifest.DigestZone(m.Digest)
	week := notify.Monday(now, zone)
	from := week.AddDate(0, 0, -7)
	to := c.groupUsers(ctx, []string{"group:" + operatorGroup})
	if len(to) == 0 {
		return nil, nil
	}
	lines, err := c.adminLines(ctx, m, from, week, now)
	if err != nil {
		return nil, err
	}
	day := week.In(zone).Format(time.DateOnly)
	return []notify.AgentAlert{{Kind: notify.KindAdminWeek, Key: "adminweek-" + day, To: to, Text: ask.AdminWeekText(from.In(zone).Format(time.DateOnly), lines)}}, nil
}

func (c *Cell) adminLines(ctx context.Context, m manifest.Tenant, from, to, now time.Time) ([]ask.AdminLine, error) {
	out, err := c.auditLines(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out = append(out, c.rolloutLines(from, to)...) // SB-5106
	spend, err := c.monthSpend(ctx, m, now, nil)
	if err != nil {
		return nil, err
	}
	for _, s := range spend {
		if s.Capped {
			out = append(out, ask.AdminLine{Section: "Caps and limits", Text: s.Agent + " is at its monthly cap", Link: "/v1/admin/agents/value"})
		}
	}
	if m.Limits != nil {
		people, err := c.personSpent(ctx, now.UTC().Format("2006-01"))
		if err != nil {
			return nil, err
		}
		if b := ask.Bands(people, m.Limits.MonthlyUSDPerPerson); b.AtLimit != "0" {
			out = append(out, ask.AdminLine{Section: "Caps and limits", Text: "people at the personal limit this month: " + b.AtLimit, Link: "/v1/admin/ai/usage"})
		}
	}
	if g, err := c.Guardrails(ctx, now); err == nil && len(g.Doubled) > 0 {
		out = append(out, ask.AdminLine{Section: "Guardrails", Text: "injection flags doubled last week in " + strings.Join(g.Doubled, ", "), Link: "/v1/admin/guardrails"})
	}
	if c.Reviews != nil {
		rvs, err := c.Reviews.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, rv := range rvs {
			if rv.State == reviews.Open {
				out = append(out, ask.AdminLine{Section: "Access reviews due", Text: fmt.Sprintf("%s %s in %s, due %s", rv.ID, rv.Quarter, rv.Space, rv.Due), Link: "/v1/admin/reviews/" + rv.ID + "/evidence"})
			}
		}
	}
	sens, err := c.sensitiveLines(ctx) // SB-5407
	if err != nil {
		return nil, err
	}
	out = append(out, sens...)
	checks, err := c.Checks("")
	if err != nil {
		return nil, err
	}
	for _, ch := range checks {
		if !ch.Made.Before(from) && ch.Made.Before(to) {
			out = append(out, ask.AdminLine{Section: "Checks back", Text: fmt.Sprintf("%s %s: %s", ch.Change.Kind, ch.Change.Ref, ch.Verdict), Link: "/v1/admin/checks/" + ch.ID})
		}
	}
	spaces, err := c.Usage.Spend().SpentBySpace(ctx, c.Config.Tenant, now.UTC().Format("2006-01"))
	if err != nil {
		return nil, err
	}
	top := make([]string, 0, len(spaces))
	for k := range spaces {
		top = append(top, k)
	}
	sort.Slice(top, func(i, j int) bool {
		return spaces[top[i]] > spaces[top[j]] || spaces[top[i]] == spaces[top[j]] && top[i] < top[j]
	})
	for i, k := range top {
		if i == 3 {
			break
		}
		out = append(out, ask.AdminLine{Section: "Top spaces by spend this month", Text: fmt.Sprintf("%s $%.2f", strings.TrimPrefix(k, "spc_"), spaces[k]), Link: "/v1/admin/costs/spaces"})
	}
	order := map[string]int{"Changed": 0, "Caps and limits": 1, "Pauses": 2, "Guardrails": 3, "Access reviews due": 4, "Checks back": 5, "Top spaces by spend this month": 6}
	sort.SliceStable(out, func(i, j int) bool { return order[out[i].Section] < order[out[j].Section] })
	return out, nil
}

// auditLines are the week's audit rows the admin's week names: proposals approved,
// manifests applied, agents paused and limits lifted.
func (c *Cell) auditLines(ctx context.Context, from, to time.Time) ([]ask.AdminLine, error) {
	var out []ask.AdminLine
	rows, err := c.Audit.Store.Range(ctx, 1, 1<<30)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		at, err := time.Parse(time.RFC3339Nano, r.At)
		if err != nil || at.Before(from) || !at.Before(to) {
			continue
		}
		switch {
		case r.Kind == "config" && r.State == proposals.Approved:
			what := "a change"
			if p, ok, err := c.Proposals.Get(ctx, r.Subject); err == nil && ok {
				what = ask.ProposalChange(p.Diff)
			}
			out = append(out, ask.AdminLine{Section: "Changed", Text: fmt.Sprintf("%s %s approved by %s (%s)", r.Subject, what, r.Actor, strings.TrimPrefix(r.Space, "spc_")), Link: auditLink(r.Seq)})
		case r.Kind == "config" && r.Subject == "manifest" && r.State == "applied":
			out = append(out, ask.AdminLine{Section: "Changed", Text: "manifest applied by " + r.Actor + ": " + r.Reason, Link: auditLink(r.Seq)})
		case r.Kind == "kill" && strings.HasPrefix(r.Subject, "agent:") && r.State == "killed":
			out = append(out, ask.AdminLine{Section: "Pauses", Text: r.Subject + " paused: " + r.Reason, Link: auditLink(r.Seq)})
		case r.Kind == "access" && strings.HasPrefix(r.Subject, "limit:") && r.State == "lifted":
			out = append(out, ask.AdminLine{Section: "Caps and limits", Text: "a personal limit lifted in " + strings.TrimPrefix(r.Space, "spc_"), Link: auditLink(r.Seq)})
		}
	}
	return out, nil
}

// learnAlerts are the checks due and the admin's week, told as the agent alerts are.
// The counting runs at most once an hour (learnEvery); in between, only what is kept is told.
func (c *Cell) learnAlerts(ctx context.Context, now time.Time) ([]notify.AgentAlert, error) {
	ran := filepath.Join(c.checksDir(), "ran")
	if b, err := os.ReadFile(ran); err == nil {
		if last, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b))); err == nil && now.Sub(last) >= 0 && now.Sub(last) < learnEvery {
			return c.checkAlerts(ctx, now)
		}
	}
	if err := os.MkdirAll(c.checksDir(), 0o700); err != nil {
		return nil, err
	}
	if err := writeAtomic(ran, []byte(now.UTC().Format(time.RFC3339)+"\n")); err != nil {
		return nil, err
	}
	out, err := c.runChecks(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("checks: %w", err)
	}
	week, err := c.adminWeek(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("admin's week: %w", err)
	}
	return append(out, week...), nil
}

// Inventory is the signed inventory export (SB-5007).
func (c *Cell) Inventory(ctx context.Context) (inventory.Signed, error) {
	now := time.Now()
	m := c.Manifest()
	from := now.AddDate(0, 0, -90)
	in := inventory.Inputs{Tenant: c.Config.Tenant, Release: c.Release, Now: now, Manifest: m}
	var err error
	if in.Traces, err = c.Report.Traces(ctx, c.Config.Tenant, from, now); err != nil {
		return inventory.Signed{}, err
	}
	if in.Questions, err = c.Traces.StewardQuestionsBetween(ctx, c.Config.Tenant, from, now); err != nil {
		return inventory.Signed{}, err
	}
	for _, a := range api.AgentRegistry(m, "", nil, nil, nil, false) {
		in.Agents = append(in.Agents, inventory.Agent{Name: a.Name, Spaces: a.Spaces})
	}
	if rs, ok := c.Traces.(ask.RoutineStore); ok {
		if in.Routines, err = rs.Routines(ctx, c.Config.Tenant); err != nil {
			return inventory.Signed{}, err
		}
	}
	if ss, ok := c.Traces.(ask.SuggestedStore); ok {
		for _, n := range sortedSpaces(m) {
			sg, found, err := ss.Suggested(ctx, c.Config.Tenant, ask.SpaceKey(n))
			if err != nil {
				return inventory.Signed{}, err
			}
			if found {
				in.Starters = append(in.Starters, sg)
			}
		}
	}
	if rr, ok := c.Traces.(ask.RhythmStore); ok {
		if in.RhythmRuns, err = rr.RhythmRuns(ctx, c.Config.Tenant); err != nil {
			return inventory.Signed{}, err
		}
	}
	if c.Proposals != nil {
		if in.Proposals, err = c.Proposals.List(ctx); err != nil {
			return inventory.Signed{}, err
		}
	}
	sg, err := c.AnchorSigner(ctx)
	if err != nil {
		return inventory.Signed{}, err
	}
	return inventory.Sign(inventory.Build(in), sg, now)
}

// inventoryCP is GET /v1/cell/inventory for sb inventory export (SB-5007).
func (c *Cell) inventoryCP(w http.ResponseWriter, r *http.Request) {
	if !c.guardControlPlane(w, r) {
		return
	}
	inv, err := c.Inventory(r.Context())
	if err != nil {
		refuseJSON(w, errs.EUNAVAIL, "The inventory can't be made right now.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(inv)
}
