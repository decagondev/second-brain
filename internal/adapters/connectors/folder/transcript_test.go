package folder

import (
	"strings"
	"testing"
)

// SB-3701: each cue is its start time as a heading and its speaker and text; a broken cue
// is skipped and counted.
func TestTranscriptText(t *testing.T) {
	vtt := "WEBVTT\n\nNOTE made by the meeting tool\n\n1\n00:00:01.000 --> 00:00:04.000\n<v Ann Lee>We ship on Friday.</v>\n\n" +
		"00:12:40.500 --> 00:12:44.000 align:start\nBen: The cutoff moves to 4pm.\n\nnot a timing\ntext\n\n01:02:03.000 --> 01:02:05.000\n\n"
	got, skipped := TranscriptText([]byte(vtt))
	want := "### 00:00:01\nAnn Lee: We ship on Friday.\n\n### 00:12:40\nBen: The cutoff moves to 4pm.\n\n2 cues couldn't be read."
	if got != want || skipped != 2 {
		t.Fatalf("%d skipped\n%s", skipped, got)
	}
	srt := "1\r\n00:00:05,250 --> 00:00:07,000\r\nAgreed, <i>Friday</i>.\r\n\r\n2\r\n00:01:00,000 --> 00:01:02,000\r\nCarla: Done\r\n"
	got, skipped = TranscriptText([]byte(srt))
	if skipped != 0 || got != "### 00:00:05\nAgreed, Friday.\n\n### 00:01:00\nCarla: Done" {
		t.Fatalf("%d\n%s", skipped, got)
	}
	if !Document("Ops/standup.vtt") || !Document("Ops/standup.SRT") {
		t.Fatal("a transcript isn't a document")
	}
	if got, _ := TranscriptText([]byte{0xff, 0xfe}); got != "" || strings.Contains(got, "###") {
		t.Fatal("non-UTF-8 read")
	}
}
