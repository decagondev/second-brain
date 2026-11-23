// Package jobsvc is a cell's jobs service (sb-runtime(7), Schedules, exactly; SB-1504): it
// runs every scheduled job, records each finish, and runs the watchdog that makes a missed
// run loud. A job runs when the local clock reaches its minute, or when the platform asks
// for it by name (Cloud Scheduler, through `sb-jobs run JOB`).
package jobsvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	accessfs "example.com/second-brain/internal/adapters/access/localfs"
	"example.com/second-brain/internal/adapters/alert/webhook"
	backupfs "example.com/second-brain/internal/adapters/backup/localfs"
	"example.com/second-brain/internal/adapters/backup/s3"
	wrapfs "example.com/second-brain/internal/adapters/keywrap/localfs"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/offsite"
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
	// HTTP posts the weekly report; nil is a client with a 30-second timeout (tests set a
	// TLS test server's).
	HTTP *http.Client
	// Send is where notifications go; nil is the cell's chat bot (wire.Cell.Notifier).
	Send notify.Sender
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
		"notify":    s.notify,
		"evals":     s.evals,
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

// Loop ticks once a minute, at the start of each minute, until ctx ends. The audit stream,
// when one is set, runs beside it.
func (s *Service) Loop(ctx context.Context) {
	go s.Stream(ctx)
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

// signer is the cell's anchor key (wire.Cell.AnchorSigner).
func (s *Service) signer(ctx context.Context) (audit.Signer, error) { return s.Cell.AnchorSigner(ctx) }

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
	worm, err := s.Cell.WormTarget(ctx)
	if err != nil {
		return fmt.Errorf("jobs: the write-once store: %w", err)
	}
	if err := worm.Put(ctx, name, b, now.Add(AnchorRetention)); err != nil && !errors.Is(err, backupfs.ErrExists) && !errors.Is(err, s3.ErrExists) {
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
		if b.Worm {
			t, err := c.WormTarget(ctx)
			if err != nil {
				return fmt.Errorf("jobs: the write-once store: %w", err)
			}
			ts = append(ts, t)
			continue
		}
		var t backup.Target = backupfs.Dir{Root: b.Dir, Label: b.Dir, Media: b.Media, Away: b.Offsite}
		if b.S3 != "" {
			bk, err := c.OffsiteBucket(ctx, b)
			if err != nil {
				return fmt.Errorf("jobs: the off-box bucket: %w", err)
			}
			t = bk
		}
		if b.Seal != "" {
			pub, err := offsite.ParsePublic(b.Seal)
			if err != nil {
				return fmt.Errorf("jobs: backups: %w", err)
			}
			t = offsite.Target{Target: t, To: pub}
		}
		ts = append(ts, t)
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
		if err := os.WriteFile(filepath.Join(dir, week+".md"), []byte(md), 0o600); err != nil {
			return err
		}
		return s.deliver(ctx, week, md)
	})
	return err
}

// deliver posts the weekly report to the report webhook, or the alert webhook when that is
// the only one set (SB-1608). A delivery that fails fails the run, so the watchdog sees it.
func (s *Service) deliver(ctx context.Context, week, md string) error {
	a := s.Cell.Config.Alerts
	to := a.ReportWebhook
	if to == "" {
		to = a.Webhook
	}
	if to == "" {
		return nil // the file is the delivery
	}
	hc := s.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	err := webhook.Sender{URL: to, HTTP: hc}.Post(ctx, webhook.Report{Kind: "weekly_report", Tenant: s.Cell.Config.Tenant, Week: week, Markdown: md})
	if err != nil {
		return fmt.Errorf("jobs: the weekly report wasn't delivered: %w", err)
	}
	return nil
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
	if err := s.statement(ctx, res); err != nil {
		return err
	}
	if len(res.Unlisted) > 0 {
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev1", Kind: "access_path", Tenant: c.Config.Tenant, Where: "job:access", Ref: res.Unlisted[0].ID})
		return fmt.Errorf("%w: %d unlisted", ErrAccess, len(res.Unlisted))
	}
	return nil
}

// statement records what the access check found (sb-selfhost(7), LAW-17 on one machine):
// the statement itself under STATE/access/, and its hash in an audit row, so it is signed
// by the next anchor and can't be changed afterwards without the chain showing it.
func (s *Service) statement(ctx context.Context, res access.Result) error {
	c := s.Cell
	now := s.now().UTC()
	st := struct {
		Tenant    string        `json:"tenant"`
		CheckedAt string        `json:"checked_at"`
		Retain    []string      `json:"retain"`
		Paths     []access.Path `json:"paths"`
		Unlisted  []access.Path `json:"unlisted"`
	}{c.Config.Tenant, now.Format(time.RFC3339), append([]string{}, c.Config.Custody.Retain...), append([]access.Path{}, res.Paths...),
		append([]access.Path{}, res.Unlisted...)}
	b, err := json.MarshalIndent(st, "", " ")
	if err != nil {
		return err
	}
	dir := filepath.Join(c.Config.State, "access")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "statement-"+now.Format("2006-01-02")+".json"), b, 0o600); err != nil {
		return fmt.Errorf("jobs: the access statement can't be written: %w", err)
	}
	sum := sha256.Sum256(b)
	state := "Checked"
	if len(res.Unlisted) > 0 {
		state = "Unlisted"
	}
	_, err = c.Audit.Write(ctx, audit.Event{Actor: "sys_jobs", Kind: "access", Subject: "access_statement:" + now.Format("2006-01-02"), State: state,
		PayloadSHA256: hex.EncodeToString(sum[:]), Reason: fmt.Sprintf("paths=%d,unlisted=%d", len(res.Paths), len(res.Unlisted))})
	return err
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

// notify tells stewards about new questions and askers about replies (sb-ask(3),
// Notifications; SB-2101, SB-2102).
func (s *Service) notify(ctx context.Context) error {
	send := s.Send
	if send == nil {
		send = s.Cell.Notifier()
	}
	_, err := s.Cell.NotifyJob(send).Run(ctx)
	return err
}

// evals runs the tenant's cases from wrong citations on the release the cell runs, and
// holds that release if it regresses one (sb-evals(7); SB-2204).
func (s *Service) evals(ctx context.Context) error {
	_, err := s.Cell.RunEvals(ctx)
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
	s.boxAlerts(ctx)
	return nil
}

// boxAlerts raises one Sev-2 a day for each failing check of a self-hosted box (SB-1802),
// from the same report the console shows. Warnings and unknowns show in the console only.
func (s *Service) boxAlerts(ctx context.Context) {
	c := s.Cell
	r, err := c.Health(ctx, s.now())
	if err != nil || r.Box == nil {
		return
	}
	day := s.now().Format("2006-01-02")
	for _, l := range r.Box.Checks {
		if l.Verdict != box.Fail {
			continue
		}
		sum := sha256.Sum256([]byte(l.Check))
		if first, err := c.Usage.First(ctx, "box-"+day+"-"+hex.EncodeToString(sum[:6])); err != nil || !first {
			continue
		}
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev2", Kind: "box_check", Tenant: c.Config.Tenant, Where: l.Check, Ref: day})
	}
}
