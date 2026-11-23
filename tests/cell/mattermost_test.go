package cell

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-2005 through a running cell: a Mattermost slash command from a directory user gets
// an ephemeral answer through the server's response_url; a wrong token gets 401; someone
// the directory doesn't know gets the refusal.
func TestMattermostThroughARunningCell(t *testing.T) {
	replies := make(chan string, 4)
	mm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.Header.Get("Authorization") == "Bearer bot-token" && r.URL.Path == "/api/v4/users/u_ann":
			_, _ = w.Write([]byte(`{"email":"ann@acme.example"}`))
		case r.Method == "GET" && r.URL.Path == "/api/v4/users/u_eve":
			_, _ = w.Write([]byte(`{"email":"eve@acme.example"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/hooks/commands/"):
			b, _ := io.ReadAll(r.Body)
			replies <- string(b)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(mm.Close)
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, celltest.Manifest, map[string]string{"mattermost": `{"command_token":"cmd-token","bot_token":"bot-token"}`})
		cfg.Mattermost.Base, cfg.Mattermost.TeamID = mm.URL, "team1"
		return cfg
	})
	r.signIn(t, "ann")
	send := func(token, user string) int {
		v := url.Values{"token": {token}, "team_id": {"team1"}, "user_id": {user}, "command": {"/brain"}, "text": {"refund limit?"},
			"channel_name": {"town-square"}, "response_url": {mm.URL + "/hooks/commands/x1"}}
		code, _ := r.do(t, "POST", "/chat/mattermost", v.Encode(), "Content-Type", "application/x-www-form-urlencoded")
		return code
	}
	next := func() map[string]string {
		select {
		case b := <-replies:
			var m map[string]string
			_ = json.Unmarshal([]byte(b), &m)
			return m
		case <-time.After(20 * time.Second):
			t.Fatal("no reply through the response_url")
			return nil
		}
	}
	if code := send("wrong", "u_ann"); code != http.StatusUnauthorized {
		t.Fatalf("a wrong token: %d", code)
	}
	if code := send("cmd-token", "u_ann"); code != http.StatusOK {
		t.Fatalf("ann's command: %d", code)
	}
	if m := next(); m["response_type"] != "ephemeral" || !strings.Contains(m["text"], "_Trace tr_") {
		t.Fatalf("ann's answer: %v", m)
	}
	if code := send("cmd-token", "u_eve"); code != http.StatusOK {
		t.Fatalf("eve's command: %d", code)
	}
	if m := next(); !strings.Contains(m["text"], "can't match your Mattermost account") {
		t.Fatalf("eve, unknown to the directory, got: %v", m)
	}
}
