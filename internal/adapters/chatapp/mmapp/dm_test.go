package mmapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/second-brain/internal/core/notify"
)

func TestDMSendsADirectMessage(t *testing.T) {
	var post map[string]string
	var direct []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer bot-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v4/users/me":
			_, _ = io.WriteString(w, `{"id":"bot1","is_bot":true}`)
		case "/api/v4/users/email/ann@acme.example":
			_, _ = io.WriteString(w, `{"id":"u1"}`)
		case "/api/v4/users/email/left@acme.example":
			_, _ = io.WriteString(w, `{"id":"u2","delete_at":5}`)
		case "/api/v4/channels/direct":
			_ = json.NewDecoder(r.Body).Decode(&direct)
			_, _ = io.WriteString(w, `{"id":"ch1"}`)
		case "/api/v4/posts":
			_ = json.NewDecoder(r.Body).Decode(&post)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"p1"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	d := DM{Base: srv.URL, Token: func(context.Context) (string, error) { return "bot-token", nil }, Client: srv.Client()}
	m := notify.Message{Text: "1 of your questions has a reply", Link: "https://brain.acme.example/?tab=questions"}
	if err := d.Send(context.Background(), notify.Person{Email: "ann@acme.example"}, m); err != nil {
		t.Fatal(err)
	}
	if len(direct) != 2 || direct[0] != "bot1" || direct[1] != "u1" || post["channel_id"] != "ch1" ||
		post["message"] != `1 of your questions has a reply [Open](https://brain.acme.example/?tab=questions)` {
		t.Errorf("direct %v, post %v", direct, post)
	}
	for _, email := range []string{"nobody@acme.example", "left@acme.example"} {
		if err := d.Send(context.Background(), notify.Person{Email: email}, m); !errors.Is(err, notify.ErrNoSuchPerson) {
			t.Errorf("%s: %v", email, err)
		}
	}
	if got := DMText(notify.Message{Text: "2 new questions for the a_b space"}); got != `2 new questions for the a\_b space` {
		t.Error(got)
	}
}
