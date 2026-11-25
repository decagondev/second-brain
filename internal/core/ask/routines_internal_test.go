package ask

import (
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
)

// SB-4302: a daily routine runs at 08:00 on working days, a weekly one at 08:00 on its day,
// in the digest zone, once per slot.
func TestRoutineSlots(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	sat := time.Date(2026, 11, 28, 12, 0, 0, 0, berlin) // a Saturday
	if got := RoutineSlot(Routine{Every: EveryDaily}, sat, berlin); !got.Equal(time.Date(2026, 11, 27, 8, 0, 0, 0, berlin)) {
		t.Fatalf("daily on a Saturday: %v", got)
	}
	wedEarly := time.Date(2026, 11, 25, 7, 0, 0, 0, berlin)
	if got := RoutineSlot(Routine{Every: EveryWeekly, Day: "wednesday"}, wedEarly, berlin); !got.Equal(time.Date(2026, 11, 18, 8, 0, 0, 0, berlin)) {
		t.Fatalf("weekly before its hour: %v", got)
	}
	r := Routine{Every: EveryWeekly, Day: "wednesday", Last: time.Date(2026, 11, 20, 0, 0, 0, 0, berlin)}
	if RoutineDue(r, wedEarly, berlin) {
		t.Fatal("due before its slot")
	}
	if !RoutineDue(r, wedEarly.Add(2*time.Hour), berlin) {
		t.Fatal("not due after its slot")
	}
	r.Last = wedEarly.Add(2 * time.Hour)
	if RoutineDue(r, wedEarly.Add(5*time.Hour), berlin) {
		t.Fatal("due twice in one slot")
	}
}

// SB-4302: the claims that cite a new source come first, in their order.
func TestNewFirst(t *testing.T) {
	cl := func(text, cite string) answer.Claim {
		return answer.Claim{Text: text, Cites: []answer.Cite{{Cite: cite}}}
	}
	got := newFirst([]answer.Claim{cl("a", "docs:x@1"), cl("b", "docs:y@2"), cl("c", "docs:z@1")}, map[string]bool{"docs:y@2": true, "docs:z@1": true})
	if got[0].Text != "b" || got[1].Text != "c" || got[2].Text != "a" {
		t.Fatalf("order: %+v", got)
	}
}
