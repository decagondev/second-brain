package gdrive

import "testing"

// SB-5306: a Meet transcript is known by its name, and its text keeps the speakers' lines
// and drops Meet's timestamps.
func TestMeetTranscripts(t *testing.T) {
	if !IsMeetTranscript("Weekly ops sync (2026-11-20 10:00 GMT) - Transcript") || IsMeetTranscript("Transcript of nothing") {
		t.Fatal("IsMeetTranscript")
	}
	got := MeetText("\ufeffAttendees\r\n00:00:00\r\nAnn: We ship on Friday.\r\n\r\n00:01:05\r\nBob: I'll book the vans.\r\n")
	if got != "Attendees\nAnn: We ship on Friday.\nBob: I'll book the vans." {
		t.Fatalf("%q", got)
	}
}
