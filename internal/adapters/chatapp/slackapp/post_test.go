package slackapp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SB-5304: Post posts in the channel as the bot and returns the message's timestamp.
func TestPostInAChannel(t *testing.T) {
	var posted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			_ = json.NewDecoder(r.Body).Decode(&posted)
			if posted["channel"] == "#ops" {
				_, _ = io.WriteString(w, `{"ok":true,"ts":"1700000000.0001"}`)
				return
			}
			_, _ = io.WriteString(w, `{"ok":false,"error":"not_in_channel"}`)
		}
	}))
	defer srv.Close()
	d := DM{Base: srv.URL, Token: func(context.Context) (string, error) { return "xoxb-test", nil }, Client: srv.Client()}
	ts, err := d.Post(context.Background(), "#ops", "Sent by ann@acme.example via Second Brain:\n<b>bold</b>")
	if err != nil || ts != "1700000000.0001" || posted["text"] != "Sent by ann@acme.example via Second Brain:\n&lt;b&gt;bold&lt;/b&gt;" {
		t.Fatalf("%q %v %v", ts, err, posted)
	}
	if _, err := d.Post(context.Background(), "#elsewhere", "x"); err == nil {
		t.Fatal("a channel the bot isn't in was posted to")
	}
}
