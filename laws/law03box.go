package laws

import (
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/core/box"
)

// LAW-03 for a self-hosted machine's health (SB-1801, SB-1802): a fact the host step
// couldn't gather, or facts too old to trust, are never judged healthy, and the doctor
// never exits 0 with a failing or unknown line.

// boxJudge is the system under test: the real one is box.Check; a broken one is passed
// by the self-tests.
type boxJudge func(box.Facts, time.Time) []box.Line

func randomFacts(r *rand.Rand, now time.Time) (box.Facts, bool) {
	t := func(d time.Duration) *time.Time { x := now.Add(d); return &x }
	f := box.Facts{At: now.Add(-time.Duration(r.Intn(180)) * time.Minute)}
	missing := false
	if r.Intn(4) > 0 {
		for _, s := range box.Services {
			f.Services = append(f.Services, box.Service{Name: s, State: "running"})
		}
	} else {
		missing = true
	}
	if r.Intn(4) > 0 {
		for _, n := range box.Timers {
			f.Timers = append(f.Timers, box.Timer{Name: n, Enabled: true, Result: "success"})
		}
	} else {
		missing = true
	}
	if r.Intn(4) > 0 {
		f.Disks = []box.Disk{{Path: "/opt/second-brain", UsedPct: float64(r.Intn(100))}}
	} else {
		missing = true
	}
	switch r.Intn(3) {
	case 0:
		f.CertExpires = t(time.Duration(r.Intn(120)-10) * 24 * time.Hour)
	case 1:
		f.CertManaged = true
	default:
		missing = true
	}
	if r.Intn(4) > 0 {
		f.Offsite = &box.Offsite{Path: "/mnt/nas", Mounted: true, Writable: r.Intn(5) > 0}
	} else {
		missing = true
	}
	if r.Intn(4) > 0 {
		f.Ready = []int{200, 204, 503}[r.Intn(3)]
	} else {
		missing = true
	}
	if r.Intn(4) > 0 {
		e := r.Intn(2) == 0
		f.Encrypted = &e
	} else {
		missing = true
	}
	if r.Intn(4) > 0 {
		f.LastDump, f.LastOffsite = t(-time.Duration(r.Intn(72))*time.Hour), t(-time.Duration(r.Intn(72))*time.Hour)
	} else {
		missing = true
	}
	if r.Intn(4) > 0 { // the restore drill (SB-1803)
		f.LastDrill, f.DrillOK = t(-time.Duration(r.Intn(70*24))*time.Hour), r.Intn(5) > 0
	} else {
		missing = true
	}
	if r.Intn(4) > 0 { // sealing and the recovery kit (SB-1901, SB-1902)
		sealed := r.Intn(5) > 0
		f.Sealed, f.Recovery = &sealed, []string{"kit", "kit", "pending"}[r.Intn(3)]
	} else {
		missing = true
	}
	return f, missing
}

func law03Box(r *rand.Rand, judge boxJudge) error {
	now := time.Unix(1_800_000_000, 0).UTC()
	f, missing := randomFacts(r, now)
	lines := judge(f, now)
	stale := now.Sub(f.At) > box.FactsMaxAge
	w := box.Worst(lines)
	if (missing || stale) && (w == box.Pass || w == box.Warn) {
		return fmt.Errorf("facts with a gap (missing=%v stale=%v) judged %s", missing, stale, w)
	}
	if stale {
		for _, l := range lines {
			if l.Verdict != box.Unknown {
				return fmt.Errorf("stale facts judged %s on %s", l.Verdict, l.Check)
			}
		}
	}
	for _, l := range lines {
		if (l.Verdict == box.Fail || l.Verdict == box.Unknown) && box.ExitCode(lines) == 0 {
			return fmt.Errorf("the doctor exits 0 with %s on %s", l.Verdict, l.Check)
		}
	}
	return nil
}
