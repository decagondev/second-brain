package mmapp

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/ask"
)

// Follow up in the thread, Mattermost (sb-ask(3), The chat app (Mattermost); SB-5801): an
// outgoing webhook for the channels the bot is in posts every new message to
// POST /chat/mattermost/events with its own token. A reply whose thread's root is the bot's
// post continues that person's conversation in the thread, through the Slack app's rules,
// answered to them alone: an ephemeral post in the thread, or a direct message for anything
// labelled confidential or restricted (LAW-32).

// Threads is the bot's side of a thread on the server.
type Threads interface {
	slackapp.Replier
	Bot(ctx context.Context) (string, error)
	// Root is a post's thread root and the root's author; "" for a post that isn't a reply.
	Root(ctx context.Context, post string) (root, author string, err error)
}

// events handles the outgoing webhook: the token must match, and only replies under the
// bot's own posts are answered. The webhook's reply is always empty.
func (a *App) events(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var in struct {
		Token       string `json:"token"`
		TeamID      string `json:"team_id"`
		ChannelID   string `json:"channel_id"`
		ChannelName string `json:"channel_name"`
		UserID      string `json:"user_id"`
		PostID      string `json:"post_id"`
		Text        string `json:"text"`
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		_ = json.Unmarshal(body, &in)
	} else if f, err := url.ParseQuery(string(body)); err == nil {
		in.Token, in.TeamID, in.ChannelID, in.ChannelName = f.Get("token"), f.Get("team_id"), f.Get("channel_id"), f.Get("channel_name")
		in.UserID, in.PostID, in.Text = f.Get("user_id"), f.Get("post_id"), f.Get("text")
	}
	want, err := a.tokenFor(r.Context())
	if err != nil || want == "" || !hmac.Equal([]byte(in.Token), []byte(want)) || a.TeamID == "" || in.TeamID != a.TeamID {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
	if a.Threads == nil || in.UserID == "" || in.PostID == "" {
		return
	}
	a.run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), slackapp.AnswerTimeout)
		defer cancel()
		bot, err := a.Threads.Bot(ctx)
		if err != nil || bot == "" || in.UserID == bot {
			return
		}
		root, author, err := a.Threads.Root(ctx, in.PostID)
		if err != nil || root == "" || author != bot {
			return // not a reply under the brain's own post
		}
		a.Rules.FollowUp(ctx, slackapp.Followup{User: in.UserID, Channel: in.ChannelID, ChannelName: in.ChannelName, Thread: root, Text: in.Text,
			Key: "mattermost/" + a.TeamID + "/" + in.ChannelID + "/" + root}, a.Threads, func(q string, r ask.Response) string { return Render(q, r).Text })
	})
}

func (a *App) tokenFor(ctx context.Context) (string, error) {
	if a.EventsToken == nil {
		return "", nil
	}
	return a.EventsToken(ctx)
}

var _ Threads = DM{}

// Bot is the bot's own user id.
func (d DM) Bot(ctx context.Context) (string, error) {
	var me struct {
		ID string `json:"id"`
	}
	if err := d.call(ctx, http.MethodGet, "users/me", nil, &me); err != nil {
		return "", err
	}
	return me.ID, nil
}

// Root implements Threads.
func (d DM) Root(ctx context.Context, post string) (string, string, error) {
	var p struct {
		RootID string `json:"root_id"`
	}
	if err := d.call(ctx, http.MethodGet, "posts/"+url.PathEscape(post), nil, &p); err != nil || p.RootID == "" {
		return "", "", err
	}
	var root struct {
		UserID string `json:"user_id"`
	}
	if err := d.call(ctx, http.MethodGet, "posts/"+url.PathEscape(p.RootID), nil, &root); err != nil {
		return "", "", err
	}
	return p.RootID, root.UserID, nil
}

// Ephemeral implements slackapp.Replier: a post only user sees, in the thread.
func (d DM) Ephemeral(ctx context.Context, channel, user, thread, text string) error {
	var out struct {
		ID string `json:"id"`
	}
	return d.call(ctx, http.MethodPost, "posts/ephemeral", map[string]any{"user_id": user,
		"post": map[string]string{"channel_id": channel, "message": text, "root_id": thread}}, &out)
}

// Direct implements slackapp.Replier: a direct message from the bot.
func (d DM) Direct(ctx context.Context, user, text string) error {
	bot, err := d.Bot(ctx)
	if err != nil || bot == "" {
		return err
	}
	var ch struct {
		ID string `json:"id"`
	}
	if err := d.call(ctx, http.MethodPost, "channels/direct", []string{bot, user}, &ch); err != nil || ch.ID == "" {
		return err
	}
	var post struct {
		ID string `json:"id"`
	}
	return d.call(ctx, http.MethodPost, "posts", map[string]string{"channel_id": ch.ID, "message": text}, &post)
}
