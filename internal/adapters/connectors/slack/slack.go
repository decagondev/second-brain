// Package slack is the read-only Slack connector, `chat: slack` (sb-connector(3), SB-204).
// Documents are channel messages; direct messages are never read. Slack has no change
// feed, so changes come from Events API deliveries the cell verifies and keeps in an inbox.
package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

const (
	// DefaultBase is Slack's Web API.
	DefaultBase    = "https://slack.com/api"
	ACLDeadline    = 800 * time.Millisecond
	ACLConcurrency = 8
	kind           = "chat"
	// PublicToken is the prefilter entry for a public channel: any full workspace member.
	PublicToken = "slack:members"
)

// Creds are the app's bot token and signing secret, read from the vault at call time.
type Creds struct {
	BotToken      string `json:"bot_token"`
	SigningSecret string `json:"signing_secret"`
}

// Registrar learns secret values so logs can redact them.
type Registrar interface{ Register(secret string) }

// Connector talks to one Slack workspace.
type Connector struct {
	Base    string
	HTTP    *http.Client
	Creds   func(ctx context.Context) (string, error) // vault://tenants/T/connectors/chat (JSON)
	Redact  Registrar
	Exclude []string // channel name patterns, e.g. "#hr-*"
	Inbox   Inbox
}

var _ ports.Connector = (*Connector)(nil)

// ErrNotSupported is Act on this read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: slack connector is read-only")

func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

func (c *Connector) creds(ctx context.Context) (Creds, error) {
	raw, err := c.Creds(ctx)
	if err != nil {
		return Creds{}, err
	}
	var k Creds
	if err := json.Unmarshal([]byte(raw), &k); err != nil || k.BotToken == "" || k.SigningSecret == "" {
		return Creds{}, errors.New("slack: credentials are not {bot_token, signing_secret}")
	}
	if c.Redact != nil {
		c.Redact.Register(k.BotToken)
		c.Redact.Register(k.SigningSecret)
	}
	return k, nil
}

// slackError is an API answer with ok=false, or an HTTP failure.
type slackError struct {
	status int
	code   string
}

func (e *slackError) Error() string { return fmt.Sprintf("slack: %d %s", e.status, e.code) }

func (c *Connector) call(ctx context.Context, method string, q url.Values, v any) error {
	k, err := c.creds(ctx)
	if err != nil {
		return err
	}
	wait := 20 * time.Millisecond
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+"/"+method+"?"+q.Encode(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+k.BotToken)
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		if resp.StatusCode == 429 {
			select {
			case <-ctx.Done():
				return &slackError{429, "ratelimited"}
			case <-time.After(wait):
			}
			wait *= 2
			continue
		}
		if resp.StatusCode != 200 {
			return &slackError{resp.StatusCode, "http"}
		}
		var head struct {
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &head); err != nil {
			return &slackError{200, "bad_json"}
		}
		if !head.OK {
			return &slackError{200, head.Error}
		}
		return json.Unmarshal(body, v)
	}
}

type channel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsPrivate  bool   `json:"is_private"`
	IsIM       bool   `json:"is_im"`
	IsMPIM     bool   `json:"is_mpim"`
	IsMember   bool   `json:"is_member"`
	IsArchived bool   `json:"is_archived"`
}

type message struct {
	TS       string `json:"ts"`
	ThreadTS string `json:"thread_ts"`
	User     string `json:"user"`
	Text     string `json:"text"`
	Edited   *struct {
		TS string `json:"ts"`
	} `json:"edited"`
	ReplyCount int `json:"reply_count"`
}

func (c *Connector) excluded(ch channel) bool {
	if ch.IsIM || ch.IsMPIM || strings.HasPrefix(ch.ID, "D") {
		return true // direct messages are never read
	}
	for _, p := range c.Exclude {
		if ok, _ := path.Match(strings.TrimPrefix(p, "#"), ch.Name); ok {
			return true
		}
	}
	return false
}

func (c *Connector) channelInfo(ctx context.Context, id string) (channel, error) {
	var r struct {
		Channel channel `json:"channel"`
	}
	err := c.call(ctx, "conversations.info", url.Values{"channel": {id}}, &r)
	return r.Channel, err
}

// docID is CHANNEL/THREAD/TS (THREAD equals TS for a root message).
func docID(ch, thread, ts string) string {
	if thread == "" {
		thread = ts
	}
	return ch + "/" + thread + "/" + ts
}

func splitID(id string) (ch, thread, ts string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("slack: bad document id %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func micros(ts string) (int, error) {
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return 0, err
	}
	return int(f * 1e6), nil
}

func (c *Connector) message(ctx context.Context, ch, thread, ts string) (message, error) {
	var r struct {
		Messages []message `json:"messages"`
	}
	q := url.Values{"channel": {ch}, "latest": {ts}, "oldest": {ts}, "inclusive": {"true"}, "limit": {"1"}}
	method := "conversations.history"
	if thread != ts {
		method = "conversations.replies"
		q.Set("ts", thread)
	}
	if err := c.call(ctx, method, q, &r); err != nil {
		return message{}, err
	}
	for _, m := range r.Messages {
		if m.TS == ts {
			return m, nil
		}
	}
	return message{}, &slackError{200, "message_not_found"}
}

// Fetch reads one message as the app.
func (c *Connector) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	chID, thread, ts, err := splitID(ref.ID)
	if err != nil {
		return ports.Document{}, err
	}
	ch, err := c.channelInfo(ctx, chID)
	if err != nil {
		return ports.Document{}, err
	}
	if c.excluded(ch) {
		return ports.Document{}, fmt.Errorf("slack: %s is excluded", chID)
	}
	m, err := c.message(ctx, chID, thread, ts)
	if err != nil {
		return ports.Document{}, err
	}
	vts := m.TS
	if m.Edited != nil && m.Edited.TS != "" {
		vts = m.Edited.TS
	}
	v, err := micros(vts)
	if err != nil {
		return ports.Document{}, fmt.Errorf("slack: bad timestamp on %s", ref.ID)
	}
	pre, err := c.prefilter(ctx, ch)
	if err != nil {
		return ports.Document{}, err
	}
	return ports.Document{Ref: ref, Version: v, Title: "#" + ch.Name, Text: m.Text, ACLPrefilter: pre}, nil
}

func (c *Connector) prefilter(ctx context.Context, ch channel) ([]string, error) {
	if !ch.IsPrivate {
		return []string{PublicToken}, nil
	}
	ids, err := c.members(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	var out []string
	for id := range ids {
		var u struct {
			User user `json:"user"`
		}
		if err := c.call(ctx, "users.info", url.Values{"user": {id}}, &u); err == nil && u.User.Profile.Email != "" {
			out = append(out, strings.ToLower(u.User.Profile.Email))
		}
	}
	return out, nil
}

type user struct {
	ID                string `json:"id"`
	Deleted           bool   `json:"deleted"`
	IsRestricted      bool   `json:"is_restricted"`
	IsUltraRestricted bool   `json:"is_ultra_restricted"`
	Profile           struct {
		Email string `json:"email"`
	} `json:"profile"`
}

func (c *Connector) members(ctx context.Context, ch string) (map[string]bool, error) {
	out := map[string]bool{}
	cursor := ""
	for {
		var r struct {
			Members  []string `json:"members"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		q := url.Values{"channel": {ch}, "limit": {"1000"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if err := c.call(ctx, "conversations.members", q, &r); err != nil {
			return nil, err
		}
		for _, m := range r.Members {
			out[m] = true
		}
		if r.Metadata.NextCursor == "" {
			return out, nil
		}
		cursor = r.Metadata.NextCursor
	}
}

// ACL answers per message from channel visibility and membership, live.
func (c *Connector) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs)) // zero value is Unknown (LAW-03)
	if u.Email == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	var who struct {
		User user `json:"user"`
	}
	err := c.call(ctx, "users.lookupByEmail", url.Values{"email": {u.Email}}, &who)
	var se *slackError
	switch {
	case errors.As(err, &se) && se.code == "users_not_found":
		for i := range out {
			out[i] = ports.VerdictDeny // no Slack account: can't open anything
		}
		return out, nil
	case err != nil:
		return out, nil
	}
	if who.User.Deleted {
		for i := range out {
			out[i] = ports.VerdictDeny
		}
		return out, nil
	}
	guest := who.User.IsRestricted || who.User.IsUltraRestricted
	var mu sync.Mutex
	chans := map[string]*struct {
		once   sync.Once
		ch     channel
		member bool
		err    error
	}{}
	entry := func(id string) *struct {
		once   sync.Once
		ch     channel
		member bool
		err    error
	} {
		mu.Lock()
		defer mu.Unlock()
		e, ok := chans[id]
		if !ok {
			e = &struct {
				once   sync.Once
				ch     channel
				member bool
				err    error
			}{}
			chans[id] = e
		}
		return e
	}
	sem := make(chan struct{}, ACLConcurrency)
	var wg sync.WaitGroup
	for i, r := range refs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			chID, thread, ts, err := splitID(id)
			if err != nil {
				return
			}
			e := entry(chID)
			e.once.Do(func() {
				e.ch, e.err = c.channelInfo(ctx, chID)
				if e.err == nil && (e.ch.IsPrivate || guest) {
					var ms map[string]bool
					ms, e.err = c.members(ctx, chID)
					e.member = ms[who.User.ID]
				}
			})
			if e.err != nil || c.excluded(e.ch) {
				return // Unknown
			}
			if _, err := c.message(ctx, chID, thread, ts); err != nil {
				return // gone or unanswerable: Unknown
			}
			v := ports.VerdictDeny
			if (!e.ch.IsPrivate && !guest) || e.member {
				v = ports.VerdictAllow
			}
			mu.Lock()
			out[i] = v
			mu.Unlock()
		}(i, r.ID)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil // late is Unknown
	}
	return out, nil
}

// Act is not supported: read-only.
func (c *Connector) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}
