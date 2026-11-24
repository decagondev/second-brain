package slackapp

import (
	"context"
	"strings"
	"testing"
)

// SB-3207: while an admin's notice stands, it is the first line of every answer, escaped.
func TestNoticeComesFirst(t *testing.T) {
	a, posted := newApp(directory, &agent{})
	a.Notice = func(context.Context) string { return "Notice: Upgrade at 18:00 & read-only after" }
	send(a, "/chat/slack/commands", command("UALICE", "refund limit?"), now, "")
	send(a, "/chat/slack/commands", command("UCAROL", "refund limit?"), now, "") // no match: still first
	if len(*posted) != 2 {
		t.Fatalf("posted %d", len(*posted))
	}
	for _, m := range *posted {
		if !strings.HasPrefix(m.Text, "*Notice: Upgrade at 18:00 &amp; read-only after*\n") {
			t.Fatalf("text %q", m.Text)
		}
	}
	if b := (*posted)[0].Blocks; len(b) == 0 || !strings.HasPrefix(b[0].Text.Text, "*Notice: ") || !strings.Contains(b[0].Text.Text, "refund limit?") {
		t.Fatalf("blocks %+v", b)
	}
	a.Notice = func(context.Context) string { return "" }
	send(a, "/chat/slack/commands", command("UALICE", "refund limit?"), now, "")
	if strings.Contains((*posted)[2].Text, "Notice:") {
		t.Fatalf("no notice once it ends: %q", (*posted)[2].Text)
	}
}
