package box

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// SB-2605: every line that isn't a pass names a fix, and every runbook section a fix
// names is a heading of the runbook.
func TestEveryFixIsThereAndPointsSomewhereReal(t *testing.T) {
	md, err := os.ReadFile("../../../" + Runbook)
	if err != nil {
		t.Fatal(err)
	}
	sections := map[string]bool{}
	for _, line := range strings.Split(string(md), "\n") {
		if h, ok := strings.CutPrefix(line, "## "); ok {
			slug := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(h), "-"), "-")
			sections[slug] = true
		}
	}
	broken := []func(f *Facts){
		func(f *Facts) { f.Services = nil; f.Timers = nil; f.Disks = nil; f.Offsite = nil },
		func(f *Facts) {
			f.Services = f.Services[1:]
			f.Timers[0].Enabled = false
			f.Timers[1].Result = "failed"
			f.Disks[0].UsedPct = 95
			f.Offsite.OnRoot = true
		},
		func(f *Facts) {
			f.Disks[0].UsedPct = 85
			f.Offsite.Writable = false
			f.LastDump = nil
			f.LastOffsite = tp(-60 * time.Hour)
		},
		func(f *Facts) {
			f.Offsite = &Offsite{Path: "s3://b", S3: true, Answer: "refused"}
			f.LastDrill = nil
			f.Sealed = bp(false)
		},
		func(f *Facts) {
			f.Offsite = &Offsite{Path: "s3://b", S3: true}
			f.DrillOK = false
			f.Recovery = "none"
			f.Held = "rel_x"
		},
		func(f *Facts) {
			f.LastDrill = tp(-40 * 24 * time.Hour)
			f.Recovery = "pending"
			f.CertExpires = tp(24 * time.Hour)
			f.Ready = 503
		},
		func(f *Facts) {
			f.LastDrill = tp(-70 * 24 * time.Hour)
			f.SharesOnBox = 1
			f.CertExpires = nil
			f.Encrypted = nil
			f.Sealed = nil
		},
		func(f *Facts) {
			f.Recovery = ""
			f.CertExpires = tp(10 * 24 * time.Hour)
			f.Ready = 0
			f.Encrypted = bp(false)
		},
		func(f *Facts) { f.At = now.Add(-3 * time.Hour) },
	}
	ref := regexp.MustCompile(regexp.QuoteMeta(Runbook) + `#([a-z0-9-]+)`)
	named := 0
	for i, mut := range broken {
		f := healthy()
		mut(&f)
		for _, l := range Check(f, now) {
			if l.Verdict == Pass {
				continue
			}
			if l.Fix == "" {
				t.Errorf("case %d: %s (%s) has no fix", i, l.Check, l.Verdict)
			}
			for _, m := range ref.FindAllStringSubmatch(l.Fix, -1) {
				named++
				if !sections[m[1]] {
					t.Errorf("case %d: %s names %s#%s, which the runbook doesn't have", i, l.Check, Runbook, m[1])
				}
			}
		}
	}
	if named == 0 {
		t.Fatal("no fix named a runbook section")
	}
}
