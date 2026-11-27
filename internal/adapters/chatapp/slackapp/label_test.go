package slackapp

import (
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
)

// SB-5405, LAW-32: a confidential answer names its label, and may not go to a channel.
func TestConfidentialAnswerStaysWithThePerson(t *testing.T) {
	m := Render("plan?", ask.Response{Trace: "tr_1", Label: "confidential"})
	if !strings.Contains(m.Text, "Confidential: cites confidential documents") || m.ResponseType != "ephemeral" {
		t.Fatalf("render: %+v", m)
	}
	if ask.MayPostInChannel("confidential") || ask.MayPostInChannel("restricted") || !ask.MayPostInChannel("internal") || !ask.MayPostInChannel("") {
		t.Fatal("the channel gate")
	}
	if strings.Contains(Render("menu?", ask.Response{Trace: "tr_2", Label: "internal"}).Text, "Internal:") {
		t.Fatal("an internal answer named its label")
	}
}
