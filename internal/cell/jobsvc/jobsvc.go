// Package jobsvc is a cell's jobs service (sb-runtime(7), Schedules, exactly; SB-1504): it
// runs every scheduled job, records each finish, and runs the watchdog that makes a missed
// run loud. A job runs when the local clock reaches its minute, or when the platform asks
// for it by name (Cloud Scheduler, through `sb-jobs run JOB`).
package jobsvc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	accessfs "example.com/second-brain/internal/adapters/access/localfs"
	backupfs "example.com/second-brain/internal/adapters/backup/localfs"
	wrapfs "example.com/second-brain/internal/adapters/keywrap/localfs"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/report"
	"example.com/second-brain/internal/core/reviews"
	"example.com/second-brain/internal/core/schedule"
)

// AnchorRetention is how long an anchor is kept, write-once (ADR-0012).
const AnchorRetention = 7 * 365 * 24 * time.Hour

// Errors.
var (
	ErrUnknownJob = errors.New("jobs: no such job")
	ErrNoBackups  = errors.New("jobs: no backup copies are configured (sb-cell(5), backups)")
	ErrAccess     = errors.New("jobs: LAW-17 a factory access path isn't in the hand-off record")
)

// Service runs one cell's jobs.
type Service struct {
	Cell   *wire.Cell
	Ingest *ingestsvc.Service
	// Now is the clock; nil is the real one.
	Now func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Jobs are every job by name, as schedule.Jobs lists them.
func (s *Service) Jobs() map[string]func(ctx context.Context) error {
	return map[string]func(ctx context.Context) error{
		"sync":      s.sync,
		"anchor":    s.anchor,
		"retention": s.retention,
		"backup":    s.backup,
		"report":    s.report,
		"access":    s.access,
		"review":    s.review,
		"drift":     s.drift,
	}
}

// Run runs one job by name and records its finish only if it succeeded: a failed run
// leaves the last finish where it was, so the watchdog sees it.
func (s *Service) Run(ctx context.Context, name string) error {
	f, ok := s.Jobs()[name]
	if !ok {
		return fmt.Errorf("%w %q", ErrUnknownJob, name)
	}
	c := s.Cell
	start := s.now()
	err := f(ctx)
	labels := map[string]string{"job": name, "result": "ok"}
	if err != nil {
		labels["result"] = "failed"
		c.Log.Error("job failed", "job", name, "error", err.Error())
	} else if rerr := c.Runs.Finished(ctx, name, s.now()); rerr != nil {
		err = fmt.Errorf("jobs: %s ran, but its finish can't be recorded: %w", name, rerr)
		labels["result"] = "unrecorded"
	}
	c.Metrics.Add("sb_job_runs_total", "Scheduled job runs, by job and result (sb-runtime(7)).", labels, 1)
	c.Metrics.Set("sb_job_last_seconds", "How long a job's last run took.", map[string]string{"job": name}, s.now().Sub(start).Seconds())
	return err
}

// Watchdog compares every job's last finish with its schedule and alerts on each overdue
// one (sb-runtime(7)): Sev-1 for a job that carries a law.
func (s *Service) Watchdog(ctx context.Context) []schedule.Alert {
	c := s.Cell
	alerts := schedule.Watch(ctx, c.Runs, s.now(), c.Metrics)
	for _, a := range alerts {
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: a.Severity, Kind: a.Kind, Tenant: c.Config.Tenant, Where: "job:" + a.Job, Ref: a.Law})
	}
	return alerts
}

// Tick runs every job due in the minute that holds t, then the watchdog if it is due. A
// job that fails doesn't stop the others; the failures are returned by job.
func (s *Service) Tick(ctx context.Context, t time.Time) map[string]error {
	out := map[string]error{}
	for _, j := range schedule.Jobs {
		due, err := schedule.Due(j.Cron, t)
		if err != nil {
			out[j.Name] = err
			continue
		}
		if due {
			if err := s.Run(ctx, j.Name); err != nil {
				out[j.Name] = err
			}
		}
	}
	if t.UTC().Minute()%int(schedule.WatchdogEvery/time.Minute) == 0 {
		s.Watchdog(ctx)
	}
	return out
}

// Loop ticks once a minute, at the start of each minute, until ctx ends.
func (s *Service) Loop(ctx context.Context) {
	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		t := time.NewTimer(next.Sub(now))
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case at := <-t.C:
			s.Tick(ctx, at)
		}
	}
}

func (s *Service) sync(ctx context.Context) error {
	fails := s.Ingest.SyncAll(ctx)
	if len(fails) == 0 {
		return nil
	}
	kinds := make([]string, 0, len(fails))
	for k := range fails {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	var msgs []string
	for _, k := range kinds {
		msgs = append(msgs, k+": "+fails[k].Error())
	}
	return fmt.Errorf("jobs: sync failed for %s", strings.Join(msgs, "; "))
}

// signer is the cell's anchor key (sb-audit(5), Anchors), read at each run. Locally its
// seed is vault://tenants/TENANT/audit/anchor-seed (32 bytes, base64); on GCP it is a data
// key wrapped by the cell's own KMS key.
func (s *Service) signer(ctx context.Context) (audit.Signer, error) {
	raw, err := s.Cell.Secret("audit/anchor-seed")(ctx)
	if err != nil {
		return audit.Signer{}, fmt.Errorf("jobs: the anchor key can't be read: %w", err)
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return audit.Signer{}, fmt.Errorf("jobs: the anchor seed isn't base64")
	}
	return audit.NewSigner(s.Cell.Config.Tenant, seed)
}

// lastRow is the audit log's last row, or the zero row for an empty log.
func lastRow(ctx context.Context, st audit.Store) (audit.Row, error) {
	var last audit.Row
	for from := int64(1); ; {
		rows, err := st.Range(ctx, from, 1000)
		if err != nil {
			return audit.Row{}, err
		}
		if len(rows) == 0 {
			return last, nil
		}
		last = rows[len(rows)-1]
		from = last.Seq + 1
	}
}

func (s *Service) worm() backupfs.Dir {
	return backupfs.Dir{Root: filepath.Join(s.Cell.Config.State, "worm"), Label: "worm", Media: "disk"}
}

// anchor signs the audit log's last row and writes the anchor, write-once, for seven years.
// An anchor already written for that row is the same work done: it is not an error.
func (s *Service) anchor(ctx context.Context) error {
	sg, err := s.signer(ctx)
	if err != nil {
		return err
	}
	last, err := lastRow(ctx, s.Cell.Audit.Store)
	if err != nil {
		return fmt.Errorf("jobs: the audit log can't be read: %w", err)
	}
	now := s.now()
	a := sg.Sign(last.Seq, last.Hash, now)
	b, _ := json.Marshal(a)
	name := fmt.Sprintf("anchors/%012d.json", last.Seq)
	if err := s.worm().Put(ctx, name, b, now.Add(AnchorRetention)); err != nil && !errors.Is(err, backupfs.ErrExists) {
		return fmt.Errorf("jobs: the anchor can't be written: %w", err)
	}
	return nil
}

func (s *Service) retention(ctx context.Context) error {
	r, err := s.Cell.Parts().Retention.Sweep(ctx)
	if err == nil && len(r.Failed) > 0 {
		err = fmt.Errorf("jobs: retention failed for %s", strings.Join(r.Failed, ", "))
	}
	return err
}

func (s *Service) backup(ctx context.Context) error {
	c := s.Cell
	if len(c.Config.Backups) == 0 {
		return ErrNoBackups
	}
	var ts []backup.Target
	for _, b := range c.Config.Backups {
		ts = append(ts, backupfs.Dir{Root: b.Dir, Label: b.Dir, Media: b.Media, Away: b.Offsite})
	}
	if err := backup.Check(ts); err != nil {
		return err
	}
	sg, err := s.signer(ctx)
	if err != nil {
		return err
	}
	_, err = backup.Job{Tenant: c.Config.Tenant, Audit: c.Audit.Store, Signer: sg, Approvals: c.Actions, Targets: ts, Now: s.now}.Run(ctx)
	return err
}

// report writes last week's leadership report to STATE/reports/WEEK.md.
func (s *Service) report(ctx context.Context) error {
	c := s.Cell
	price := report.Price{InputPerM: c.Config.Model.Price.InPerM, OutputPerM: c.Config.Model.Price.OutPerM}
	week, _, _ := report.WeekOf(s.now().AddDate(0, 0, -7))
	_, err := report.Weekly(ctx, c.Report, c.Config.Tenant, s.now(), price, func(_ context.Context, md string) error {
		dir := filepath.Join(c.Config.State, "reports")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, week+".md"), []byte(md), 0o600)
	})
	return err
}

// access is the LAW-17 re-check for a tenant-custody cell; a factory-custody cell has no
// hand-off record to check against, and the run says so by finishing.
func (s *Service) access(ctx context.Context) error {
	c := s.Cell
	if !c.Config.Custody.Tenant {
		return nil
	}
	sources := []access.Source{access.Wraps{Store: wrapfs.Dir{Root: filepath.Join(c.Config.State, "keywraps")}},
		accessfs.Dir{Root: filepath.Join(c.Config.State, "iam")}}
	res, err := access.Watch(ctx, c.Config.Tenant, sources, c.Config.Custody.Retain, c.Metrics)
	if err != nil {
		return err
	}
	if len(res.Unlisted) > 0 {
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev1", Kind: "access_path", Tenant: c.Config.Tenant, Where: "job:access", Ref: res.Unlisted[0].ID})
		return fmt.Errorf("%w: %d unlisted", ErrAccess, len(res.Unlisted))
	}
	return nil
}

type roster struct{ c *wire.Cell }

func (r roster) Members(ctx context.Context, group string) ([]string, error) {
	return r.c.Directory.Members(ctx, r.c.Config.Tenant, strings.TrimPrefix(group, "group:"))
}

// review opens the quarter's access reviews that don't exist yet (sb-spaces(7)).
func (s *Service) review(ctx context.Context) error {
	c := s.Cell
	rows, err := c.Audit.Store.Range(ctx, 1, 1<<30)
	if err != nil {
		return fmt.Errorf("jobs: the audit log can't be read: %w", err)
	}
	_, err = reviews.Run(ctx, c.Manifest(), s.now(), roster{c}, reviews.FromAudit(rows), c.Reviews, c.NewID)
	return err
}

// drift compares what the cell runs with the manifest last applied and sends one
// manifest_drift alert per finding (sb-tenant(5), Drift). Drift is reported, never fixed.
func (s *Service) drift(ctx context.Context) error {
	c := s.Cell
	fs, err := c.Drift(ctx)
	if err != nil {
		return err
	}
	c.Metrics.Set("sb_manifest_drift", "Differences between the running cell and its applied manifest (LAW-16).",
		map[string]string{"tenant": c.Config.Tenant}, float64(len(fs)))
	for _, f := range fs {
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev2", Kind: "manifest_drift", Tenant: c.Config.Tenant, Where: f.Key(), Ref: f.Message()})
	}
	return nil
}
