package box

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// The disk runway (SB-3206): days of samples kept, the fewest that give an estimate, and
// the runway that warns and the one that fails.
const (
	RunwayKeepDays = 14
	RunwayMinDays  = 3
	RunwayWarnDays = 30
	RunwayFailDays = 7
)

// Sample is one disk's use on one day (UTC), in bytes.
type Sample struct {
	Day  string `json:"day"`
	Path string `json:"path"`
	Used uint64 `json:"used"`
	Size uint64 `json:"size"`
}

// AddSamples adds today's use of each disk that has its bytes, one sample a day per disk
// (the newest wins), and keeps the last RunwayKeepDays days, oldest first.
func AddSamples(hist []Sample, disks []Disk, at time.Time) []Sample {
	day := at.UTC().Format("2006-01-02")
	oldest := at.UTC().AddDate(0, 0, -(RunwayKeepDays - 1)).Format("2006-01-02")
	by := map[[2]string]Sample{}
	for _, s := range hist {
		if s.Day >= oldest && s.Day <= day {
			by[[2]string{s.Day, s.Path}] = s
		}
	}
	for _, d := range disks {
		if d.SizeBytes > 0 {
			by[[2]string{day, d.Path}] = Sample{Day: day, Path: d.Path, Used: d.UsedBytes, Size: d.SizeBytes}
		}
	}
	out := make([]Sample, 0, len(by))
	for _, s := range by {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day < out[j].Day
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// Runway is one disk's days until full at the growth from its first sample to its last.
// ok is false with fewer than RunwayMinDays days of samples; growing is false when the disk
// isn't growing, which has no runway.
func Runway(hist []Sample, path string) (days float64, ok, growing bool) {
	var ss []Sample
	for _, s := range hist {
		if s.Path == path {
			ss = append(ss, s)
		}
	}
	if len(ss) < RunwayMinDays {
		return 0, false, false
	}
	first, last := ss[0], ss[len(ss)-1]
	a, errA := time.Parse("2006-01-02", first.Day)
	b, errB := time.Parse("2006-01-02", last.Day)
	span := b.Sub(a).Hours() / 24
	if errA != nil || errB != nil || span < RunwayMinDays-1 {
		return 0, false, false
	}
	if last.Used <= first.Used || last.Size <= last.Used {
		return 0, true, last.Size <= last.Used // full is no runway at all
	}
	perDay := float64(last.Used-first.Used) / span
	return float64(last.Size-last.Used) / perDay, true, true
}

// checkRunway judges the shortest runway of the disks in the history; no history, as on a
// box whose host step predates it, is no line.
func checkRunway(f Facts, add adder) {
	if len(f.DiskHistory) == 0 {
		return
	}
	paths := map[string]bool{}
	for _, s := range f.DiskHistory {
		paths[s.Path] = true
	}
	names := make([]string, 0, len(paths))
	for p := range paths {
		names = append(names, p)
	}
	sort.Strings(names)
	shortest, where, known := math.Inf(1), "", false
	for _, p := range names {
		days, ok, growing := Runway(f.DiskHistory, p)
		if !ok {
			continue
		}
		known = true
		if growing && days < shortest {
			shortest, where = days, p
		}
	}
	fix := "free space on " + where + ", or give it a bigger disk" + see("disk-runway")
	switch {
	case !known:
		add("disk runway", Pass, "not enough history yet", "")
	case where == "":
		add("disk runway", Pass, "not growing", "")
	case shortest < RunwayFailDays:
		add("disk runway", Fail, fmt.Sprintf("%s full in about %.0f days", where, math.Floor(shortest)), fix)
	case shortest < RunwayWarnDays:
		add("disk runway", Warn, fmt.Sprintf("%s full in about %.0f days", where, math.Floor(shortest)), fix)
	default:
		add("disk runway", Pass, fmt.Sprintf("%s full in about %.0f days", where, math.Floor(shortest)), "")
	}
}
