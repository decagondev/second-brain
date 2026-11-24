package slackapp

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/tenancy"
)

type saver struct {
	got  []ask.ThreadMessage
	link string
	who  string
}

func (s *saver) SaveThread(_ context.Context, sc tenancy.Scope, link string, ms []ask.ThreadMessage) (ask.StewardQuestion, error) {
	s.got, s.link, s.who = ms, link, sc.User()
	return ask.StewardQuestion{ID: "qs_1", Space: "company"}, nil
}

func saveAction(user string) url.Values {
	p, _ := json.Marshal(map[string]any{"type": "message_action", "callback_id": SaveCallback, "team": map[string]string{"id": team},
		"user": map[string]string{"id": user}, "channel": map[string]string{"id": "C1", "name": "ops"}, "response_url": hook,
		"message": map[string]string{"ts": "1700000000.000200", "thread_ts": "1700000000.000100"}})
	return url.Values{"payload": {string(p)}}
}

// SB-3703: the shortcut saves the message's whole thread, each author by email, as the
// person who pressed it; someone the cell can't match saves nothing.
func TestSaveToTheBrain(t *testing.T) {
	a, posted := newApp(directory, &agent{})
	sv := &saver{}
	var root string
	a.Saver = sv
	a.Thread = func(_ context.Context, ch, ts string) ([]RawMessage, error) {
		root = ch + "/" + ts
		return []RawMessage{{User: "UALICE", Text: "When is the cutoff?"}, {User: "UBOB", Text: "3pm."}, {User: "UNONE", Text: "Thanks"}}, nil
	}
	if w := send(a, "/chat/slack/interactions", saveAction("UALICE"), now, ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if root != "C1/1700000000.000100" || sv.who != "alice" || sv.link != "https://slack.com/archives/C1/p1700000000000100" {
		t.Fatalf("saved %q as %q at %q", root, sv.who, sv.link)
	}
	if len(sv.got) != 3 || sv.got[0].Email != "alice@acme.example" || sv.got[1].Email != "bob@acme.example" || sv.got[2].Email != "" {
		t.Fatalf("messages %+v", sv.got)
	}
	if len(*posted) != 1 || (*posted)[0].Text != "Sent to the stewards of company." || (*posted)[0].ResponseType != "ephemeral" {
		t.Fatalf("reply %+v", *posted)
	}
	sv.got = nil
	send(a, "/chat/slack/interactions", saveAction("UCAROL"), now, "")
	if sv.got != nil || (*posted)[1].Text != MsgNoMatch {
		t.Fatalf("an inactive user saved a thread: %+v", *posted)
	}
	if ask.ThreadText([]ask.ThreadMessage{{Email: "", Text: "a\nb"}}) != "someone: a b" {
		t.Fatal(ask.ThreadText([]ask.ThreadMessage{{Text: "a\nb"}}))
	}
}
