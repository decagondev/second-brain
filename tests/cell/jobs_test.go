package cell

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/jobsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/holds"
	"example.com/second-brain/internal/core/offsite"
	"example.com/second-brain/internal/core/schedule"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func jobsCell(t *testing.T, seed bool) (running, *jobsvc.Service, *clock, *alerts) {
	t.Helper()
	r, _ := docsCell(t)
	if seed {
		p := filepath.Join(r.cell.Config.Vault, "vault", "tenants", tenant, "audit", "anchor-seed")
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := &alerts{}
	r.cell.Alerter = got
	c := &clock{time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC)} // a Monday
	ing := &ingestsvc.Service{Cell: r.cell, Alert: got, Now: c.now}
	return r, &jobsvc.Service{Cell: r.cell, Ingest: ing, Now: c.now}, c, got
}

func finished(t *testing.T, c *wire.Cell, job string) (time.Time, bool) {
	t.Helper()
	at, found, err := c.Runs.LastFinish(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	return at, found
}

// TestJobsRunOnTheirSchedule: a fake clock walks a day; each job runs at its minute and
// records its finish; one that fails records nothing, so the watchdog alerts on it.
func TestJobsRunOnTheirSchedule(t *testing.T) {
	r, svc, c, got := jobsCell(t, true)
	ctx := context.Background()
	walk := func(from time.Time, minutes int) {
		c.t = from
		for i := 0; i < minutes; i++ {
			svc.Tick(ctx, c.t)
			c.t = c.t.Add(time.Minute)
		}
	}
	walk(c.t, 24*60)                                        // a Monday: the weekly report
	walk(time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC), 6*60) // the 1st of a quarter: access and review
	for _, j := range []string{"sync", "anchor", "retention", "report", "access", "review"} {
		if _, ok := finished(t, r.cell, j); !ok {
			t.Errorf("%s never finished in a day that held its schedule", j)
		}
	}
	if _, ok := finished(t, r.cell, "backup"); ok {
		t.Fatal("backup finished with no copies configured")
	}
	anchors, _ := filepath.Glob(filepath.Join(r.cell.Config.State, "worm", "anchors", "*.json"))
	if len(anchors) == 0 {
		t.Fatal("no anchor written")
	}
	if _, err := os.Stat(filepath.Join(r.cell.Config.State, "reports", "2027-W09.md")); err != nil {
		t.Fatalf("weekly report: %v", err)
	}
	missed := map[string]string{}
	for _, a := range got.got {
		if a.Kind == "job_missed" {
			missed[strings.TrimPrefix(a.Where, "job:")] = a.Severity
		}
	}
	if missed["backup"] != "sev1" {
		t.Fatalf("the failed backup wasn't alerted as Sev-1: %v", missed)
	}
	if _, ok := missed["sync"]; ok {
		t.Fatalf("a job that ran was alerted: %v", missed)
	}
}

func TestWatchdogOnANewCell(t *testing.T) {
	_, svc, _, got := jobsCell(t, false)
	alerts := svc.Watchdog(context.Background())
	if len(alerts) != len(schedule.Jobs) || len(got.got) != len(schedule.Jobs) {
		t.Fatalf("a cell with no finishes must have every job overdue: %d alerts, %d sent", len(alerts), len(got.got))
	}
}

func TestAnchorWithoutAKeyFails(t *testing.T) {
	r, svc, _, _ := jobsCell(t, false)
	if err := svc.Run(context.Background(), "anchor"); err == nil {
		t.Fatal("anchor ran without its key")
	}
	if _, ok := finished(t, r.cell, "anchor"); ok {
		t.Fatal("a failed anchor recorded a finish")
	}
	if err := svc.Run(context.Background(), "nope"); !errors.Is(err, jobsvc.ErrUnknownJob) {
		t.Fatalf("unknown job: %v", err)
	}
}

// TestRetentionKeepsHeldRecords: the retention job reads legal holds when it runs
// (PT-0011): a held trace stays, and goes once the hold is released.
func TestRetentionKeepsHeldRecords(t *testing.T) {
	r, svc, c, _ := jobsCell(t, true)
	ctx := context.Background()
	old := ask.Trace{ID: "tr_old", Tenant: tenant, User: "usr_ann", Space: "company", At: c.t.AddDate(-2, 0, 0)}
	if err := r.cell.Traces.PutTrace(ctx, old); err != nil {
		t.Fatal(err)
	}
	hs := holds.Service{Store: r.cell.Holds, Audit: r.cell.Audit, NewID: r.cell.NewID, Now: c.now}
	h, err := hs.Place(ctx, "usr_counsel", "matter-1", "", []string{"usr_ann"}, "litigation")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(ctx, "retention"); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := r.cell.Traces.GetTrace(ctx, tenant, "tr_old"); !found {
		t.Fatal("LAW-13: retention deleted a held trace")
	}
	if _, err := hs.Release(ctx, "usr_counsel", h.ID, "settled"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(ctx, "retention"); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := r.cell.Traces.GetTrace(ctx, tenant, "tr_old"); found {
		t.Fatal("a trace past retention with no hold was kept")
	}
}

func TestBackupWritesThreeCopies(t *testing.T) {
	r, svc, _, _ := jobsCell(t, true)
	base := t.TempDir()
	r.cell.Config.Backups = []wire.Backup{{Dir: filepath.Join(base, "a"), Media: "disk"}, {Dir: filepath.Join(base, "b"), Media: "nas"},
		{Dir: filepath.Join(base, "c"), Media: "object-store", Offsite: true}}
	if err := svc.Run(context.Background(), "backup"); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"a", "b", "c"} {
		if fs, _ := filepath.Glob(filepath.Join(base, d, "*", "*")); len(fs) == 0 {
			if fs2, _ := filepath.Glob(filepath.Join(base, d, "*")); len(fs2) == 0 {
				t.Errorf("copy %s is empty", d)
			}
		}
	}
	r.cell.Config.Backups = r.cell.Config.Backups[:2]
	if err := svc.Run(context.Background(), "backup"); err == nil {
		t.Fatal("a backup with two copies passed the 3-2-1 rule")
	}
}

// SB-1605: on a tenant-custody cell the monthly access check leaves a statement under
// state/access, and an audit row with its hash, which the next anchor signs (LAW-17 on one
// machine). An unlisted path still fails the job and alerts.
func TestAccessStatement(t *testing.T) {
	r, svc, c, got := jobsCell(t, true)
	ctx := context.Background()
	r.cell.Config.Custody.Tenant = true
	r.cell.Config.Custody.Retain = []string{"release-pull"}
	iam := filepath.Join(r.cell.Config.State, "iam")
	if err := os.MkdirAll(iam, 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(paths string) {
		if err := os.WriteFile(filepath.Join(iam, tenant+".json"), []byte(paths), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(`[{"kind":"pull-token","id":"release-pull","grant":"release-pull"}]`)
	if err := svc.Run(ctx, "access"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(r.cell.Config.State, "access", "statement-"+c.t.Format("2006-01-02")+".json"))
	if err != nil || !strings.Contains(string(b), `"release-pull"`) || !strings.Contains(string(b), `"unlisted": []`) {
		t.Fatalf("statement %s %v", b, err)
	}
	rows, _ := r.cell.Audit.Store.Range(ctx, 1, 1<<20)
	last := rows[len(rows)-1]
	if last.Kind != "access" || last.State != "Checked" || len(last.PayloadSHA256) != 64 {
		t.Fatalf("audit row %+v", last)
	}
	write(`[{"kind":"pull-token","id":"release-pull","grant":"release-pull"},{"kind":"ssh-key","id":"vendor-laptop"}]`)
	if err := svc.Run(ctx, "access"); !errors.Is(err, jobsvc.ErrAccess) {
		t.Fatalf("an unlisted path: %v", err)
	}
	rows, _ = r.cell.Audit.Store.Range(ctx, 1, 1<<20)
	if rows[len(rows)-1].State != "Unlisted" || len(got.got) == 0 {
		t.Fatalf("unlisted: row %+v, alerts %+v", rows[len(rows)-1], got.got)
	}
}

// SB-1901: the off-box copy is sealed to the recovery key; the copies on the box stay as
// they are, and the sealed one opens with the recovery key to the same bytes.
func TestBackupSealsTheOffBoxCopy(t *testing.T) {
	r, svc, _, _ := jobsCell(t, true)
	base := t.TempDir()
	key, err := offsite.Generate(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	r.cell.Config.Backups = []wire.Backup{{Dir: filepath.Join(base, "a"), Media: "disk"}, {Dir: filepath.Join(base, "b"), Media: "nas"},
		{Dir: filepath.Join(base, "c"), Media: "disk", Offsite: true, Seal: key.Public().String()}}
	if err := svc.Run(context.Background(), "backup"); err != nil {
		t.Fatal(err)
	}
	plain, _ := filepath.Glob(filepath.Join(base, "a", "*", "*", "audit.ndjson"))
	sealed, _ := filepath.Glob(filepath.Join(base, "c", "*", "*", "audit.ndjson.sealed"))
	if len(plain) != 1 || len(sealed) != 1 {
		t.Fatalf("copies: on the box %v, off it %v", plain, sealed)
	}
	// Every object is sealed (export, approvals and their .sha256); the adapter's .retain
	// markers hold only a date.
	all, _ := filepath.Glob(filepath.Join(base, "c", "*", "*", "*"))
	n := 0
	for _, f := range all {
		switch {
		case strings.HasSuffix(f, ".sealed"):
			n++
		case !strings.HasSuffix(f, ".sealed.retain"):
			t.Fatalf("%s reached the off-box copy unsealed", f)
		}
	}
	if n != 4 {
		t.Fatalf("%d sealed objects off the box, want 4: %v", n, all)
	}
	want, _ := os.ReadFile(plain[0])
	f, _ := os.Open(sealed[0])
	defer func() { _ = f.Close() }()
	var got bytes.Buffer
	if err := offsite.Open(&got, f, key); err != nil || !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("the sealed copy doesn't open to the box's copy: %v", err)
	}
}
