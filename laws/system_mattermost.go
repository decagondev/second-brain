package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"

	"example.com/second-brain/internal/adapters/chatapp/mmapp"
	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	dirmem "example.com/second-brain/internal/adapters/directory/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// The Mattermost slash command (SB-2005) under the chat laws: LAW-01 (an answer cites only
// what the asker can open, and only the asker sees it) and LAW-12 (someone deprovisioned,
// or not in the directory, gets no answer). The real Mattermost app, the Slack app's
// mapping rules, the directory and the cell's ask agent.

// mmSystem's inactive is the world user deprovisioned before they ask; keepInactive is
// a mutation whose directory lookup ignores that.
type mmSystem struct {
	inactive     string
	keepInactive bool
	// share presses Share as the asker and as someone else (SB-2105); leakShare is a
	// mutation whose channel post carries the claims.
	share, leakShare bool
}

// mmCapture keeps the agent's last answer, to look for its claims in a share.
type mmCapture struct {
	slackapp.Asker
	last ask.Response
}

func (c *mmCapture) Ask(ctx context.Context, s tenancy.Scope, req ask.Request) (ask.Response, error) {
	r, err := c.Asker.Ask(ctx, s, req)
	c.last = r
	return r, err
}

type activeAnyway struct{ ports.UserFinder }

func (f activeAnyway) UsersByEmail(ctx context.Context, tenant, email string) ([]ports.User, error) {
	us, err := f.UserFinder.UsersByEmail(ctx, tenant, email)
	for i := range us {
		us[i].Active = true
	}
	return us, err
}

var mmCite = regexp.MustCompile("`([^`@]+)@[0-9]+`")

// ask sends the slash command as the world user's Mattermost account.
func (m mmSystem) ask(w *World, userID, question string) (ChatReply, error) {
	if askPoliciesErr != nil {
		return ChatReply{}, askPoliciesErr
	}
	ctx := context.Background()
	tenant := w.user(userID).Tenant
	dir := dirmem.New()
	emails := map[string]string{}
	for _, u := range w.Users {
		if u.Tenant != tenant {
			continue
		}
		email := u.ID + "@" + tenant + ".example"
		emails[ChatUser(u.ID)] = email
		if err := dir.PutUser(ctx, ports.User{ID: u.ID, Tenant: tenant, UserName: email, Emails: []string{email}, Active: u.ID != m.inactive}); err != nil {
			return ChatReply{}, err
		}
	}
	agent, err := askSystem{}.agent(ctx, w, tenant)
	if err != nil {
		return ChatReply{}, err
	}
	var finder ports.UserFinder = dir
	if m.keepInactive {
		finder = activeAnyway{dir}
	}
	var final mmapp.Message
	var shared []mmapp.Message
	posts := 0
	seen := &mmCapture{Asker: agent}
	app := &mmapp.App{Tenant: tenant, Base: "https://chat." + tenant + ".example", TeamID: "team" + tenant,
		Token: func(context.Context) (string, error) { return "law-command-token", nil },
		Rules: &slackapp.App{Tenant: tenant, Users: finder, Issuer: tenancy.NewIssuer(tenant),
			Email:  func(_ context.Context, mu string) (string, error) { return emails[mu], nil },
			Groups: func(_ context.Context, _, user string) ([]string, error) { return w.user(user).Groups, nil }},
		Agent: seen,
		Post: func(_ context.Context, _ string, msg mmapp.Message) error {
			if msg.ResponseType == "in_channel" {
				if m.leakShare {
					for _, c := range seen.last.Answer.Claims {
						msg.Text += "\n" + c.Text
					}
				}
				shared = append(shared, msg)
				return nil
			}
			final = msg
			posts++
			return nil
		},
		Go: func(f func()) { f() }}
	if m.share {
		app.ActionURL = "https://brain." + tenant + ".example/chat/mattermost/actions"
	}
	mux := http.NewServeMux()
	app.Register(mux)
	body := url.Values{"token": {"law-command-token"}, "team_id": {"team" + tenant}, "user_id": {ChatUser(userID)}, "command": {"/brain"},
		"channel_name": {"town-square"}, "text": {question}, "response_url": {"https://chat." + tenant + ".example/hooks/commands/x1"}}.Encode()
	req := httptest.NewRequest("POST", "/chat/mattermost", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || posts != 1 {
		return ChatReply{}, fmt.Errorf("mattermost: HTTP %d, %d replies", rec.Code, posts)
	}
	rep := ChatReply{VisibleTo: "channel"}
	if final.ResponseType == "ephemeral" {
		rep.VisibleTo = "ephemeral:" + ChatUser(userID)
	}
	for _, c := range mmCite.FindAllStringSubmatch(final.Text, -1) {
		rep.Docs = append(rep.Docs, c[1])
	}
	if m.share && len(final.Attachments) == 1 && len(final.Attachments[0].Actions) == 1 {
		sc := final.Attachments[0].Actions[0].Integration.Context
		other := ChatUser(userID + "x")
		for _, u := range w.Users {
			if u.Tenant == tenant && u.ID != userID && u.ID != m.inactive {
				other = ChatUser(u.ID) // a colleague the directory knows
				break
			}
		}
		for _, who := range []string{other, ChatUser(userID)} { // someone else first
			b, _ := json.Marshal(map[string]any{"user_id": who, "team_id": "team" + tenant, "context": sc})
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest("POST", "/chat/mattermost/actions", strings.NewReader(string(b))))
			if rec.Code != http.StatusOK {
				return rep, fmt.Errorf("share: HTTP %d", rec.Code)
			}
			if who != ChatUser(userID) && len(shared) > 0 {
				return rep, fmt.Errorf("someone other than %s shared their answer to the channel (LAW-01)", userID)
			}
		}
		if len(shared) != 1 {
			return rep, fmt.Errorf("%s pressed Share and %d messages reached the channel", userID, len(shared))
		}
		for _, c := range seen.last.Answer.Claims {
			if len(c.Text) > 8 && (strings.Contains(shared[0].Text, escapeMM(c.Text)) || strings.Contains(shared[0].Text, c.Text)) {
				return rep, fmt.Errorf("the shared message carries a claim's text, not just its cites (LAW-01): %q", c.Text)
			}
		}
		for _, c := range mmCite.FindAllStringSubmatch(shared[0].Text, -1) {
			rep.Shared = append(rep.Shared, c[1])
		}
	}
	if final.Text == mmapp.MsgNoMatch {
		rep.Docs = append(rep.Docs, "(refused)")
	}
	return rep, nil
}

// escapeMM is claim text as the Mattermost app escapes it.
func escapeMM(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune("\\`*_{}[]()<>#+-!|~@:", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// law01MattermostShare: the Share button posts only the cites the asker was shown, only
// when the asker presses it (SB-2105).
func law01MattermostShare(w *World, leak bool) error {
	for _, u := range w.Users {
		rep, err := mmSystem{share: true, leakShare: leak}.ask(w, u.ID, "anything")
		if err != nil {
			return err
		}
		mine := map[string]bool{}
		for _, d := range rep.Docs {
			mine[d] = true
		}
		for _, d := range rep.Shared {
			if !mine[d] {
				return fmt.Errorf("%s's share cites %s, which their answer didn't", u.ID, d)
			}
		}
	}
	return nil
}

// law12Mattermost: a deprovisioned person, or one the directory doesn't know, asks and
// gets the refusal and no answer; everyone else gets an answer.
func law12Mattermost(w *World, keepInactive bool) error {
	for _, u := range w.Users {
		rep, err := mmSystem{inactive: u.ID, keepInactive: keepInactive}.ask(w, u.ID, "anything")
		if err != nil {
			return err
		}
		if len(rep.Docs) != 1 || rep.Docs[0] != "(refused)" {
			return fmt.Errorf("%s was deprovisioned and still got an answer in Mattermost (LAW-12): %v", u.ID, rep.Docs)
		}
	}
	return nil
}
