package wire

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/mmapp"
	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/adapters/smtpmail"
	"example.com/second-brain/internal/api"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// stewardFacts are who answers each space's questions (ask.SpaceFacts), from m.
func stewardFacts(m manifest.Tenant) ask.SpaceFacts {
	return func(space string) ([]string, string, bool) {
		if space == "company" {
			return []string{m.QuestionsTo}, "", m.QuestionsTo != ""
		}
		sp, ok := m.Spaces[strings.TrimPrefix(space, "spc_")]
		return sp.Stewards, sp.Label, ok && len(sp.Stewards) > 0
	}
}

// Notifier is what notifications go through (sb-ask(3), Notifications): the Mattermost
// bot when the cell has one, else the Slack bot, else email (SB-2307), else none.
func (c *Cell) Notifier() notify.Sender {
	client := &http.Client{Timeout: 10 * time.Second}
	if mm := c.Config.Mattermost; mm.Base != "" {
		return mmapp.DM{Base: mm.Base, Token: slackField(c.Secret("mattermost"), "bot_token"), Client: client}
	}
	if cc, ok := c.Config.Connectors["chat"]; ok {
		return slackapp.DM{Base: cc.Base, Token: slackField(c.Secret("connectors/chat"), "bot_token"), Client: client}
	}
	if e := c.Config.Email; e.Server != "" {
		return smtpmail.Mail{Server: e.Server, From: e.From, Login: c.mailLogin()}
	}
	return nil
}

// claimant says whether a claim's holder is an active member of the stewards now, by the
// directory as it is now (SB-2502, LAW-21), with the email to show.
func (c *Cell) claimant(ctx context.Context, user string, stewards []string) (string, bool, error) {
	u, found, err := c.Directory.User(ctx, c.Config.Tenant, user)
	if err != nil || !found || !u.Active || len(u.Emails) == 0 {
		return "", false, err
	}
	for _, g := range stewards {
		name, ok := strings.CutPrefix(g, "group:")
		if !ok {
			continue
		}
		ids, err := c.Directory.Members(ctx, c.Config.Tenant, name)
		if err != nil {
			return "", false, err
		}
		for _, id := range ids {
			if id == user {
				return u.Emails[0], true, nil
			}
		}
	}
	return "", false, nil
}

// mailLogin reads the mail server's sign-in from vault://tenants/TENANT/email, as
// {"username", "password"}; no such secret is no sign-in.
func (c *Cell) mailLogin() func(context.Context) (string, string, error) {
	return func(ctx context.Context) (string, string, error) {
		raw, err := c.Secret("email")(ctx)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", "", nil
			}
			return "", "", err
		}
		var v struct{ Username, Password string }
		if err := json.Unmarshal([]byte(raw), &v); err != nil || v.Username == "" || v.Password == "" {
			return "", "", errors.New("wire: the email secret needs a username and a password")
		}
		return v.Username, v.Password, nil
	}
}

// accessNote tells a document's owner that someone asked for access to it (SB-2205), by the
// cell's chat bot; nil when the cell has none.
func (c *Cell) accessNote() ask.AccessNote {
	return func(ctx context.Context, owner, asker string, doc ask.Result) error {
		send := c.Send
		if send == nil {
			send = c.Notifier()
		}
		if send == nil {
			return errors.New("wire: the cell has no chat bot to tell owners with")
		}
		us, err := c.Directory.UsersByEmail(ctx, c.Config.Tenant, owner)
		if err != nil {
			return err
		}
		var to []ports.User
		for _, u := range us {
			if u.Active {
				to = append(to, u)
			}
		}
		if len(to) != 1 {
			return nil // nobody single to tell; the asker is told it was sent all the same
		}
		p, err := c.Traces.NotifyPrefs(ctx, c.Config.Tenant, to[0].ID)
		if err != nil {
			return err
		}
		if !p.Wants(notify.KindAccess) {
			return nil
		}
		title, link := doc.Title, doc.URL
		if title == "" {
			title = doc.Cite
		}
		if link == "" {
			link = strings.TrimRight(c.Config.BaseURL, "/") + "/"
		}
		err = send.Send(ctx, notify.Person{ID: to[0].ID, Email: owner}, notify.Message{Kind: notify.KindAccess,
			Text: asker + " asked a question that your document " + title + " answers. Consider sharing it with them.", Link: link})
		if errors.Is(err, notify.ErrNoSuchPerson) {
			return nil
		}
		return err
	}
}

// NotifyJob is the cell's notify job, with the applied manifest.
func (c *Cell) NotifyJob(send notify.Sender) notify.Job {
	m := c.Manifest()
	zone, _ := manifest.DigestZone(m.Digest)
	weekly := zone
	if weekly == nil {
		weekly = time.UTC
	}
	return notify.Job{Zone: weekly, Digest: zone != nil, Askers: c.askers, Changes: c.digestChanges, Watches: c.Traces, Sources: c.Sources, Tenant: c.Config.Tenant, Base: c.Config.BaseURL, Questions: c.Traces, Facts: stewardFacts(m),
		Dir: c.Directory, Prefs: c.Traces, Send: send, Clock: notifyClock{filepath.Join(c.Config.State, "notify")},
		Now: time.Now, Lookback: time.Duration(m.RetentionDays) * 24 * time.Hour,
		Failed: func(kind string) {
			c.Metrics.Add("sb_notify_failed_total", "Notifications that couldn't be sent, by kind (sb-ask(3)).", map[string]string{"kind": kind}, 1)
		}}
}

// notifyClock keeps each part's last run in a file under the cell's state.
type notifyClock struct{ dir string }

func (n notifyClock) Last(_ context.Context, part string) (time.Time, bool, error) {
	b, err := os.ReadFile(filepath.Join(n.dir, part))
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b)))
	return t, err == nil, err
}

func (n notifyClock) Mark(_ context.Context, part string, t time.Time) error {
	if err := os.MkdirAll(n.dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(n.dir, part)
	if err := os.WriteFile(path+".tmp", []byte(t.UTC().Format(time.RFC3339Nano)+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// ScopeAs is the scope a person would have now, in space (or company): their groups read
// from the directory at this moment. Someone not active has none.
func (c *Cell) ScopeAs(ctx context.Context, user, space string) (tenancy.Scope, error) {
	tenant := c.Config.Tenant
	u, found, err := c.Directory.User(ctx, tenant, user)
	if err != nil {
		return tenancy.Scope{}, err
	}
	if !found || !u.Active {
		return tenancy.Scope{}, errors.New("wire: not an active user")
	}
	groups, err := c.Directory.GroupsOf(ctx, tenant, user)
	if err != nil {
		return tenancy.Scope{}, err
	}
	if space == "" || space == "company" {
		return c.Auth.Issuer.Issue(tenant, user, groups)
	}
	return c.Auth.Issuer.IssueIn(tenant, user, groups, space)
}

func (c *Cell) askers(ctx context.Context, from, to time.Time) ([]string, error) {
	ts, err := c.Traces.Between(ctx, c.Config.Tenant, from, to)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, t := range ts {
		if !seen[t.User] {
			seen[t.User] = true
			out = append(out, t.User)
		}
	}
	return out, nil
}

// digestChanges is what changed this week, company-wide, as the person now (SB-2103).
func (c *Cell) digestChanges(ctx context.Context, user string) ([]string, int, error) {
	s, err := c.ScopeAs(ctx, user, "company")
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.Parts().Agent.Changes(ctx, s, 7)
	if err != nil {
		return nil, 0, err
	}
	var titles []string
	for _, r := range resp.Items {
		t := r.Title
		if t == "" {
			t = r.Cite
		}
		titles = append(titles, t)
	}
	more := 0
	if len(titles) > notify.MaxDigest {
		more, titles = len(titles)-notify.MaxDigest, titles[:notify.MaxDigest]
	}
	return titles, more, nil
}

// Sources is the hash of a question's search results as the person, in space, now (SB-2104).
func (c *Cell) Sources(ctx context.Context, user, space, question string) (string, error) {
	s, err := c.ScopeAs(ctx, user, space)
	if err != nil {
		return "", err
	}
	resp, err := c.Parts().Agent.Search(ctx, s, ask.SearchRequest{Query: question})
	if err != nil {
		return "", err
	}
	cites := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		cites = append(cites, r.Cite)
	}
	return ask.SourcesHash(cites), nil
}

// channel names what Notifier sends through.
func (c *Cell) channel() string {
	_, chat := c.Config.Connectors["chat"]
	switch {
	case c.Config.Mattermost.Base != "":
		return "mattermost"
	case chat:
		return "slack"
	case c.Config.Email.Server != "":
		return "email"
	}
	return ""
}

// testTo sends the fixed test notification to p through the cell's real channel (SB-2602).
func (c *Cell) testTo(ctx context.Context, p notify.Person) (bool, string, error) {
	send := c.Notifier()
	if send == nil {
		return false, "", api.ErrNoChannel
	}
	err := send.Send(ctx, p, notify.Message{Kind: notify.KindTest, Text: notify.TestText, Link: strings.TrimSuffix(c.Config.BaseURL, "/") + "/?tab=questions"})
	if errors.Is(err, notify.ErrNoSuchPerson) {
		return false, c.channel(), nil
	}
	return err == nil, c.channel(), err
}

// testSelf is the caller's own test: their directory user, if active with an email.
func (c *Cell) testSelf(ctx context.Context, user string) (bool, string, error) {
	u, found, err := c.Directory.User(ctx, c.Config.Tenant, user)
	if err != nil {
		return false, "", err
	}
	if !found || !u.Active || len(u.Emails) == 0 || u.Emails[0] == "" {
		return false, "", api.ErrNoPerson
	}
	return c.testTo(ctx, notify.Person{ID: u.ID, Email: u.Emails[0]})
}

// testEmail is an operator's test to the one active directory user with email.
func (c *Cell) testEmail(ctx context.Context, email string) (bool, string, error) {
	us, err := c.Directory.UsersByEmail(ctx, c.Config.Tenant, email)
	if err != nil {
		return false, "", err
	}
	var active []ports.User
	for _, u := range us {
		if u.Active {
			active = append(active, u)
		}
	}
	if len(active) != 1 {
		return false, "", api.ErrNoPerson
	}
	return c.testTo(ctx, notify.Person{ID: active[0].ID, Email: email})
}
