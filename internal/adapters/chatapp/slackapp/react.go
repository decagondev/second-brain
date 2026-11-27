package slackapp

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

// React to save (sb-ask(3), The chat app (Slack); SB-5806): the reaction sb-cell(5)
// chat.save_reaction names (default brain), added to a message in a channel the app is in,
// sends the person who added it one direct message offering Save for the stewards or Ask
// about this thread. Only the reaction is read until they press one; then the thread is
// read as data (LAW-09) and saved, or asked about, as them, in the channel's space. A
// reaction in a channel the app isn't in reads nothing.

// DefaultSaveReaction is chat.save_reaction's default.
const DefaultSaveReaction = "brain"

// The offer and its buttons.
const (
	MsgReactOffer   = "You reacted to a thread. Save it for the space's stewards, or ask the brain about it? Nothing is read until you choose."
	ActionReactSave = "react_save"
	ActionReactAsk  = "react_ask"
	// ThreadQuestion is what Ask about this thread asks, with the thread as the file.
	ThreadQuestion = "What does this thread discuss, and what do our sources say about it?"
)

// BlockReplier sends a direct message with buttons.
type BlockReplier interface {
	DirectBlocks(ctx context.Context, user, text string, blocks []Block) error
}

func (a *App) saveReaction() string {
	if a.SaveReaction != "" {
		return a.SaveReaction
	}
	return DefaultSaveReaction
}

// reacted offers the person who added the save reaction what to do with the thread.
func (a *App) reacted(ctx context.Context, user, reaction, channel, ts string) {
	br, ok := a.Replies.(BlockReplier)
	if reaction != a.saveReaction() || !ok {
		return
	}
	name, err := a.Replies.ChannelName(ctx, channel)
	if err != nil || name == "" || name == "directmessage" {
		return // a channel the app isn't in is never read
	}
	v := channel + "|" + ts + "|" + name
	_ = br.DirectBlocks(ctx, user, MsgReactOffer, []Block{
		{Type: "section", Text: &Text{Type: "mrkdwn", Text: MsgReactOffer}},
		{Type: "actions", Elements: []Element{
			{Type: "button", Text: &Text{Type: "plain_text", Text: "Save for the stewards"}, ActionID: ActionReactSave, Value: v},
			{Type: "button", Text: &Text{Type: "plain_text", Text: "Ask about this thread"}, ActionID: ActionReactAsk, Value: v},
		}},
	})
}

// reactChoice handles a press of one of the offer's buttons: only now is the thread read.
func (a *App) reactChoice(user, action, value, responseURL string) {
	parts := strings.SplitN(value, "|", 3)
	if len(parts) != 3 || a.Thread == nil || !slackURL(responseURL) {
		return
	}
	channel, ts, name := parts[0], parts[1], parts[2]
	read := func(ctx context.Context) ([]RawMessage, error) { return a.Thread(ctx, channel, ts) }
	a.run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), AnswerTimeout)
		defer cancel()
		if action == ActionReactSave {
			link := "https://slack.com/archives/" + url.PathEscape(channel) + "/p" + strings.ReplaceAll(ts, ".", "")
			_ = a.post(ctx, responseURL, Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: a.SaveThread(ctx, user, name, link, read)})
			return
		}
		_ = a.post(ctx, responseURL, a.askThread(ctx, user, name, read))
	})
}

// askThread asks ThreadQuestion as user in the channel's space, the thread as a file read for
// this ask alone (SB-3802).
func (a *App) askThread(ctx context.Context, user, channel string, read func(ctx context.Context) ([]RawMessage, error)) Message {
	s, err := a.Scope(ctx, user)
	if err != nil {
		return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: MsgNoMatch}
	}
	s, msg := a.InSpace(ctx, s, channel)
	if msg != "" {
		return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: msg}
	}
	raw, err := read(ctx)
	if err != nil || len(raw) == 0 {
		return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: MsgSaveFail}
	}
	var b strings.Builder
	for i, m := range raw {
		if i == ask.MaxThreadMessages {
			break
		}
		b.WriteString(m.Text + "\n")
	}
	resp, err := a.Agent.Ask(tenancy.WithClient(ctx, tenancy.ClientChat), s, ask.Request{Question: ThreadQuestion, Language: a.AnswerLanguage(),
		File: &ask.FileUpload{Name: "thread.txt", Data: []byte(b.String())}})
	if err != nil {
		return Message{ResponseType: "ephemeral", ReplaceOriginal: true, Text: ErrText(err)}
	}
	return Render(ThreadQuestion, resp)
}

var _ BlockReplier = DM{}

// DirectBlocks implements BlockReplier.
func (d DM) DirectBlocks(ctx context.Context, user, text string, blocks []Block) error {
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
	if err := d.call(ctx, http.MethodPost, "chat.postMessage", map[string]any{"channel": im.Channel.ID, "text": text, "blocks": blocks}, &sent); err != nil {
		return err
	}
	if !sent.OK {
		return errSlack("chat.postMessage", sent.Err)
	}
	return nil
}
