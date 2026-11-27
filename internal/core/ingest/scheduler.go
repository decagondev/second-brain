package ingest

import (
	"context"
	"encoding/json"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Schedule constants (sb-ingest(7), Sync).
const (
	SyncEvery     = 60 * time.Second
	SyncJitterMax = 10 * time.Second
	FreshnessSLA  = 15 * time.Minute
)

// Recorder receives sync outcomes for metrics (obs.Metrics in production).
type Recorder interface {
	SyncSucceeded(tenant, connector string, at time.Time)
	SyncFailed(tenant, connector string, err error)
}

// Syncer runs one connector's sync with a durable cursor.
type Syncer struct {
	Pipeline  Pipeline
	Connector string // the connector's kind, e.g. docs
	Cursors   ports.CursorStore
	Recorder  Recorder
	Now       func() time.Time
	// Progress is told a first sync's checkpoints (SB-6004), for health; nil tells nobody.
	Progress func(ctx context.Context, connector string, cp Checkpoint)
}

// firstKey is where a first sync's checkpoint is kept, beside the connector's cursor (CONNECTOR--first).
func (s Syncer) firstKey() string { return s.Connector + "--first" }

// first runs a first sync (no cursor yet) from its checkpoint, keeping one every
// CheckpointEvery changes; the checkpoint goes once the cursor is kept.
func (s Syncer) first(ctx context.Context) error {
	t := s.Pipeline.Tenant
	var at Checkpoint
	if raw, found, err := s.Cursors.Cursor(ctx, t, s.firstKey()); err != nil {
		return err
	} else if found && raw != "" {
		if json.Unmarshal([]byte(raw), &at) != nil {
			at = Checkpoint{} // a checkpoint that can't be read starts over: the work is idempotent
		}
	}
	next, err := s.Pipeline.SyncFrom(ctx, "", at.Done, func(ctx context.Context, cp Checkpoint) error {
		if s.Progress != nil {
			s.Progress(ctx, s.Connector, cp)
		}
		b, _ := json.Marshal(cp)
		return s.Cursors.SaveCursor(ctx, t, s.firstKey(), ports.Cursor(b))
	})
	if err != nil {
		return err
	}
	if err := s.Cursors.SaveCursor(ctx, t, s.Connector, next); err != nil {
		return err
	}
	return s.Cursors.SaveCursor(ctx, t, s.firstKey(), "")
}

// Once syncs from the stored cursor and stores the new cursor only after the whole sync
// succeeded. On failure the old cursor stays, so the next run repeats the (idempotent) work.
func (s Syncer) Once(ctx context.Context) error {
	t := s.Pipeline.Tenant
	cur, _, err := s.Cursors.Cursor(ctx, t, s.Connector)
	if err == nil && cur == "" {
		err = s.first(ctx)
	} else if err == nil {
		var next ports.Cursor
		next, err = s.Pipeline.Sync(ctx, cur)
		if err == nil {
			err = s.Cursors.SaveCursor(ctx, t, s.Connector, next)
		}
	}
	if s.Recorder != nil {
		if err != nil {
			s.Recorder.SyncFailed(t, s.Connector, err)
		} else {
			s.Recorder.SyncSucceeded(t, s.Connector, s.Now())
		}
	}
	return err
}

// Run syncs every SyncEvery plus jitter until ctx ends. jitter returns a duration in
// [0, SyncJitterMax); errors are recorded, never fatal.
func (s Syncer) Run(ctx context.Context, jitter func() time.Duration) {
	for {
		_ = s.Once(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(SyncEvery + jitter()):
		}
	}
}
