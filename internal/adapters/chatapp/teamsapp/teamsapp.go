// Package teamsapp is the Microsoft Teams entry point to the ask agent (sb-ask(3), The chat
// app (Teams); SB-5904), beside the Slack and Mattermost apps and under their rules. Teams
// posts each activity to POST /chat/teams with a token Microsoft signs; the app checks it
// against Microsoft's published keys with the standard library and refuses anything else.
// A mention in a channel or group chat, and a message in a personal chat, is asked as the
// one active directory user the sender's Entra id maps to. Teams has no message only one
// person sees, so an answer always goes to the person's personal chat with the bot: in a
// channel the reply chain gets only a line saying so. A follow-up in a reply chain continues
// the person's own conversation in it (SB-5801), and a confidential answer never goes
// anywhere but that personal chat (LAW-32).
package teamsapp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/mmapp"
	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/tenancy"
)

// The Bot Framework's addresses and the token's issuer.
const (
	DefaultOpenID = "https://login.botframework.com/v1/.well-known/openidconfiguration"
	DefaultLogin  = "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token"
	Issuer        = "https://api.botframework.com"
	BotScope      = "https://api.botframework.com/.default"
	// Skew is how far a token's times may be off; KeysFor how long the keys are kept.
	Skew    = 5 * time.Minute
	KeysFor = 24 * time.Hour
)

// Replies the app sends.
const (
	MsgSentDirect = "I've sent you the answer in our personal chat."
	MsgNoChannel  = "I can't tell which channel this is, so I can't answer here. Ask me in our personal chat."
)

// App is one cell's Teams bot.
type App struct {
	Tenant string
	// AppID is the bot's Microsoft app id: the audience every token must carry.
	AppID string
	// Creds return vault://tenants/TENANT/teams, {"app_id", "app_password"}, at call time.
	Creds func(ctx context.Context) (string, error)
	// OpenID and Login are the Bot Framework's (a fake's in tests); "" is Microsoft's.
	OpenID, Login string
	HTTP          *http.Client
	// Rules are the Slack app's: Email (here an Entra object id's email), Users, Groups,
	// Issuer and Spaces map the person and the channel as for Slack and Mattermost.
	Rules *slackapp.App
	Agent slackapp.Asker
	Go    func(func())
	Now   func() time.Time
	// Masked is told how many strings of a reply were masked (SB-5102); nil is told nothing.
	Masked func(ctx context.Context, n int)

	mu       sync.Mutex
	keys     map[string]*rsa.PublicKey
	keysAt   time.Time
	botToken string
	botUntil time.Time
}

// Register mounts the endpoint.
func (a *App) Register(mux *http.ServeMux) { mux.HandleFunc("POST /chat/teams", a.activity) }

func (a *App) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *App) http() *http.Client {
	if a.HTTP == nil {
		return http.DefaultClient
	}
	return a.HTTP
}

// Activity is the part of a Bot Framework activity the app reads.
type Activity struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Text       string `json:"text"`
	ServiceURL string `json:"serviceUrl"`
	ChannelID  string `json:"channelId"`
	From       struct {
		ID          string `json:"id"`
		AADObjectID string `json:"aadObjectId"`
	} `json:"from"`
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Conversation struct {
		ID       string `json:"id"`
		Type     string `json:"conversationType"`
		TenantID string `json:"tenantId"`
	} `json:"conversation"`
	ChannelData struct {
		Channel *struct {
			Name string `json:"name"`
		} `json:"channel"`
	} `json:"channelData"`
}

func (a *App) activity(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var act Activity
	if json.Unmarshal(body, &act) != nil {
		http.Error(w, "bad activity", http.StatusBadRequest)
		return
	}
	tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || a.verify(r.Context(), tok, act.ServiceURL) != nil || !serviceURL(act.ServiceURL) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
	if act.Type != "message" || act.ChannelID != "msteams" || act.From.AADObjectID == "" {
		return
	}
	run := a.Go
	if run == nil {
		run = func(f func()) { go f() }
	}
	run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), slackapp.AnswerTimeout)
		defer cancel()
		a.answer(ctx, act)
	})
}

// serviceURL is a Bot Framework service Teams replies through: https, or the loopback in a
// test.
func serviceURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.User == nil && (u.Scheme == "https" || u.Scheme == "http" && strings.HasPrefix(u.Host, "127.0.0.1:"))
}

// mentionTag is Teams' mention of the bot, which isn't part of the question.
var mentionTag = regexp.MustCompile(`(?s)<at>.*?</at>`)

// answer asks the activity's question as its person, and answers them in their personal
// chat with the bot.
func (a *App) answer(ctx context.Context, act Activity) {
	q := strings.TrimSpace(mentionTag.ReplaceAllString(act.Text, ""))
	if q == "" {
		return
	}
	personal := act.Conversation.Type == "personal"
	channel := "directmessage"
	if !personal {
		if act.ChannelData.Channel == nil || act.ChannelData.Channel.Name == "" {
			_ = a.reply(ctx, act, MsgNoChannel)
			return
		}
		channel = strings.TrimPrefix(act.ChannelData.Channel.Name, "#")
	}
	text := a.ask(ctx, act, q, channel, personal)
	if personal {
		_ = a.reply(ctx, act, text)
		return
	}
	if err := a.direct(ctx, act, text); err == nil {
		_ = a.reply(ctx, act, MsgSentDirect)
	}
}

func (a *App) ask(ctx context.Context, act Activity, q, channel string, personal bool) string {
	s, err := a.Rules.Scope(ctx, act.From.AADObjectID)
	if err != nil {
		return slackapp.MsgNoMatch
	}
	s, msg := a.Rules.InSpace(ctx, s, channel)
	if msg != "" {
		return msg
	}
	req := ask.Request{Question: q, Language: a.Rules.AnswerLanguage()}
	if !personal { // a reply chain is one conversation per person (SB-5801)
		req.Thread = "teams/" + act.Conversation.TenantID + "/" + act.Conversation.ID
	}
	resp, err := a.Agent.Ask(tenancy.WithClient(ctx, tenancy.ClientChat), s, req)
	if err != nil {
		return slackapp.ErrText(err)
	}
	text := mmapp.Render(q, resp).Text
	if line := a.Rules.NoticeLine(ctx); line != "" {
		text = "**" + line + "**\n" + text
	}
	return text
}

// reply posts text in the activity's conversation, under it.
func (a *App) reply(ctx context.Context, act Activity, text string) error {
	u := strings.TrimRight(act.ServiceURL, "/") + "/v3/conversations/" + url.PathEscape(act.Conversation.ID) + "/activities/" + url.PathEscape(act.ID)
	return a.post(ctx, u, map[string]any{"type": "message", "textFormat": "markdown", "text": text, "replyToId": act.ID}, nil)
}

// direct opens (or finds) the personal chat with the activity's sender and posts text there.
func (a *App) direct(ctx context.Context, act Activity, text string) error {
	var conv struct {
		ID string `json:"id"`
	}
	base := strings.TrimRight(act.ServiceURL, "/") + "/v3/conversations"
	if err := a.post(ctx, base, map[string]any{"bot": map[string]string{"id": act.Recipient.ID}, "members": []map[string]string{{"id": act.From.ID}},
		"tenantId": act.Conversation.TenantID, "isGroup": false}, &conv); err != nil {
		return err
	}
	if conv.ID == "" {
		return errors.New("teamsapp: no personal chat")
	}
	return a.post(ctx, base+"/"+url.PathEscape(conv.ID)+"/activities", map[string]any{"type": "message", "textFormat": "markdown", "text": text}, nil)
}

func (a *App) post(ctx context.Context, u string, body any, out any) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(body)
	if masked, n := ingest.MaskJSON(b); n > 0 { // SB-5102: nothing sensitive goes out
		b = masked
		if a.Masked != nil {
			a.Masked(ctx, n)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.http().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("teamsapp: %s: HTTP %d", u, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
	}
	return nil
}

// token is the bot's token for the Bot Framework, from its app id and password.
func (a *App) token(ctx context.Context) (string, error) {
	a.mu.Lock()
	if a.botToken != "" && a.now().Before(a.botUntil) {
		t := a.botToken
		a.mu.Unlock()
		return t, nil
	}
	a.mu.Unlock()
	raw, err := a.Creds(ctx)
	if err != nil {
		return "", err
	}
	var cr struct {
		AppID    string `json:"app_id"`
		Password string `json:"app_password"`
	}
	if json.Unmarshal([]byte(raw), &cr) != nil || cr.AppID == "" || cr.Password == "" {
		return "", errors.New("teamsapp: the credentials aren't {app_id, app_password}")
	}
	login := a.Login
	if login == "" {
		login = DefaultLogin
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {cr.AppID}, "client_secret": {cr.Password}, "scope": {BotScope}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, login, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.http().Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t) != nil || t.AccessToken == "" {
		return "", fmt.Errorf("teamsapp: bot token: HTTP %d", resp.StatusCode)
	}
	a.mu.Lock()
	a.botToken, a.botUntil = t.AccessToken, a.now().Add(time.Duration(t.ExpiresIn)*time.Second-time.Minute)
	a.mu.Unlock()
	return t.AccessToken, nil
}

// ErrToken is an activity whose token isn't Microsoft's, for this bot, now.
var ErrToken = errors.New("teamsapp: the token isn't valid for this bot")

// verify checks an RS256 token against the Bot Framework's keys: the signature, the issuer,
// this bot as the audience, its times, and that it names the activity's serviceUrl.
func (a *App) verify(ctx context.Context, tok, service string) error {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 || a.AppID == "" {
		return ErrToken
	}
	var head struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if decode(parts[0], &head) != nil || head.Alg != "RS256" || head.Kid == "" {
		return ErrToken
	}
	key, err := a.key(ctx, head.Kid)
	if err != nil {
		return ErrToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return ErrToken
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig) != nil {
		return ErrToken
	}
	var claims struct {
		Iss        string          `json:"iss"`
		Aud        json.RawMessage `json:"aud"`
		Exp        int64           `json:"exp"`
		Nbf        int64           `json:"nbf"`
		ServiceURL string          `json:"serviceurl"`
	}
	if decode(parts[1], &claims) != nil || claims.Iss != Issuer || !audience(claims.Aud, a.AppID) {
		return ErrToken
	}
	now := a.now()
	if claims.Exp == 0 || now.After(time.Unix(claims.Exp, 0).Add(Skew)) || (claims.Nbf != 0 && now.Add(Skew).Before(time.Unix(claims.Nbf, 0))) {
		return ErrToken
	}
	if claims.ServiceURL != "" && strings.TrimRight(claims.ServiceURL, "/") != strings.TrimRight(service, "/") {
		return ErrToken
	}
	return nil
}

func decode(seg string, v any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func audience(raw json.RawMessage, want string) bool {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one == want
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		for _, x := range many {
			if x == want {
				return true
			}
		}
	}
	return false
}

// key is the signing key kid, from the keys kept for KeysFor, read again once for a kid
// they don't hold.
func (a *App) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	a.mu.Lock()
	k, ok := a.keys[kid]
	fresh := a.now().Sub(a.keysAt) < KeysFor
	a.mu.Unlock()
	if ok && fresh {
		return k, nil
	}
	keys, err := a.fetchKeys(ctx)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.keys, a.keysAt = keys, a.now()
	a.mu.Unlock()
	if k, ok := keys[kid]; ok {
		return k, nil
	}
	return nil, ErrToken
}

func (a *App) getJSON(ctx context.Context, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := a.http().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teamsapp: %s: HTTP %d", u, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(v)
}

func (a *App) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	oid := a.OpenID
	if oid == "" {
		oid = DefaultOpenID
	}
	var conf struct {
		JWKS string `json:"jwks_uri"`
	}
	if err := a.getJSON(ctx, oid, &conf); err != nil || !strings.HasPrefix(conf.JWKS, "https://") && !strings.HasPrefix(conf.JWKS, "http://127.0.0.1:") {
		return nil, ErrToken
	}
	var set struct {
		Keys []struct {
			Kty, Kid, N, E string
		} `json:"keys"`
	}
	if err := a.getJSON(ctx, conf.JWKS, &set); err != nil {
		return nil, err
	}
	out := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil || len(e) == 0 || len(e) > 4 {
			continue
		}
		ev := 0
		for _, b := range e {
			ev = ev<<8 | int(b)
		}
		out[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: ev}
	}
	return out, nil
}
