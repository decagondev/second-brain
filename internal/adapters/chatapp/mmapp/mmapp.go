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
}

// Message is a slash command response (Mattermost's response_type and text).
type Message struct {
	ResponseType string `json:"response_type"` // "ephemeral"
	Text         string `json:"text"`
}

// MsgNoMatch is sent to someone the directory doesn't know as one active user.
const MsgNoMatch = "I can't match your Mattermost account to your company sign-in, so I can't answer. Ask your admin to check your email in the directory."

// Register mounts the endpoint on mux.
func (a *App) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /chat/mattermost", a.command)
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
		_ = a.post(ctx, responseURL, a.answer(ctx, user, channel, question))
	})
}

// answer maps the person and the channel's space, asks, and renders the reply.
func (a *App) answer(ctx context.Context, user, channel, question string) Message {
	s, err := a.Rules.Scope(ctx, user)
	if err != nil {
		return Message{ResponseType: "ephemeral", Text: MsgNoMatch}
	}
	s, msg := a.Rules.InSpace(ctx, s, channel)
	if msg != "" {
		return Message{ResponseType: "ephemeral", Text: msg}
	}
	resp, err := a.Agent.Ask(ctx, s, ask.Request{Question: question})
	if err != nil {
		return Message{ResponseType: "ephemeral", Text: slackapp.ErrText(err)}
	}
	return Render(question, resp)
}

// Render is the ephemeral answer in Mattermost's Markdown: claims with citation links,
// uncited claims labelled, who to ask, the notice, the refused count and the trace.
func Render(question string, r ask.Response) Message {
	var b strings.Builder
	b.WriteString("**" + escape(question) + "**\n")
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
