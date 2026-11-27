package teamsapp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

const tenant = "acme-logistics"

type users []ports.User

func (u users) UsersByEmail(_ context.Context, t, email string) ([]ports.User, error) {
	var out []ports.User
	for _, x := range u {
		for _, e := range x.Emails {
			if strings.EqualFold(e, email) && x.Tenant == t {
				out = append(out, x)
			}
		}
	}
	return out, nil
}

type agent struct {
	reqs  []ask.Request
	label string
}

func (a *agent) Ask(_ context.Context, _ tenancy.Scope, r ask.Request) (ask.Response, error) {
	a.reqs = append(a.reqs, r)
	return ask.Response{Trace: "tr_1", Label: a.label, Answer: ask.Answer{Claims: []answer.Claim{{Text: "Cutoff is 3pm", Cites: []answer.Cite{{Cite: "docs:dock@1", URL: "https://sp.example/dock", Title: "Dock hours"}}}}}}, nil
}
func (a *agent) Trace(context.Context, tenancy.Scope, string) (ask.Trace, error) {
	return ask.Trace{}, ask.ErrNotFound
}

// botFramework is a recorded fake of the Bot Framework: its OpenID keys, the bot token and
// the conversations a reply goes to.
type botFramework struct {
	srv  *httptest.Server
	key  *rsa.PrivateKey
	mu   sync.Mutex
	sent []string // URL path | text
}

func newBotFramework(t *testing.T) *botFramework {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	b := &botFramework{key: k}
	b.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/openid":
			_ = json.NewEncoder(w).Encode(map[string]string{"jwks_uri": b.srv.URL + "/keys"})
		case r.URL.Path == "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": "k1",
				"n": base64.RawURLEncoding.EncodeToString(k.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes())}}})
		case r.URL.Path == "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "bot-token", "expires_in": 3600})
		case r.Header.Get("Authorization") != "Bearer bot-token":
			http.Error(w, "no", http.StatusUnauthorized)
		case r.URL.Path == "/v3/conversations":
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "a:personal-sam"})
		default:
			body, _ := io.ReadAll(r.Body)
			var m struct{ Text string }
			_ = json.Unmarshal(body, &m)
			b.mu.Lock()
			b.sent = append(b.sent, r.URL.Path+"|"+m.Text)
			b.mu.Unlock()
		}
	}))
	t.Cleanup(b.srv.Close)
	return b
}

func (b *botFramework) sign(t *testing.T, claims map[string]any, kid string) string {
	enc := func(v any) string {
		j, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(j)
	}
	head := enc(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	body := enc(claims)
	sum := sha256.Sum256([]byte(head + "." + body))
	sig, err := rsa.SignPKCS1v15(rand.Reader, b.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return head + "." + body + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func setup(t *testing.T) (*App, *agent, *botFramework, *http.ServeMux) {
	b := newBotFramework(t)
	ag := &agent{}
	dir := users{{ID: "sam", Tenant: tenant, Emails: []string{"sam@acme.example"}, Active: true}}
	rules := &slackapp.App{Tenant: tenant, Users: dir, Issuer: tenancy.NewIssuer(tenant),
		Email: func(_ context.Context, id string) (string, error) {
			if id == "aad-sam" {
				return "sam@acme.example", nil
			}
			return "", errors.New("no such user")
		},
		Groups: func(context.Context, string, string) ([]string, error) { return []string{"staff"}, nil }}
	a := &App{Tenant: tenant, AppID: "bot-app", OpenID: b.srv.URL + "/openid", Login: b.srv.URL + "/token", HTTP: b.srv.Client(), Rules: rules, Agent: ag,
		Creds: func(context.Context) (string, error) { return `{"app_id":"bot-app","app_password":"pw"}`, nil },
		Go:    func(f func()) { f() }}
	mux := http.NewServeMux()
	a.Register(mux)
	return a, ag, b, mux
}

func (b *botFramework) activity(kind, text string) Activity {
	var act Activity
	act.Type, act.ID, act.Text, act.ServiceURL, act.ChannelID = "message", "act1", text, b.srv.URL, "msteams"
	act.From.ID, act.From.AADObjectID, act.Recipient.ID = "29:sam", "aad-sam", "28:bot"
	act.Conversation.ID, act.Conversation.Type, act.Conversation.TenantID = "19:ops@thread.tacv2;messageid=100", kind, "t-acme"
	if kind == "channel" {
		act.ChannelData.Channel = &struct {
			Name string `json:"name"`
		}{Name: "ops"}
	} else {
		act.Conversation.ID = "a:personal-sam"
	}
	return act
}

func send(t *testing.T, mux *http.ServeMux, act Activity, tok string) int {
	body, _ := json.Marshal(act)
	r := httptest.NewRequest("POST", "/chat/teams", bytes.NewReader(body))
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w.Code
}

// SB-5904: an activity signed by the Bot Framework for this bot is answered; anything else
// is refused. A channel mention gets its answer in the personal chat and a line in the reply
// chain, and its reply chain is one conversation; a personal chat is answered there.
func TestAskInTeams(t *testing.T) {
	_, ag, b, mux := setup(t)
	claims := func(aud, svc string) map[string]any {
		return map[string]any{"iss": Issuer, "aud": aud, "exp": time.Now().Add(time.Hour).Unix(), "nbf": time.Now().Add(-time.Minute).Unix(), "serviceurl": svc}
	}
	good := b.sign(t, claims("bot-app", b.srv.URL), "k1")
	ch := b.activity("channel", "<at>Brain</at> when is the cutoff?")
	if code := send(t, mux, ch, ""); code != http.StatusUnauthorized {
		t.Fatalf("unsigned: %d", code)
	}
	for _, bad := range []string{b.sign(t, claims("other-bot", b.srv.URL), "k1"), b.sign(t, claims("bot-app", "https://evil.example"), "k1"),
		b.sign(t, claims("bot-app", b.srv.URL), "k9"), good[:len(good)-4] + "AAAA"} {
		if code := send(t, mux, ch, bad); code != http.StatusUnauthorized {
			t.Fatalf("a bad token was taken: %d", code)
		}
	}
	if len(ag.reqs) != 0 {
		t.Fatal("asked on a refused activity")
	}
	if code := send(t, mux, ch, good); code != http.StatusOK {
		t.Fatalf("signed: %d", code)
	}
	if len(ag.reqs) != 1 || ag.reqs[0].Question != "when is the cutoff?" || ag.reqs[0].Thread != "teams/t-acme/19:ops@thread.tacv2;messageid=100" {
		t.Fatalf("asked %+v", ag.reqs)
	}
	sent := strings.Join(b.sent, "\n")
	if !strings.Contains(sent, "/v3/conversations/a:personal-sam/activities|") || !strings.Contains(sent, "Cutoff is 3pm") ||
		!strings.Contains(sent, "messageid=100/activities/act1|"+MsgSentDirect) {
		t.Fatalf("sent %v", b.sent)
	}
	for _, s := range b.sent {
		if strings.Contains(s, "thread.tacv2") && strings.Contains(s, "Cutoff") {
			t.Fatalf("an answer was posted in the channel: %s", s)
		}
	}
	b.sent = nil
	ag.label = "confidential"
	if code := send(t, mux, b.activity("personal", "the plan?"), good); code != http.StatusOK || len(b.sent) != 1 ||
		!strings.Contains(b.sent[0], "/v3/conversations/a:personal-sam/activities/act1|") || ag.reqs[1].Thread != "" {
		t.Fatalf("personal: %v %+v", b.sent, ag.reqs)
	}
}
