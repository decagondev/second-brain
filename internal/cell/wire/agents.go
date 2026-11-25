package wire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/costs"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
)

// Agents on a schedule and when something happens (sb-agents(7); SB-4005, SB-4006).

// briefDays are the spaces' brief days, by space key.
func updateSpaces(m manifest.Tenant) []string {
	var out []string
	for n, sp := range m.Spaces {
		if sp.Updates {
			out = append(out, ask.SpaceKey(n))
		}
	}
	sort.Strings(out)
	return out
}

// requestCategories are a space's request categories, by name (SB-4502).
func requestCategories(m manifest.Tenant) func(name string) []string {
	return func(name string) []string {
		if r := m.Spaces[name].Requests; r != nil {
			return r.Categories
		}
		return nil
	}
}

// leaderSpaces are the spaces that get the leader's week, by key (SB-4503).
func leaderSpaces(m manifest.Tenant) []string {
	var out []string
	for n := range m.Spaces {
		if m.LeaderWeek(n) {
			out = append(out, ask.SpaceKey(n))
		}
	}
	sort.Strings(out)
	return out
}

func briefDays(m manifest.Tenant) map[string]time.Weekday {
	out := map[string]time.Weekday{}
	for n, sp := range m.Spaces {
		if sp.Brief != nil {
			out[ask.SpaceKey(n)] = sp.Brief.Weekday()
		}
	}
	return out
}

// overBudget is whether this month's model spend has reached the tenant's budget; a meter
// that can't be read is an error, never under budget.
func (c *Cell) overBudget(ctx context.Context) (bool, error) {
	m := c.Manifest()
	if m.BudgetUSDMonth <= 0 {
		return false, nil
	}
	spent, err := c.Usage.Spend().Spent(ctx, c.Config.Tenant, time.Now().UTC().Format("2006-01"))
	if err != nil {
		return false, err
	}
	return spent >= float64(m.BudgetUSDMonth), nil
}

// branchSpend is this month's model spend of each space's branch, by name: what ran in the
// space and in its teams (SB-4105). A meter that can't be read is an error.
func (c *Cell) branchSpend(ctx context.Context, m manifest.Tenant, month string) (map[string]float64, error) {
	by, err := c.Usage.Spend().SpentBySpace(ctx, c.Config.Tenant, month)
	if err != nil {
		return nil, err
	}
	out := map[string]float64{}
	for key, usd := range by {
		name := strings.TrimPrefix(key, "spc_")
		if _, ok := m.Spaces[name]; !ok {
			continue
		}
		out[name] += usd
		if p := m.Spaces[name].Parent; p != "" {
			out[p] += usd
		}
	}
	return out, nil
}

// spaceOver is whether space key's branch, or its department's, has spent its own
// budget_usd_month this month; unreadable spend is an error, never under budget.
func (c *Cell) spaceOver(ctx context.Context, key string) (bool, error) {
	m := c.Manifest()
	name := strings.TrimPrefix(key, "spc_")
	sp, ok := m.Spaces[name]
	if !ok || (sp.BudgetUSDMonth == 0 && (sp.Parent == "" || m.Spaces[sp.Parent].BudgetUSDMonth == 0)) {
		return false, nil
	}
	spent, err := c.branchSpend(ctx, m, time.Now().UTC().Format("2006-01"))
	if err != nil {
		return false, err
	}
	for _, n := range []string{name, sp.Parent} {
		if b := m.Spaces[n].BudgetUSDMonth; n != "" && b > 0 && spent[n] >= float64(b) {
			return true, nil
		}
	}
	return false, nil
}

// spaceSpend is each space's spend in month, own and branch, with its budget and, for
// the running month after costs.ForecastMinDays, a straight-line month-end forecast.
func (c *Cell) spaceSpend(ctx context.Context, month string) ([]api.SpaceSpend, error) {
	from, to, err := costs.ParseMonth(month)
	if err != nil {
		return nil, err
	}
	m := c.Manifest()
	by, err := c.Usage.Spend().SpentBySpace(ctx, c.Config.Tenant, month)
	if err != nil {
		return nil, err
	}
	branch, err := c.branchSpend(ctx, m, month)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	gone := now.Sub(from)
	out := []api.SpaceSpend{}
	for _, name := range sortedSpaces(m) {
		sp := m.Spaces[name]
		row := api.SpaceSpend{Space: name, Parent: sp.Parent, OwnUSD: cents(by[ask.SpaceKey(name)]), SpentUSD: cents(branch[name]), BudgetUSD: sp.BudgetUSDMonth}
		if now.Before(to) && gone >= costs.ForecastMinDays*24*time.Hour {
			f := cents(branch[name] * float64(to.Sub(from)) / float64(gone))
			row.ForecastUSD = &f
		}
		out = append(out, row)
	}
	return out, nil
}

func cents(v float64) float64 { return math.Round(v*100) / 100 }

// spaceBudgets are the spaces with a budget and the share of it their branch spent this
// month, for the notify job's warnings (SB-4105).
func (c *Cell) spaceBudgets(ctx context.Context) ([]notify.SpaceBudget, error) {
	m := c.Manifest()
	month := time.Now().UTC().Format("2006-01")
	var out []notify.SpaceBudget
	var spent map[string]float64
	for _, name := range sortedSpaces(m) {
		b := m.Spaces[name].BudgetUSDMonth
		if b <= 0 {
			continue
		}
		if spent == nil {
			var err error
			if spent, err = c.branchSpend(ctx, m, month); err != nil {
				return nil, err
			}
		}
		out = append(out, notify.SpaceBudget{Space: ask.SpaceKey(name), Month: month, Share: spent[name] / float64(b)})
	}
	return out, nil
}

// firstSteward is the first active steward of space, by email, as a user id; "" is none.
func (c *Cell) firstSteward(ctx context.Context, m manifest.Tenant, key string) (string, error) {
	people, err := c.stewardPeople(m)(ctx, key)
	if err != nil {
		return "", err
	}
	for _, p := range people {
		us, err := c.Directory.UsersByEmail(ctx, c.Config.Tenant, p.Email)
		if err != nil {
			return "", err
		}
		for _, u := range us {
			if u.Active {
				return u.ID, nil
			}
		}
	}
	return "", nil
}

// runBrief runs a space's brief as its first active steward, and puts the draft in its
// queue; why is the reason it didn't (notify.Brief*), "" when it did.
func (c *Cell) runBrief(ctx context.Context, key string) (string, error) {
	sp, ok := c.Manifest().Spaces[strings.TrimPrefix(key, "spc_")]
	if !ok || sp.Brief == nil {
		return "", nil // gone since the job was built
	}
	if over, err := c.spaceOver(ctx, key); err != nil || over {
		return notify.BriefSpaceBudget, err
	}
	return c.runAs(ctx, key, ask.DraftBrief, "Weekly brief: "+sp.Brief.Agent, ask.Request{Question: ask.BriefQuestion, Agent: sp.Brief.Agent})
}

// runUpdate drafts a space's weekly status update with the update agent (SB-4102).
func (c *Cell) runUpdate(ctx context.Context, key string) (string, error) {
	if sp, ok := c.Manifest().Spaces[strings.TrimPrefix(key, "spc_")]; !ok || !sp.Updates {
		return "", nil
	}
	if over, err := c.spaceOver(ctx, key); err != nil || over {
		return notify.BriefSpaceBudget, err
	}
	return c.runAs(ctx, key, ask.DraftUpdate, "Update: week of "+time.Now().UTC().AddDate(0, 0, -7).Format(time.DateOnly),
		ask.Request{Question: ask.UpdateQuestion, Agent: ask.AgentUpdate})
}

// runLeader drafts a department's leader's week as a branch ask of its first active
// steward (SB-4503).
func (c *Cell) runLeader(ctx context.Context, key string) (string, error) {
	if !c.Manifest().LeaderWeek(strings.TrimPrefix(key, "spc_")) {
		return "", nil
	}
	if over, err := c.spaceOver(ctx, key); err != nil || over {
		return notify.BriefSpaceBudget, err
	}
	return c.runAsIn(ctx, key, ask.DraftLeaderWeek, "Leader's week: week of "+time.Now().UTC().AddDate(0, 0, -7).Format(time.DateOnly),
		ask.Request{Question: ask.LeaderWeekQuestion, Agent: ask.AgentLeaderWeek}, true)
}

// branchOf is ctx with user's scopes in the teams under space key they are a member of, as
// the auth middleware issues them for ?branch=true (SB-3903).
func (c *Cell) branchOf(ctx context.Context, user, key string) (context.Context, error) {
	tenant := c.Config.Tenant
	groups, err := c.Directory.GroupsOf(ctx, tenant, user)
	if err != nil {
		return nil, err
	}
	teams, err := spaces.Teams(ctx, c.Spaces, tenant, groups, key)
	if err != nil {
		return nil, err
	}
	var branch []tenancy.Scope
	for _, id := range teams {
		in, err := c.ScopeAs(ctx, user, id)
		if err != nil {
			return nil, err
		}
		branch = append(branch, in)
	}
	return tenancy.WithBranch(ctx, branch), nil
}

// budgetShare is the share of space name's budget its branch spent this month (SB-4503).
func (c *Cell) budgetShare(ctx context.Context, name string) (float64, bool, error) {
	m := c.Manifest()
	b := m.Spaces[name].BudgetUSDMonth
	if b <= 0 {
		return 0, false, nil
	}
	spent, err := c.branchSpend(ctx, m, time.Now().UTC().Format("2006-01"))
	if err != nil {
		return 0, false, err
	}
	return spent[name] / float64(b), true, nil
}

// runAs runs req in space key as its first active steward and puts the answer in its queue
// as a draft of kind; why is the reason it didn't (notify.Brief*), "" when it did.
func (c *Cell) runAs(ctx context.Context, key, kind, title string, req ask.Request) (string, error) {
	return c.runAsIn(ctx, key, kind, title, req, false)
}

// runAsIn is runAs, across the space's branch when branch is set.
func (c *Cell) runAsIn(ctx context.Context, key, kind, title string, req ask.Request, branch bool) (string, error) {
	return c.runDraft(ctx, key, kind, title, req, branch, "")
}

// runDraft is runAsIn for one document (KIND:ID) the draft is about (ask.AgentDraftFor).
func (c *Cell) runDraft(ctx context.Context, key, kind, title string, req ask.Request, branch bool, doc string) (string, error) {
	m := c.Manifest()
	agent := req.Agent
	if agent == "" {
		agent = "ask"
	}
	if c.agentKilled(agent) {
		return notify.BriefKilled, nil
	}
	over, err := c.overBudget(ctx)
	if err != nil || over {
		return notify.BriefBudget, err
	}
	user, err := c.firstSteward(ctx, m, key)
	if err != nil || user == "" {
		return notify.BriefNoSteward, err
	}
	s, err := c.ScopeAs(ctx, user, key)
	if err != nil {
		return notify.BriefNoSteward, nil //nolint:nilerr // not active by now: nobody to run it as
	}
	if branch {
		if ctx, err = c.branchOf(ctx, user, key); err != nil {
			return "", err
		}
	}
	_, err = c.Parts().Stewards.AgentDraftFor(ctx, s, kind, title, req, doc)
	switch {
	case errors.Is(err, ask.ErrInvalid):
		return notify.BriefNothing, nil
	case errors.Is(err, ask.ErrKilled): // paused in this space (SB-4702)
		return notify.BriefKilled, nil
	case errors.Is(err, ask.ErrBudget): // at its monthly cap (SB-4703)
		return notify.BriefBudget, nil
	case err != nil:
		return "", err
	}
	return "", nil
}

// welcomeKey names one person's welcome to one space.
func welcomeKey(user, space string) string {
	sum := sha256.Sum256([]byte(user + "|" + space))
	return hex.EncodeToString(sum[:12])
}

func (c *Cell) welcomeDir() string { return filepath.Join(c.Config.State, "welcome") }

// joined queues one welcome brief for each user SCIM added to group, in each space that
// group is a member of; never twice for the same person and space (SB-4006).
func (c *Cell) joined(_ context.Context, group string, users []string) {
	m := c.Manifest()
	for name, sp := range m.Spaces {
		if !contains(sp.Members, "group:"+group) {
			continue
		}
		for _, u := range users {
			k := welcomeKey(u, name)
			if _, err := os.Stat(filepath.Join(c.welcomeDir(), "done", k)); err == nil {
				continue
			}
			b, _ := json.Marshal(notify.Welcome{ID: k, User: u, Space: name})
			if err := os.MkdirAll(c.welcomeDir(), 0o700); err == nil {
				_ = os.WriteFile(filepath.Join(c.welcomeDir(), k+".json"), b, 0o600) // a lost welcome is no harm; SCIM never fails for it
			}
		}
	}
}

// welcomes are the waiting welcome briefs, each from the start-here lists of its space and
// the department above it, resolved as the person now: what they can't open is counted.
func (c *Cell) welcomes(ctx context.Context) ([]notify.Welcome, error) {
	ents, err := os.ReadDir(c.welcomeDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m := c.Manifest()
	var out []notify.Welcome
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(c.welcomeDir(), e.Name()))
		if err != nil {
			return nil, err
		}
		var w notify.Welcome
		if json.Unmarshal(b, &w) != nil || w.ID+".json" != e.Name() {
			continue
		}
		w.Text = c.welcomeText(ctx, m, w)
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// welcomeText is the brief's one line, or "" when there's nothing they can open.
func (c *Cell) welcomeText(ctx context.Context, m manifest.Tenant, w notify.Welcome) string {
	items := m.Effective(w.Space).StartHere
	if _, ok := m.Spaces[w.Space]; !ok || len(items) == 0 {
		return ""
	}
	s, err := c.ScopeAs(ctx, w.User, ask.SpaceKey(w.Space))
	if err != nil {
		return ""
	}
	res, err := c.Parts().Agent.StartHere(ctx, s, items)
	if err != nil || len(res.Items) == 0 {
		return ""
	}
	var titles []string
	for _, it := range res.Items {
		t := it.Title
		if t == "" {
			t = it.Cite
		}
		titles = append(titles, t)
	}
	more := 0
	if len(titles) > notify.MaxDigest {
		titles, more = titles[:notify.MaxDigest], len(titles)-notify.MaxDigest
	}
	text := fmt.Sprintf("Welcome to %s. Start here: %s", w.Space, strings.Join(titles, "; "))
	if more > 0 {
		text += fmt.Sprintf(" and %d more", more)
	}
	return text
}

// welcomed audits a welcome brief, writes its trace, and marks it done.
func (c *Cell) welcomed(ctx context.Context, w notify.Welcome) error {
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: w.User, Kind: "access", Subject: "welcome:" + w.ID, State: "welcomed",
		Reason: "welcome brief to a new member", Space: "spc_" + w.Space}); err != nil {
		return err
	}
	if err := c.Traces.PutTrace(ctx, ask.Trace{ID: "tr_welcome_" + w.ID, Tenant: c.Config.Tenant, User: w.User, Space: "spc_" + w.Space,
		At: time.Now().UTC(), Response: ask.Response{Agent: ask.AgentWelcome}}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(c.welcomeDir(), "done"), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(c.welcomeDir(), "done", w.ID), nil, 0o600); err != nil {
		return err
	}
	return os.Remove(filepath.Join(c.welcomeDir(), w.ID+".json"))
}

// newTranscripts are the transcripts first seen in [from, to), once for each space that
// holds one (SB-4006).
func (c *Cell) newTranscripts(ctx context.Context, from, to time.Time) ([]notify.NewDoc, error) {
	lk, ok := c.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	chunks, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{SeenSince: from})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []notify.NewDoc
	for _, ch := range chunks {
		ref := ch.Kind + ":" + ch.SourceID
		if ask.TypeOf(ch) != "transcript" || !ch.Seen.Before(to) {
			continue
		}
		for _, sp := range ch.Spaces {
			if seen[sp+"|"+ref] {
				continue
			}
			seen[sp+"|"+ref] = true
			title := ch.Title
			if title == "" {
				title = ch.SourceID
			}
			out = append(out, notify.NewDoc{Space: sp, Ref: ref, Title: title, Closed: ask.IsClosed(ch.Text)})
		}
	}
	return out, nil
}

// sourceChanged notes the documents with a new version first seen in [from, to) on the
// agent drafts of the spaces that list on: [source-changed], then makes the day's redrafts
// (SB-4905). Only a version first seen after a draft was made is a change to it.
func (c *Cell) sourceChanged(ctx context.Context, from, to time.Time) error {
	m := c.Manifest()
	var spaces []string
	for _, n := range sortedSpaces(m) {
		if m.Spaces[n].Watches(ask.SourceChanged) {
			spaces = append(spaces, ask.SpaceKey(n))
		}
	}
	if len(spaces) == 0 {
		return nil
	}
	if lk, ok := c.Index.(ports.DocLookup); ok {
		chunks, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{SeenSince: from})
		if err != nil {
			return err
		}
		changed := map[string]time.Time{}
		for _, ch := range chunks {
			ref := ch.Kind + ":" + ch.SourceID
			if ch.Seen.Before(to) && ch.Seen.After(changed[ref]) {
				changed[ref] = ch.Seen
			}
		}
		if _, err := c.Parts().Stewards.NoteSourceChanges(ctx, spaces, changed); err != nil {
			return err
		}
	}
	_, err := c.Parts().Stewards.RedraftChanged(ctx, spaces)
	return err
}

// runDecisions drafts a new transcript's decisions as its space's first active steward.
func (c *Cell) runDecisions(ctx context.Context, d notify.NewDoc) (string, error) {
	if _, ok := c.Manifest().Spaces[strings.TrimPrefix(d.Space, "spc_")]; !ok {
		return "", nil
	}
	return c.runAs(ctx, d.Space, ask.DraftDecisions, "Decisions: "+d.Title,
		ask.Request{Question: ask.DecisionsQuestion, Documents: []string{d.Ref}})
}

// departmental are the spaces a scope's user is a member of, by their groups, that run a
// department agent, with the glossary they use (SB-4007).
func departmental(m manifest.Tenant) func(ctx context.Context, s tenancy.Scope) []ask.DeptSpace {
	return func(_ context.Context, s tenancy.Scope) []ask.DeptSpace {
		var out []ask.DeptSpace
		for name, sp := range m.Spaces {
			member := false
			for _, g := range s.Groups() {
				member = member || contains(sp.Members, "group:"+g)
			}
			if !member {
				continue
			}
			agents := append([]string{}, sp.Agents...)
			sort.Strings(agents)
			for _, a := range agents {
				d, ok := ask.Departments[a]
				if !ok || !contains(m.Agents, a) || d.Template != sp.Template || (d.Restricted && sp.Label != "restricted") {
					continue
				}
				out = append(out, ask.DeptSpace{Name: name, Agent: a, Glossary: m.Effective(name).Glossary})
				break
			}
		}
		return out
	}
}

// accountSpaces are the sales and support spaces a user is a member of now that run their
// account agent, as keys (SB-4605).
func (c *Cell) accountSpaces(ctx context.Context, user string) []string {
	s, err := c.ScopeAs(ctx, user, "company")
	if err != nil {
		return nil
	}
	m := c.Manifest()
	var out []string
	for _, ds := range departmental(m)(ctx, s) {
		if ds.Agent == "account-brief" || ds.Agent == "case-brief" {
			out = append(out, ask.SpaceKey(ds.Name))
		}
	}
	sort.Strings(out)
	return out
}

// policyNotices are the spaces' policies to tell their readers about (SB-4103).
func (c *Cell) policyNotices(ctx context.Context) ([]ask.PolicyNotice, error) {
	m := c.Manifest()
	names := make([]string, 0, len(m.Spaces))
	for n := range m.Spaces {
		names = append(names, n)
	}
	sort.Strings(names)
	return c.Parts().Stewards.PolicyNotices(ctx, names)
}

// sortedSpaces are the manifest's space names, sorted.
func sortedSpaces(m manifest.Tenant) []string {
	out := make([]string, 0, len(m.Spaces))
	for n := range m.Spaces {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// newIncidents are the incident tickets first seen in [from, to): a first chunk whose
// "Labels:" line names one of a space's incident_labels, once for each such space that
// holds it (SB-4204), marked closed when that version is (SB-4604).
func (c *Cell) newIncidents(ctx context.Context, from, to time.Time) ([]notify.NewDoc, error) {
	lk, ok := c.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	chunks, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{SeenSince: from})
	if err != nil {
		return nil, err
	}
	m := c.Manifest()
	seen := map[string]bool{}
	var out []notify.NewDoc
	for _, ch := range chunks {
		if ch.Start != 0 || !ch.Seen.Before(to) || len(ask.IncidentLabels(ch.Text)) == 0 {
			continue
		}
		ref := ch.Kind + ":" + ch.SourceID
		for _, sp := range ch.Spaces {
			s, ok := m.Spaces[strings.TrimPrefix(sp, "spc_")]
			if !ok || seen[sp+"|"+ref] || !ask.IsIncident(ch.Text, s.Incidents()) {
				continue
			}
			seen[sp+"|"+ref] = true
			title := ch.Title
			if title == "" {
				title = ch.SourceID
			}
			out = append(out, notify.NewDoc{Space: sp, Ref: ref, Title: title, Closed: ask.IsClosed(ch.Text)})
		}
	}
	return out, nil
}

// runIncident drafts an incident's brief as its space's first active steward (SB-4204).
func (c *Cell) runIncident(ctx context.Context, d notify.NewDoc) (string, error) {
	if _, ok := c.Manifest().Spaces[strings.TrimPrefix(d.Space, "spc_")]; !ok {
		return "", nil
	}
	return c.runAs(ctx, d.Space, ask.DraftIncident, "Incident: "+d.Title,
		ask.Request{Question: ask.IncidentQuestion(d.Title), Agent: ask.AgentIncident})
}

// runReview drafts a closed incident's review as its space's first active steward, with the
// ticket's action items for that steward to confirm (SB-4604).
func (c *Cell) runReview(ctx context.Context, d notify.NewDoc) (string, error) {
	if _, ok := c.Manifest().Spaces[strings.TrimPrefix(d.Space, "spc_")]; !ok {
		return "", nil
	}
	return c.runDraft(ctx, d.Space, ask.DraftReview, "Review: "+d.Title,
		ask.Request{Question: ask.ReviewQuestion(d.Title), Agent: ask.AgentReview}, false, d.Ref)
}

// runCheckins drafts, for each goal not done whose owner is one active user and a steward
// of its space, the week's check-in as that owner; it returns the goals gone three weeks
// without a published check-in (SB-4207).
func (c *Cell) runCheckins(ctx context.Context) ([]notify.SilentGoal, error) {
	lk, ok := c.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	m := c.Manifest()
	var silent []notify.SilentGoal
	for _, name := range sortedSpaces(m) {
		key := ask.SpaceKey(name)
		chunks, err := lk.Docs(ctx, c.Config.Tenant, ports.DocQuery{Space: key})
		if err != nil {
			return nil, err
		}
		stewards, err := c.stewardPeople(m)(ctx, key)
		if err != nil {
			return nil, err
		}
		queue, err := c.Traces.StewardQueue(ctx, c.Config.Tenant, key)
		if err != nil {
			return nil, err
		}
		for _, ch := range latest(chunks) {
			g, isGoal := ask.ParseGoal(ch.Text)
			if ch.Start != 0 || !isGoal || g.Status == "done" || g.Owner == "" {
				continue
			}
			steward := false
			for _, p := range stewards {
				steward = steward || strings.EqualFold(p.Email, g.Owner)
			}
			ids, err := c.activeByEmail(ctx, g.Owner)
			if err != nil {
				return nil, err
			}
			if len(ids) != 1 || !steward {
				continue // no one owner who could publish it: nothing drafted, nobody told
			}
			goal := ch.Kind + ":" + ch.SourceID
			if ask.Silent(ch.Seen, ask.LastCheckin(queue, goal), time.Now()) {
				silent = append(silent, notify.SilentGoal{Owner: ids[0], Space: key, Goal: goal})
			}
			if c.agentKilled(ask.AgentCheckin) {
				continue
			}
			if over, err := c.spaceOver(ctx, key); err != nil || over {
				if err != nil {
					return nil, err
				}
				continue
			}
			s, err := c.ScopeAs(ctx, ids[0], key)
			if err != nil {
				continue
			}
			title := ch.Title
			if title == "" {
				title = ch.SourceID
			}
			_, err = c.Parts().Stewards.AgentDraftFor(ctx, s, ask.DraftCheckin, "Check-in: "+title,
				ask.Request{Question: ask.CheckinQuestion(title), Agent: ask.AgentCheckin}, goal)
			if err != nil && !errors.Is(err, ask.ErrInvalid) && !errors.Is(err, ask.ErrKilled) && !errors.Is(err, ask.ErrBudget) {
				return nil, err
			}
		}
	}
	return silent, nil
}

// latest keeps each document's newest version's chunks.
func latest(chunks []ports.Chunk) []ports.Chunk {
	newest := map[string]int{}
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		if c.Version > newest[k] {
			newest[k] = c.Version
		}
	}
	var out []ports.Chunk
	for _, c := range chunks {
		if c.Version == newest[c.Kind+":"+c.SourceID] {
			out = append(out, c)
		}
	}
	return out
}

// tell sends one person one notification of kind, if they want that kind (SB-4206).
func (c *Cell) tell(ctx context.Context, user, kind, text string) error {
	send := c.Send
	if send == nil {
		send = c.Notifier()
	}
	if send == nil {
		return errors.New("wire: the cell has nothing to send notifications through")
	}
	u, found, err := c.Directory.User(ctx, c.Config.Tenant, user)
	if err != nil || !found || !u.Active || len(u.Emails) == 0 {
		return fmt.Errorf("wire: no one active to tell: %v", err)
	}
	p, err := c.Traces.NotifyPrefs(ctx, c.Config.Tenant, user)
	if err != nil {
		return err
	}
	if !p.Wants(kind) || p.Paused(time.Now()) {
		return nil
	}
	return send.Send(ctx, notify.Person{ID: u.ID, Email: u.Emails[0]}, notify.Message{Kind: kind, Text: text,
		Link: strings.TrimSuffix(c.Config.BaseURL, "/") + "/?tab=legwork"})
}

// inboxApprovals are the actions waiting on a person's approval group, for their inbox
// (SB-4301): the workflow's name and a link to the approval page, nothing of the payload.
func inboxApprovals(bus *toolbus.Bus) func(ctx context.Context, s tenancy.Scope) ([]ask.InboxItem, error) {
	return func(ctx context.Context, s tenancy.Scope) ([]ask.InboxItem, error) {
		as, err := bus.Pending(ctx, s)
		if err != nil {
			return nil, err
		}
		out := []ask.InboxItem{}
		for _, a := range as {
			out = append(out, ask.InboxItem{Kind: ask.InboxApproval, ID: a.ID, Space: strings.TrimPrefix(a.Space, "spc_"),
				Title: a.Workflow, At: a.ExpiresAt.Add(-approvals.TTL), Due: a.ExpiresAt.UTC().Format(time.DateOnly), Link: "/?action=" + a.ID})
		}
		return out, nil
	}
}

// ownerActive says whether an agent's owner has an active person (SB-4402): an email is
// one active user, group:NAME any active member. A directory error is not active.
func (c *Cell) ownerActive(ctx context.Context, owner string) bool {
	name, isGroup := strings.CutPrefix(strings.TrimSpace(owner), "group:")
	if owner == "" {
		return false
	}
	if !isGroup {
		ids, err := c.activeByEmail(ctx, owner)
		return err == nil && len(ids) == 1
	}
	ids, err := c.Directory.Members(ctx, c.Config.Tenant, name)
	if err != nil {
		return false
	}
	for _, id := range ids {
		u, found, err := c.Directory.User(ctx, c.Config.Tenant, id)
		if err == nil && found && u.Active {
			return true
		}
	}
	return false
}

// catchUpOffer is, once after a person's notification pause ended, when it began, marking
// it offered so the inbox offers Catch me up once (SB-4506).
func (c *Cell) catchUpOffer(ctx context.Context, user string) (time.Time, bool, error) {
	p, err := c.Traces.NotifyPrefs(ctx, c.Config.Tenant, user)
	if err != nil {
		return time.Time{}, false, err
	}
	since, ok := p.CatchUpDue(time.Now())
	if !ok {
		return time.Time{}, false, nil
	}
	p.CatchUpOffered = true
	if err := c.Traces.PutNotifyPrefs(ctx, c.Config.Tenant, user, p); err != nil {
		return time.Time{}, false, err
	}
	return since, true, nil
}

// spaceSkills are space name's skills with its department's (SB-4601), by name; From names
// the department of an inherited one.
func spaceSkills(m manifest.Tenant, name string) []ask.Skill {
	sp, ok := m.Spaces[name]
	if !ok {
		return nil
	}
	eff := m.Effective(name)
	out := make([]ask.Skill, 0, len(eff.Skills))
	for n, sk := range eff.Skills {
		from := ""
		if _, own := sp.Skills[n]; !own {
			from = sp.Parent
		}
		out = append(out, ask.Skill{Name: n, Instruction: sk.Instruction, From: from})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// rescope is user as a scope in space now, with their branch scopes for a branch draft, for
// a redraft (SB-4602).
func (c *Cell) rescope(ctx context.Context, user, space string, branch bool) (context.Context, tenancy.Scope, error) {
	s, err := c.ScopeAs(ctx, user, space)
	if err != nil {
		return ctx, tenancy.Scope{}, err
	}
	if branch {
		if ctx, err = c.branchOf(ctx, user, space); err != nil {
			return ctx, tenancy.Scope{}, err
		}
	}
	return ctx, s, nil
}
