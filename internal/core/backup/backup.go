// Package backup backs up a cell's audit chain and approval store 3-2-1 to write-once
// targets, and runs the restore drill (sb-audit(7), Retention and backup; SB-412).
package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
)

// Retention is how long every copy is kept, write-once.
const Retention = 7 * 365 * 24 * time.Hour

// MaxLag is how far behind the live chain the latest copy may be at drill time.
const MaxLag = 24 * time.Hour

// Target is one write-once copy.
type Target interface {
	Name() string
	Medium() string // e.g. "disk", "object-store", "tape"
	Offsite() bool
	Put(ctx context.Context, name string, data []byte, retainUntil time.Time) error // refuses an existing name
	Get(ctx context.Context, name string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]string, error)
}

// ErrRule is a target set that isn't 3-2-1.
var ErrRule = errors.New("backup: not 3-2-1")

// Check enforces three copies, two media, one off-site.
func Check(ts []Target) error {
	media := map[string]bool{}
	offsite := false
	for _, t := range ts {
		media[t.Medium()] = true
		offsite = offsite || t.Offsite()
	}
	switch {
	case len(ts) < 3:
		return fmt.Errorf("%w: %d copies, need at least 3", ErrRule, len(ts))
	case len(media) < 2:
		return fmt.Errorf("%w: %d kind of media, need at least 2", ErrRule, len(media))
	case !offsite:
		return fmt.Errorf("%w: no off-site copy", ErrRule)
	}
	return nil
}

// Approvals lists every action of a tenant, for the snapshot.
type Approvals interface {
	All(ctx context.Context, tenant string) ([]approvals.Action, error)
}

// Job is one cell's nightly backup.
type Job struct {
	Tenant    string
	Audit     audit.Store
	Signer    audit.Signer
	Approvals Approvals
	Targets   []Target
	Now       func() time.Time
}

// Result names what one run wrote.
type Result struct {
	Day    string
	Rows   int
	SHA256 map[string]string // object name -> hex
}

// Run writes the audit export and the approvals snapshot to every target.
func (j Job) Run(ctx context.Context) (Result, error) {
	if err := Check(j.Targets); err != nil {
		return Result{}, err
	}
	now := j.Now().UTC()
	day := now.Format("2006-01-02")
	var exp bytes.Buffer
	n, err := audit.Export(ctx, j.Audit, j.Signer, 1, now, &exp)
	if err != nil {
		return Result{}, fmt.Errorf("backup: export: %w", err)
	}
	acts, err := j.Approvals.All(ctx, j.Tenant)
	if err != nil {
		return Result{}, fmt.Errorf("backup: approvals: %w", err)
	}
	sort.Slice(acts, func(a, b int) bool { return acts[a].ID < acts[b].ID })
	var snap bytes.Buffer
	enc := json.NewEncoder(&snap)
	for _, a := range acts {
		if err := enc.Encode(a); err != nil {
			return Result{}, err
		}
	}
	res := Result{Day: day, Rows: n, SHA256: map[string]string{}}
	objects := map[string][]byte{
		j.Tenant + "/" + day + "/audit.ndjson":    exp.Bytes(),
		j.Tenant + "/" + day + "/approvals.jsonl": snap.Bytes(),
	}
	names := make([]string, 0, len(objects))
	for k := range objects {
		names = append(names, k)
	}
	sort.Strings(names)
	until := now.Add(Retention)
	for _, name := range names {
		sum := sha256.Sum256(objects[name])
		h := hex.EncodeToString(sum[:])
		res.SHA256[name] = h
		for _, t := range j.Targets {
			if err := t.Put(ctx, name, objects[name], until); err != nil {
				return res, fmt.Errorf("backup: %s: %s: %w", t.Name(), name, err)
			}
			if err := t.Put(ctx, name+".sha256", []byte(h+"\n"), until); err != nil {
				return res, fmt.Errorf("backup: %s: %s.sha256: %w", t.Name(), name, err)
			}
		}
	}
	return res, nil
}

// CopyResult is one copy's drill outcome.
type CopyResult struct {
	Target  string
	Object  string
	LastSeq int64
	OK      bool
	Problem string
}

// DrillReport is the restore drill's outcome.
type DrillReport struct {
	Copies []CopyResult
	Passed bool
}

// Drill fetches the latest audit export from every copy, checks its sha256, verifies the
// chain with the cell's pinned key, and checks the copies agree and aren't behind
// liveLastSeq by more than MaxLag (liveAt is when that row was written).
func Drill(ctx context.Context, tenant, pinnedKey string, ts []Target, liveLastSeq int64, liveRowsSince func(seq int64) (time.Time, bool), now time.Time) DrillReport {
	rep := DrillReport{Passed: len(ts) > 0}
	hashes := map[string]bool{}
	for _, t := range ts {
		cr := CopyResult{Target: t.Name()}
		fail := func(p string) {
			cr.Problem, rep.Passed = p, false
			rep.Copies = append(rep.Copies, cr)
		}
		names, err := t.List(ctx, tenant+"/")
		if err != nil {
			fail("list: " + err.Error())
			continue
		}
		latest := ""
		for _, n := range names {
			if strings.HasSuffix(n, "/audit.ndjson") && n > latest {
				latest = n
			}
		}
		if latest == "" {
			fail("no audit export")
			continue
		}
		cr.Object = latest
		data, err := t.Get(ctx, latest)
		if err != nil {
			fail("get: " + err.Error())
			continue
		}
		want, err := t.Get(ctx, latest+".sha256")
		sum := sha256.Sum256(data)
		got := hex.EncodeToString(sum[:])
		if err != nil || strings.TrimSpace(string(want)) != got {
			fail("sha256 doesn't match")
			continue
		}
		r, err := audit.Verify(bytes.NewReader(data), tenant, pinnedKey)
		if err != nil {
			fail("chain: " + strings.TrimPrefix(err.Error(), "audit: "))
			continue
		}
		cr.LastSeq = r.LastSeq
		if cr.LastSeq < liveLastSeq && liveRowsSince != nil {
			if at, ok := liveRowsSince(cr.LastSeq + 1); ok && now.Sub(at) > MaxLag {
				fail(fmt.Sprintf("behind the live chain: copy ends at seq %d, live at %d", cr.LastSeq, liveLastSeq))
				continue
			}
		}
		hashes[latest+"@"+got] = true
		cr.OK = true
		rep.Copies = append(rep.Copies, cr)
	}
	objects := map[string]bool{}
	for k := range hashes {
		objects[strings.SplitN(k, "@", 2)[0]] = true
	}
	if len(hashes) > len(objects) {
		rep.Passed = false // two copies hold different bytes for the same night
		rep.Copies = append(rep.Copies, CopyResult{Target: "(all)", Problem: "copies disagree"})
	}
	return rep
}

// Report is the drill's Markdown summary.
func (r DrillReport) Report(tenant string) string {
	var b strings.Builder
	verdict := "passed"
	if !r.Passed {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "### Restore drill: %s\n\nCell `%s`.\n\n| Copy | Object | Last seq | Result |\n|---|---|---|---|\n", verdict, tenant)
	for _, c := range r.Copies {
		res := "pass"
		if !c.OK {
			res = "FAIL: " + c.Problem
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %s |\n", c.Target, c.Object, c.LastSeq, res)
	}
	return b.String()
}
