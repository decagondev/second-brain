// Package mmapp is the Mattermost entry point to the ask agent (sb-ask(3), The chat app
// (Mattermost); SB-2005): a slash command that calls POST /chat/mattermost. It follows the
// Slack app's rules (slackapp): the person is mapped by email to exactly one active
// directory user at every request, a space's channels ask in that space, and the answer
// goes only to the asker, through Mattermost's response_url.
package mmapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
)

// App is one cell's Mattermost slash command.
type App struct {
	Tenant string
	// Base is the Mattermost server, e.g. https://chat.acme.example: its REST API is read
	// for a user's email, and response URLs must be its own /hooks/commands/ URLs.
	Base   string
	TeamID string // the tenant's Mattermost team
	// Token returns the slash command's token (vault://tenants/TENANT/mattermost,
	// command_token), at call time.
	Token func(ctx context.Context) (string, error)
	// Rules are the Slack app's: its Email (here the Mattermost user's), Users, Groups,
	// Issuer and Spaces map the person and the channel, the same way for both.
	Rules *slackapp.App
	Agent slackapp.Asker
	// Post sends a message to a response_url; nil means a real HTTP POST.
	Post func(ctx context.Context, responseURL string, msg Message) error
	// Go runs the answer after the acknowledgement; nil means a goroutine.
	Go func(func())
	// ActionURL is the cell's POST /chat/mattermost/actions, for the Share button
	// (SB-2105); "" shows no button.
	ActionURL string
}

// Message is a slash command response (Mattermost's response_type, text and attachments).
type Message struct {
	ResponseType string       `json:"response_type"` // "ephemeral" or "in_channel"
	Text         string       `json:"text"`
	Attachments  []Attachment `json:"attachments,omitempty"`
}

// Attachment is a message attachment with interactive buttons.
type Attachment struct {
	Actions []Action `json:"actions"`
}

// Action is one button: Mattermost posts its context to the integration's URL.
type Action struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Integration Integration `json:"integration"`
}

// Integration is where a button's press goes, with what.
type Integration struct {
	URL     string       `json:"url"`
	Context ShareContext `json:"context"`
}

// ShareContext is the Share button's context, signed with the command token, since
// Mattermost doesn't sign action requests.
type ShareContext struct {
	Trace       string `json:"trace"`
	User        string `json:"user"`
	ResponseURL string `json:"response_url"`
	Sig         string `json:"sig"`
}

// Messages for a press of Share.
const (
	MsgShareFail = "Only the person who asked can share this answer."
	MsgShared    = "Shared in the channel."
	MsgShareGone = "This answer can't be shared any more. Ask again to share it."
)

// MsgNoMatch is sent to someone the directory doesn't know as one active user.
const MsgNoMatch = "I can't match your Mattermost account to your company sign-in, so I can't answer. Ask your admin to check your email in the directory."

// Register mounts the endpoint on mux.
func (a *App) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /chat/mattermost", a.command)
	mux.HandleFunc("POST /chat/mattermost/actions", a.action)
}

// verify reads the command and checks its token and team.
func (a *App) verify(r *http.Request) (url.Values, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		return nil, false
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, false
	}
	want, err := a.Token(r.Context())
	if err != nil || want == "" {
		return nil, false
	}
	got := form.Get("token")
	if h, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Token "); ok && got == "" {
		got = h
	}
	if !hmac.Equal([]byte(got), []byte(want)) || a.TeamID == "" || form.Get("team_id") != a.TeamID {
		return nil, false
	}
	return form, true
}

func (a *App) command(w http.ResponseWriter, r *http.Request) {
	form, ok := a.verify(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	question := strings.TrimSpace(form.Get("text"))
	if question == "" {
		reply(w, Message{ResponseType: "ephemeral", Text: "Ask a question: " + form.Get("command") + " What is our refund approval limit?"})
		return
	}
	responseURL := form.Get("response_url")
	if !a.ownURL(responseURL) {
		http.Error(w, "bad response_url", http.StatusBadRequest)
		return
	}
	reply(w, Message{ResponseType: "ephemeral", Text: slackapp.MsgLooking})
	user, channel := form.Get("user_id"), form.Get("channel_name")
	a.run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), slackapp.AnswerTimeout)
		defer cancel()
		m := a.answer(ctx, user, channel, question)
		if trace := m.trace; trace != "" && a.ActionURL != "" {
			if tok, err := a.Token(ctx); err == nil && tok != "" {
				c := ShareContext{Trace: trace, User: user, ResponseURL: responseURL}
				c.Sig = sign(tok, c)
				m.Attachments = []Attachment{{Actions: []Action{{ID: "share", Name: "Share in channel", Integration: Integration{URL: a.ActionURL, Context: c}}}}}
			}
		}
		_ = a.post(ctx, responseURL, m.Message)
	})
}

type answered struct {
	Message
	trace string // the answer's, when there is one to share
}

// answer maps the person and the channel's space, asks, and renders the reply.
func (a *App) answer(ctx context.Context, user, channel, question string) answered {
	s, err := a.Rules.Scope(ctx, user)
	if err != nil {
		return answered{Message: Message{ResponseType: "ephemeral", Text: MsgNoMatch}}
	}
	s, msg := a.Rules.InSpace(ctx, s, channel)
	if msg != "" {
		return answered{Message: Message{ResponseType: "ephemeral", Text: msg}}
	}
	resp, err := a.Agent.Ask(ctx, s, ask.Request{Question: question})
	if err != nil {
		return answered{Message: Message{ResponseType: "ephemeral", Text: slackapp.ErrText(err)}}
	}
	return answered{Render(question, resp), resp.Trace}
}

func sign(token string, c ShareContext) string {
	m := hmac.New(sha256.New, []byte(token))
	_, _ = m.Write([]byte(c.Trace + "\n" + c.User + "\n" + c.ResponseURL))
	return hex.EncodeToString(m.Sum(nil))
}

// action handles the Share button: only the asker, with their own scope read again, and
// only the citation links reach the channel (SB-2105).
func (a *App) action(w http.ResponseWriter, r *http.Request) {
	var p struct {
		UserID  string       `json:"user_id"`
		TeamID  string       `json:"team_id"`
		Context ShareContext `json:"context"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&p); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tok, err := a.Token(r.Context())
	c := p.Context
	if err != nil || tok == "" || c.Sig == "" || !hmac.Equal([]byte(sign(tok, c)), []byte(c.Sig)) ||
		a.TeamID == "" || p.TeamID != a.TeamID || !a.ownURL(c.ResponseURL) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	done := func(text string) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"ephemeral_text": text})
	}
	if p.UserID != c.User {
		done(MsgShareFail)
		return
	}
	s, err := a.Rules.Scope(r.Context(), p.UserID)
	if err != nil {
		done(MsgNoMatch)
		return
	}
	t, err := a.Agent.Trace(r.Context(), s, c.Trace)
	if err != nil || t.User != s.User() {
		done(MsgShareFail)
		return
	}
	if err := a.post(r.Context(), c.ResponseURL, Shared(t)); err != nil {
		done(MsgShareGone)
		return
	}
	done(MsgShared)
}

// Shared is the in-channel message: the citation links only, never claim text, so each
// reader opens the sources under their own access (the Slack rule, SB-308).
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
	text := "Shared an answer (trace " + escape(t.ID) + "). Open the sources to read it with your own access."
	if len(cites) > 0 {
		text += "\nSources: " + links(cites)
	}
	return Message{ResponseType: "in_channel", Text: text}
}

// Render is the ephemeral answer in Mattermost's Markdown: claims with citation links,
// uncited claims labelled, who to ask, the notice, the refused count and the trace.
func Render(question string, r ask.Response) Message {
	var b strings.Builder
	b.WriteString("**" + escape(question) + "**\n")
	for _, w := range ask.Warnings(r) { // before the answer (SB-2701, SB-2704)
		b.WriteString("_" + escape(w) + "_\n")
	}
	for _, c := range r.Answer.Claims {
		if c.Uncited || len(c.Cites) == 0 {
			b.WriteString("- _Uncited:_ " + escape(c.Text) + "\n")
			continue
		}
		b.WriteString("- " + escape(c.Text) + " " + links(c.Cites) + "\n")
	}
	for _, p := range r.People {
		who := p.Email
		if p.Title != "" {
			who += " (" + p.Title + ")"
		}
		b.WriteString("_Ask " + escape(who) + "_\n")
	}
	if r.Notice != nil {
		b.WriteString("\n_" + notices[*r.Notice] + "_\n")
	}
	if r.Refused > 0 {
		fmt.Fprintf(&b, "_%d source(s) were not shown because you can't open them._\n", r.Refused)
	}
	fmt.Fprintf(&b, "_Trace %s_", r.Trace)
	return Message{ResponseType: "ephemeral", Text: b.String()}
}

var notices = map[string]string{
	ask.NoticeNoAccess:    "Some sources you don't have access to may answer this. Ask their owner.",
	ask.NoticeNoSources:   "Nothing indexed answers this yet.",
	ask.NoticeMostUncited: "Most of this answer isn't backed by your sources.",
}

func links(cs []answer.Cite) string {
	var out []string
	for _, c := range cs {
		if c.URL != "" && strings.HasPrefix(c.URL, "https://") {
			out = append(out, "[`"+strings.ReplaceAll(c.Cite, "`", "")+"`]("+strings.NewReplacer("(", "%28", ")", "%29", " ", "%20").Replace(c.URL)+")")
		} else {
			out = append(out, "`"+strings.ReplaceAll(c.Cite, "`", "")+"`")
		}
	}
	return "[" + strings.Join(out, ", ") + "]"
}

// escape keeps text from becoming Markdown, links or mentions.
func escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune("\\`*_{}[]()<>#+-!|~@:", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ownURL accepts only the Mattermost server's own command response URLs.
func (a *App) ownURL(raw string) bool {
	u, err := url.Parse(raw)
	base, berr := url.Parse(a.Base)
	return err == nil && berr == nil && a.Base != "" && u.User == nil && u.Scheme == base.Scheme && u.Host == base.Host &&
		(u.Scheme == "https" || u.Scheme == "http" && strings.HasPrefix(u.Host, "127.0.0.1:")) &&
		strings.HasPrefix(u.Path, strings.TrimSuffix(base.Path, "/")+"/hooks/commands/")
}

func (a *App) run(f func()) {
	if a.Go != nil {
		a.Go(f)
		return
	}
	go f()
}

func (a *App) post(ctx context.Context, responseURL string, m Message) error {
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
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("mmapp: response_url HTTP %d", resp.StatusCode)
	}
	return nil
}

func reply(w http.ResponseWriter, m Message) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// APIEmail returns an Email function that reads a Mattermost user's email with the bot's
// token (GET /api/v4/users/ID). A deleted user, a bot, or a hidden email is no email.
func APIEmail(base string, token func(ctx context.Context) (string, error), client *http.Client) func(ctx context.Context, user string) (string, error) {
	return func(ctx context.Context, user string) (string, error) {
		if user == "" || strings.ContainsAny(user, "/?#%") {
			return "", fmt.Errorf("mmapp: bad user id")
		}
		tok, err := token(ctx)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(base, "/")+"/api/v4/users/"+url.PathEscape(user), nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("mmapp: users: HTTP %d", resp.StatusCode)
		}
		var out struct {
			Email    string `json:"email"`
			DeleteAt int64  `json:"delete_at"`
			IsBot    bool   `json:"is_bot"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
			return "", err
		}
		if out.DeleteAt != 0 || out.IsBot || out.Email == "" {
			return "", fmt.Errorf("mmapp: users: no usable email")
		}
		return out.Email, nil
	}
}
