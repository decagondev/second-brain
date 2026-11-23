package slackapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/notify"
)

func TestDMSendsADirectMessage(t *testing.T) {
	var posted map[string]any
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer xoxb-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/users.lookupByEmail":
			switch r.URL.Query().Get("email") {
			case "ann@acme.example":
				_, _ = io.WriteString(w, `{"ok":true,"user":{"id":"U1"}}`)
			case "bot@acme.example":
				_, _ = io.WriteString(w, `{"ok":true,"user":{"id":"U2","is_bot":true}}`)
			default:
				_, _ = io.WriteString(w, `{"ok":false,"error":"users_not_found"}`)
			}
		case "/conversations.open":
			_, _ = io.WriteString(w, `{"ok":true,"channel":{"id":"D1"}}`)
		case "/chat.postMessage":
			_ = json.NewDecoder(r.Body).Decode(&posted)
			_, _ = io.WriteString(w, `{"ok":true}`)
		}
	}))
	defer srv.Close()
	d := DM{Base: srv.URL, Token: func(context.Context) (string, error) { return "xoxb-test", nil }, Client: srv.Client()}
	m := notify.Message{Kind: notify.KindQuestions, Text: "2 new questions for the <finance> space", Link: "https://brain.acme.example/?steward=finance"}
	if err := d.Send(context.Background(), notify.Person{ID: "usr_ann", Email: "ann@acme.example"}, m); err != nil {
		t.Fatal(err)
	}
	if posted["channel"] != "D1" || posted["text"] != "2 new questions for the &lt;finance&gt; space <https://brain.acme.example/?steward=finance|Open>" {
		t.Errorf("posted %v", posted)
	}
	if strings.Join(calls, ",") != "/users.lookupByEmail,/conversations.open,/chat.postMessage" {
		t.Errorf("calls %v", calls)
	}
	for _, email := range []string{"nobody@acme.example", "bot@acme.example"} {
		if err := d.Send(context.Background(), notify.Person{Email: email}, m); !errors.Is(err, notify.ErrNoSuchPerson) {
			t.Errorf("%s: %v", email, err)
		}
	}
	bad := DM{Base: srv.URL, Token: func(context.Context) (string, error) { return "wrong", nil }, Client: srv.Client()}
	if err := bad.Send(context.Background(), notify.Person{Email: "ann@acme.example"}, m); err == nil || errors.Is(err, notify.ErrNoSuchPerson) {
		t.Errorf("a refused token: %v", err)
	}
}

func TestDMTextDropsANonHTTPSLink(t *testing.T) {
	if got := DMText(notify.Message{Text: "1 of your questions has a reply", Link: "javascript:alert(1)"}); got != "1 of your questions has a reply" {
		t.Error(got)
	}
}
