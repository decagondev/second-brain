package slackapp

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
)

// SB-3304: a claim whose sources disagree says so in words, with its links.
func TestSourcesDisagree(t *testing.T) {
	r := ask.Response{Trace: "tr_1", Answer: ask.Answer{Claims: []answer.Claim{{Text: "Returns take 14 or 30 days.", Disagree: true,
		Cites: []answer.Cite{{Cite: "docs:uk@1"}, {Cite: "docs:eu@1"}}}}}}
	text := Render("returns?", r).Text
	if !strings.Contains(text, "_Sources disagree:_ Returns take") || !strings.Contains(text, "docs:eu@1") {
		t.Fatalf("%q", text)
	}
}
