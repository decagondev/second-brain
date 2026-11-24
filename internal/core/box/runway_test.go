package box

import (
	"strings"
	"testing"
	"time"
)

const gb = 1 << 30

func days(path string, used ...uint64) []Sample {
	start := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	var out []Sample
	for i, u := range used {
		out = append(out, Sample{Day: start.AddDate(0, 0, i).Format("2006-01-02"), Path: path, Used: u * gb, Size: 100 * gb})
	}
	return out
}

func runwayLine(t *testing.T, hist []Sample) Line {
	t.Helper()
	var got []Line
	checkRunway(Facts{DiskHistory: hist}, func(check string, v Verdict, detail, fix string) {
		got = append(got, Line{Check: check, Verdict: v, Detail: detail, Fix: fix})
	})
	if len(got) != 1 {
		t.Fatalf("lines: %+v", got)
	}
	return got[0]
}

// SB-3206: steady growth gives the right estimate; a shrinking disk has no runway; two
// days of history is not enough; under 30 days warns and under 7 fails.
func TestDiskRunway(t *testing.T) {
	if d, ok, growing := Runway(days("/var", 50, 51, 52, 53), "/var"); !ok || !growing || d != 47 {
		t.Fatalf("1 GB a day with 47 free: %v %v %v", d, ok, growing)
	}
	if _, ok, growing := Runway(days("/var", 60, 55, 50), "/var"); !ok || growing {
		t.Fatal("a shrinking disk has a runway")
	}
	if _, ok, _ := Runway(days("/var", 50, 60), "/var"); ok {
		t.Fatal("two days gave an estimate")
	}
	if l := runwayLine(t, days("/var", 50, 60)); l.Verdict != Pass || l.Detail != "not enough history yet" {
		t.Fatalf("two days: %+v", l)
	}
	if l := runwayLine(t, days("/var", 60, 55, 50)); l.Verdict != Pass || l.Detail != "not growing" {
		t.Fatalf("shrinking: %+v", l)
	}
	if l := runwayLine(t, days("/var", 40, 42, 44)); l.Verdict != Warn || !strings.Contains(l.Detail, "about 28 days") || !strings.Contains(l.Fix, "#disk-runway") {
		t.Fatalf("28 days: %+v", l)
	}
	if l := runwayLine(t, days("/var", 70, 80, 90)); l.Verdict != Fail || !strings.Contains(l.Detail, "about 1 days") {
		t.Fatalf("1 day: %+v", l)
	}
	if l := runwayLine(t, append(days("/a", 10, 10, 10), days("/b", 40, 42, 44)...)); l.Verdict != Warn || !strings.HasPrefix(l.Detail, "/b") {
		t.Fatalf("the shortest of two: %+v", l)
	}
	var none []Line
	checkRunway(Facts{}, func(check string, v Verdict, detail, fix string) { none = append(none, Line{}) })
	if len(none) != 0 {
		t.Fatal("a line with no history")
	}
}

// SB-3206: one sample a day per disk, the newest wins, the last 14 days kept.
func TestAddSamples(t *testing.T) {
	at := time.Date(2027, 3, 20, 10, 0, 0, 0, time.UTC)
	hist := days("/var", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19) // 2027-03-01 to 03-19
	got := AddSamples(hist, []Disk{{Path: "/var", UsedBytes: 20 * gb, SizeBytes: 100 * gb}, {Path: "/x", UsedPct: 3}}, at)
	if len(got) != RunwayKeepDays || got[0].Day != "2027-03-07" || got[len(got)-1].Day != "2027-03-20" || got[len(got)-1].Used != 20*gb {
		t.Fatalf("%+v", got)
	}
	again := AddSamples(got, []Disk{{Path: "/var", UsedBytes: 21 * gb, SizeBytes: 100 * gb}}, at.Add(time.Hour))
	if len(again) != RunwayKeepDays || again[len(again)-1].Used != 21*gb {
		t.Fatalf("the same day twice: %+v", again[len(again)-1])
	}
}
