package slackapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

// Save to the brain (sb-ask(3), The chat app (Slack); SB-3703): a message shortcut sends
// the message's thread to the stewards of the channel's space.

// SaveCallback is the message shortcut's callback id.
const SaveCallback = "save_to_brain"

// Saver keeps a saved thread for the stewards.
type Saver interface {
	SaveThread(ctx context.Context, s tenancy.Scope, link string, ms []ask.ThreadMessage) (ask.StewardQuestion, error)
}

// RawMessage is one chat message as the chat app reads it: its author's chat id and text.
type RawMessage struct {
	User, Text string
}

// Replies the save sends.
const (
	MsgSaveFail = "The thread couldn't be saved just now. Try again in a minute."
)

// SavedTo is the reply once a thread is saved.
func SavedTo(q ask.StewardQuestion) string {
	return "Sent to the stewards of " + strings.TrimPrefix(q.Space, "spc_") + "."
}

// SaveThread maps the person and the channel's space, reads the thread and saves it: what
// the Slack shortcut and the Mattermost command share.
func (a *App) SaveThread(ctx context.Context, chatUser, channel, link string, read func(ctx context.Context) ([]RawMessage, error)) string {
	if a.Saver == nil {
		return MsgSaveFail
	}
	s, err := a.Scope(ctx, chatUser)
	if err != nil {
		return MsgNoMatch
	}
	s, msg := a.InSpace(ctx, s, channel)
	if msg != "" {
		return msg
	}
	raw, err := read(ctx)
	if err != nil || len(raw) == 0 {
		return MsgSaveFail
	}
	emails := map[string]string{}
	var ms []ask.ThreadMessage
	for i, m := range raw {
		if i == ask.MaxThreadMessages {
			break
		}
		e, ok := emails[m.User]
		if !ok && a.Email != nil {
			if got, err := a.Email(ctx, m.User); err == nil {
				e = got
			}
			emails[m.User] = e
		}
		ms = append(ms, ask.ThreadMessage{Email: e, Text: m.Text})
	}
	q, err := a.Saver.SaveThread(ctx, s, link, ms)
	if err != nil {
		return ErrText(err)
	}
	return SavedTo(q)
}

// saveAction handles the shortcut: an immediate 200, then the reply through response_url.
func (a *App) saveAction(w http.ResponseWriter, user, channelID, channelName, ts, threadTS, responseURL string) {
	w.WriteHeader(http.StatusOK)
	if !slackURL(responseURL) || a.Thread == nil {
		return
	}
	root := threadTS
	if root == "" {
		root = ts
	}
	link := "https://slack.com/archives/" + url.PathEscape(channelID) + "/p" + strings.ReplaceAll(root, ".", "")
	a.run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), AnswerTimeout)
		defer cancel()
		text := a.SaveThread(ctx, user, channelName, link, func(ctx context.Context) ([]RawMessage, error) { return a.Thread(ctx, channelID, root) })
		_ = a.post(ctx, responseURL, Message{ResponseType: "ephemeral", Text: text})
	})
}

// WebThread reads a thread with conversations.replies and the bot's token.
func WebThread(base string, token func(ctx context.Context) (string, error), client *http.Client) func(ctx context.Context, channel, ts string) ([]RawMessage, error) {
	if base == "" {
		base = "https://slack.com/api"
	}
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context, channel, ts string) ([]RawMessage, error) {
		tok, err := token(ctx)
		if err != nil {
			return nil, err
		}
		q := url.Values{"channel": {channel}, "ts": {ts}, "limit": {fmt.Sprint(ask.MaxThreadMessages)}}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/conversations.replies?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		var out struct {
			OK       bool   `json:"ok"`
			Err      string `json:"error"`
			Messages []struct {
				User  string `json:"user"`
				Text  string `json:"text"`
				BotID string `json:"bot_id"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
			return nil, err
		}
		if !out.OK {
			return nil, fmt.Errorf("slackapp: conversations.replies: %s", out.Err)
		}
		var ms []RawMessage
		for _, m := range out.Messages {
			if m.BotID == "" && strings.TrimSpace(m.Text) != "" {
				ms = append(ms, RawMessage{User: m.User, Text: m.Text})
			}
		}
		return ms, nil
	}
}
