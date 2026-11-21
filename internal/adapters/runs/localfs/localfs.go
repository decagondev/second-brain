// Package localfs records each scheduled job's last finish in a file (sb-runtime(7),
// Schedules; SB-1504): ROOT/JOB holds an RFC 3339 time, written atomically, so every
// service of a cell reads the same record. A cell on GCP keeps it in its database.
package localfs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
