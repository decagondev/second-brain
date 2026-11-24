package health

import "testing"

// SB-3205: a doubled week warns, a flat one doesn't, and no durations is no line.
func TestAnswerTime(t *testing.T) {
	flat := []int{900, 1000, 1100, 1200, 4000}
	doubled := []int{1800, 2000, 2200, 2400, 8000}
	if a := AnswerFrom(flat, flat); a == nil || a.Warning || a.P95Seconds != 4 {
		t.Fatalf("flat: %+v", a)
	}
	if a := AnswerFrom(doubled, flat); a == nil || !a.Warning || a.P95Seconds != 8 || a.P95SecondsBefore != 4 {
		t.Fatalf("doubled: %+v", a)
	}
	if AnswerFrom(nil, nil) != nil {
		t.Fatal("a line with no durations")
	}
	r := Report{AnswerTime: AnswerFrom(doubled, flat)}
	found := false
	for _, c := range r.Checks() {
		found = found || (c.Name == "answer_time" && c.Warning)
	}
	if !found {
		t.Fatal("answer_time isn't a warning check")
	}
}
