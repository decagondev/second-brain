// Package health is the cell's health from its own records (sb-admin(3), GET
// /v1/admin/health; SB-1607): each scheduled job's last finish and whether it is overdue,
// decided exactly as the watchdog decides it, and each connector's last successful sync,
// its failures since, and whether it is stale. It reads finish times and counts only.
package health

import (
	"example.com/second-brain/internal/core/gateway"

	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
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
	// Failure is the last failure's class while it stands (SB-3201): auth, rate_limited,
	// unreachable or source_error, never a URL, body or message.
	Failure string `json:"failure,omitempty"`
	// First is a first sync under way (SB-6004): how far it has come and when it started;
	// nil once a sync succeeds.
	First *FirstSync `json:"first_sync,omitempty"`
}

// FirstSync is a first sync's progress: Done of Total changes since Started.
type FirstSync struct {
	Done    int       `json:"done"`
	Total   int       `json:"total"`
	Started time.Time `json:"started"`
}

// FirstProgress is a first sync on the health report: Items of Of, and about how many
// minutes are left at the pace so far (nil until there is a pace).
type FirstProgress struct {
	Items       int  `json:"items"`
	Of          int  `json:"of"`
	LeftMinutes *int `json:"left_minutes"`
}

// Progress is a first sync's place on the report at now.
func (f FirstSync) Progress(now time.Time) *FirstProgress {
	p := &FirstProgress{Items: f.Done, Of: f.Total}
	if el := now.Sub(f.Started); f.Done > 0 && el > 0 && f.Total >= f.Done {
		left := int(time.Duration(float64(el) * float64(f.Total-f.Done) / float64(f.Done)).Minutes())
		p.LeftMinutes = &left
	}
	return p
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
	// Failure is the class of the last failed sync, and Fix the runbook section for it
	// (SB-3201); both are absent once a sync succeeds.
	Failure string `json:"failure,omitempty"`
	Fix     string `json:"fix,omitempty"`
	// FirstSync is a first sync under way (SB-6004).
	FirstSync *FirstProgress `json:"first_sync,omitempty"`
}

// ConnectorRunbook is the runbook whose sections fix each class of sync failure.
const ConnectorRunbook = "docs/runbooks/connectors.md"

// Report is the cell's health, overdue and stale items first.
type Report struct {
	Jobs       []Job       `json:"jobs"`
	Connectors []Connector `json:"connectors"`
	At         time.Time   `json:"at"`
	// Box is the machine's health on a self-hosted cell (SB-1802); absent on a managed one.
	Box *Box `json:"box,omitempty"`
	// Notify is the notify job's last run (SB-2607); absent on a cell with no channel.
	Notify *Notify `json:"notify,omitempty"`
	// Index is the nightly count of the index's leftovers (SB-2905), and Prompt the input
	// tokens per model call (SB-2906). Counts only.
	Index  *Index  `json:"index,omitempty"`
	Prompt *Prompt `json:"prompt,omitempty"`
	// Unstewarded are the spaces with no active steward now, and those the directory
	// couldn't answer for (SB-3102); absent in a cell without spaces.
	Unstewarded *Unstewarded `json:"spaces_unstewarded,omitempty"`
	// AnswerTime is seconds per answer, this week and the week before (SB-3205).
	AnswerTime *AnswerTime `json:"answer_time,omitempty"`
	// Model is the main model's fallback: gated, switched until a time, and whether no
	// model could answer the last call (SB-5601, SB-5602).
	Model *gateway.State `json:"model,omitempty"`
	// UnmappedLabels are the Purview label ids the docs connector met that labels.microsoft
	// doesn't map, by how many items carried each (SB-5902): never which documents.
	UnmappedLabels map[string]int `json:"unmapped_labels,omitempty"`
}

// Unstewarded is health's spaces with no active steward; either list warns.
type Unstewarded struct {
	Spaces  []string `json:"spaces"`
	Unknown []string `json:"unknown"`
	Warning bool     `json:"warning"`
}

// IndexStale is how old the nightly leftover count may be before it warns.
const IndexStale = 48 * time.Hour

// Index is the last count of chunks of documents their source no longer lists (Gone), and
// chunks no space holds (Unheld).
type Index struct {
	At      *time.Time `json:"at"`
	Gone    int        `json:"gone"`
	Unheld  int        `json:"unheld"`
	Warning bool       `json:"warning"`
}

// IndexFrom judges the last count: above zero, missing or older than IndexStale warns.
func IndexFrom(at time.Time, gone, unheld int, found bool, now time.Time) *Index {
	if !found {
		return &Index{Warning: true}
	}
	t := at.UTC()
	return &Index{At: &t, Gone: gone, Unheld: unheld, Warning: gone+unheld > 0 || now.Sub(at) > IndexStale}
}

// Prompt is the 95th percentile of input tokens per model call this week and the week
// before; Warning is this week's over 1.5 times the week before's.
type Prompt struct {
	P95       int  `json:"p95"`
	P95Before int  `json:"p95_before"`
	Warning   bool `json:"warning"`
}

// P95 is the nearest-rank 95th percentile of xs; 0 for none.
func P95(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	s := append([]int(nil), xs...)
	sort.Ints(s)
	rank := (95*len(s) + 99) / 100 // ceil(0.95 n)
	return s[rank-1]
}

// Grew reports this week's p95 over 1.5 times the week before's.
func Grew(now, before int) bool { return before > 0 && 2*now > 3*before }

// AnswerTime is the 95th percentile of seconds per answer this week and the week before,
// and whether it is over 1.5 times as long (SB-3205).
type AnswerTime struct {
	P95Seconds       float64 `json:"p95_seconds"`
	P95SecondsBefore float64 `json:"p95_seconds_before"`
	Warning          bool    `json:"warning"`
}

// AnswerFrom is the answer-time watch from the week's and the week before's durations in
// milliseconds; nil when there are none.
func AnswerFrom(week, before []int) *AnswerTime {
	if len(week)+len(before) == 0 {
		return nil
	}
	now, was := P95(week), P95(before)
	return &AnswerTime{P95Seconds: float64(now) / 1000, P95SecondsBefore: float64(was) / 1000, Warning: Grew(now, was)}
}

// PromptFrom is the prompt watch from the week's and the week before's input tokens.
func PromptFrom(week, before []int) *Prompt {
	p := &Prompt{P95: P95(week), P95Before: P95(before)}
	p.Warning = Grew(p.P95, p.P95Before)
	return p
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
		if s.First != nil {
			c.FirstSync = s.First.Progress(now)
		}
		if s.Failures > 0 {
			c.Failure = ingest.KnownClass(s.Failure)
			c.Fix = ConnectorRunbook + "#" + strings.ReplaceAll(c.Failure, "_", "-")
		}
		r.Connectors = append(r.Connectors, c)
	}
	sort.SliceStable(r.Jobs, func(a, b int) bool { return r.Jobs[a].Overdue && !r.Jobs[b].Overdue })
	sort.SliceStable(r.Connectors, func(a, b int) bool { return r.Connectors[a].Stale && !r.Connectors[b].Stale })
	return r, nil
}
