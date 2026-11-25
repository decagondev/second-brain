package ask

import (
	"testing"
	"time"
)

// SB-4204: an incident is a ticket whose Labels line, near its top, names a label.
func TestIncidentLabels(t *testing.T) {
	text := "INC-42 Payments down\n\nLabels: Sev-1, payments\n\nThe queue stopped."
	if got := IncidentLabels(text); len(got) != 2 || got[0] != "sev-1" {
		t.Fatalf("labels: %v", got)
	}
	if !IsIncident(text, []string{"SEV-1"}) || IsIncident(text, []string{"sev-2"}) {
		t.Fatal("matching")
	}
	if IsIncident("a\nb\nc\nd\ne\nLabels: sev-1", []string{"sev-1"}) {
		t.Fatal("a Labels line below the top made an incident")
	}
}

// SB-4207: a goal is silent three weeks after its last published check-in, or after it was
// first seen when it has none.
func TestSilentGoals(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	qs := []StewardQuestion{
		{Thread: &SavedThread{Kind: DraftCheckin, Goal: "docs:g1"}, Reply: &StewardReply{Note: "rc_1", NoteAt: now.Add(-10 * 24 * time.Hour)}},
		{Thread: &SavedThread{Kind: DraftCheckin, Goal: "docs:g2"}}, // drafted, never published
	}
	if Silent(now.Add(-60*24*time.Hour), LastCheckin(qs, "docs:g1"), now) {
		t.Fatal("published 10 days ago and silent")
	}
	if !Silent(now.Add(-22*24*time.Hour), LastCheckin(qs, "docs:g2"), now) {
		t.Fatal("never published in 22 days and not silent")
	}
	if Silent(now.Add(-5*24*time.Hour), time.Time{}, now) {
		t.Fatal("a new goal is silent")
	}
}
