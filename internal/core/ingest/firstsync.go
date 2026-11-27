package ingest

import (
	"context"
	"errors"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// A big tenant's first sync (SB-6004; sb-ingest(7), First syncs): syncs are paced, wait on
// a source's 429 instead of failing, and a first sync keeps a checkpoint every
// CheckpointEvery items so a restart resumes where it stopped.
const (
	DefaultPerMinute = 600
	CheckpointEvery  = 500
	MaxRetries       = 8
	MaxWait          = 5 * time.Minute
)

// RetryAfter is an error that says how long the source asked to wait (its Retry-After).
type RetryAfter interface {
	RetryAfter() time.Duration
}

// Pace spaces a pipeline's reads to PerMinute a minute (0 is DefaultPerMinute). Sleep and
// Now are the clock; nil is the real one. One Pace belongs to one connector.
type Pace struct {
	PerMinute int
	Sleep     func(ctx context.Context, d time.Duration) error
	Now       func() time.Time
	last      time.Time
}

func (p *Pace) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

func (p *Pace) sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	if p.Sleep != nil {
		return p.Sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// wait holds the next read until its turn.
func (p *Pace) wait(ctx context.Context) error {
	n := p.PerMinute
	if n <= 0 {
		n = DefaultPerMinute
	}
	gap := time.Minute / time.Duration(n)
	if !p.last.IsZero() {
		if err := p.sleep(ctx, p.last.Add(gap).Sub(p.now())); err != nil {
			return err
		}
	}
	p.last = p.now()
	return nil
}

// throttled says whether err is a source's 429, and how long to wait before the next try.
func throttled(err error, try int) (time.Duration, bool) {
	var se ports.StatusError
	if !errors.As(err, &se) || se.HTTPStatus() != 429 {
		return 0, false
	}
	d := time.Second << try
	var ra RetryAfter
	if errors.As(err, &ra) && ra.RetryAfter() > 0 {
		d = ra.RetryAfter()
	}
	return min(d, MaxWait), true
}

// patient reads ref at its pace, waiting out a source's 429s up to MaxRetries times.
func (p Pipeline) patient(ctx context.Context, ref ports.SourceRef, force bool) error {
	pace := p.Pace
	if pace == nil {
		pace = &Pace{PerMinute: 1 << 30} // unpaced
	}
	for try := 0; ; try++ {
		if err := pace.wait(ctx); err != nil {
			return err
		}
		err := p.Ingest(ctx, ref, force)
		d, ok := throttled(err, try)
		if !ok || try >= MaxRetries {
			return err
		}
		if p.Throttled != nil {
			p.Throttled(p.Kind)
		}
		if err := pace.sleep(ctx, d); err != nil {
			return err
		}
	}
}

// Checkpoint is how far a first sync has come: Done of Total changes handled.
type Checkpoint struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// Progress receives a first sync's checkpoints; an error stops the sync (it resumes from
// the last checkpoint kept).
type Progress func(ctx context.Context, cp Checkpoint) error

// SyncFrom is Sync that skips the first skip changes, which a checkpoint says are done, and
// calls progress at the start and every CheckpointEvery changes. The connector lists a
// first sync's changes in the same order each time, so skipping resumes it.
func (p Pipeline) SyncFrom(ctx context.Context, cursor ports.Cursor, skip int, progress Progress) (ports.Cursor, error) {
	return p.sync(ctx, cursor, skip, progress)
}
