package features

import (
	"testing"
	"time"
)

// SB-5106: a pilot feature is on for the pilot group for 14 days, then for everyone, unless
// it is held; an unknown feature is off.
func TestPilotHoldAndRelease(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	st, changed := State{}.Seen(now)
	if !changed || st.Since["pause-link"] != now {
		t.Fatalf("seen %+v", st)
	}
	if again, changed := st.Seen(now.Add(time.Hour)); changed || again.Since["pause-link"] != now {
		t.Fatal("seen moved the date")
	}
	if st.For("pause-link", []string{"finance"}, now) || !st.For("pause-link", []string{"pilot"}, now) {
		t.Fatal("pilot gate")
	}
	later := now.AddDate(0, 0, PilotDays)
	if !st.For("pause-link", nil, later) || st.Of(List[1], now).OnAt != later {
		t.Fatalf("after %d days: %+v", PilotDays, st.Of(List[1], now))
	}
	st.Held = map[string]bool{"pause-link": true}
	if st.For("pause-link", nil, later) || !st.For("pause-link", []string{"pilot"}, later) || st.Of(List[1], later).State != "held" {
		t.Fatal("hold")
	}
	if st.For("no-such-feature", []string{"pilot"}, later) {
		t.Fatal("an unknown feature is on")
	}
}
