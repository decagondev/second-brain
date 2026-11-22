package slackapp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

const (
	tenant = "acme-logistics"
	team   = "T123"
	secret = "signing-secret-123"
	hook   = "https://hooks.slack.com/commands/T123/1/abc"
)

var now = time.Unix(1_800_000_000, 0)

type users []ports.User

func (u users) UsersByEmail(_ context.Context, t, email string) ([]ports.User, error) {
	var out []ports.User
	for _, x := range u {
		if x.Tenant == t && strings.EqualFold(x.Emails[0], email) {
			out = append(out, x)
		}
	}
	return out, nil
}

type agent struct {
	asked  []string // scope user ids
	spaces []string // scope spaces
	err    error
}

func (a *agent) Ask(_ context.Context, s tenancy.Scope, r ask.Request) (ask.Response, error) {
	a.asked = append(a.asked, s.User())
	a.spaces = append(a.spaces, s.Space())
	if a.err != nil {
		return ask.Response{}, a.err
	}
	n := ask.NoticeMostUncited
	return ask.Response{Trace: "tr_1", Notice: &n, Refused: 2, Answer: ask.Answer{Claims: []answer.Claim{
		{Text: "Limit is 500 <!channel>", Cites: []answer.Cite{{Cite: "docs:refunds@3", URL: "https://drive.example/r?v=3", Title: "Refund policy",
			Owner: &answer.Person{Email: "ann@acme-logistics.example", Title: "Support lead"}}, {Cite: "tickets:LIN-7@2"}}},
		{Text: "Ask finance.", Cites: []answer.Cite{}, Uncited: true},
	}}, People: []ask.PersonRef{{Email: "ann@acme-logistics.example", Title: "Support lead", Cites: 1}}}, nil
}

func (a *agent) Trace(_ context.Context, s tenancy.Scope, id string) (ask.Trace, error) {
	if id != "tr_1" || s.User() != "alice" {
		return ask.Trace{}, ask.ErrNotFound
	}
	r, _ := a.Ask(context.Background(), s, ask.Request{})
	return ask.Trace{ID: "tr_1", User: "alice", Tenant: tenant, Response: r}, nil
}

func newApp(dir users, ag *agent) (*App, *[]Message) {
	var posted []Message
	email := map[string]string{"UALICE": "alice@acme.example", "UBOB": "bob@acme.example", "UCAROL": "carol@acme.example", "UDAVE": "dave@acme.example"}
	return &App{Tenant: tenant, TeamID: team,
		Secret: func(context.Context) (string, error) { return secret, nil },
		Email: func(_ context.Context, u string) (string, error) {
			if e, ok := email[u]; ok {
				return e, nil
			}
			return "", errors.New("no such user")
		},
		Users:  dir,
		Groups: func(context.Context, string, string) ([]string, error) { return []string{"staff"}, nil },
		Issuer: tenancy.NewIssuer(tenant),
		Agent:  ag,
		Post: func(_ context.Context, u string, m Message) error {
			if u != hook {
				return errors.New("wrong url")
			}
			posted = append(posted, m)
			return nil
		},
		Go:  func(f func()) { f() },
		Now: func() time.Time { return now }}, &posted
}

var directory = users{
	{ID: "alice", Tenant: tenant, Emails: []string{"alice@acme.example"}, Active: true},
	{ID: "bob1", Tenant: tenant, Emails: []string{"bob@acme.example"}, Active: true},
	{ID: "bob2", Tenant: tenant, Emails: []string{"BOB@acme.example"}, Active: true},
	{ID: "carol", Tenant: tenant, Emails: []string{"carol@acme.example"}, Active: false},
	{ID: "dave", Tenant: "beta-freight", Emails: []string{"dave@acme.example"}, Active: true},
}

func send(a *App, path string, form url.Values, at time.Time, sig string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	a.Register(mux)
	body := form.Encode()
	ts := strconv.FormatInt(at.Unix(), 10)
	if sig == "" {
		sig = Sign(secret, ts, []byte(body))
	}
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", sig)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func command(user, text string) url.Values {
	return url.Values{"team_id": {team}, "user_id": {user}, "command": {"/ask"}, "text": {text}, "response_url": {hook}}
}

func TestAnswerIsEphemeralWithLinks(t *testing.T) {
	ag := &agent{}
	a, posted := newApp(directory, ag)
	w := send(a, "/chat/slack/commands", command("UALICE", "refund limit?"), now, "")
	var ack Message
	_ = json.Unmarshal(w.Body.Bytes(), &ack)
	if w.Code != 200 || ack.ResponseType != "ephemeral" || ack.Text != MsgLooking {
		t.Fatalf("ack %d %+v", w.Code, ack)
	}
	if len(*posted) != 1 || len(ag.asked) != 1 || ag.asked[0] != "alice" {
		t.Fatalf("posted %d, asked as %v", len(*posted), ag.asked)
	}
	m := (*posted)[0]
	for _, want := range []string{"<https://drive.example/r?v=3|docs:refunds@3>", "`tickets:LIN-7@2`", "_Uncited:_ Ask finance.", "&lt;!channel&gt;", "2 source(s) were not shown", "Trace tr_1", "Most of this answer",
		"_Ask ann@acme-logistics.example (Support lead), owner of Refund policy_"} {
		if !strings.Contains(m.Text, want) {
			t.Errorf("reply lacks %q:\n%s", want, m.Text)
		}
	}
	if m.ResponseType != "ephemeral" || strings.Contains(m.Text, "<!channel>") {
		t.Errorf("reply must be ephemeral and escaped: %+v", m)
	}
}

func TestNoAnswerWithoutExactlyOneActiveUser(t *testing.T) {
	for _, u := range []string{"UBOB", "UCAROL", "UDAVE", "UNOBODY"} { // two matches, inactive, other tenant, unknown
		ag := &agent{}
		a, posted := newApp(directory, ag)
		send(a, "/chat/slack/commands", command(u, "q"), now, "")
		if len(ag.asked) != 0 || len(*posted) != 1 || (*posted)[0].Text != MsgNoMatch {
			t.Errorf("%s: asked %v, posted %+v", u, ag.asked, *posted)
		}
	}
}

func TestRefusesUnverifiedRequests(t *testing.T) {
	cases := map[string]func() *httptest.ResponseRecorder{}
	ag := &agent{}
	a, posted := newApp(directory, ag)
	cases["bad signature"] = func() *httptest.ResponseRecorder {
		return send(a, "/chat/slack/commands", command("UALICE", "q"), now, "v0=deadbeef")
	}
	cases["too old"] = func() *httptest.ResponseRecorder {
		return send(a, "/chat/slack/commands", command("UALICE", "q"), now.Add(-6*time.Minute), "")
	}
	cases["other workspace"] = func() *httptest.ResponseRecorder {
		f := command("UALICE", "q")
		f.Set("team_id", "T999")
		return send(a, "/chat/slack/commands", f, now, "")
	}
	for name, c := range cases {
		if w := c(); w.Code != 401 {
			t.Errorf("%s: %d", name, w.Code)
		}
	}
	f := command("UALICE", "q")
	f.Set("response_url", "https://evil.example/hook")
	if w := send(a, "/chat/slack/commands", f, now, ""); w.Code != 400 {
		t.Errorf("foreign response_url: %d", w.Code)
	}
	if len(ag.asked) != 0 || len(*posted) != 0 {
		t.Error("an unverified request reached the agent")
	}
}

func share(user string) url.Values {
	p, _ := json.Marshal(map[string]any{"team": map[string]string{"id": team}, "user": map[string]string{"id": user},
		"response_url": hook, "actions": []map[string]string{{"action_id": "share", "value": "tr_1"}}})
	return url.Values{"payload": {string(p)}}
}

func TestShareCarriesNoClaimText(t *testing.T) {
	a, posted := newApp(directory, &agent{})
	if w := send(a, "/chat/slack/interactions", share("UALICE"), now, ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	m := (*posted)[0]
	if m.ResponseType != "in_channel" || strings.Contains(m.Text, "Limit is 500") || strings.Contains(m.Text, "finance") {
		t.Fatalf("share leaked claim text: %+v", m)
	}
	if !strings.Contains(m.Text, "docs:refunds@3") || !strings.Contains(m.Text, "tickets:LIN-7@2") {
		t.Errorf("share lacks the citation links: %s", m.Text)
	}
}

func TestOnlyTheAskerShares(t *testing.T) {
	dir := append(users{{ID: "erin", Tenant: tenant, Emails: []string{"erin@acme.example"}, Active: true}}, directory...)
	a, posted := newApp(dir, &agent{})
	base := a.Email
	a.Email = func(ctx context.Context, u string) (string, error) {
		if u == "UERIN" {
			return "erin@acme.example", nil
		}
		return base(ctx, u)
	}
	send(a, "/chat/slack/interactions", share("UERIN"), now, "")
	if len(*posted) != 1 || (*posted)[0].Text != MsgShareFail || (*posted)[0].ResponseType != "ephemeral" {
		t.Fatalf("%+v", *posted)
	}
}

func TestAgentErrorsAreEphemeral(t *testing.T) {
	a, posted := newApp(directory, &agent{err: ask.ErrKilled})
	send(a, "/chat/slack/commands", command("UALICE", "q"), now, "")
	if m := (*posted)[0]; m.ResponseType != "ephemeral" || !strings.Contains(m.Text, "switched off") {
		t.Fatalf("%+v", m)
	}
}

func TestWebEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer xoxb-test" || r.URL.Path != "/users.info" {
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
			return
		}
		switch r.URL.Query().Get("user") {
		case "UALICE":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"profile":{"email":"alice@acme.example"}}}`))
		case "UBOT":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"is_bot":true,"profile":{"email":"bot@acme.example"}}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":false,"error":"user_not_found"}`))
		}
	}))
	defer srv.Close()
	f := WebEmail(srv.URL, func(context.Context) (string, error) { return "xoxb-test", nil }, srv.Client())
	if e, err := f(context.Background(), "UALICE"); err != nil || e != "alice@acme.example" {
		t.Fatalf("%q %v", e, err)
	}
	for _, u := range []string{"UBOT", "UNOBODY"} {
		if _, err := f(context.Background(), u); err == nil {
			t.Errorf("%s should have no usable email", u)
		}
	}
}
