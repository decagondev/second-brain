// Package slackapp is the Slack entry point to the ask agent (sb-ask(3), The chat app;
// SB-308): the /ask slash command and the Share button. Replies go through Slack's
// response_url, ephemeral to the asker; the bot needs only the `commands` scope.
package slackapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"

	"example.com/second-brain/internal/core/ingest"
)

// MaxSkew is how old a signed request may be.
const MaxSkew = 5 * time.Minute

// AnswerTimeout bounds one answer.
const AnswerTimeout = 60 * time.Second

// Asker is the ask agent as the app uses it.
type Asker interface {
	Ask(ctx context.Context, s tenancy.Scope, req ask.Request) (ask.Response, error)
	Trace(ctx context.Context, s tenancy.Scope, id string) (ask.Trace, error)
}

// App is one cell's Slack app.
type App struct {
	Tenant string
	TeamID string // the tenant's Slack workspace
	// Secret returns the signing secret (vault://tenants/TENANT/connectors/chat), at call time.
	Secret func(ctx context.Context) (string, error)
	// Email returns a Slack user's profile email (users.info); see WebEmail.
	Email  func(ctx context.Context, slackUser string) (string, error)
	Users  ports.UserFinder
	Groups func(ctx context.Context, tenant, user string) ([]string, error)
	Issuer tenancy.Issuer // the cell's one issuer (SB-108)
	Agent  Asker
	// Spaces is the cell's space registry (SB-1007): a space's channels ask in it. nil is
	// a cell without spaces, where every channel asks company-wide.
	Spaces ports.SpaceStore
	// Post sends a message to a response_url; nil means a real HTTPS POST.
	Post func(ctx context.Context, responseURL string, msg Message) error
	// Masked is told how many strings of a reply were masked (SB-5102); nil is told nothing.
	Masked func(ctx context.Context, n int)
	// Go runs the answer after the acknowledgement; nil means a goroutine.
	Go  func(func())
	Now func() time.Time
	// Language is the manifest's language, which answers are written in (SB-3003); nil
	// leaves it to the model.
	Language func() string
	// Notice is the cell's notice to everyone as a first line, "Notice: …", or "" when none
	// stands (SB-3207); nil is none.
	Notice func(ctx context.Context) string
	// Saver keeps saved threads, and Thread reads one (SB-3703); nil saves none.
	Saver  Saver
	Thread func(ctx context.Context, channel, ts string) ([]RawMessage, error)
}

// NoticeLine is the first line every answer starts with while an admin's notice stands.
func (a *App) NoticeLine(ctx context.Context) string {
	if a == nil || a.Notice == nil {
		return ""
	}
	return a.Notice(ctx)
}

// WithNotice puts the notice first in m, as plain text.
func (a *App) WithNotice(ctx context.Context, m Message) Message {
	line := a.NoticeLine(ctx)
	if line == "" {
		return m
	}
	line = "*" + escape(line) + "*\n"
	m.Text = line + m.Text
	if len(m.Blocks) > 0 && m.Blocks[0].Text != nil {
		t := *m.Blocks[0].Text
		t.Text = line + t.Text
		m.Blocks = append([]Block{{Type: m.Blocks[0].Type, Text: &t, Elements: m.Blocks[0].Elements}}, m.Blocks[1:]...)
	}
	return m
}

// AnswerLanguage is the language chat answers are asked in.
func (a *App) AnswerLanguage() string {
	if a == nil || a.Language == nil {
		return ""
	}
	return a.Language()
}

// Message is a response_url payload.
type Message struct {
	ResponseType    string  `json:"response_type"` // "ephemeral" or "in_channel"
	Text            string  `json:"text"`
	ReplaceOriginal bool    `json:"replace_original,omitempty"`
	Blocks          []Block `json:"blocks,omitempty"`
}

// Block is the subset of Slack Block Kit the app sends.
type Block struct {
	Type     string    `json:"type"`
	Text     *Text     `json:"text,omitempty"`
	Elements []Element `json:"elements,omitempty"`
}

// Text is a Block Kit text object.
type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Element is a Block Kit button.
type Element struct {
	Type     string `json:"type"`
	Text     *Text  `json:"text,omitempty"`
	ActionID string `json:"action_id,omitempty"`
	Value    string `json:"value,omitempty"`
}

// Replies the app sends (no content, no names).
const (
	MsgLooking = "Looking that up…"
	MsgNoMatch = "I can't match your Slack account to your company sign-in, so I can't answer. Ask your admin to check your email in the directory."
	MsgUsage   = "Ask a question: /ask What is our refund approval limit?"
	// MsgNotMember answers a /ask in a space's channel from someone outside the space.
	MsgNotMember = "This channel asks the brain of a space you aren't a member of, so I can't answer here. Ask its stewards for membership, or ask somewhere else for a company-wide answer."
	// MsgSpacesDown answers when the space registry can't say which space a channel is in.
	MsgSpacesDown = "Answers are unavailable right now. Try again in a minute."
	MsgShareFail  = "Only the person who asked can share this answer."
)

// ErrNoMatch is a chat user who isn't exactly one active directory user.
var ErrNoMatch = errors.New("slackapp: chat user doesn't map to one active directory user")

// Register mounts the endpoints on mux.
func (a *App) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /chat/slack/commands", a.command)
	mux.HandleFunc("POST /chat/slack/interactions", a.interaction)
}

// verify reads and checks a Slack request; it returns the form on success.
func (a *App) verify(r *http.Request) (url.Values, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		return nil, false
	}
	ts := r.Header.Get("X-Slack-Request-Timestamp")
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, false
	}
	if d := a.Now().Sub(time.Unix(sec, 0)); d > MaxSkew || d < -MaxSkew {
		return nil, false
	}
	secret, err := a.Secret(r.Context())
	if err != nil || secret == "" {
		return nil, false
	}
	if !hmac.Equal([]byte(Sign(secret, ts, body)), []byte(r.Header.Get("X-Slack-Signature"))) {
		return nil, false
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, false
	}
	return form, true
}

// Sign is Slack's v0 request signature (used by tests and the fake).
func Sign(secret, ts string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte("v0:" + ts + ":"))
	m.Write(body)
	return "v0=" + hex.EncodeToString(m.Sum(nil))
}

func (a *App) command(w http.ResponseWriter, r *http.Request) {
	form, ok := a.verify(r)
	if !ok || form.Get("team_id") != a.TeamID || a.TeamID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	question := strings.TrimSpace(form.Get("text"))
	if question == "" {
		reply(w, Message{ResponseType: "ephemeral", Text: MsgUsage})
		return
	}
	user, responseURL := form.Get("user_id"), form.Get("response_url")
	if !slackURL(responseURL) {
		http.Error(w, "bad response_url", http.StatusBadRequest)
		return
	}
	reply(w, Message{ResponseType: "ephemeral", Text: MsgLooking})
	a.run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), AnswerTimeout)
		defer cancel()
		_ = a.post(ctx, responseURL, a.WithNotice(ctx, a.answer(ctx, user, form.Get("channel_name"), question)))
	})
}

// answer maps the person and the channel's space, asks, and renders an ephemeral reply.
func (a *App) answer(ctx context.Context, slackUser, channel, question string) Message {
	s, err := a.Scope(ctx, slackUser)
	if err != nil {
		return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: MsgNoMatch}
	}
	s, msg := a.InSpace(ctx, s, channel)
	if msg != "" {
		return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: msg}
	}
	resp, err := a.Agent.Ask(tenancy.WithClient(ctx, tenancy.ClientChat), s, ask.Request{Question: question, Language: a.AnswerLanguage()})
	if err != nil {
		return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: ErrText(err)}
	}
	return Render(question, resp)
}

// InSpace puts the ask in the space whose channels list this channel (sb-ask(3), The chat
// app). An unmapped channel, or a direct message, stays company-wide; a mapped channel for
// someone outside its space is refused, never asked company-wide instead.
func (a *App) InSpace(ctx context.Context, s tenancy.Scope, channel string) (tenancy.Scope, string) {
	if a.Spaces == nil || channel == "" || channel == "directmessage" || channel == "privategroup" {
		return s, ""
	}
	all, err := a.Spaces.Spaces(ctx, a.Tenant)
	if err != nil {
		return tenancy.Scope{}, MsgSpacesDown
	}
	for _, sp := range all {
		for _, ch := range sp.Channels {
			if ch != "#"+channel {
				continue
			}
			id, err := spaces.Resolve(ctx, a.Spaces, a.Tenant, s.Groups(), sp.Name)
			if errors.Is(err, spaces.ErrUnavailable) {
				return tenancy.Scope{}, MsgSpacesDown
			}
			if err != nil {
				return tenancy.Scope{}, MsgNotMember
			}
			in, err := a.Issuer.IssueIn(a.Tenant, s.User(), s.Groups(), id)
			if err != nil {
				return tenancy.Scope{}, MsgNoMatch
			}
			return in.WithAttrs(s.Attrs()), ""
		}
	}
	return s, ""
}

// Scope maps a chat user to exactly one active directory user and issues their scope.
// It is checked on every request and never cached (sb-identity(7), Chat users).
func (a *App) Scope(ctx context.Context, slackUser string) (tenancy.Scope, error) {
	if slackUser == "" {
		return tenancy.Scope{}, ErrNoMatch
	}
	email, err := a.Email(ctx, slackUser)
	if err != nil || email == "" {
		return tenancy.Scope{}, ErrNoMatch
	}
	users, err := a.Users.UsersByEmail(ctx, a.Tenant, email)
	if err != nil || len(users) != 1 || !users[0].Active || users[0].Tenant != a.Tenant {
		return tenancy.Scope{}, ErrNoMatch
	}
	groups, err := a.Groups(ctx, a.Tenant, users[0].ID)
	if err != nil {
		return tenancy.Scope{}, ErrNoMatch
	}
	s, err := a.Issuer.Issue(a.Tenant, users[0].ID, groups)
	if err != nil {
		return tenancy.Scope{}, ErrNoMatch
	}
	// The directory's attributes, as the web path reads them at every request (SB-1401).
	return s.WithAttrs(users[0].Attributes()), nil
}

// Render is the ephemeral answer: claims with citation links, uncited claims labelled,
// then notice, refused count and trace, and a Share button carrying only the trace id.
func Render(question string, r ask.Response) Message {
	var b strings.Builder
	for _, w := range ask.Warnings(r) { // first, so they are read before the answer (SB-2701, SB-2704)
		b.WriteString("_" + w + "_\n")
	}
	if p := r.Pinned; p != nil { // SB-3601: first, then the brain's answer
		b.WriteString("_" + escape(ask.PinnedLead(*p)) + "_ " + escape(p.Text) + " (" + escape(strings.Join(p.Cites, ", ")) + ")\n")
	}
	for _, c := range r.Answer.Claims {
		if c.Uncited || len(c.Cites) == 0 {
			b.WriteString("• _Uncited:_ " + escape(c.Text) + "\n")
			continue
		}
		lead := ""
		if c.Disagree { // SB-3304
			lead = "_Sources disagree:_ "
		}
		b.WriteString("• " + lead + escape(c.Text) + " " + links(c.Cites) + "\n")
	}
	for _, p := range r.People {
		who := p.Email
		if p.Title != "" {
			who += " (" + p.Title + ")"
		}
		b.WriteString("_Ask " + escape(who) + ", owner of " + escape(ownedTitles(r, p.Email)) + "_\n")
	}
	if v := r.Verified; v != nil { // SB-3607
		b.WriteString("_" + ask.VerifiedLine(*v) + "_\n")
	}
	if r.Notice != nil {
		b.WriteString("\n_" + notices[*r.Notice] + "_\n")
	}
	if r.Refused > 0 {
		fmt.Fprintf(&b, "_%d source(s) were not shown because you can't open them._\n", r.Refused)
	}
	if m := ask.MarkLine(r.Agent, r.Release); m != "" { // SB-4706, LAW-28
		b.WriteString("_" + escape(m) + "_\n")
	}
	fmt.Fprintf(&b, "_Trace %s_", r.Trace)
	text := b.String()
	return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: text, Blocks: []Block{
		{Type: "section", Text: &Text{Type: "mrkdwn", Text: "*" + escape(question) + "*\n" + text}},
		{Type: "actions", Elements: []Element{{Type: "button", Text: &Text{Type: "plain_text", Text: "Share in channel"}, ActionID: "share", Value: r.Trace}}},
	}}
}

// ownedTitles names the documents of one owner the answer cites (SB-1702), by title or cite.
func ownedTitles(r ask.Response, email string) string {
	var names []string
	seen := map[string]bool{}
	for _, c := range r.Answer.Claims {
		for _, ci := range c.Cites {
			if ci.Owner == nil || ci.Owner.Email != email || seen[ci.Cite] {
				continue
			}
			seen[ci.Cite] = true
			n := ci.Title
			if n == "" {
				n = ci.Cite
			}
			names = append(names, n)
		}
	}
	return strings.Join(names, ", ")
}

var notices = map[string]string{
	ask.NoticeNoAccess:    "Some sources you don't have access to may answer this. Ask their owner.",
	ask.NoticeNoSources:   "Nothing indexed answers this yet.",
	ask.NoticeMostUncited: "Most of this answer isn't backed by your sources.",
}

func links(cs []answer.Cite) string {
	var out []string
	for _, c := range cs {
		label := escape(c.Cite)
		if c.URL != "" && strings.HasPrefix(c.URL, "https://") {
			out = append(out, "<"+escapeURL(c.URL)+"|"+label+">")
		} else {
			out = append(out, "`"+label+"`")
		}
		if m := ask.CiteMark(c.GeneratedBy, c.ApprovedBy, c.Edited, c.AskedBy, c.EditedBy); m != "" { // SB-4706
			out[len(out)-1] += " (" + escape(m) + ")"
		}
	}
	return "[" + strings.Join(out, ", ") + "]"
}

// escape keeps text from becoming Slack markup (links, mentions).
func escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func escapeURL(u string) string {
	return strings.NewReplacer("|", "%7C", ">", "%3E", "<", "%3C").Replace(u)
}

// ErrText is an ask error as a chat reply: its reason, never its detail.
func ErrText(err error) string {
	var over *ask.RateError
	switch {
	case errors.Is(err, ask.ErrInvalid):
		return "That question can't be asked: " + strings.TrimPrefix(err.Error(), "ask: EINVAL: ")
	case errors.Is(err, ask.ErrKilled):
		return "Answers are switched off for now."
	case errors.As(err, &over):
		return fmt.Sprintf("You've asked a lot in the last minute. Wait %d seconds, then ask again.", int(math.Ceil(max(over.Wait.Seconds(), 1))))
	default:
		return "Answers are unavailable right now. Try again in a minute."
	}
}

// interaction handles the Share button: only the asker, with their own scope.
func (a *App) interaction(w http.ResponseWriter, r *http.Request) {
	form, ok := a.verify(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var p struct {
		Type        string              `json:"type"`
		CallbackID  string              `json:"callback_id"`
		Team        struct{ ID string } `json:"team"`
		User        struct{ ID string } `json:"user"`
		ResponseURL string              `json:"response_url"`
		Channel     struct{ ID, Name string }
		Message     struct {
			TS       string `json:"ts"`
			ThreadTS string `json:"thread_ts"`
		} `json:"message"`
		Actions []struct {
			ActionID string `json:"action_id"`
			Value    string `json:"value"`
		} `json:"actions"`
	}
	if json.Unmarshal([]byte(form.Get("payload")), &p) != nil || p.Team.ID != a.TeamID || a.TeamID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if p.Type == "message_action" && p.CallbackID == SaveCallback { // SB-3703
		a.saveAction(w, p.User.ID, p.Channel.ID, p.Channel.Name, p.Message.TS, p.Message.ThreadTS, p.ResponseURL)
		return
	}
	if len(p.Actions) != 1 || p.Actions[0].ActionID != "share" || !slackURL(p.ResponseURL) {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	trace := p.Actions[0].Value
	a.run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), AnswerTimeout)
		defer cancel()
		s, err := a.Scope(ctx, p.User.ID)
		if err != nil {
			_ = a.post(ctx, p.ResponseURL, Message{ResponseType: "ephemeral", Text: MsgNoMatch})
			return
		}
		t, err := a.Agent.Trace(ctx, s, trace)
		if err != nil || t.User != s.User() {
			_ = a.post(ctx, p.ResponseURL, Message{ResponseType: "ephemeral", Text: MsgShareFail})
			return
		}
		_ = a.post(ctx, p.ResponseURL, Shared(t))
	})
}

// Shared is the in-channel message: the citation links only, never claim text, so each
// reader opens the sources under their own access.
func Shared(t ask.Trace) Message {
	seen := map[string]bool{}
	var cites []answer.Cite
	for _, c := range t.Response.Answer.Claims {
		for _, ci := range c.Cites {
			if !seen[ci.Cite] {
				seen[ci.Cite] = true
				cites = append(cites, ci)
			}
		}
	}
	text := "Shared an answer (trace " + t.ID + "). Open the sources to read it with your own access."
	if t.Response.Agent != "" { // SB-4706, LAW-28
		text = "Shared an answer written by " + escape(t.Response.Agent) + markVersion(t.Response.Release) + " (trace " + t.ID + "). Open the sources to read it with your own access."
	}
	if len(cites) > 0 {
		text += "\nSources: " + links(cites)
	}
	return Message{ResponseType: "in_channel", Text: text}
}

func (a *App) run(f func()) {
	if a.Go != nil {
		a.Go(f)
		return
	}
	go f()
}

func (a *App) post(ctx context.Context, responseURL string, m Message) error {
	m = a.mask(ctx, m) // SB-5102: nothing sensitive goes out
	if a.Post != nil {
		return a.Post(ctx, responseURL, m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responseURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("slackapp: response_url HTTP %d", resp.StatusCode)
	}
	return nil
}

// slackURL accepts only Slack's own response URLs (no request forgery to other hosts).
func slackURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "hooks.slack.com" && u.User == nil
}

func reply(w http.ResponseWriter, m Message) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// WebEmail returns an Email function that calls Slack's users.info with the bot token
// (users:read.email). Base is https://slack.com/api unless a test points it at a fake.
func WebEmail(base string, token func(ctx context.Context) (string, error), client *http.Client) func(ctx context.Context, user string) (string, error) {
	if base == "" {
		base = "https://slack.com/api"
	}
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context, user string) (string, error) {
		tok, err := token(ctx)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/users.info?"+url.Values{"user": {user}}.Encode(), nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		var out struct {
			OK   bool   `json:"ok"`
			Err  string `json:"error"`
			User struct {
				Deleted bool `json:"deleted"`
				IsBot   bool `json:"is_bot"`
				Profile struct {
					Email string `json:"email"`
				} `json:"profile"`
			} `json:"user"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
			return "", err
		}
		if !out.OK || out.User.Deleted || out.User.IsBot || out.User.Profile.Email == "" {
			return "", fmt.Errorf("slackapp: users.info: no usable email (%s)", out.Err)
		}
		return out.User.Profile.Email, nil
	}
}

// markVersion is ", version V", or "" without one.
func markVersion(v string) string {
	if v == "" {
		return ""
	}
	return ", version " + escape(v)
}

// mask is m with every string masked by ingest.MaskOut (sb-ingest(7), Nothing sensitive goes
// out; SB-5102), and Masked told how many changed.
func (a *App) mask(ctx context.Context, m Message) Message {
	b, err := json.Marshal(m)
	if err != nil {
		return m
	}
	out, n := ingest.MaskJSON(b)
	if n == 0 {
		return m
	}
	var masked Message
	if err := json.Unmarshal(out, &masked); err != nil {
		return Message{ResponseType: m.ResponseType, Text: "This reply couldn't be sent safely."}
	}
	if a.Masked != nil {
		a.Masked(ctx, n)
	}
	return masked
}
