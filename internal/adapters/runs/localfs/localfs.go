// Package localfs records each scheduled job's last finish in a file (sb-runtime(7),
// Schedules; SB-1504): ROOT/JOB holds an RFC 3339 time, written atomically, so every
// service of a cell reads the same record. A cell on GCP keeps it in its database.
package localfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/core/health"
)

// Dir is rooted at Root.
type Dir struct{ Root string }

var safe = regexp.MustCompile(`^[a-z][a-z-]*$`)

func (d Dir) path(job string) (string, error) {
	if !safe.MatchString(job) {
		return "", fmt.Errorf("runs: unsafe job name %q", job)
	}
	return filepath.Join(d.Root, job), nil
}

// LastFinish implements schedule.Runs. A record that can't be read or parsed is an error,
// so the watchdog counts the job overdue.
func (d Dir) LastFinish(_ context.Context, job string) (time.Time, bool, error) {
	p, err := d.path(job)
	if err != nil {
		return time.Time{}, false, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b)))
	if err != nil {
		return time.Time{}, false, fmt.Errorf("runs: %s: %w", job, err)
	}
	return at, true, nil
}

// Finished records that job finished at at.
func (d Dir) Finished(_ context.Context, job string, at time.Time) error {
	p, err := d.path(job)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d.Root, 0o700); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(at.UTC().Format(time.RFC3339Nano)+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Syncs keeps each connector's sync record (health.Syncs; SB-1607) as ROOT/sync-KIND.json:
// the ingest service writes it after each sync, and the console's health reads it.
type Syncs struct{ Root string }

func (s Syncs) path(kind string) (string, error) {
	if !safe.MatchString(kind) {
		return "", fmt.Errorf("runs: unsafe connector kind %q", kind)
	}
	return filepath.Join(s.Root, "sync-"+kind+".json"), nil
}

// Sync implements health.Syncs.
func (s Syncs) Sync(_ context.Context, kind string) (health.Sync, bool, error) {
	p, err := s.path(kind)
	if err != nil {
		return health.Sync{}, false, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return health.Sync{}, false, nil
	}
	if err != nil {
		return health.Sync{}, false, err
	}
	var x health.Sync
	if err := json.Unmarshal(b, &x); err != nil {
		return health.Sync{}, false, fmt.Errorf("runs: sync-%s.json: %w", kind, err)
	}
	return x, true, nil
}

// Succeeded records a successful sync, clearing the failures.
func (s Syncs) Succeeded(ctx context.Context, kind string, at time.Time) error {
	return s.update(ctx, kind, func(x *health.Sync) { x.LastSuccess, x.Failures, x.Failure = at.UTC(), 0, "" })
}

// Failed records a failed sync and its class (SB-3201): the class only, never a message.
func (s Syncs) Failed(ctx context.Context, kind string, at time.Time, class string) error {
	return s.update(ctx, kind, func(x *health.Sync) { x.LastFailure, x.Failure = at.UTC(), class; x.Failures++ })
}

func (s Syncs) update(ctx context.Context, kind string, f func(*health.Sync)) error {
	x, _, err := s.Sync(ctx, kind)
	if err != nil {
		x = health.Sync{} // an unreadable record is replaced by the next sync's
	}
	f(&x)
	p, err := s.path(kind)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(x)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
