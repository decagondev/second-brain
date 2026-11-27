package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Event is one Events API event the connector cares about (ids only, never text).
type Event struct {
	Type     string `json:"type"`    // message, message_changed, message_deleted, member_joined_channel, member_left_channel
	Channel  string `json:"channel"` // channel id
	TS       string `json:"ts"`      // the message's ts
	ThreadTS string `json:"thread_ts,omitempty"`
}

// Inbox holds verified events in arrival order; the connector's cursor is a position in it.
type Inbox interface {
	Append(ctx context.Context, e Event) error
	Read(ctx context.Context, from int) (events []Event, next int, err error)
	Len(ctx context.Context) (int, error)
}

// MemoryInbox is an in-process Inbox (development and tests); cells keep it in their database.
type MemoryInbox struct {
	mu     sync.Mutex
	events []Event
}

func (m *MemoryInbox) Append(_ context.Context, e Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return nil
}

func (m *MemoryInbox) Read(_ context.Context, from int) ([]Event, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if from < 0 || from > len(m.events) {
		return nil, from, errors.New("slack: cursor outside the inbox")
	}
	return append([]Event(nil), m.events[from:]...), len(m.events), nil
}

func (m *MemoryInbox) Len(context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events), nil
}

// MaxSkew is how old a signed delivery may be.
const MaxSkew = 5 * time.Minute

// EventsHandler receives Slack's Events API deliveries: it verifies the v0 signature and
// timestamp, answers url_verification, and appends supported events to the inbox. A bad
// signature is 401 and nothing is stored.
func (c *Connector) EventsHandler(now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		k, err := c.creds(r.Context())
		if err != nil {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		ts := r.Header.Get("X-Slack-Request-Timestamp")
		sec, err := strconv.ParseInt(ts, 10, 64)
		if err != nil || now().Sub(time.Unix(sec, 0)).Abs() > MaxSkew || !validSignature(k.SigningSecret, ts, body, r.Header.Get("X-Slack-Signature")) {
			http.Error(w, "bad signature", http.StatusUnauthorized)
			return
		}
		var env struct {
			Type      string `json:"type"`
			Challenge string `json:"challenge"`
			Event     struct {
				Type     string `json:"type"`
				Subtype  string `json:"subtype"`
				Channel  string `json:"channel"`
				TS       string `json:"ts"`
				ThreadTS string `json:"thread_ts"`
				Message  *struct {
					TS       string `json:"ts"`
					ThreadTS string `json:"thread_ts"`
				} `json:"message"`
				DeletedTS string `json:"deleted_ts"`
			} `json:"event"`
		}
		if err := json.Unmarshal(body, &env); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if env.Type == "url_verification" {
			_ = json.NewEncoder(w).Encode(map[string]string{"challenge": env.Challenge})
			return
		}
		if c.Also != nil { // SB-5801, SB-5806: the chat app's part of the same delivery
			c.Also(body)
		}
		e := env.Event
		var ev Event
		switch {
		case e.Type == "message" && e.Subtype == "message_changed" && e.Message != nil:
			ev = Event{Type: "message_changed", Channel: e.Channel, TS: e.Message.TS, ThreadTS: e.Message.ThreadTS}
		case e.Type == "message" && e.Subtype == "message_deleted":
			ev = Event{Type: "message_deleted", Channel: e.Channel, TS: e.DeletedTS, ThreadTS: e.ThreadTS}
		case e.Type == "message" && (e.Subtype == "" || e.Subtype == "thread_broadcast"):
			ev = Event{Type: "message", Channel: e.Channel, TS: e.TS, ThreadTS: e.ThreadTS}
		case e.Type == "member_joined_channel" || e.Type == "member_left_channel":
			ev = Event{Type: e.Type, Channel: e.Channel}
		default:
			w.WriteHeader(http.StatusOK) // not ours; acknowledge so Slack doesn't retry
			return
		}
		if err := c.Inbox.Append(r.Context(), ev); err != nil {
			http.Error(w, "unavailable", http.StatusServiceUnavailable) // Slack retries
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func validSignature(secret, ts string, body []byte, got string) bool {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte("v0:" + ts + ":"))
	m.Write(body)
	want := "v0=" + hex.EncodeToString(m.Sum(nil))
	return hmac.Equal([]byte(want), []byte(got))
}

// Sign is what Slack does to a delivery (used by the fake and tests).
func Sign(secret, ts string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte("v0:" + ts + ":"))
	m.Write(body)
	return "v0=" + hex.EncodeToString(m.Sum(nil))
}

// Changes reads the inbox from the cursor. An empty cursor lists every message in every
// channel the app is in, and returns the inbox position taken before the listing.
func (c *Connector) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	if cur == "" {
		return c.everything(ctx)
	}
	from, err := strconv.Atoi(string(cur))
	if err != nil {
		return nil, cur, errors.New("slack: bad cursor")
	}
	evs, next, err := c.Inbox.Read(ctx, from)
	if err != nil {
		return nil, cur, err
	}
	var set ports.ChangeSet
	names := map[string]channel{}
	for _, e := range evs {
		ch, ok := names[e.Channel]
		if !ok {
			if ch, err = c.channelInfo(ctx, e.Channel); err != nil {
				return nil, cur, err
			}
			names[e.Channel] = ch
		}
		if c.excluded(ch) {
			continue
		}
		ref := ports.SourceRef{Kind: kind, ID: docID(e.Channel, e.ThreadTS, e.TS)}
		switch e.Type {
		case "message", "message_changed":
			set = append(set, ports.Change{Ref: ref})
		case "message_deleted":
			set = append(set, ports.Change{Ref: ref, Deleted: true})
		case "member_joined_channel", "member_left_channel":
			ids, err := c.channelMessages(ctx, ch.ID)
			if err != nil {
				return nil, cur, err
			}
			for _, id := range ids {
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: id}, ACLOnly: true})
			}
		}
	}
	return set, ports.Cursor(strconv.Itoa(next)), nil
}

func (c *Connector) everything(ctx context.Context) (ports.ChangeSet, ports.Cursor, error) {
	pos, err := c.Inbox.Len(ctx)
	if err != nil {
		return nil, "", err
	}
	var set ports.ChangeSet
	cursor := ""
	for {
		var r struct {
			Channels []channel `json:"channels"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		q := url.Values{"types": {"public_channel,private_channel"}, "exclude_archived": {"false"}, "limit": {"1000"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if err := c.call(ctx, "conversations.list", q, &r); err != nil {
			return nil, "", err
		}
		for _, ch := range r.Channels {
			if !ch.IsMember || c.excluded(ch) {
				continue
			}
			ids, err := c.channelMessages(ctx, ch.ID)
			if err != nil {
				return nil, "", err
			}
			for _, id := range ids {
				set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: id}})
			}
		}
		if r.Metadata.NextCursor == "" {
			return set, ports.Cursor(strconv.Itoa(pos)), nil
		}
		cursor = r.Metadata.NextCursor
	}
}

// channelMessages lists every message id in a channel, thread replies included.
func (c *Connector) channelMessages(ctx context.Context, ch string) ([]string, error) {
	var ids []string
	cursor := ""
	for {
		var r struct {
			Messages []message `json:"messages"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		q := url.Values{"channel": {ch}, "limit": {"200"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if err := c.call(ctx, "conversations.history", q, &r); err != nil {
			return nil, err
		}
		for _, m := range r.Messages {
			ids = append(ids, docID(ch, m.TS, m.TS))
			if m.ReplyCount > 0 {
				var rep struct {
					Messages []message `json:"messages"`
				}
				if err := c.call(ctx, "conversations.replies", url.Values{"channel": {ch}, "ts": {m.TS}}, &rep); err != nil {
					return nil, err
				}
				for _, x := range rep.Messages {
					if x.TS != m.TS {
						ids = append(ids, docID(ch, m.TS, x.TS))
					}
				}
			}
		}
		if r.Metadata.NextCursor == "" {
			return ids, nil
		}
		cursor = r.Metadata.NextCursor
	}
}
