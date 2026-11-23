// Package health is the cell's health from its own records (sb-admin(3), GET
// /v1/admin/health; SB-1607): each scheduled job's last finish and whether it is overdue,
// decided exactly as the watchdog decides it, and each connector's last successful sync,
// its failures since, and whether it is stale. It reads finish times and counts only.
package health

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/schedule"
)

// FreshnessSLA is how old a connector's last success may be before it is stale: the
// ingest freshness target (sb-ingest(7)).
const FreshnessSLA = ingest.FreshnessSLA

// Sync is one connector's sync record.
type Sync struct {
	LastSuccess time.Time `json:"last_success"`
	LastFailure time.Time `json:"last_failure"`
	Failures    int       `json:"failures"` // since the last success
}

// Syncs keeps each connector's sync record. found is false for a connector that has never
// synced; an unreadable record is an error, never "never synced".
type Syncs interface {
	Sync(ctx context.Context, kind string) (s Sync, found bool, err error)
}

// Job is one scheduled job's health.
type Job struct {
	Name       string     `json:"name"`
	Law        string     `json:"law,omitempty"`
	LastFinish *time.Time `json:"last_finish"`
	Overdue    bool       `json:"overdue"`
	Unreadable bool       `json:"unreadable,omitempty"`
}

// Connector is one connector's health.
type Connector struct {
	Kind        string     `json:"kind"`
	LastSuccess *time.Time `json:"last_success"`
	LastFailure *time.Time `json:"last_failure"`
	Failures    int        `json:"failures"`
	Stale       bool       `json:"stale"`
}

// Report is the cell's health, overdue and stale items first.
type Report struct {
	Jobs       []Job       `json:"jobs"`
	Connectors []Connector `json:"connectors"`
	At         time.Time   `json:"at"`
	// Box is the machine's health on a self-hosted cell (SB-1802); absent on a managed one.
	Box *Box `json:"box,omitempty"`
	// Notify is the notify job's last run (SB-2607); absent on a cell with no channel.
	Notify *Notify `json:"notify,omitempty"`
}

// NotifyStale is how long the notify job, every five minutes, may go without a run.
const NotifyStale = time.Hour

// Notify is the notify job's last run, counts only.
type Notify struct {
	LastRun *time.Time `json:"last_run"`
	Via     string     `json:"via"`
	Sent    int        `json:"sent"`
	Skipped int        `json:"skipped"`
	Failed  int        `json:"failed"`
	Warning bool       `json:"warning"`
	Reason  string     `json:"reason,omitempty"`
}

// NotifyFrom judges the last run's record for a cell sending through via ("" is none).
func NotifyFrom(rec notify.RunRecord, found bool, via string, now time.Time) *Notify {
	if via == "" {
		return nil
	}
	n := &Notify{Via: via}
	switch {
	case !found:
		n.Warning, n.Reason = true, "the notify job hasn't run"
		return n
	case rec.Error || rec.Failed > 0:
		n.Warning, n.Reason = true, "the last run failed"
	case now.Sub(rec.At) > NotifyStale:
		n.Warning, n.Reason = true, "no run in the last hour"
	}
	at := rec.At.UTC()
	n.LastRun, n.Sent, n.Skipped, n.Failed = &at, rec.Sent, rec.Skipped, rec.Failed
	return n
}

// Box is a self-hosted machine's checks, judged from the host step's facts.
type Box struct {
	FactsAt *time.Time  `json:"facts_at"`
	Verdict box.Verdict `json:"verdict"`
	Checks  []box.Line  `json:"checks"`
}

// BoxFrom judges the host step's facts: no facts, or facts that don't parse, are one
// Unknown line, never an empty and healthy-looking section.
func BoxFrom(raw []byte, found bool, now time.Time) *Box {
	if !found {
		return &Box{Verdict: box.Unknown, Checks: []box.Line{{Check: "host step", Verdict: box.Unknown,
			Detail: "no facts from the hourly host step yet", Fix: "systemctl status sb-selfhost-update.timer"}}}
	}
	f, err := box.Parse(raw)
	if err != nil {
		return &Box{Verdict: box.Unknown, Checks: []box.Line{{Check: "host step", Verdict: box.Unknown,
			Detail: "the host step's facts can't be read", Fix: "systemctl status sb-selfhost-update.service"}}}
	}
	lines := box.Check(f, now)
	at := f.At.UTC()
	return &Box{FactsAt: &at, Verdict: box.Worst(lines), Checks: lines}
}

// ErrUnavailable is a store that can't be read: the report would claim health it can't see.
var ErrUnavailable = errors.New("health: EUNAVAIL")

func ptr(t time.Time, ok bool) *time.Time {
	if !ok || t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

// Build reads every job's and every connector's record. An unreadable job record is that
// job overdue, as the watchdog treats it; an unreadable connector record fails the report.
func Build(ctx context.Context, runs schedule.Runs, syncs Syncs, kinds []string, now time.Time) (Report, error) {
	r := Report{Jobs: []Job{}, Connectors: []Connector{}, At: now.UTC()}
	for _, j := range schedule.Jobs {
		at, found, err := runs.LastFinish(ctx, j.Name)
		h := Job{Name: j.Name, Law: j.Law, LastFinish: ptr(at, found && err == nil), Unreadable: err != nil}
		h.Overdue = err != nil || !found || now.Sub(at) > j.Grace()
		r.Jobs = append(r.Jobs, h)
	}
	for _, k := range kinds {
		s, found, err := syncs.Sync(ctx, k)
		if err != nil {
			return Report{}, fmt.Errorf("%w: connector %s: %v", ErrUnavailable, k, err)
		}
		c := Connector{Kind: k, LastSuccess: ptr(s.LastSuccess, found), LastFailure: ptr(s.LastFailure, found), Failures: s.Failures}
		c.Stale = !found || s.LastSuccess.IsZero() || now.Sub(s.LastSuccess) > FreshnessSLA
		r.Connectors = append(r.Connectors, c)
	}
	sort.SliceStable(r.Jobs, func(a, b int) bool { return r.Jobs[a].Overdue && !r.Jobs[b].Overdue })
	sort.SliceStable(r.Connectors, func(a, b int) bool { return r.Connectors[a].Stale && !r.Connectors[b].Stale })
	return r, nil
}
