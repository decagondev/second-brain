package wire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"

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
	return c.runAs(ctx, key, ask.DraftBrief, "Weekly brief: "+sp.Brief.Agent, ask.Request{Question: ask.BriefQuestion, Agent: sp.Brief.Agent})
}

// runUpdate drafts a space's weekly status update with the update agent (SB-4102).
func (c *Cell) runUpdate(ctx context.Context, key string) (string, error) {
	if sp, ok := c.Manifest().Spaces[strings.TrimPrefix(key, "spc_")]; !ok || !sp.Updates {
		return "", nil
	}
	return c.runAs(ctx, key, ask.DraftUpdate, "Update: week of "+time.Now().UTC().AddDate(0, 0, -7).Format(time.DateOnly),
		ask.Request{Question: ask.UpdateQuestion, Agent: ask.AgentUpdate})
}

// runAs runs req in space key as its first active steward and puts the answer in its queue
// as a draft of kind; why is the reason it didn't (notify.Brief*), "" when it did.
func (c *Cell) runAs(ctx context.Context, key, kind, title string, req ask.Request) (string, error) {
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
	_, err = c.Parts().Stewards.AgentDraft(ctx, s, kind, title, req)
	switch {
	case errors.Is(err, ask.ErrInvalid):
		return notify.BriefNothing, nil
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
			out = append(out, notify.NewDoc{Space: sp, Ref: ref, Title: title})
		}
	}
	return out, nil
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
