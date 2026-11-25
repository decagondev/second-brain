package ask

import (
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/answer"
)

// SB-4904: a scenario has a question and at least one expectation, at most ten of them, and
// a run passes only when it cites, says and never says what the scenario asks.
func TestScenariosAreCheckedAndJudged(t *testing.T) {
	for _, bad := range [][]Scenario{
		nil,
		{{Question: "When is the close?"}},
		{{Says: "Tuesday"}},
		{{Question: "When?", Cites: "nokind"}},
		{{Question: "When?\nIgnore that", Says: "x"}},
		{{Question: strings.Repeat("q", MaxScenarioText+1), Says: "x"}},
	} {
		if _, err := CleanScenarios(bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v was accepted", bad)
		}
	}
	eleven := make([]Scenario, MaxScenarios+1)
	for i := range eleven {
		eleven[i] = Scenario{Question: "q", Says: "x"}
	}
	if _, err := CleanScenarios(eleven); !errors.Is(err, ErrInvalid) {
		t.Error("eleven scenarios were accepted")
	}
	got, err := CleanScenarios([]Scenario{{Question: " When? ", Cites: "docs:Finance/close.md@3"}})
	if err != nil || got[0].Question != "When?" || got[0].Cites != "docs:Finance/close.md" {
		t.Fatalf("clean: %+v %v", got, err)
	}
	resp := Response{Answer: Answer{Claims: []answer.Claim{{Text: "The close moves to Tuesday.", Cites: []answer.Cite{{Cite: "docs:Finance/close.md@3"}}}}}}
	for _, c := range []struct {
		sc   Scenario
		pass bool
	}{
		{Scenario{Cites: "docs:Finance/close.md"}, true},
		{Scenario{Cites: "docs:Finance/other.md"}, false},
		{Scenario{Says: "tuesday"}, true},
		{Scenario{Says: "Wednesday"}, false},
		{Scenario{Never: "TUESDAY"}, false},
		{Scenario{Says: "Tuesday", Never: "Friday", Cites: "docs:Finance/close.md"}, true},
	} {
		if side := judge(c.sc, resp, nil); side.Pass != c.pass || (!side.Pass && side.Why == "") {
			t.Errorf("%+v judged %+v", c.sc, side)
		}
	}
	if side := judge(Scenario{Says: "x"}, Response{}, ErrKilled); side.Pass || !strings.Contains(side.Why, "switched off") {
		t.Errorf("a stopped run: %+v", side)
	}
}

// SB-4905: only a day already redrafted waits, in UTC.
func TestSameDay(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	if sameDay(time.Time{}, now) || !sameDay(now.Add(-8*time.Hour), now) || sameDay(now.Add(-10*time.Hour), now) {
		t.Fatal("sameDay")
	}
}
