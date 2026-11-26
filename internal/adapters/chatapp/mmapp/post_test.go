package mmapp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SB-5304: Post finds the channel by name in the team and posts there as the bot.
func TestPostInAChannel(t *testing.T) {
	var post map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/teams/team1/channels/name/ops":
			_, _ = io.WriteString(w, `{"id":"ch9"}`)
		case "/api/v4/posts":
			_ = json.NewDecoder(r.Body).Decode(&post)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"p9"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	d := DM{Base: srv.URL, TeamID: "team1", Token: func(context.Context) (string, error) { return "bot-token", nil }, Client: srv.Client()}
	id, err := d.Post(context.Background(), "#ops", "Sent by ann@acme.example via Second Brain:\nThe dock opens at seven.")
	if err != nil || id != "p9" || post["channel_id"] != "ch9" || post["message"] == "" {
		t.Fatalf("%q %v %v", id, err, post)
	}
	if _, err := d.Post(context.Background(), "#nowhere", "x"); err == nil {
		t.Fatal("a channel the team hasn't was posted to")
	}
}
