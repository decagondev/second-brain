package laws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/adapters/backup/localfs"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
	"example.com/second-brain/internal/core/gameday"
)

// LAW-11 through a disaster (SB-613): when a cell's database is lost and restored from its
// 3-2-1 copies, the restored chain is row for row the chain that was backed up, it verifies
// with the cell's pinned key, and the next row written continues it. A tampered copy is
// skipped, never restored; if every copy is tampered, nothing is restored.

// DRWorld is a cell with backups that can lose its database.
type DRWorld interface {
	Write(n int)              // n audit rows, each with an approval
	Backup() error            // the nightly backup to three copies
	Tamper(copy int)          // change one byte of a copy's latest export
	Lose()                    // the database is gone: empty stores
	Restore() (string, error) // the copy restored from
	Backed() []string         // the row hashes at the last backup
	Chain() []string          // the row hashes in the store now
	Verify() error            // export the store now and verify it with the pinned key
	Continue() error          // write one row on the restored store, then Verify
}

// NewDRWorld builds the system under test: the real backup job, local write-once copies and
// the game-day restore.
var NewDRWorld = func(dir string) DRWorld { return newDRWorld(dir, false) }

type drWorld struct {
	dir     string
	rows    *auditmem.Store
	acts    *actmem.Store
	log     *audit.Log
	signer  audit.Signer
	targets []localfs.Dir
	backed  []string
	now     time.Time
	n       int
	trust   bool // mutant: restore the first copy without checking it
}

func newDRWorld(dir string, trust bool) *drWorld {
	signer, _ := audit.NewSigner("acme-logistics", bytes.Repeat([]byte{7}, 32))
	w := &drWorld{dir: dir, signer: signer, trust: trust, now: time.Date(2027, 3, 2, 1, 0, 0, 0, time.UTC),
		targets: []localfs.Dir{{Root: filepath.Join(dir, "a"), Label: "primary", Media: "disk"},
			{Root: filepath.Join(dir, "b"), Label: "nas", Media: "nas"},
			{Root: filepath.Join(dir, "c"), Label: "offsite", Media: "object-store", Away: true}}}
	w.Lose()
	return w
}

func (w *drWorld) Lose() {
	w.rows, w.acts = auditmem.New(), actmem.New()
	w.log = &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: w.rows, Now: func() time.Time { return w.now }}
}

func (w *drWorld) Write(n int) {
	for i := 0; i < n; i++ {
		w.n++
		w.now = w.now.Add(time.Minute)
		id := fmt.Sprintf("tc_%d", w.n)
		_, _ = w.log.Write(context.Background(), audit.Event{Actor: "usr_ann", Kind: "approval", Subject: id, Reason: "ok"})
		_ = w.acts.Put(context.Background(), approvals.Action{ID: id, Tenant: "acme-logistics", State: approvals.Ran})
	}
}

func (w *drWorld) ts() []backup.Target {
	out := make([]backup.Target, len(w.targets))
	for i, t := range w.targets {
		out[i] = t
	}
	return out
}

func (w *drWorld) Backup() error {
	job := backup.Job{Tenant: "acme-logistics", Audit: w.rows, Signer: w.signer, Approvals: w.acts, Targets: w.ts(), Now: func() time.Time { return w.now }}
	if _, err := job.Run(context.Background()); err != nil {
		return err
	}
	w.backed = w.Chain()
	return nil
}

func (w *drWorld) Tamper(i int) {
	p := filepath.Join(w.targets[i].Root, "acme-logistics", w.now.Format("2006-01-02"), "audit.ndjson")
	_ = os.Chmod(p, 0o600)
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	_ = os.WriteFile(p, bytes.Replace(b, []byte(`"reason":"ok"`), []byte(`"reason":"no"`), 1), 0o600)
}

func (w *drWorld) Restore() (string, error) {
	if w.trust {
		p := filepath.Join(w.targets[0].Root, "acme-logistics", w.now.Format("2006-01-02"), "audit.ndjson")
		return w.targets[0].Label, w.importRaw(p)
	}
	res, err := gameday.Restore(context.Background(), "acme-logistics", w.signer.Key(), w.ts(), w.rows, w.acts)
	return res.Copy, err
}

// importRaw is the mutant's restore: copy the rows in without checking anything.
func (w *drWorld) importRaw(p string) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	for _, line := range bytes.Split(bytes.TrimSpace(b), []byte("\n")) {
		if bytes.Contains(line, []byte(`"anchor":true`)) {
			continue
		}
		var row audit.Row
		if err := json.Unmarshal(line, &row); err != nil {
			return err
		}
		_, _ = w.rows.Append(context.Background(), func(int64, string) (audit.Row, error) { return row, nil })
	}
	return nil
}

func (w *drWorld) Backed() []string { return w.backed }

func (w *drWorld) Chain() []string {
	rs, _ := w.rows.Range(context.Background(), 1, 1<<20)
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = fmt.Sprintf("%d:%s", r.Seq, r.Hash)
	}
	return out
}

func (w *drWorld) Verify() error {
	var buf bytes.Buffer
	if _, err := audit.Export(context.Background(), w.rows, w.signer, 1, w.now, &buf); err != nil {
		return err
	}
	_, err := audit.Verify(&buf, "acme-logistics", w.signer.Key())
	return err
}

func (w *drWorld) Continue() error {
	w.Write(1)
	return w.Verify()
}

func law11Restore(r *rand.Rand, w DRWorld) error {
	w.Write(1 + r.Intn(20))
	if err := w.Backup(); err != nil {
		return err
	}
	w.Write(r.Intn(5)) // written after the backup: lost with the database
	bad := map[int]bool{}
	switch r.Intn(4) {
	case 1:
		bad[r.Intn(3)] = true
	case 2:
		bad[0], bad[1+r.Intn(2)] = true, true
	case 3:
		bad[0], bad[1], bad[2] = true, true, true
	}
	labels := []string{"primary", "nas", "offsite"}
	for i := range bad {
		w.Tamper(i)
	}
	w.Lose()
	used, err := w.Restore()
	if len(bad) == 3 {
		if err == nil || len(w.Chain()) != 0 {
			return fmt.Errorf("every copy was tampered, yet the restore used %q and wrote %d rows", used, len(w.Chain()))
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("restore with copies %v tampered: %v", bad, err)
	}
	for i, l := range labels {
		if bad[i] && used == l {
			return fmt.Errorf("restored from the tampered copy %s", l)
		}
	}
	if fmt.Sprint(w.Chain()) != fmt.Sprint(w.Backed()) {
		return fmt.Errorf("the restored chain isn't the backed-up chain:\n%v\n%v", w.Chain(), w.Backed())
	}
	if err := w.Verify(); err != nil {
		return fmt.Errorf("the restored chain doesn't verify: %v", err)
	}
	if err := w.Continue(); err != nil {
		return fmt.Errorf("the chain doesn't continue after the restore: %v", err)
	}
	return nil
}
