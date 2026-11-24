package ingest

import (
	"context"
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
}

// Once syncs from the stored cursor and stores the new cursor only after the whole sync
// succeeded. On failure the old cursor stays, so the next run repeats the (idempotent) work.
func (s Syncer) Once(ctx context.Context) error {
	t := s.Pipeline.Tenant
	cur, _, err := s.Cursors.Cursor(ctx, t, s.Connector)
	if err == nil {
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
