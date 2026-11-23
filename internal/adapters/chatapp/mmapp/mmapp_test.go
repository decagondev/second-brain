package mmapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

const tenant = "acme-logistics"

type asker struct{ asked []string }

func (a *asker) Ask(_ context.Context, s tenancy.Scope, req ask.Request) (ask.Response, error) {
	a.asked = append(a.asked, s.User()+": "+req.Question)
	return ask.Response{Trace: "tr_1", Answer: ask.Answer{Claims: []answer.Claim{
		{Text: "Refunds over 500 need a lead @channel [x](https://evil.example)", Cites: []answer.Cite{{Cite: "docs:refunds@2", URL: "https://drive.example/d/1"}}},
		{Text: "Ask finance.", Uncited: true},
	}}}, nil
}
func (a *asker) Trace(ctx context.Context, s tenancy.Scope, id string) (ask.Trace, error) {
	r, _ := a.Ask(ctx, s, ask.Request{})
	a.asked = a.asked[:len(a.asked)-1]
	return ask.Trace{ID: id, User: "usr_ann", Response: r}, nil
}

// users is a directory: users by email.
type users []ports.User

func (us users) UsersByEmail(_ context.Context, _, email string) ([]ports.User, error) {
	var out []ports.User
	for _, u := range us {
		if u.Emails[0] == email {
			out = append(out, u)
		}
	}
	return out, nil
}

// fakeMattermost serves GET /api/v4/users/ID for a fixed set of users.
func fakeMattermost(t *testing.T) *httptest.Server {
	users := map[string]map[string]any{
		"u_ann": {"email": "ann@acme.example"},
		"u_old": {"email": "old@acme.example", "delete_at": 1},
		"u_bot": {"email": "bot@acme.example", "is_bot": true},
		"u_eve": {"email": "eve@acme.example"},
		"u_dan": {"email": "dan@acme.example"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer bot-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		u, ok := users[strings.TrimPrefix(r.URL.Path, "/api/v4/users/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(u)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func app(t *testing.T) (*App, *asker, *[]Message, *http.ServeMux) {
	srv := fakeMattermost(t)
	dir := users{{ID: "usr_ann", Tenant: tenant, Emails: []string{"ann@acme.example"}, Active: true},
		{ID: "usr_dan", Tenant: tenant, Emails: []string{"dan@acme.example"}, Active: false}}
	ag := &asker{}
	var posts []Message
	a := &App{Tenant: tenant, Base: srv.URL, TeamID: "team1",
		Token: func(context.Context) (string, error) { return "cmd-token", nil },
		Rules: &slackapp.App{Tenant: tenant, Users: dir, Issuer: tenancy.NewIssuer(tenant), Groups: func(context.Context, string, string) ([]string, error) { return []string{"support"}, nil },
			Email: APIEmail(srv.URL, func(context.Context) (string, error) { return "bot-token", nil }, srv.Client())},
		Agent: ag,
		Post:  func(_ context.Context, _ string, m Message) error { posts = append(posts, m); return nil },
		Go:    func(f func()) { f() }}
	mux := http.NewServeMux()
	a.Register(mux)
	return a, ag, &posts, mux
}

func send(mux *http.ServeMux, v url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/chat/mattermost", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func cmd(a *App, user, text string) url.Values {
	return url.Values{"token": {"cmd-token"}, "team_id": {"team1"}, "user_id": {user}, "command": {"/brain"},
		"text": {text}, "channel_name": {"town-square"}, "response_url": {a.Base + "/hooks/commands/abc"}}
}

// SB-2005: the answer goes only to the asker, mapped by email to one active directory user.
func TestCommandAnswersPrivately(t *testing.T) {
	a, ag, posts, mux := app(t)
	rec := send(mux, cmd(a, "u_ann", "refund limit?"))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ephemeral"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if len(ag.asked) != 1 || ag.asked[0] != "usr_ann: refund limit?" || len(*posts) != 1 {
		t.Fatalf("asked %v, posts %d", ag.asked, len(*posts))
	}
	m := (*posts)[0]
	if m.ResponseType != "ephemeral" || !strings.Contains(m.Text, "[`docs:refunds@2`](https://drive.example/d/1)") || !strings.Contains(m.Text, "_Uncited:_ Ask finance.") {
		t.Fatalf("reply %+v", m)
	}
	if strings.Contains(m.Text, " @channel") || strings.Contains(m.Text, "[x](https://evil.example)") {
		t.Fatalf("claim text became markup: %q", m.Text)
	}
}

// Someone the directory doesn't know, deprovisioned, deleted in Mattermost, or a bot, gets
// the refusal and nothing is asked.
func TestCommandRefusesUnknownPeople(t *testing.T) {
	a, ag, posts, mux := app(t)
	for _, u := range []string{"u_eve", "u_dan", "u_old", "u_bot", "u_nobody", "../u_ann"} {
		if rec := send(mux, cmd(a, u, "refund limit?")); rec.Code != 200 {
			t.Fatalf("%s: %d", u, rec.Code)
		}
	}
	if len(ag.asked) != 0 || len(*posts) != 6 {
		t.Fatalf("asked %v, posts %d", ag.asked, len(*posts))
	}
	for _, m := range *posts {
		if m.Text != MsgNoMatch || m.ResponseType != "ephemeral" {
			t.Fatalf("%+v", m)
		}
	}
}

// A wrong token or team is 401 with no reply; a response_url off the server is refused.
func TestCommandIsVerified(t *testing.T) {
	a, ag, posts, mux := app(t)
	for name, v := range map[string]url.Values{"token": cmd(a, "u_ann", "q"), "team": cmd(a, "u_ann", "q"), "none": cmd(a, "u_ann", "q")} {
		switch name {
		case "token":
			v.Set("token", "cmd-tokeX")
		case "team":
			v.Set("team_id", "team2")
		default:
			v.Del("token")
		}
		if rec := send(mux, v); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: %d", name, rec.Code)
		}
	}
	for _, u := range []string{"https://evil.example/hooks/commands/x", a.Base + "/api/v4/posts", "http://user:pw@" + strings.TrimPrefix(a.Base, "http://") + "/hooks/commands/x"} {
		v := cmd(a, "u_ann", "q")
		v.Set("response_url", u)
		if rec := send(mux, v); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", u, rec.Code)
		}
	}
	if len(ag.asked) != 0 || len(*posts) != 0 {
		t.Fatalf("asked %v, posts %d", ag.asked, len(*posts))
	}
	a.Token = func(context.Context) (string, error) { return "", nil }
	if rec := send(mux, cmd(a, "u_ann", "q")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token in the vault: %d", rec.Code)
	}
}

func press(mux *http.ServeMux, user string, c ShareContext) *httptest.ResponseRecorder {
	b, _ := json.Marshal(map[string]any{"user_id": user, "team_id": "team1", "context": c})
	req := httptest.NewRequest("POST", "/chat/mattermost/actions", strings.NewReader(string(b)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// shareButton asks as ann and returns the Share button's context.
func shareButton(t *testing.T, a *App, mux *http.ServeMux, posts *[]Message) ShareContext {
	t.Helper()
	a.ActionURL = "https://brain.acme.example/chat/mattermost/actions"
	send(mux, cmd(a, "u_ann", "refund limit?"))
	m := (*posts)[len(*posts)-1]
	if len(m.Attachments) != 1 || len(m.Attachments[0].Actions) != 1 || m.Attachments[0].Actions[0].Integration.URL != a.ActionURL {
		t.Fatalf("no Share button: %+v", m)
	}
	return m.Attachments[0].Actions[0].Integration.Context
}

// SB-2105, the Slack Share test against Mattermost: the channel gets the citation links
// only, never the claim text.
func TestShareCarriesNoClaimText(t *testing.T) {
	a, _, posts, mux := app(t)
	c := shareButton(t, a, mux, posts)
	rec := press(mux, "u_ann", c)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), MsgShared) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	m := (*posts)[len(*posts)-1]
	if m.ResponseType != "in_channel" || strings.Contains(m.Text, "Refunds over 500") || strings.Contains(m.Text, "finance") {
		t.Fatalf("share leaked claim text: %+v", m)
	}
	if !strings.Contains(m.Text, "[`docs:refunds@2`](https://drive.example/d/1)") {
		t.Errorf("share lacks the citation links: %s", m.Text)
	}
}

// Only the asker shares, and a forged or altered context is refused.
func TestOnlyTheAskerShares(t *testing.T) {
	a, _, posts, mux := app(t)
	c := shareButton(t, a, mux, posts)
	n := len(*posts)
	if rec := press(mux, "u_eve", c); rec.Code != 200 || !strings.Contains(rec.Body.String(), MsgShareFail) {
		t.Fatalf("someone else: %d %s", rec.Code, rec.Body)
	}
	forged := c
	forged.User = "u_eve"
	if rec := press(mux, "u_eve", forged); rec.Code != 401 {
		t.Fatalf("a forged context: %d", rec.Code)
	}
	elsewhere := c
	elsewhere.ResponseURL = "https://evil.example/hooks/commands/x"
	elsewhere.Sig = sign("cmd-token", elsewhere)
	if rec := press(mux, "u_ann", elsewhere); rec.Code != 401 {
		t.Fatalf("a foreign response URL: %d", rec.Code)
	}
	if len(*posts) != n {
		t.Fatalf("something was posted: %+v", (*posts)[n:])
	}
}
