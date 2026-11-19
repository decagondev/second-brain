// Package gameday is the disaster-recovery game day (sb-observability(7), Disaster-recovery
// game day, exactly; SB-613): restore a lost cell's audit chain and approvals from its 3-2-1
// copies into empty stores, and score the day against RTO 4 h and RPO 15 min.
package gameday

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
)

// Targets.
const (
	RTO = 4 * time.Hour
	RPO = 15 * time.Minute
)

// Errors.
var (
	ErrNoCopy    = errors.New("gameday: no copy passes the restore checks")
	ErrDisagree  = errors.New("gameday: copies that pass hold different bytes; a human chooses")
	ErrNotEmpty  = errors.New("gameday: restore only into empty stores")
	ErrChainGap  = errors.New("gameday: a row doesn't continue the chain")
	ErrSnapshot  = errors.New("gameday: the approvals snapshot can't be read")
	ErrAppending = errors.New("gameday: a restored row can't be written")
)

// Actions is the approvals store a restore writes into.
type Actions interface {
	Put(ctx context.Context, a approvals.Action) error
	All(ctx context.Context, tenant string) ([]approvals.Action, error)
}

// Restored is what Restore did.
type Restored struct {
	Copy    string              `json:"copy"`
	Object  string              `json:"object"`
	Rows    int                 `json:"rows"`
	LastSeq int64               `json:"last_seq"`
	LastAt  time.Time           `json:"last_at"`
	Actions int                 `json:"actions"`
	Skipped []backup.CopyResult `json:"skipped,omitempty"`
}

// Restore checks every copy and restores the first that passes into empty stores.
func Restore(ctx context.Context, tenant, pinnedKey string, ts []backup.Target, rows audit.Store, acts Actions) (Restored, error) {
	if have, err := rows.Range(ctx, 1, 1); err != nil || len(have) > 0 {
		return Restored{}, ErrNotEmpty
	}
	if have, err := acts.All(ctx, tenant); err != nil || len(have) > 0 {
		return Restored{}, ErrNotEmpty
	}
	rep := backup.Drill(ctx, tenant, pinnedKey, ts, 0, nil, time.Time{}) // no live chain to lag behind
	var res Restored
	var use backup.Target
	for _, c := range rep.Copies {
		if c.Target == "(all)" {
			return Restored{}, ErrDisagree
		}
		if !c.OK {
			res.Skipped = append(res.Skipped, c)
			continue
		}
		if use == nil {
			for _, t := range ts {
				if t.Name() == c.Target {
					use = t
				}
			}
			res.Copy, res.Object = c.Target, c.Object
		}
	}
	if use == nil {
		return res, ErrNoCopy
	}
	data, err := use.Get(ctx, res.Object)
	if err != nil {
		return res, fmt.Errorf("%w: %v", ErrNoCopy, err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if bytes.Contains(line, []byte(`"anchor":true`)) {
			continue
		}
		var row audit.Row
		if err := json.Unmarshal(line, &row); err != nil {
			return res, fmt.Errorf("%w: %v", ErrChainGap, err)
		}
		if _, err := rows.Append(ctx, func(lastSeq int64, lastHash string) (audit.Row, error) {
			if row.Seq != lastSeq+1 || row.Prev != lastHash {
				return audit.Row{}, fmt.Errorf("%w: seq %d after %d", ErrChainGap, row.Seq, lastSeq)
			}
			if h, err := audit.HashOf(row); err != nil || h != row.Hash {
				return audit.Row{}, fmt.Errorf("%w: seq %d hash", ErrChainGap, row.Seq)
			}
			return row, nil
		}); err != nil {
			return res, fmt.Errorf("%w: %v", ErrAppending, err)
		}
		res.Rows++
		res.LastSeq = row.Seq
		if at, err := time.Parse(time.RFC3339Nano, row.At); err == nil {
			res.LastAt = at
		}
	}
	if err := sc.Err(); err != nil {
		return res, fmt.Errorf("%w: %v", ErrChainGap, err)
	}
	snapName := strings.TrimSuffix(res.Object, "audit.ndjson") + "approvals.jsonl"
	snap, err := use.Get(ctx, snapName)
	if err != nil {
		return res, fmt.Errorf("%w: %v", ErrSnapshot, err)
	}
	dec := json.NewDecoder(bytes.NewReader(snap))
	for dec.More() {
		var a approvals.Action
		if err := dec.Decode(&a); err != nil || a.Tenant != tenant {
			return res, ErrSnapshot
		}
		if err := acts.Put(ctx, a); err != nil {
			return res, fmt.Errorf("%w: %v", ErrSnapshot, err)
		}
		res.Actions++
	}
	return res, nil
}

// Score is one game day's result.
type Score struct {
	Tenant   string
	Failure  time.Time // T0
	Answered time.Time // the restored cell passed its smoke test
	LastRow  time.Time // the last audit row in the restored chain
	Chain    bool      // verifies, row for row, and continues (LAW-11)
	Index    bool      // the rebuilt index equals the one before (LAW-13)
	Region   bool      // every copy used and the restored cell are in the pinned region
}

// RTO and RPO as measured.
func (s Score) RTO() time.Duration { return s.Answered.Sub(s.Failure) }
func (s Score) RPO() time.Duration { return s.Failure.Sub(s.LastRow) }

type line struct {
	name, value string
	ok          bool
}

func (s Score) lines() []line {
	yes := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	return []line{
		{"RTO", s.RTO().Round(time.Second).String(), !s.Answered.IsZero() && s.RTO() >= 0 && s.RTO() <= RTO},
		{"RPO", s.RPO().Round(time.Second).String(), !s.LastRow.IsZero() && s.RPO() >= 0 && s.RPO() <= RPO},
		{"Chain", yes(s.Chain), s.Chain},
		{"Index", yes(s.Index), s.Index},
		{"Region", yes(s.Region), s.Region},
	}
}

// Pass is true only if every line passes.
func (s Score) Pass() bool {
	for _, l := range s.lines() {
		if !l.ok {
			return false
		}
	}
	return true
}

// Markdown is the score for the retro.
func (s Score) Markdown() string {
	var b strings.Builder
	verdict := "passed"
	if !s.Pass() {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "### DR game day: %s\n\nCell `%s`, T0 %s.\n\n| Measure | Result | Pass |\n|---|---|---|\n", verdict, s.Tenant, s.Failure.UTC().Format(time.RFC3339))
	for _, l := range s.lines() {
		p := "pass"
		if !l.ok {
			p = "FAIL"
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", l.name, l.value, p)
	}
	return b.String()
}

// SameIndex reports whether two (cite, ACL) snapshots hold the same set.
func SameIndex(before, after []string) bool {
	a := append([]string(nil), before...)
	c := append([]string(nil), after...)
	sort.Strings(a)
	sort.Strings(c)
	return len(a) > 0 && strings.Join(a, "\n") == strings.Join(c, "\n")
}
