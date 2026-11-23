package slackapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"example.com/second-brain/internal/core/notify"
)

// DM sends notifications as the bot's direct messages (sb-ask(3), Notifications;
// SB-2101): users.lookupByEmail, conversations.open, chat.postMessage. It needs the bot
// scopes users:read.email, im:write and chat:write.
type DM struct {
	Base   string // the Web API, "" for https://slack.com/api
	Token  func(ctx context.Context) (string, error)
	Client *http.Client
}

var _ notify.Sender = DM{}

func (d DM) call(ctx context.Context, method, path string, body any, out any) error {
	base := d.Base
	if base == "" {
		base = "https://slack.com/api"
	}
	tok, err := d.Token(ctx)
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(base, "/")+"/"+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	c := d.Client
	if c == nil {
		c = http.DefaultClient
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slackapp: %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

// Send implements notify.Sender.
func (d DM) Send(ctx context.Context, to notify.Person, m notify.Message) error {
	var who struct {
		OK   bool   `json:"ok"`
		Err  string `json:"error"`
		User struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
			IsBot   bool   `json:"is_bot"`
		} `json:"user"`
	}
	if err := d.call(ctx, http.MethodGet, "users.lookupByEmail?"+url.Values{"email": {to.Email}}.Encode(), nil, &who); err != nil {
		return err
	}
	if !who.OK && who.Err == "users_not_found" || who.OK && (who.User.ID == "" || who.User.Deleted || who.User.IsBot) {
		return notify.ErrNoSuchPerson
	}
	if !who.OK {
		return fmt.Errorf("slackapp: users.lookupByEmail: %s", who.Err)
	}
	var im struct {
		OK      bool   `json:"ok"`
		Err     string `json:"error"`
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}
	if err := d.call(ctx, http.MethodPost, "conversations.open", map[string]string{"users": who.User.ID}, &im); err != nil {
		return err
	}
	if !im.OK || im.Channel.ID == "" {
		return fmt.Errorf("slackapp: conversations.open: %s", im.Err)
	}
	var sent struct {
		OK  bool   `json:"ok"`
		Err string `json:"error"`
	}
	if err := d.call(ctx, http.MethodPost, "chat.postMessage", map[string]any{"channel": im.Channel.ID, "text": DMText(m),
		"unfurl_links": false, "unfurl_media": false}, &sent); err != nil {
		return err
	}
	if !sent.OK {
		return fmt.Errorf("slackapp: chat.postMessage: %s", sent.Err)
	}
	return nil
}

// DMText is a notification in Slack's mrkdwn: the text escaped, then the link.
func DMText(m notify.Message) string {
	esc := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	text := esc.Replace(m.Text)
	if strings.HasPrefix(m.Link, "https://") {
		text += " <" + strings.NewReplacer("|", "%7C", ">", "%3E", "<", "%3C").Replace(m.Link) + "|Open>"
	}
	return text
}
