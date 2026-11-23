package cell

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/notify"
)

// SB-2602 through a running cell: Send me a test goes through the cell's real channel (a
// fake Mattermost here) with exactly the fixed text, at most once a minute; an operator's
// test by email needs the operator role.
func TestSendMeATestThroughARunningCell(t *testing.T) {
	var mu sync.Mutex
	var posts []string
	mm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/users/me":
			_, _ = io.WriteString(w, `{"id":"bot1","is_bot":true}`)
		case "/api/v4/users/email/ann@acme.example":
			_, _ = io.WriteString(w, `{"id":"u1"}`)
		case "/api/v4/channels/direct":
			_, _ = io.WriteString(w, `{"id":"ch1"}`)
		case "/api/v4/posts":
			var p map[string]string
			_ = json.NewDecoder(r.Body).Decode(&p)
			mu.Lock()
			posts = append(posts, p["message"])
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"p1"}`)
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
	code, body := r.do(t, "POST", "/v1/me/notifications/test", "")
	if code != 200 || !strings.Contains(body, `"sent":true`) || !strings.Contains(body, `"via":"mattermost"`) {
		t.Fatalf("test: %d %s", code, body)
	}
	mu.Lock()
	got := append([]string(nil), posts...)
	mu.Unlock()
	if len(got) != 1 || !strings.HasPrefix(got[0], notify.TestText+" [Open](") {
		t.Fatalf("the chat got %q", got)
	}
	if code, body := r.do(t, "POST", "/v1/me/notifications/test", ""); code != http.StatusTooManyRequests {
		t.Fatalf("a second test within a minute: %d %s", code, body)
	}
	if code, _ := r.do(t, "POST", "/v1/admin/notifications/test", `{"to":"ann@acme.example"}`); code != http.StatusForbidden {
		t.Fatalf("a member's test to someone else: %d", code)
	}
	r.signIn(t, "bob") // the chat doesn't know bob: the cell says the channel didn't take it
	if code, body := r.do(t, "POST", "/v1/me/notifications/test", ""); code != 200 || !strings.Contains(body, `"sent":false`) || !strings.Contains(body, `"via":"mattermost"`) {
		t.Fatalf("bob's test: %d %s", code, body)
	}
}
