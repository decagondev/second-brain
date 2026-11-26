package mmapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"example.com/second-brain/internal/core/notify"
)

// DM sends notifications as the Mattermost bot's direct messages (sb-ask(3),
// Notifications; SB-2101): the user by email, a direct channel with the bot, one post.
type DM struct {
	Base   string // the Mattermost server
	TeamID string // the team whose channels Post finds by name (SB-5304)
	Token  func(ctx context.Context) (string, error)
	Client *http.Client
}

var _ notify.Sender = DM{}

// errNotFound is a 404 from the server.
var errNotFound = errors.New("mmapp: not found")

func (d DM) call(ctx context.Context, method, path string, body, out any) error {
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
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(d.Base, "/")+"/api/v4/"+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("mmapp: %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

// Send implements notify.Sender.
func (d DM) Send(ctx context.Context, to notify.Person, m notify.Message) error {
	var bot, user struct {
		ID       string `json:"id"`
		DeleteAt int64  `json:"delete_at"`
		IsBot    bool   `json:"is_bot"`
	}
	if err := d.call(ctx, http.MethodGet, "users/me", nil, &bot); err != nil || bot.ID == "" {
		return fmt.Errorf("mmapp: the bot's own user: %v", err)
	}
	err := d.call(ctx, http.MethodGet, "users/email/"+url.PathEscape(to.Email), nil, &user)
	if errors.Is(err, errNotFound) || err == nil && (user.ID == "" || user.DeleteAt != 0 || user.IsBot) {
		return notify.ErrNoSuchPerson
	}
	if err != nil {
		return err
	}
	var ch struct {
		ID string `json:"id"`
	}
	if err := d.call(ctx, http.MethodPost, "channels/direct", []string{bot.ID, user.ID}, &ch); err != nil || ch.ID == "" {
		return fmt.Errorf("mmapp: direct channel: %v", err)
	}
	var post struct {
		ID string `json:"id"`
	}
	return d.call(ctx, http.MethodPost, "posts", map[string]string{"channel_id": ch.ID, "message": DMText(m)}, &post)
}

// DMText is a notification in Mattermost's Markdown: the text escaped, then the link.
func DMText(m notify.Message) string {
	text := escape(m.Text)
	if strings.HasPrefix(m.Link, "https://") {
		text += " [Open](" + strings.NewReplacer("(", "%28", ")", "%29", " ", "%20").Replace(m.Link) + ")"
	}
	if strings.HasPrefix(m.Pause, "https://") { // SB-5107
		text += " [Pause this agent here](" + strings.NewReplacer("(", "%28", ")", "%29", " ", "%20").Replace(m.Pause) + ")"
	}
	return text
}

// Post posts text in channel (#name) of the team, as the bot (SB-5304), and returns the
// post's id. The bot must be in the channel.
func (d DM) Post(ctx context.Context, channel, text string) (string, error) {
	var ch struct {
		ID string `json:"id"`
	}
	name := strings.TrimPrefix(channel, "#")
	if err := d.call(ctx, http.MethodGet, "teams/"+url.PathEscape(d.TeamID)+"/channels/name/"+url.PathEscape(name), nil, &ch); err != nil || ch.ID == "" {
		return "", fmt.Errorf("mmapp: channel %s: %v", name, err)
	}
	var post struct {
		ID string `json:"id"`
	}
	if err := d.call(ctx, http.MethodPost, "posts", map[string]string{"channel_id": ch.ID, "message": escape(text)}, &post); err != nil {
		return "", err
	}
	return post.ID, nil
}
