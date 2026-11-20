// Package schedule is a cell's timed jobs and the watchdog that makes a missed run loud
// (sb-runtime(7), Schedules, exactly; SB-904). Times are UTC.
package schedule

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Job is one timed job.
type Job struct {
	Name     string
	Cron     string        // for Cloud Scheduler
	Interval time.Duration // the longest normal gap between finishes
	Law      string        // the law it carries, or ""
}

// Jobs are every cell's, as sb-runtime(7) lists them.
var Jobs = []Job{
	{Name: "sync", Cron: "* * * * *", Interval: time.Minute, Law: "LAW-13"},
	{Name: "anchor", Cron: "5 * * * *", Interval: time.Hour, Law: "LAW-11"},
	{Name: "retention", Cron: "0 3 * * *", Interval: 24 * time.Hour},
	{Name: "backup", Cron: "0 2 * * *", Interval: 24 * time.Hour, Law: "LAW-11"},
	{Name: "report", Cron: "0 7 * * 1", Interval: 7 * 24 * time.Hour},
	{Name: "access", Cron: "0 4 1 * *", Interval: 31 * 24 * time.Hour, Law: "LAW-17"},
}

// Grace is how late a job may be before it is overdue: twice its interval, but at most a
// day for the weekly and monthly jobs.
func (j Job) Grace() time.Duration {
	g := 2 * j.Interval
	if j.Interval >= 7*24*time.Hour {
		g = j.Interval + 24*time.Hour
	}
	return g
}

// Severity is the missed-run alert's: Sev-1 for a job that carries a law.
func (j Job) Severity() string {
	if j.Law != "" {
		return "sev1"
	}
	return "sev2"
}

// Runs records each job's last finish.
type Runs interface {
	LastFinish(ctx context.Context, job string) (at time.Time, found bool, err error)
}

// Alert is one missed run.
type Alert struct {
	Kind     string `json:"kind"` // job_missed
	Job      string `json:"job"`
	Severity string `json:"severity"`
	Law      string `json:"law,omitempty"`
	Since    string `json:"since,omitempty"` // last finish, or "" if none or unreadable
}

// Gauge sets a metric (obs.Metrics).
type Gauge interface {
	Set(name, help string, labels map[string]string, v float64)
}

// Watch checks every job. A job with no recorded finish, or whose record can't be read, is
// overdue: unknown is never fine.
func Watch(ctx context.Context, runs Runs, now time.Time, g Gauge) []Alert {
	var out []Alert
	for _, j := range Jobs {
		at, found, err := runs.LastFinish(ctx, j.Name)
		overdue := err != nil || !found || now.Sub(at) > j.Grace()
		v := 0.0
		if overdue {
			v = 1
			a := Alert{Kind: "job_missed", Job: j.Name, Severity: j.Severity(), Law: j.Law}
			if err == nil && found {
				a.Since = at.UTC().Format(time.RFC3339)
			}
			out = append(out, a)
		}
		if g != nil {
			g.Set("sb_job_overdue", "1 when a scheduled job has missed its run (sb-runtime(7)).", map[string]string{"job": j.Name}, v)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Job < out[b].Job })
	return out
}

// Find returns a job by name.
func Find(name string) (Job, error) {
	for _, j := range Jobs {
		if j.Name == name {
			return j, nil
		}
	}
	return Job{}, fmt.Errorf("schedule: no job %q", name)
}
