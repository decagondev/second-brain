package slackapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

// Follow up in the thread (sb-ask(3), The chat app (Slack); SB-5801): a reply in the thread
// under one of the brain's messages continues that person's conversation in the thread, one
// per thread and person, re-authorised as whoever replied, with the conversation's last
// earlier questions sent as before (SB-2901). The answer goes to the person alone: an
// ephemeral reply in the thread, or a direct message when it cites something labelled
// confidential or restricted (LAW-32). A reply from someone else starts their own.

// Replier answers one person in a chat app.
type Replier interface {
	// Ephemeral shows text to user alone, in thread of channel.
	Ephemeral(ctx context.Context, channel, user, thread, text string) error
	// Direct sends text to user in a direct message.
	Direct(ctx context.Context, user, text string) error
}

// Thread replies.
const (
	// MsgThreadLabel is the ephemeral line when the answer went to a direct message.
	MsgThreadLabel = "This answer cites confidential documents, so I've sent it to you directly."
)

// Followup is one reply in a thread under the brain's message, as a chat app read it.
type Followup struct {
	User, Channel, ChannelName, Thread, Text string
	// Key is the thread's key across the cell: the app's name, its workspace, the channel and
	// the thread's root.
	Key string
}

// mention is a leading mention of the bot, which isn't part of the question.
var mention = regexp.MustCompile(`^\s*(<@[A-Z0-9]+>|@[a-z0-9._-]+)\s*`)

// FollowUp asks f's question in its thread's conversation as its person, and answers them
// alone through r with the text render makes. Nothing is posted for anyone else to read.
func (a *App) FollowUp(ctx context.Context, f Followup, r Replier, render func(question string, resp ask.Response) string) {
	q := strings.TrimSpace(mention.ReplaceAllString(f.Text, ""))
	if q == "" {
		return
	}
	s, err := a.Scope(ctx, f.User)
	if err != nil {
		_ = r.Ephemeral(ctx, f.Channel, f.User, f.Thread, MsgNoMatch)
		return
	}
	s, msg := a.InSpace(ctx, s, f.ChannelName)
	if msg != "" {
		_ = r.Ephemeral(ctx, f.Channel, f.User, f.Thread, msg)
		return
	}
	resp, err := a.Agent.Ask(tenancy.WithClient(ctx, tenancy.ClientChat), s, ask.Request{Question: q, Thread: f.Key, Language: a.AnswerLanguage()})
	if err != nil {
		_ = r.Ephemeral(ctx, f.Channel, f.User, f.Thread, ErrText(err))
		return
	}
	text := render(q, resp)
	if line := a.NoticeLine(ctx); line != "" {
		text = line + "\n" + text
	}
	text = a.mask(ctx, Message{Text: text}).Text                     // SB-5102: nothing sensitive goes out
	if !ask.MayPostInChannel(resp.Label) && !direct(f.ChannelName) { // LAW-32: to the person directly
		if err := r.Direct(ctx, f.User, text); err == nil {
			_ = r.Ephemeral(ctx, f.Channel, f.User, f.Thread, MsgThreadLabel)
		}
		return
	}
	_ = r.Ephemeral(ctx, f.Channel, f.User, f.Thread, text)
}

// direct says a channel is a direct message with the bot.
func direct(name string) bool { return name == "directmessage" || name == "" }

// Event is the Slack app's part of an Events API delivery the chat connector has already
// verified (SB-5801, SB-5806): a reply in a thread under the bot's message is answered, a
// save reaction is offered, and anything else is ignored. Bot messages never are.
func (a *App) Event(body []byte) {
	if a.Replies == nil {
		return
	}
	var env struct {
		Type   string `json:"type"`
		TeamID string `json:"team_id"`
		Event  struct {
			Type         string `json:"type"`
			Subtype      string `json:"subtype"`
			User         string `json:"user"`
			BotID        string `json:"bot_id"`
			Text         string `json:"text"`
			Channel      string `json:"channel"`
			ChannelType  string `json:"channel_type"`
			TS           string `json:"ts"`
			ThreadTS     string `json:"thread_ts"`
			ParentUserID string `json:"parent_user_id"`
			Reaction     string `json:"reaction"`
			Item         struct {
				Channel string `json:"channel"`
				TS      string `json:"ts"`
			} `json:"item"`
		} `json:"event"`
	}
	if json.Unmarshal(body, &env) != nil || env.Type != "event_callback" || a.TeamID == "" || env.TeamID != a.TeamID {
		return
	}
	e := env.Event
	switch {
	case e.Type == "message" && e.Subtype == "" && e.BotID == "" && e.User != "" && e.ThreadTS != "" && e.ThreadTS != e.TS && e.ParentUserID != "":
		a.run(func() {
			ctx, cancel := context.WithTimeout(context.Background(), AnswerTimeout)
			defer cancel()
			bot, err := a.Replies.Bot(ctx)
			if err != nil || bot == "" || e.ParentUserID != bot {
				return // not under the brain's own message
			}
			name, err := a.Replies.ChannelName(ctx, e.Channel)
			if err != nil {
				return // a channel the app can't see is never answered in
			}
			a.FollowUp(ctx, Followup{User: e.User, Channel: e.Channel, ChannelName: name, Thread: e.ThreadTS, Text: e.Text,
				Key: "slack/" + a.TeamID + "/" + e.Channel + "/" + e.ThreadTS}, a.Replies, func(q string, r ask.Response) string { return Render(q, r).Text })
		})
	case e.Type == "message" && e.ChannelType == "im" && e.Subtype == "" && e.BotID == "" && e.User != "" && e.ThreadTS == "":
		a.run(func() { // a direct message to the brain is a question (SB-5805)
			ctx, cancel := context.WithTimeout(context.Background(), AnswerTimeout)
			defer cancel()
			if bot, err := a.Replies.Bot(ctx); err != nil || e.User == bot {
				return
			}
			a.directQuestion(ctx, e.User, e.Text)
		})
	case e.Type == "reaction_added" && e.User != "" && e.Item.Channel != "" && e.Item.TS != "":
		a.run(func() {
			ctx, cancel := context.WithTimeout(context.Background(), AnswerTimeout)
			defer cancel()
			a.reacted(ctx, e.User, e.Reaction, e.Item.Channel, e.Item.TS)
		})
	}
}

// Replies is the bot's side of a thread: who it is, a channel's name, and the replies.
type Replies interface {
	Replier
	Bot(ctx context.Context) (string, error)
	ChannelName(ctx context.Context, id string) (string, error)
}

var _ Replies = DM{}

// Bot is the bot's own user id (auth.test).
func (d DM) Bot(ctx context.Context) (string, error) {
	var out struct {
		OK     bool   `json:"ok"`
		Err    string `json:"error"`
		UserID string `json:"user_id"`
	}
	if err := d.call(ctx, http.MethodPost, "auth.test", map[string]string{}, &out); err != nil {
		return "", err
	}
	if !out.OK {
		return "", errSlack("auth.test", out.Err)
	}
	return out.UserID, nil
}

// ChannelName is a channel's name, or "directmessage" for a direct message with the bot
// (conversations.info); a channel the bot isn't in is an error.
func (d DM) ChannelName(ctx context.Context, id string) (string, error) {
	var out struct {
		OK      bool   `json:"ok"`
		Err     string `json:"error"`
		Channel struct {
			Name     string `json:"name"`
			IsIM     bool   `json:"is_im"`
			IsMember bool   `json:"is_member"`
		} `json:"channel"`
	}
	if err := d.call(ctx, http.MethodGet, "conversations.info?"+url.Values{"channel": {id}}.Encode(), nil, &out); err != nil {
		return "", err
	}
	if !out.OK {
		return "", errSlack("conversations.info", out.Err)
	}
	if out.Channel.IsIM {
		return "directmessage", nil
	}
	if !out.Channel.IsMember {
		return "", errSlack("conversations.info", "not_in_channel")
	}
	return out.Channel.Name, nil
}

// Ephemeral implements Replier (chat.postEphemeral, in the thread).
func (d DM) Ephemeral(ctx context.Context, channel, user, thread, text string) error {
	var out struct {
		OK  bool   `json:"ok"`
		Err string `json:"error"`
	}
	body := map[string]any{"channel": channel, "user": user, "text": text}
	if thread != "" {
		body["thread_ts"] = thread
	}
	if err := d.call(ctx, http.MethodPost, "chat.postEphemeral", body, &out); err != nil {
		return err
	}
	if !out.OK {
		return errSlack("chat.postEphemeral", out.Err)
	}
	return nil
}

// Direct implements Replier (conversations.open, chat.postMessage).
func (d DM) Direct(ctx context.Context, user, text string) error {
	var im struct {
		OK      bool   `json:"ok"`
		Err     string `json:"error"`
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}
	if err := d.call(ctx, http.MethodPost, "conversations.open", map[string]string{"users": user}, &im); err != nil {
		return err
	}
	if !im.OK || im.Channel.ID == "" {
		return errSlack("conversations.open", im.Err)
	}
	var sent struct {
		OK  bool   `json:"ok"`
		Err string `json:"error"`
	}
	if err := d.call(ctx, http.MethodPost, "chat.postMessage", map[string]any{"channel": im.Channel.ID, "text": text, "unfurl_links": false, "unfurl_media": false}, &sent); err != nil {
		return err
	}
	if !sent.OK {
		return errSlack("chat.postMessage", sent.Err)
	}
	return nil
}

type slackErr struct{ method, code string }

func (e slackErr) Error() string { return "slackapp: " + e.method + ": " + e.code }

func errSlack(method, code string) error { return slackErr{method, code} }

// directQuestion answers a direct message to the brain, company-wide, in that direct
// message; the person's first one of the day starts with where they left off (SB-5805).
func (a *App) directQuestion(ctx context.Context, user, text string) {
	q := strings.TrimSpace(mention.ReplaceAllString(text, ""))
	if q == "" {
		return
	}
	s, err := a.Scope(ctx, user)
	if err != nil {
		_ = a.Replies.Direct(ctx, user, MsgNoMatch)
		return
	}
	var lines []string
	if a.LeftOff != nil && a.firstToday(user) {
		if lo, err := a.LeftOff(ctx, s); err == nil {
			for _, l := range ask.LeftOffLines(lo, strings.TrimSuffix(a.Base, "/")) {
				lines = append(lines, "_"+escape(l)+"_")
			}
		}
	}
	resp, err := a.Agent.Ask(tenancy.WithClient(ctx, tenancy.ClientChat), s, ask.Request{Question: q, Language: a.AnswerLanguage()})
	out := ErrText(err)
	if err == nil {
		out = Render(q, resp).Text
	}
	if len(lines) > 0 {
		out = strings.Join(lines, "\n") + "\n\n" + out
	}
	if line := a.NoticeLine(ctx); line != "" {
		out = line + "\n" + out
	}
	_ = a.Replies.Direct(ctx, user, a.mask(ctx, Message{Text: out}).Text)
}

// firstToday says this is user's first direct message to the app today (UTC). The record
// is the process's own and holds only chat ids and a date.
func (a *App) firstToday(user string) bool {
	now := time.Now
	if a.Now != nil {
		now = a.Now
	}
	day := now().UTC().Format(time.DateOnly)
	a.seenMu.Lock()
	defer a.seenMu.Unlock()
	if a.seen == nil {
		a.seen = map[string]string{}
	}
	if a.seen[user] == day {
		return false
	}
	a.seen[user] = day
	return true
}
