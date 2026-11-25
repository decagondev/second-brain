package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

// SB-5101: each owner hears once per document version, one message a run, and a second
// run tells no one again.
func TestSuspectsTellOwnersOnce(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, s, _ := world(now)
	j.Suspects = func(context.Context) ([]SuspectDoc, error) {
		return []SuspectDoc{{Cite: "docs:a@1", Space: "finance", Owner: "usr_ann"}, {Cite: "docs:b@2", Space: "finance", Owner: "usr_ann"},
			{Cite: "docs:a@1", Space: "ops", Owner: "usr_ann"}, {Cite: "docs:c@1", Space: "finance", Owner: "usr_gone"}}, nil
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	var mine []string
	for _, g := range s.got {
		if g.msg.Kind == KindSuspect {
			mine = append(mine, g.to.ID+": "+g.msg.Text)
		}
	}
	if len(mine) != 1 || !strings.HasPrefix(mine[0], "usr_ann: 2 of your documents") || strings.Contains(mine[0], "docs:") {
		t.Fatalf("told %v", mine)
	}
	s.got = nil
	j.Now = func() time.Time { return now.Add(time.Hour) }
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, g := range s.got {
		if g.msg.Kind == KindSuspect {
			t.Fatalf("told twice: %v", g)
		}
	}
}

// SB-5102: Masking masks a message's text before it goes, and says how many changed.
func TestMaskingSender(t *testing.T) {
	s := &sender{}
	n := 0
	m := Masking{Next: s, Masked: func(_ context.Context, k int) { n += k }}
	key := "AKIA" + strings.Repeat("Q7", 8)
	if err := m.Send(context.Background(), Person{ID: "usr_ann", Email: "ann@acme.example"}, Message{Kind: KindReplies, Text: "Key " + key + ", card 4111 1111 1111 1111"}); err != nil {
		t.Fatal(err)
	}
	if len(s.got) != 1 || strings.Contains(s.got[0].msg.Text, key) || strings.Contains(s.got[0].msg.Text, "4111 1111 1111 1111") || n != 1 {
		t.Fatalf("sent %+v, masked %d", s.got, n)
	}
}

// SB-5107: a notice about one agent's run in one space carries the pause link only for a
// person who may pause it there.
func TestPauseLinkOnlyForThoseWhoMay(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, s, _ := world(now)
	j.AgentAlerts = func(context.Context, time.Time) ([]AgentAlert, error) {
		return []AgentAlert{{Kind: KindCheck, Key: "check-1", To: []string{"usr_ann", "usr_bob"}, Text: "Did it help?", Agent: "case-brief", Space: "support"}}, nil
	}
	j.MayPause = func(_ context.Context, user, space, agent string) bool {
		return user == "usr_ann" && space == "support" && agent == "case-brief"
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{}
	for _, g := range s.got {
		if g.msg.Kind == KindCheck {
			links[g.to.ID] = g.msg.Pause
		}
	}
	if links["usr_ann"] != "https://brain.acme.example/?pause=case-brief&space=support" || links["usr_bob"] != "" || len(links) != 2 {
		t.Fatalf("links %v", links)
	}
}
