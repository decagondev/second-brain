package mmapp

import (
	"context"
	"strings"
	"testing"
)

// SB-3207: while an admin's notice stands, it is the first line of every answer.
func TestNoticeComesFirst(t *testing.T) {
	a, _, posts, mux := app(t)
	a.Rules.Notice = func(context.Context) string { return "Notice: Upgrade at 18:00." }
	send(mux, cmd(a, "u_ann", "refund limit?"))
	if len(*posts) != 1 || !strings.HasPrefix((*posts)[0].Text, "**Notice: Upgrade at 18:00.**\n") {
		t.Fatalf("posts %+v", *posts)
	}
}
