package mmapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/ask"
)

// React to save and direct questions, Mattermost (sb-ask(3), The chat app (Mattermost);
// SB-6005), from the bot's websocket. The save reaction (sb-cell(5) chat.save_reaction,
// default brain) on a post in a channel the bot is in sends the person one direct message
// offering Save for the stewards or Ask about this thread, as in Slack (SB-5806): only the
// reaction is read until they press one. A post in a direct message with the bot is a
// question, answered as its sender (SB-5805).

// Channels says what the bot knows of a channel.
type Channels interface {
	// Channel is a channel's name and type (O, P, D, G) when the bot is in it; an error
	// when it isn't, which means the channel is never read.
	Channel(ctx context.Context, id string) (name, kind string, err error)
	// DirectWith sends a direct message with buttons.
	DirectWith(ctx context.Context, user, text string, attachments []Attachment) error
}

// Kinds of a press from the reaction's offer.
const (
	KindReactSave = "react_save"
	KindReactAsk  = "react_ask"
)

func (a *App) saveReaction() string {
	if a.SaveReaction != "" {
		return a.SaveReaction
	}
	return slackapp.DefaultSaveReaction
}

// Event handles one websocket event; anything but a save reaction and a direct message to
// the bot is ignored, and so is everything the bot posted.
func (a *App) Event(ctx context.Context, e WSEvent) {
	ch, ok := a.Threads.(Channels)
	if a.Threads == nil || !ok {
		return
	}
	bot, err := a.Threads.Bot(ctx)
	if err != nil || bot == "" {
		return
	}
	switch e.Event {
	case "reaction_added":
		var r struct {
			UserID    string `json:"user_id"`
			PostID    string `json:"post_id"`
			EmojiName string `json:"emoji_name"`
		}
		if json.Unmarshal([]byte(e.Str("reaction")), &r) != nil || r.UserID == "" || r.UserID == bot || r.EmojiName != a.saveReaction() || !postIDRe.MatchString(r.PostID) {
			return
		}
		name, kind, err := ch.Channel(ctx, e.Broadcast.ChannelID)
		if err != nil || name == "" || kind == "D" || kind == "G" {
			return // a channel the bot isn't in is never read
		}
		tok, err := a.Token(ctx)
		if err != nil || tok == "" || a.ActionURL == "" {
			return
		}
		button := func(id, label, kind string) Action {
			c := ShareContext{Kind: kind, User: r.UserID, Post: r.PostID, Channel: e.Broadcast.ChannelID, ChannelName: name}
			c.Sig = sign(tok, c)
			return Action{ID: id, Name: label, Integration: Integration{URL: a.ActionURL, Context: c}}
		}
		_ = ch.DirectWith(ctx, r.UserID, slackapp.MsgReactOffer, []Attachment{{Actions: []Action{
			button("reactsave", "Save for the stewards", KindReactSave), button("reactask", "Ask about this thread", KindReactAsk)}}})
	case "posted":
		var p struct {
			UserID    string `json:"user_id"`
			ChannelID string `json:"channel_id"`
			Message   string `json:"message"`
			RootID    string `json:"root_id"`
			Type      string `json:"type"`
		}
		if e.Str("channel_type") != "D" || json.Unmarshal([]byte(e.Str("post")), &p) != nil || p.UserID == "" || p.UserID == bot || p.Type != "" || p.RootID != "" {
			return
		}
		if _, kind, err := ch.Channel(ctx, p.ChannelID); err != nil || kind != "D" {
			return
		}
		a.Rules.DirectQuestion(ctx, p.UserID, p.Message, a.Threads, func(q string, r ask.Response) string { return Render(q, r).Text })
	}
}

// reactPress handles a press of the offer's buttons, signed by the cell: only now is the
// thread read, as the person who pressed, who must be the one who reacted.
func (a *App) reactPress(ctx context.Context, user string, c ShareContext) string {
	if user != c.User {
		return MsgShareFail
	}
	if c.Kind == KindReactSave {
		return a.save(ctx, user, c.Channel, c.ChannelName, c.Post)
	}
	if a.Thread == nil {
		return MsgSaveUsage
	}
	read := func(ctx context.Context) ([]slackapp.RawMessage, error) {
		ps, err := a.Thread(ctx, c.Post)
		if err != nil {
			return nil, err
		}
		var out []slackapp.RawMessage
		for _, p := range ps {
			if p.ChannelID == c.Channel && p.Type == "" && p.Message != "" {
				out = append(out, slackapp.RawMessage{User: p.UserID, Text: p.Message})
			}
		}
		return out, nil
	}
	text := a.Rules.AskThread(ctx, user, c.ChannelName, read, func(q string, r ask.Response) string { return Render(q, r).Text })
	if a.Threads != nil {
		_ = a.Threads.Direct(ctx, user, text)
	}
	return "The answer is in your direct messages."
}

var _ Channels = DM{}

// Channel implements Channels: the channel's name and type, if the bot is a member.
func (d DM) Channel(ctx context.Context, id string) (string, string, error) {
	var c struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := d.call(ctx, http.MethodGet, "channels/"+url.PathEscape(id), nil, &c); err != nil {
		return "", "", err
	}
	var m struct {
		UserID string `json:"user_id"`
	}
	if err := d.call(ctx, http.MethodGet, "channels/"+url.PathEscape(id)+"/members/me", nil, &m); err != nil || m.UserID == "" {
		return "", "", errNotMember
	}
	if c.Type == "D" {
		c.Name = "directmessage"
	}
	return c.Name, c.Type, nil
}

// DirectWith implements Channels.
func (d DM) DirectWith(ctx context.Context, user, text string, attachments []Attachment) error {
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
	return d.call(ctx, http.MethodPost, "posts", map[string]any{"channel_id": ch.ID, "message": text, "props": map[string]any{"attachments": []map[string]any{{"actions": attachments[0].Actions}}}}, &post)
}

type mmError string

func (e mmError) Error() string { return string(e) }

const errNotMember = mmError("mmapp: the bot isn't in that channel")
