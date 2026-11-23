package box

import (
	"testing"
	"time"
)

var now = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

func tp(d time.Duration) *time.Time { t := now.Add(d); return &t }
func bp(b bool) *bool               { return &b }

func healthy() Facts {
	var svcs []Service
	for _, s := range Services {
		svcs = append(svcs, Service{Name: s, State: "running"})
	}
	return Facts{
		At:       now.Add(-10 * time.Minute),
		Services: svcs,
		Timers: []Timer{{Name: "sb-selfhost-update.timer", Enabled: true, Result: "success"},
			{Name: "sb-selfhost-dbdump.timer", Enabled: true, Result: "success"},
			{Name: "sb-selfhost-drill.timer", Enabled: true, Result: "success"}},
		Disks:       []Disk{{Path: "/opt/second-brain", UsedPct: 40}},
		CertExpires: tp(90 * 24 * time.Hour),
		Offsite:     &Offsite{Path: "/mnt/nas", Mounted: true, Writable: true},
		Ready:       200,
		Encrypted:   bp(true),
		LastDump:    tp(-3 * time.Hour),
		LastOffsite: tp(-3 * time.Hour),
		LastDrill:   tp(-10 * 24 * time.Hour),
		DrillOK:     true,
		Sealed:      bp(true),
		Recovery:    "kit",
	}
}

func verdict(t *testing.T, lines []Line, check string) Verdict {
	t.Helper()
	for _, l := range lines {
		if l.Check == check {
			return l.Verdict
		}
	}
	t.Fatalf("no line %q in %+v", check, lines)
	return ""
}

func TestHealthyBoxPasses(t *testing.T) {
	lines := Check(healthy(), now)
	if w := Worst(lines); w != Pass {
		t.Fatalf("worst = %s: %+v", w, lines)
	}
	if ExitCode(lines) != 0 {
		t.Fatal("exit code")
	}
	for _, l := range lines {
		if l.Fix != "" {
			t.Fatalf("a passing line has a fix: %+v", l)
		}
	}
}

// Each check made to fail, one at a time.
func TestEachCheckFails(t *testing.T) {
	cases := []struct {
		name  string
		check string
		want  Verdict
		mut   func(*Facts)
	}{
		{"service down", "service api", Fail, func(f *Facts) { f.Services[1].State = "exited" }},
		{"service missing", "service worm", Fail, func(f *Facts) { f.Services = f.Services[:4] }},
		{"no services", "service api", Unknown, func(f *Facts) { f.Services = nil }},
		{"timer off", "timer sb-selfhost-update.timer", Fail, func(f *Facts) { f.Timers[0].Enabled = false }},
		{"timer failed", "timer sb-selfhost-dbdump.timer", Fail, func(f *Facts) { f.Timers[1].Result = "exit-code" }},
		{"no timers", "timer sb-selfhost-dbdump.timer", Unknown, func(f *Facts) { f.Timers = nil }},
		{"disk warn", "disk /opt/second-brain", Warn, func(f *Facts) { f.Disks[0].UsedPct = 85 }},
		{"disk fail", "disk /opt/second-brain", Fail, func(f *Facts) { f.Disks[0].UsedPct = 95 }},
		{"no disks", "disk", Unknown, func(f *Facts) { f.Disks = nil }},
		{"cert warn", "certificate", Warn, func(f *Facts) { f.CertExpires = tp(20 * 24 * time.Hour) }},
		{"cert fail", "certificate", Fail, func(f *Facts) { f.CertExpires = tp(2 * 24 * time.Hour) }},
		{"cert expired", "certificate", Fail, func(f *Facts) { f.CertExpires = tp(-time.Hour) }},
		{"cert unknown", "certificate", Unknown, func(f *Facts) { f.CertExpires = nil }},
		{"offsite on root", "offsite", Fail, func(f *Facts) { f.Offsite.OnRoot = true }},
		{"offsite unmounted", "offsite", Fail, func(f *Facts) { f.Offsite.Mounted = false }},
		{"offsite read-only", "offsite", Fail, func(f *Facts) { f.Offsite.Writable = false }},
		{"offsite unknown", "offsite", Unknown, func(f *Facts) { f.Offsite = nil }},
		{"dump late", "database dump", Warn, func(f *Facts) { f.LastDump = tp(-30 * time.Hour) }},
		{"dump old", "database dump", Fail, func(f *Facts) { f.LastDump = tp(-60 * time.Hour) }},
		{"no dump", "database dump", Unknown, func(f *Facts) { f.LastDump = nil }},
		{"offsite copy old", "off-box copy", Fail, func(f *Facts) { f.LastOffsite = tp(-72 * time.Hour) }},
		{"no drill", "restore drill", Unknown, func(f *Facts) { f.LastDrill = nil }},
		{"drill failed", "restore drill", Fail, func(f *Facts) { f.DrillOK = false }},
		{"drill late", "restore drill", Warn, func(f *Facts) { f.LastDrill = tp(-40 * 24 * time.Hour) }},
		{"drill old", "restore drill", Fail, func(f *Facts) { f.LastDrill = tp(-70 * 24 * time.Hour) }},
		{"held", "upgrades", Warn, func(f *Facts) { f.Held = "rel_2026-11-20_abcdef" }},
		{"not sealed", "off-box sealing", Warn, func(f *Facts) { f.Sealed = bp(false) }},
		{"sealing unknown", "off-box sealing", Unknown, func(f *Facts) { f.Sealed = nil }},
		{"key on the box", "recovery kit", Warn, func(f *Facts) { f.Recovery = "pending" }},
		{"no key at all", "recovery kit", Fail, func(f *Facts) { f.Recovery = "none" }},
		{"kit unknown", "recovery kit", Unknown, func(f *Facts) { f.Recovery = "" }},
		{"shares left on the box", "recovery kit", Warn, func(f *Facts) { f.SharesOnBox = 2 }},
		{"not ready", "readyz", Fail, func(f *Facts) { f.Ready = 503 }},
		{"redirect", "readyz", Fail, func(f *Facts) { f.Ready = 302 }},
		{"ready unknown", "readyz", Unknown, func(f *Facts) { f.Ready = 0 }},
		{"plain disk", "encrypted disk", Warn, func(f *Facts) { f.Encrypted = bp(false) }},
		{"disk unknown", "encrypted disk", Unknown, func(f *Facts) { f.Encrypted = nil }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := healthy()
			c.mut(&f)
			lines := Check(f, now)
			if got := verdict(t, lines, c.check); got != c.want {
				t.Fatalf("%s = %s, want %s", c.check, got, c.want)
			}
			if lines[0].Verdict != c.want {
				t.Fatalf("the worst line isn't first: %+v", lines[0])
			}
			for _, l := range lines {
				if l.Verdict != Pass && l.Fix == "" {
					t.Fatalf("no fix for %+v", l)
				}
			}
			if want := map[Verdict]int{Warn: 0, Fail: 1, Unknown: 1}[c.want]; ExitCode(lines) != want {
				t.Fatalf("exit code %d, want %d", ExitCode(lines), want)
			}
		})
	}
}

func TestReadyAcceptsNoContent(t *testing.T) {
	f := healthy()
	f.Ready = 204
	if v := verdict(t, Check(f, now), "readyz"); v != Pass {
		t.Fatal(v)
	}
}

func TestACMECertificateIsManaged(t *testing.T) {
	f := healthy()
	f.CertExpires, f.CertManaged = nil, true
	if v := verdict(t, Check(f, now), "certificate"); v != Pass {
		t.Fatal(v)
	}
}

func TestExternalDatabaseNeedsNoPostgres(t *testing.T) {
	f := healthy()
	f.Services = append(f.Services[:5], Service{Name: "postgres-external", State: "configured"})
	for _, l := range Check(f, now) {
		if l.Check == "service postgres" {
			t.Fatalf("postgres checked on an external layout: %+v", l)
		}
	}
}

// An old green is not a green.
func TestStaleFactsAreUnknown(t *testing.T) {
	f := healthy()
	f.At = now.Add(-3 * time.Hour)
	lines := Check(f, now)
	for _, l := range lines {
		if l.Verdict != Unknown {
			t.Fatalf("stale facts shown as %s: %+v", l.Verdict, l)
		}
	}
	if ExitCode(lines) != 1 {
		t.Fatal("stale facts must fail the doctor")
	}
	f.At = now.Add(time.Hour) // from the future
	if Worst(Check(f, now)) != Unknown {
		t.Fatal("facts from the future")
	}
}

func TestWorstOfNothingIsUnknown(t *testing.T) {
	if Worst(nil) != Unknown || ExitCode(nil) != 1 {
		t.Fatal("no lines must not pass")
	}
}

func TestParseRefusesUnknownFields(t *testing.T) {
	if _, err := Parse([]byte(`{"at":"2026-11-23T12:00:00Z","dissk":[]}`)); err == nil {
		t.Fatal("a typo parsed")
	}
	if _, err := Parse([]byte(`{}`)); err == nil {
		t.Fatal("facts with no time parsed")
	}
	f, err := Parse([]byte(`{"at":"2026-11-23T12:00:00Z","ready":200}`))
	if err != nil || f.Ready != 200 {
		t.Fatal(f, err)
	}
}
