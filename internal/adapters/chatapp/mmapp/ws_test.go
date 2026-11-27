package mmapp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/tenancy"
)

// wsServer is a recorded Mattermost for the bot's websocket: each connection is sent the
// next batch of events, then closed. Its REST side knows which channels the bot is in and
// records the posts the bot makes.
type wsServer struct {
	mu      sync.Mutex
	batches [][]WSEvent
	posts   []map[string]any
	pongs   int
}

func event(name, channel string, data map[string]any) WSEvent {
	e := WSEvent{Event: name, Data: map[string]json.RawMessage{}}
	e.Broadcast.ChannelID = channel
	for k, v := range data {
		b, _ := json.Marshal(v)
		if m, ok := v.(map[string]any); ok { // Mattermost sends these as JSON inside a string
			inner, _ := json.Marshal(m)
			b, _ = json.Marshal(string(inner))
		}
		e.Data[k] = b
	}
	return e
}

// serverFrame is an unmasked frame, as a server sends.
func serverFrame(fin bool, op byte, payload []byte) []byte {
	b0 := op
	if fin {
		b0 |= 0x80
	}
	b := []byte{b0}
	if len(payload) < 126 {
		b = append(b, byte(len(payload)))
	} else {
		b = append(b, 126, byte(len(payload)>>8), byte(len(payload)))
	}
	return append(b, payload...)
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer bot-token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	p := strings.TrimPrefix(r.URL.Path, "/api/v4/")
	j := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	switch {
	case p == "websocket":
		s.mu.Lock()
		var batch []WSEvent
		if len(s.batches) > 0 {
			batch, s.batches = s.batches[0], s.batches[1:]
		}
		s.mu.Unlock()
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + acceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n\r\n")
		_, _ = rw.Write(serverFrame(true, opPing, []byte("hi")))
		_ = rw.Flush()
		if _, _, pl, err := readFrame(bufio.NewReader(rw)); err == nil && string(pl) == "hi" {
			s.mu.Lock()
			s.pongs++
			s.mu.Unlock()
		}
		for _, e := range batch {
			b, _ := json.Marshal(e)
			half := len(b) / 2 // fragmented, to check continuation frames
			_, _ = rw.Write(serverFrame(false, opText, b[:half]))
			_, _ = rw.Write(serverFrame(true, opCont, b[half:]))
		}
		_, _ = rw.Write(serverFrame(true, opClose, nil))
		_ = rw.Flush()
		time.Sleep(20 * time.Millisecond)
	case p == "users/me":
		j(map[string]string{"id": "u_bot"})
	case p == "channels/c_in":
		j(map[string]string{"name": "support", "type": "O"})
	case p == "channels/c_out":
		j(map[string]string{"name": "secret", "type": "P"})
	case p == "channels/c_dm":
		j(map[string]string{"name": "u_bot__u_ann", "type": "D"})
	case p == "channels/c_in/members/me" || p == "channels/c_dm/members/me":
		j(map[string]string{"user_id": "u_bot"})
	case p == "channels/direct":
		j(map[string]string{"id": "c_dm"})
	case p == "posts" && r.Method == http.MethodPost:
		var body map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		s.mu.Lock()
		s.posts = append(s.posts, body)
		s.mu.Unlock()
		j(map[string]string{"id": "p_new"})
	case strings.HasPrefix(p, "users/"):
		j(map[string]string{"email": "ann@acme.example"})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// SB-6005: over the websocket, a save reaction in a channel the bot is in sends the reactor
// one direct message with the offer, and reads nothing else; one in a channel the bot isn't
// in does nothing; a direct message to the bot is answered as its sender; pings are
// answered; a dropped connection reconnects.
func TestWebsocketReactionsAndDirectQuestions(t *testing.T) {
	const post = "abcdefghijklmnopqrstuvwxyz"
	fake := &wsServer{batches: [][]WSEvent{
		{
			event("reaction_added", "c_in", map[string]any{"reaction": map[string]any{"user_id": "u_ann", "post_id": post, "emoji_name": "brain"}}),
			event("reaction_added", "c_out", map[string]any{"reaction": map[string]any{"user_id": "u_ann", "post_id": post, "emoji_name": "brain"}}),
			event("reaction_added", "c_in", map[string]any{"reaction": map[string]any{"user_id": "u_ann", "post_id": post, "emoji_name": "thumbsup"}}),
		},
		{
			event("posted", "c_dm", map[string]any{"channel_type": "D", "post": map[string]any{"user_id": "u_ann", "channel_id": "c_dm", "message": "What is our refund limit?"}}),
			event("posted", "c_dm", map[string]any{"channel_type": "D", "post": map[string]any{"user_id": "u_bot", "channel_id": "c_dm", "message": "my own post"}}),
		},
	}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	tok := func(context.Context) (string, error) { return "bot-token", nil }
	dir := users{{ID: "usr_ann", Tenant: tenant, Emails: []string{"ann@acme.example"}, Active: true}}
	ag := &asker{}
	a := &App{Tenant: tenant, Base: srv.URL, TeamID: "team1", Token: func(context.Context) (string, error) { return "cmd-token", nil },
		Rules: &slackapp.App{Tenant: tenant, Users: dir, Issuer: tenancy.NewIssuer(tenant), Agent: ag,
			Groups: func(context.Context, string, string) ([]string, error) { return []string{"support"}, nil },
			Email:  APIEmail(srv.URL, tok, srv.Client())},
		Agent: ag, ActionURL: "https://brain.acme.example/chat/mattermost/actions",
		Threads: DM{Base: srv.URL, TeamID: "team1", Token: tok, Client: srv.Client()},
		Thread: func(context.Context, string) ([]Post, error) {
			t.Error("the thread was read before the person chose")
			return nil, nil
		}}
	ctx, cancel := context.WithCancel(context.Background())
	connects := 0
	var mu sync.Mutex
	sock := &Socket{Base: srv.URL, Token: tok, On: a.Event, Backoff: 10 * time.Millisecond, Connected: func() {
		mu.Lock()
		connects++
		if connects == 3 {
			cancel()
		}
		mu.Unlock()
	}}
	done := make(chan struct{})
	go func() { sock.Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the socket didn't reconnect")
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.posts) != 2 {
		t.Fatalf("posts: %v", fake.posts)
	}
	offer := fake.posts[0]
	if offer["channel_id"] != "c_dm" || offer["message"] != slackapp.MsgReactOffer {
		t.Fatalf("offer: %v", offer)
	}
	b, _ := json.Marshal(offer["props"])
	if !strings.Contains(string(b), KindReactSave) || !strings.Contains(string(b), KindReactAsk) || !strings.Contains(string(b), `"sig"`) {
		t.Fatalf("offer buttons: %s", b)
	}
	if ag.asked == nil || ag.asked[0] != "usr_ann: What is our refund limit?" || len(ag.asked) != 1 {
		t.Fatalf("asked: %v", ag.asked)
	}
	if fake.pongs < 2 {
		t.Fatalf("pings answered: %d", fake.pongs)
	}
}

// SB-6005: a press of the offer is signed by the cell; a forged one, or one by someone
// else, reads nothing.
func TestReactPressIsSignedAndOwned(t *testing.T) {
	a, _, _, mux := app(t)
	a.ActionURL = "https://brain.acme.example/chat/mattermost/actions"
	read := 0
	a.Thread = func(context.Context, string) ([]Post, error) { read++; return nil, nil }
	c := ShareContext{Kind: KindReactAsk, User: "u_ann", Post: "abcdefghijklmnopqrstuvwxyz", Channel: "c_in", ChannelName: "support"}
	c.Sig = sign("cmd-token", c)
	press := func(user string, c ShareContext) string {
		b, _ := json.Marshal(map[string]any{"user_id": user, "team_id": "team1", "context": c})
		req := httptest.NewRequest("POST", "/chat/mattermost/actions", strings.NewReader(string(b)))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	forged := c
	forged.Channel = "c_out"
	if out := press("u_ann", forged); !strings.Contains(out, "unauthorized") || read != 0 {
		t.Fatalf("forged: %q %d", out, read)
	}
	if out := press("u_eve", c); !strings.Contains(out, MsgShareFail) || read != 0 {
		t.Fatalf("someone else: %q %d", out, read)
	}
	press("u_ann", c)
	if read != 1 {
		t.Fatalf("the owner's press didn't read the thread: %d", read)
	}
}
