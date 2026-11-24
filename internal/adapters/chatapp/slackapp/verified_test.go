package slackapp

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
)

// SB-3607: the chat answer says how many of its sources someone has checked.
func TestRenderSaysHowManyVerified(t *testing.T) {
	r := ask.Response{Trace: "tr_1", Answer: ask.Answer{Claims: []answer.Claim{{Text: "x", Cites: []answer.Cite{{Cite: "docs:a@1"}}}}},
		Verified: &ask.VerifiedCount{Of: 2, Marked: 1}}
	if m := Render("q", r); !strings.Contains(m.Text, "_1 of 2 sources verified._") {
		t.Fatal(m.Text)
	}
	r.Verified = nil
	if m := Render("q", r); strings.Contains(m.Text, "verified") {
		t.Fatal(m.Text)
	}
}
