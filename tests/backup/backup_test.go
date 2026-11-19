// Package backup tests the nightly backup and the restore drill end to end over real
// local write-once targets and stores (SB-412).
package backup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/adapters/backup/localfs"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
)

// A backup to three copies, then the restore drill: it passes, and fails on a tampered
// copy, naming it.
func TestBackupAndRestoreDrill(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 3, 2, 3, 0, 0, 0, time.UTC)
	rows := auditmem.New()
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: rows, Now: func() time.Time { return now }}
	for i := 0; i < 7; i++ {
		if _, err := log.Write(ctx, audit.Event{Actor: "usr_1", Kind: "approval", Subject: "tc_1", Reason: "ok"}); err != nil {
			t.Fatal(err)
		}
	}
	acts := actmem.New()
	_ = acts.Put(ctx, approvals.Action{ID: "tc_1", Tenant: "acme-logistics", State: approvals.Ran, Receipt: "linear:comment/1"})
	signer, _ := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{4}, 32))
	a := localfs.Dir{Root: t.TempDir(), Label: "primary", Media: "disk"}
	b := localfs.Dir{Root: t.TempDir(), Label: "nas", Media: "nas"}
	c := localfs.Dir{Root: t.TempDir(), Label: "offsite", Media: "object-store", Away: true}
	targets := []backup.Target{a, b, c}

	if err := backup.Check(targets[:2]); !errors.Is(err, backup.ErrRule) {
		t.Fatalf("two copies: %v", err)
	}
	if err := backup.Check([]backup.Target{a, localfs.Dir{Media: "disk"}, localfs.Dir{Media: "disk", Away: true}}); !errors.Is(err, backup.ErrRule) {
		t.Fatalf("one medium: %v", err)
	}
	if err := backup.Check([]backup.Target{a, b, localfs.Dir{Media: "disk"}}); !errors.Is(err, backup.ErrRule) {
		t.Fatalf("no off-site: %v", err)
	}

	job := backup.Job{Tenant: "acme-logistics", Audit: rows, Signer: signer, Approvals: acts, Targets: targets, Now: func() time.Time { return now }}
	res, err := job.Run(ctx)
	if err != nil || res.Rows != 7 || len(res.SHA256) != 2 {
		t.Fatalf("backup: %+v %v", res, err)
	}
	if _, err := job.Run(ctx); err == nil {
		t.Fatal("a second run the same night replaced the first")
	}
	snap, _ := c.Get(ctx, "acme-logistics/2027-03-02/approvals.jsonl")
	if !strings.Contains(string(snap), `"receipt":"linear:comment/1"`) {
		t.Fatalf("snapshot %s", snap)
	}

	rep := backup.Drill(ctx, "acme-logistics", signer.Key(), targets, 7, nil, now)
	if !rep.Passed || len(rep.Copies) != 3 {
		t.Fatalf("drill: %s", rep.Report("acme-logistics"))
	}

	// Tamper with the NAS copy (an attacker with write access to the share).
	p := filepath.Join(b.Root, "acme-logistics", "2027-03-02", "audit.ndjson")
	_ = os.Chmod(p, 0o600)
	orig, _ := os.ReadFile(p)
	_ = os.WriteFile(p, bytes.Replace(orig, []byte(`"reason":"ok"`), []byte(`"reason":"no"`), 1), 0o600)
	rep = backup.Drill(ctx, "acme-logistics", signer.Key(), targets, 7, nil, now)
	out := rep.Report("acme-logistics")
	if rep.Passed || !strings.Contains(out, "| nas |") || !strings.Contains(out, "sha256 doesn't match") {
		t.Fatalf("tampered copy:\n%s", out)
	}

	// A copy far behind the live chain fails too.
	rep = backup.Drill(ctx, "acme-logistics", signer.Key(), []backup.Target{a, c}, 50, func(int64) (time.Time, bool) { return now.Add(-48 * time.Hour), true }, now)
	if rep.Passed {
		t.Fatalf("stale copies passed:\n%s", rep.Report("acme-logistics"))
	}
}
