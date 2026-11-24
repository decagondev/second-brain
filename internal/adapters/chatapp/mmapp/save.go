package mmapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/ask"
)

// Save to the brain (sb-ask(3), The chat app (Mattermost); SB-3703): `/brain save LINK`,
// in the thread's channel, sends that thread to the stewards of the channel's space.

// MsgSaveUsage explains the save command.
const MsgSaveUsage = "To save a thread for the stewards, copy a message's link and type: save LINK, in the same channel."

var postIDRe = regexp.MustCompile(`(?:/pl/)?([a-z0-9]{26})/?$`)

// Post is one post of a thread as the Mattermost API returns it.
type Post struct {
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
	CreateAt  int64  `json:"create_at"`
	Type      string `json:"type"`
}

// postOf is the post id a save command names: "save" alone (no id), or "save" and one
// permalink or post id. Any other text is a question.
func postOf(text string) (string, bool) {
	f := strings.Fields(text)
	if len(f) == 0 || !strings.EqualFold(f[0], "save") || len(f) > 2 {
		return "", false
	}
	if len(f) == 1 {
		return "", true
	}
	m := postIDRe.FindStringSubmatch(f[1])
	if m == nil {
		return "", false
	}
	return m[1], true
}

// save answers a save command: the thread of post, which must be in channel.
func (a *App) save(ctx context.Context, user, channelID, channelName, post string) string {
	if post == "" || a.Thread == nil {
		return MsgSaveUsage
	}
	link := strings.TrimSuffix(a.Base, "/") + "/_redirect/pl/" + post
	return a.Rules.SaveThread(ctx, user, channelName, link, func(ctx context.Context) ([]slackapp.RawMessage, error) {
		ps, err := a.Thread(ctx, post)
		if err != nil {
			return nil, err
		}
		var out []slackapp.RawMessage
		for _, p := range ps {
			if p.ChannelID != channelID {
				return nil, fmt.Errorf("mmapp: the thread isn't in this channel") // never a channel the command wasn't typed in
			}
			if p.Type == "" && strings.TrimSpace(p.Message) != "" {
				out = append(out, slackapp.RawMessage{User: p.UserID, Text: p.Message})
			}
		}
		return out, nil
	})
}

// APIThread reads a post's whole thread, oldest first, with the bot's token.
func APIThread(base string, token func(ctx context.Context) (string, error), client *http.Client) func(ctx context.Context, post string) ([]Post, error) {
	return func(ctx context.Context, post string) ([]Post, error) {
		if !postIDRe.MatchString(post) || strings.Contains(post, "/") {
			return nil, fmt.Errorf("mmapp: bad post id")
		}
		tok, err := token(ctx)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(base, "/")+"/api/v4/posts/"+url.PathEscape(post)+"/thread", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("mmapp: thread: HTTP %d", resp.StatusCode)
		}
		var out struct {
			Posts map[string]Post `json:"posts"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
			return nil, err
		}
		ps := make([]Post, 0, len(out.Posts))
		for _, p := range out.Posts {
			ps = append(ps, p)
		}
		sort.Slice(ps, func(i, j int) bool { return ps[i].CreateAt < ps[j].CreateAt })
		if len(ps) > ask.MaxThreadMessages {
			ps = ps[:ask.MaxThreadMessages]
		}
		return ps, nil
	}
}
