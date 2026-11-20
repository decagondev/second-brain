// Package localfs keeps a cell's logs as one file per UTC day, DIR/YYYY-MM-DD.log, and is
// the `logs` retention store (sb-tenant(5), Retention and deletion; SB-611). Lines are the
// JSON lines internal/obs writes: ids only, never content.
package localfs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/retention"
)

// Files is safe for concurrent use. Use it as obs's writer.
type Files struct {
	Dir string
	Now func() time.Time
	mu  sync.Mutex
}

var _ retention.Store = (*Files)(nil)

const layout = "2006-01-02"

// Write appends p to today's file.
func (f *Files) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := os.MkdirAll(f.Dir, 0o700); err != nil {
		return 0, err
	}
	fh, err := os.OpenFile(filepath.Join(f.Dir, f.Now().UTC().Format(layout)+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}
	n, err := fh.Write(p)
	if cerr := fh.Close(); err == nil {
		err = cerr
	}
	return n, err
}

// Name implements retention.Store.
func (f *Files) Name() string { return "logs" }

func (f *Files) days() ([]string, error) {
	es, err := os.ReadDir(f.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range es {
		day, ok := strings.CutSuffix(e.Name(), ".log")
		if _, perr := time.Parse(layout, day); ok && perr == nil && !e.IsDir() {
			out = append(out, day)
		}
	}
	sort.Strings(out)
	return out, nil
}

func lines(b []byte) int { return bytes.Count(b, []byte("\n")) }

// Expire deletes the files whose whole day is before the cutoff; it returns the lines deleted.
// A cell has one tenant, so tenant names which cell's logs these are and nothing more.
// A log file's lines may be about anyone in any space, so while any legal hold is active
// every file is kept (SB-1207): unknown never deletes.
func (f *Files) Expire(_ context.Context, _ string, before time.Time, held retention.Held) (int, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	days, err := f.days()
	if err != nil {
		return 0, 0, err
	}
	n, kept := 0, 0
	for _, d := range days {
		t, _ := time.Parse(layout, d)
		if t.Add(24 * time.Hour).After(before) {
			continue // the file may hold a line at or after the cutoff
		}
		p := filepath.Join(f.Dir, d+".log")
		b, err := os.ReadFile(p)
		if err != nil {
			return n, kept, err
		}
		if held.Is("", "") {
			kept += lines(b)
			continue
		}
		if err := os.Remove(p); err != nil {
			return n, kept, err
		}
		n += lines(b)
	}
	return n, kept, nil
}

// Erase rewrites every file without the lines that name the user in any field.
func (f *Files) Erase(_ context.Context, _ string, user string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	days, err := f.days()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range days {
		p := filepath.Join(f.Dir, d+".log")
		b, err := os.ReadFile(p)
		if err != nil {
			return n, err
		}
		var keep bytes.Buffer
		dropped := 0
		sc := bufio.NewScanner(bytes.NewReader(b))
		sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
		for sc.Scan() {
			if names(sc.Bytes(), user) {
				dropped++
				continue
			}
			keep.Write(sc.Bytes())
			keep.WriteByte('\n')
		}
		if err := sc.Err(); err != nil {
			return n, err
		}
		if dropped == 0 {
			continue
		}
		tmp := p + ".tmp"
		if err := os.WriteFile(tmp, keep.Bytes(), 0o600); err != nil {
			return n, err
		}
		if err := os.Rename(tmp, p); err != nil {
			return n, err
		}
		n += dropped
	}
	return n, nil
}

// names reports whether a line names the user: any string value equal to the id. A line
// that isn't JSON is matched on the id as a whole word, so it is never kept by mistake.
func names(line []byte, user string) bool {
	var v any
	if json.Unmarshal(line, &v) != nil {
		for _, w := range strings.FieldsFunc(string(line), func(r rune) bool {
			return r != '_' && (r < 'a' || r > 'z') && (r < '0' || r > '9')
		}) {
			if w == user {
				return true
			}
		}
		return false
	}
	var walk func(any) bool
	walk = func(x any) bool {
		switch t := x.(type) {
		case string:
			return t == user
		case map[string]any:
			for _, e := range t {
				if walk(e) {
					return true
				}
			}
		case []any:
			for _, e := range t {
				if walk(e) {
					return true
				}
			}
		}
		return false
	}
	return walk(v)
}
