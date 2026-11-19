// Package slacktest is a recorded fake of the Slack Web API methods the connector uses,
// plus the Events API's signed deliveries. Every change returns the event body Slack
// would deliver, so tests push it through the connector's real EventsHandler.
package slacktest

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Fake is one Slack workspace.
type Fake struct {
	Server   *httptest.Server
	BotToken string
	Secret   string

	mu       sync.Mutex
	users    map[string]*User    // by id
	channels map[string]*Channel // by id
	clock    float64
	faults   []string // "500" / "timeout" on users.lookupByEmail, "404" on conversations.info
}

// User is a workspace member.
type User struct {
	ID, Email string
	Guest     bool
	Deleted   bool
}

// Channel is a public or private channel.
type Channel struct {
	ID, Name  string
	Private   bool
	BotMember bool
	Members   map[string]bool
	Messages  []*Message // roots in post order
}

// Message is one message; Replies hold its thread.
type Message struct {
	TS, ThreadTS, User, Text, EditedTS string
	Replies                            []*Message
	Deleted                            bool
}

// New starts a fake over TLS.
func New() *Fake {
	f := &Fake{BotToken: "xoxb-fake-bot-token", Secret: "fake-signing-secret-0123456789",
		users: map[string]*User{}, channels: map[string]*Channel{}, clock: 1795000000}
	mux := http.NewServeMux()
	for name, h := range map[string]func(*http.Request) any{
		"users.lookupByEmail": f.lookup, "users.info": f.userInfo, "conversations.info": f.info,
		"conversations.members": f.members, "conversations.list": f.list,
		"conversations.history": f.history, "conversations.replies": f.replies,
	} {
		h := h
		mux.HandleFunc("GET /api/"+name, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+f.BotToken {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid_auth"})
				return
			}
			out := h(r)
			if code, ok := out.(int); ok {
				w.WriteHeader(code)
				return
			}
			_ = json.NewEncoder(w).Encode(out)
		})
	}
	f.Server = httptest.NewTLSServer(mux)
	return f
}

// Base is the Web API base URL.
func (f *Fake) Base() string { return f.Server.URL + "/api" }

// CredsJSON is what the vault holds for the connector.
func (f *Fake) CredsJSON() string {
	b, _ := json.Marshal(map[string]string{"bot_token": f.BotToken, "signing_secret": f.Secret})
	return string(b)
}

func (f *Fake) tick() string { f.clock += 10; return strconv.FormatFloat(f.clock, 'f', 6, 64) }

// AddUser adds a member (guest true for single- or multi-channel guests).
func (f *Fake) AddUser(id, email string, guest bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[id] = &User{ID: id, Email: email, Guest: guest}
}

// AddChannel adds a channel the app is (or isn't) in.
func (f *Fake) AddChannel(id, name string, private, botMember bool, members ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ms := map[string]bool{}
	for _, m := range members {
		ms[m] = true
	}
	f.channels[id] = &Channel{ID: id, Name: name, Private: private, BotMember: botMember, Members: ms}
}

func ok(m map[string]any) map[string]any { m["ok"] = true; return m }
func fail(code string) map[string]any    { return map[string]any{"ok": false, "error": code} }

func (f *Fake) event(ev map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{"type": "event_callback", "event": ev})
	return b
}

// Post posts a message (thread "" for a root) and returns its ts and the event body.
func (f *Fake) Post(ch, user, thread, text string) (string, []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ts := f.tick()
	m := &Message{TS: ts, ThreadTS: thread, User: user, Text: text}
	c := f.channels[ch]
	if thread == "" {
		c.Messages = append(c.Messages, m)
	} else {
		for _, root := range c.Messages {
			if root.TS == thread {
				root.Replies = append(root.Replies, m)
			}
		}
	}
	return ts, f.event(map[string]any{"type": "message", "channel": ch, "ts": ts, "thread_ts": thread, "user": user, "text": text})
}

func (f *Fake) find(ch, ts string) *Message {
	for _, root := range f.channels[ch].Messages {
		if root.TS == ts {
			return root
		}
		for _, r := range root.Replies {
			if r.TS == ts {
				return r
			}
		}
	}
	return nil
}

// Edit edits a message and returns the event body.
func (f *Fake) Edit(ch, ts, text string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := f.find(ch, ts)
	m.Text, m.EditedTS = text, f.tick()
	return f.event(map[string]any{"type": "message", "subtype": "message_changed", "channel": ch,
		"message": map[string]any{"ts": ts, "thread_ts": m.ThreadTS, "text": text}})
}

// Delete deletes a message and returns the event body.
func (f *Fake) Delete(ch, ts string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := f.find(ch, ts)
	m.Deleted = true
	return f.event(map[string]any{"type": "message", "subtype": "message_deleted", "channel": ch, "deleted_ts": ts, "thread_ts": m.ThreadTS})
}

// Join adds a member to a channel and returns the event body.
func (f *Fake) Join(ch, user string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.channels[ch].Members[user] = true
	return f.event(map[string]any{"type": "member_joined_channel", "channel": ch, "user": user})
}

// Leave removes a member from a channel and returns the event body.
func (f *Fake) Leave(ch, user string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.channels[ch].Members, user)
	return f.event(map[string]any{"type": "member_left_channel", "channel": ch, "user": user})
}

// Deliver signs body as Slack would, for a delivery at t.
func (f *Fake) Deliver(body []byte, t time.Time) *http.Request {
	ts := strconv.FormatInt(t.Unix(), 10)
	r := httptest.NewRequest("POST", "/connectors/slack/events", bytesReader(body))
	r.Header.Set("X-Slack-Request-Timestamp", ts)
	r.Header.Set("X-Slack-Signature", sign(f.Secret, ts, body))
	return r
}

// FailNext queues faults for the next ACL lookups.
func (f *Fake) FailNext(kinds ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.faults = append(f.faults, kinds...)
}

func (f *Fake) takeFault(kinds ...string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.faults) == 0 {
		return ""
	}
	for _, k := range kinds {
		if f.faults[0] == k {
			f.faults = f.faults[1:]
			return k
		}
	}
	return ""
}

func (f *Fake) lookup(r *http.Request) any {
	switch f.takeFault("500", "timeout") {
	case "500":
		return 500
	case "timeout":
		time.Sleep(time.Second)
		return 503
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Email == r.URL.Query().Get("email") {
			return ok(map[string]any{"user": userJSON(u)})
		}
	}
	return fail("users_not_found")
}

func userJSON(u *User) map[string]any {
	return map[string]any{"id": u.ID, "deleted": u.Deleted, "is_restricted": u.Guest, "profile": map[string]string{"email": u.Email}}
}

func (f *Fake) userInfo(r *http.Request) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, found := f.users[r.URL.Query().Get("user")]; found {
		return ok(map[string]any{"user": userJSON(u)})
	}
	return fail("user_not_found")
}

func chanJSON(c *Channel) map[string]any {
	return map[string]any{"id": c.ID, "name": c.Name, "is_private": c.Private, "is_member": c.BotMember}
}

func (f *Fake) info(r *http.Request) any {
	if f.takeFault("404") == "404" {
		return fail("channel_not_found")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, found := f.channels[r.URL.Query().Get("channel")]; found {
		return ok(map[string]any{"channel": chanJSON(c)})
	}
	return fail("channel_not_found")
}

func (f *Fake) members(r *http.Request) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, found := f.channels[r.URL.Query().Get("channel")]
	if !found {
		return fail("channel_not_found")
	}
	var ms []string
	for m := range c.Members {
		ms = append(ms, m)
	}
	sort.Strings(ms)
	return ok(map[string]any{"members": ms})
}

func (f *Fake) list(*http.Request) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for id := range f.channels {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []map[string]any
	for _, id := range ids {
		out = append(out, chanJSON(f.channels[id]))
	}
	return ok(map[string]any{"channels": out})
}

func msgJSON(m *Message) map[string]any {
	j := map[string]any{"ts": m.TS, "user": m.User, "text": m.Text, "reply_count": len(m.Replies)}
	if m.ThreadTS != "" {
		j["thread_ts"] = m.ThreadTS
	}
	if m.EditedTS != "" {
		j["edited"] = map[string]string{"ts": m.EditedTS}
	}
	return j
}

func inRange(r *http.Request, ts string) bool {
	q := r.URL.Query()
	return (q.Get("latest") == "" || ts <= q.Get("latest")) && (q.Get("oldest") == "" || ts >= q.Get("oldest"))
}

func (f *Fake) history(r *http.Request) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, found := f.channels[r.URL.Query().Get("channel")]
	if !found || !c.BotMember {
		return fail("channel_not_found")
	}
	var out []map[string]any
	for _, m := range c.Messages {
		if !m.Deleted && inRange(r, m.TS) {
			out = append(out, msgJSON(m))
		}
	}
	return ok(map[string]any{"messages": out})
}

func (f *Fake) replies(r *http.Request) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, found := f.channels[r.URL.Query().Get("channel")]
	if !found || !c.BotMember {
		return fail("channel_not_found")
	}
	for _, root := range c.Messages {
		if root.TS != r.URL.Query().Get("ts") || root.Deleted {
			continue
		}
		var out []map[string]any
		for _, m := range append([]*Message{root}, root.Replies...) {
			if !m.Deleted && inRange(r, m.TS) {
				out = append(out, msgJSON(m))
			}
		}
		return ok(map[string]any{"messages": out})
	}
	return fail("thread_not_found")
}

func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

func sign(secret, ts string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte("v0:" + ts + ":"))
	m.Write(body)
	return "v0=" + hex.EncodeToString(m.Sum(nil))
}
