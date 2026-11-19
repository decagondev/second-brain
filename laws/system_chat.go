package laws

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// The real chat path (SB-308): a signed /ask from the world user's Slack account goes
// through the Slack app, the email-to-directory mapping and the cell's ask agent.
func init() {
	NewChatAsk = func() ChatAsk { return chatSystem{}.ask }
}

type chatSystem struct {
	// wrongPerson is a mutation: the mapping ignores the email and picks the tenant's
	// first directory user.
	wrongPerson bool
}

type firstUser struct{ first ports.User }

func (f firstUser) UsersByEmail(context.Context, string, string) ([]ports.User, error) {
	return []ports.User{f.first}, nil
}

var citeRe = regexp.MustCompile("`([^`@]+)@[0-9]+`")

func (c chatSystem) ask(w *World, userID, question string) (ChatReply, error) {
	if askPoliciesErr != nil {
		return ChatReply{}, askPoliciesErr
	}
	ctx := context.Background()
	asker := w.user(userID)
	tenant := asker.Tenant
	dir := dirmem.New()
	emails := map[string]string{}
	var first ports.User
	for _, u := range w.Users {
		if u.Tenant != tenant {
			continue
		}
		email := u.ID + "@" + tenant + ".example"
		emails[ChatUser(u.ID)] = email
		pu := ports.User{ID: u.ID, Tenant: tenant, UserName: email, Emails: []string{email}, Active: true}
		if first.ID == "" {
			first = pu
		}
		if err := dir.PutUser(ctx, pu); err != nil {
			return ChatReply{}, err
		}
	}
	agent, err := askSystem{}.agent(ctx, w, tenant)
	if err != nil {
		return ChatReply{}, err
	}
	var finder ports.UserFinder = dir
	if c.wrongPerson {
		finder = firstUser{first}
	}
	var final slackapp.Message
	now := time.Unix(1_800_000_000, 0)
	app := &slackapp.App{Tenant: tenant, TeamID: "T" + tenant,
		Secret: func(context.Context) (string, error) { return "law-signing-secret", nil },
		Email:  func(_ context.Context, su string) (string, error) { return emails[su], nil },
		Users:  finder,
		Groups: func(_ context.Context, _, user string) ([]string, error) { return w.user(user).Groups, nil },
		Issuer: tenancy.NewIssuer(tenant),
		Agent:  agent,
		Post:   func(_ context.Context, _ string, m slackapp.Message) error { final = m; return nil },
		Go:     func(f func()) { f() },
		Now:    func() time.Time { return now }}
	mux := http.NewServeMux()
	app.Register(mux)

	body := url.Values{"team_id": {"T" + tenant}, "user_id": {ChatUser(userID)}, "command": {"/ask"},
		"text": {question}, "response_url": {"https://hooks.slack.com/commands/T/1/x"}}.Encode()
	ts := strconv.FormatInt(now.Unix(), 10)
	req := httptest.NewRequest("POST", "/chat/slack/commands", strings.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", slackapp.Sign("law-signing-secret", ts, []byte(body)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return ChatReply{}, fmt.Errorf("chat: HTTP %d", rec.Code)
	}
	rep := ChatReply{VisibleTo: "channel"}
	if final.ResponseType == "ephemeral" {
		rep.VisibleTo = "ephemeral:" + ChatUser(userID)
	}
	for _, m := range citeRe.FindAllStringSubmatch(final.Text, -1) {
		rep.Docs = append(rep.Docs, m[1])
	}
	return rep, nil
}
