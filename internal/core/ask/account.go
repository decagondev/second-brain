package ask

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Account brief (sb-ask(3), Account brief; SB-4605): one CRM account a person can open, in a
// sales or support space, briefed from documents the cell picks by code, each checked as
// them, one cited section a group. Nothing is written to the CRM.

// KindAccount is an account brief, kept like a report.
const KindAccount = "account"

// The account brief limits.
const (
	MaxAccountGroup = 20
	AccountNotes    = 90 * 24 * time.Hour
)

// accountAgents are the agents that brief an account, by their space's template.
var accountAgents = map[string]string{"sales": "account-brief", "support": "case-brief"}

// AccountRequest is POST /v1/account-brief.
type AccountRequest struct {
	Account string `json:"account"` // crm:ID
}

// AccountFacts are an account brief's facts, counted by code, never by the model.
type AccountFacts struct {
	Name      string   `json:"name"`
	Domain    string   `json:"domain,omitempty"`
	Renewal   string   `json:"renewal,omitempty"`
	Tickets   int      `json:"tickets"`
	Notes     int      `json:"notes"`
	Documents int      `json:"documents"`
	Owners    []string `json:"owners"`
}

var (
	nameLineRe      = regexp.MustCompile(`(?im)^\s*name\s*:\s*(.+?)\s*$`)
	domainLineRe    = regexp.MustCompile(`(?im)^\s*domain\s*:\s*(\S+)\s*$`)
	renewalLineRe   = regexp.MustCompile(`(?im)^\s*renewal(?: date)?\s*:\s*(\d{4}-\d{2}-\d{2})`)
	contractTitleRe = regexp.MustCompile(`(?i)\b(contract|proposal|agreement|order form|statement of work)\b`)
)

// accountGroup is one group of an account's documents and its fixed question.
type accountGroup struct {
	name, question string
	docs           []string
}

// AccountBrief briefs s on one CRM account they can open now in s's sales or support space,
// and keeps the brief for them like a report.
func (l Legwork) AccountBrief(ctx context.Context, s tenancy.Scope, req AccountRequest) (ResearchView, error) {
	if err := l.check(s); err != nil {
		return ResearchView{}, err
	}
	agent, err := l.accountAgent(ctx, s)
	if err != nil {
		return ResearchView{}, err
	}
	kind, id, ok := strings.Cut(strings.TrimSpace(req.Account), ":")
	if !ok || kind != "crm" || id == "" {
		return ResearchView{}, fmt.Errorf("%w: account is crm:ID", ErrInvalid)
	}
	lk, ok := l.Agent.Index.(ports.DocLookup)
	if !ok {
		return ResearchView{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	chunks, err := lk.Docs(ctx, l.Agent.Tenant, ports.DocQuery{Space: s.Space()})
	if err != nil {
		return ResearchView{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	var own []ports.Chunk
	for _, c := range latest(chunks) {
		if contains(c.Spaces, s.Space()) { // the space's own documents only
			own = append(own, c)
		}
	}
	key := kind + ":" + id
	rec, open, err := l.openOne(ctx, s, own, key)
	if err != nil {
		return ResearchView{}, err
	}
	if !open {
		return ResearchView{}, fmt.Errorf("%w: no such account you can open in this space", ErrInvalid) // LAW-01
	}
	texts := docTexts(own)
	facts := accountFacts(rec, texts[key])
	groups, err := l.accountGroups(ctx, s, own, texts, key, facts)
	if err != nil {
		return ResearchView{}, err
	}
	facts.Tickets, facts.Notes, facts.Documents = len(groups[1].docs), len(groups[2].docs), len(groups[3].docs)
	r := Research{ID: l.NewID("rs_"), Tenant: l.Agent.Tenant, User: s.User(), Kind: KindAccount, Question: facts.Name, Space: s.Space(),
		State: StateDone, At: l.now(), Account: &facts}
	for _, g := range groups {
		if len(g.docs) == 0 {
			continue
		}
		resp, err := l.Agent.Ask(withOnly(ctx, g.docs), s, Request{Question: g.question, Agent: agent})
		if err != nil {
			return ResearchView{}, err
		}
		claims, _ := citedOnly(resp.Answer.Claims)
		r.Steps = append(r.Steps, g.question)
		r.Sections = append(r.Sections, Section{Question: g.question, Trace: resp.Trace, Claims: claims})
	}
	facts.Owners = sectionOwners(r.Sections)
	r.DoneAt = l.now()
	if err := l.audit(ctx, s.User(), r, KindAccount, hashOf(key)); err != nil {
		return ResearchView{}, err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return ResearchView{}, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	return view(r, r.Sections, 0), nil
}

// accountAgent is the agent that briefs an account in s's space: account-brief in a sales
// space, case-brief in a support space, when the space runs it.
func (l Legwork) accountAgent(ctx context.Context, s tenancy.Scope) (string, error) {
	if s.Space() == "company" || IsPerson(s.Space()) || l.Agent.SpaceOf == nil {
		return "", fmt.Errorf("%w: an account brief runs in a sales or support space", ErrInvalid)
	}
	sp, found, err := l.Agent.SpaceOf(ctx, s.Space())
	if err != nil {
		return "", fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	agent := accountAgents[sp.Template]
	if !found || agent == "" || !contains(sp.Agents, agent) {
		return "", fmt.Errorf("%w: an account brief runs in a sales space that runs account-brief or a support space that runs case-brief", ErrInvalid)
	}
	return agent, nil
}

// openOne is the first chunk of document key among chunks, and whether s can open it now.
func (l Legwork) openOne(ctx context.Context, s tenancy.Scope, chunks []ports.Chunk, key string) (ports.Chunk, bool, error) {
	c, ok := firsts(chunks)[key]
	if !ok {
		return ports.Chunk{}, false, nil
	}
	got, err := l.Agent.allowedDocs(ctx, s, []ports.Chunk{c})
	if err != nil {
		return ports.Chunk{}, false, err
	}
	_, open := got[key]
	return c, open, nil
}

// docTexts are each document's text at the version given, its chunks in order.
func docTexts(chunks []ports.Chunk) map[string]string {
	byDoc := map[string][]ports.Chunk{}
	for _, c := range chunks {
		k := c.Kind + ":" + c.SourceID
		byDoc[k] = append(byDoc[k], c)
	}
	out := map[string]string{}
	for k, cs := range byDoc {
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].Start < cs[j].Start })
		var b strings.Builder
		for _, c := range cs {
			b.WriteString(c.Text)
			b.WriteString("\n")
		}
		out[k] = b.String()
	}
	return out
}

// accountFacts are the name, domain and renewal date the account's record says.
func accountFacts(rec ports.Chunk, text string) AccountFacts {
	f := AccountFacts{Name: strings.TrimSpace(rec.Title), Owners: []string{}}
	if m := nameLineRe.FindStringSubmatch(text); m != nil {
		f.Name = m[1]
	}
	if f.Name == "" {
		f.Name = rec.SourceID
	}
	f.Name = cut(f.Name, 200)
	if m := domainLineRe.FindStringSubmatch(text); m != nil {
		f.Domain = strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(m[1], "https://"), "www."))
	}
	if m := renewalLineRe.FindStringSubmatch(text); m != nil {
		if _, err := time.Parse("2006-01-02", m[1]); err == nil {
			f.Renewal = m[1]
		}
	}
	return f
}

// names says a document's title or text names the account, by its name or domain.
func names(f AccountFacts, title, text string) bool {
	t := strings.ToLower(title + "\n" + text)
	return (len([]rune(f.Name)) >= 3 && strings.Contains(t, strings.ToLower(f.Name))) || (f.Domain != "" && strings.Contains(t, f.Domain))
}

// accountGroups picks, by code, the documents of each group that s can open now, newest
// first and at most MaxAccountGroup each: the record, open tickets, notes and contracts.
func (l Legwork) accountGroups(ctx context.Context, s tenancy.Scope, chunks []ports.Chunk, texts map[string]string, key string, f AccountFacts) ([]accountGroup, error) {
	first := firsts(chunks)
	var keys []string
	for k := range first {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := first[keys[i]], first[keys[j]]
		if !a.Seen.Equal(b.Seen) {
			return a.Seen.After(b.Seen)
		}
		return keys[i] < keys[j]
	})
	since := l.now().Add(-AccountNotes)
	var tickets, docs, notes []string
	for _, k := range keys {
		c := first[k]
		if k == key || !names(f, c.Title, texts[k]) {
			continue
		}
		switch {
		case c.Kind == "tickets":
			if !IsClosed(texts[k]) {
				tickets = append(tickets, k)
			}
		case contractTitleRe.MatchString(c.Title):
			docs = append(docs, k)
		case !c.Seen.Before(since):
			notes = append(notes, k)
		}
	}
	groups := []accountGroup{
		{name: "record", question: "What does the account record say about " + f.Name + "?", docs: []string{key}},
		{name: "tickets", question: "Which open tickets does " + f.Name + " have, and where does each stand?", docs: tickets},
		{name: "notes", question: "What do the last 90 days' notes and documents say about " + f.Name + "?", docs: notes},
		{name: "documents", question: "Which contracts and proposals does " + f.Name + " have, and what do they say?", docs: docs},
	}
	for i := range groups[1:] {
		g := &groups[i+1]
		open, err := l.openDocs(ctx, s, nil, g.docs, chunks)
		if err != nil {
			return nil, err
		}
		if len(open) > MaxAccountGroup {
			open = open[:MaxAccountGroup]
		}
		g.docs = open
	}
	return groups, nil
}

// sectionOwners are the owners at the source of the documents the sections' claims cite.
func sectionOwners(secs []Section) []string {
	out := []string{}
	for _, sec := range secs {
		for _, c := range sec.Claims {
			for _, ci := range c.Cites {
				if ci.Owner != nil && ci.Owner.Email != "" && !contains(out, strings.ToLower(ci.Owner.Email)) {
					out = append(out, strings.ToLower(ci.Owner.Email))
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// MaxPrepAccounts are the account briefs one meeting's prep makes at most.
const MaxPrepAccounts = 3

// prepAccounts briefs user, before a meeting, on the accounts of their sales and support
// spaces they can open whose Domain: is an attendee's address's domain; it returns the
// briefs it made.
func (l Legwork) prepAccounts(ctx context.Context, user string, attendees []string) ([]string, error) {
	if l.AccountSpaces == nil || l.ScopeIn == nil {
		return nil, nil
	}
	domains := map[string]bool{}
	for _, a := range attendees {
		if _, d, ok := strings.Cut(strings.ToLower(strings.TrimSpace(a)), "@"); ok && d != "" {
			domains[d] = true
		}
	}
	if len(domains) == 0 {
		return nil, nil
	}
	lk, ok := l.Agent.Index.(ports.DocLookup)
	if !ok {
		return nil, nil
	}
	var out []string
	for _, key := range l.AccountSpaces(ctx, user) {
		s, err := l.ScopeIn(ctx, user, key)
		if err != nil {
			continue // not active there now
		}
		chunks, err := lk.Docs(ctx, l.Agent.Tenant, ports.DocQuery{Space: key})
		if err != nil {
			return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
		var crm []ports.Chunk
		for _, c := range latest(chunks) {
			if c.Kind == "crm" && contains(c.Spaces, key) {
				crm = append(crm, c)
			}
		}
		texts := docTexts(crm)
		first := firsts(crm)
		var ks []string
		for k := range first {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			if f := accountFacts(first[k], texts[k]); f.Domain == "" || !domains[f.Domain] {
				continue
			}
			v, err := l.AccountBrief(ctx, s, AccountRequest{Account: k})
			if errors.Is(err, ErrInvalid) {
				continue // one they can't open: nothing, and nothing said
			}
			if err != nil {
				return out, err
			}
			if out = append(out, v.ID); len(out) == MaxPrepAccounts {
				return out, nil
			}
		}
	}
	return out, nil
}
