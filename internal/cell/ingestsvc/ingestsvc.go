// Package ingestsvc is a cell's ingest service (sb-ingest(7), Sync; SB-1503): it runs
// every connector's sync from its stored cursor, re-routing first, backs a failing
// connector off without stopping the others, and alerts when a connector's freshness
// passes 15 minutes.
package ingestsvc

import (
	"context"
	"errors"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// MaxBackoff is the longest wait between a failing connector's syncs.
const MaxBackoff = 15 * time.Minute

// Service syncs one cell's connectors.
type Service struct {
	Cell *wire.Cell
	// Alert sends the ingest_stale alert; nil is the cell's (wire.Cell.Alerter).
	Alert canary.Alerter
	// Now, Sleep and Jitter are the clock; nil is the real one.
	Now    func() time.Time
	Sleep  func(ctx context.Context, d time.Duration) error
	Jitter func() time.Duration

	mu      sync.Mutex
	last    map[string]time.Time // last success
	started time.Time
	stale   map[string]bool // alerted, until the next success
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Kinds are the cell's connector kinds, sorted.
func (s *Service) Kinds() []string {
	out := make([]string, 0, len(s.Cell.Parts().Connectors))
	for k := range s.Cell.Parts().Connectors {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Pipeline is the ingest pipeline for one connector kind, built from the cell's parts.
func (s *Service) Pipeline(kind string) ingest.Pipeline {
	c := s.Cell
	m := c.Metrics
	return ingest.Pipeline{Tenant: c.Config.Tenant, Release: c.Release, Connector: c.Parts().Connectors[kind], Embedder: c.Parts().Agent.Embedder,
		Index: c.Index, Kind: kind, Now: s.now,
		Spaces: func(ctx context.Context) ([]ports.Space, error) { return c.Spaces.Spaces(ctx, c.Config.Tenant) },
		SecretDropped: func(k, pattern string) {
			m.Add("sb_ingest_secrets_dropped_total", "Chunks dropped for holding a secret (SB-1103).", map[string]string{"connector": k, "pattern": pattern}, 1)
		},
		FieldsDropped: func(k string, names []string) {
			for _, n := range names {
				m.Add("sb_ingest_fields_dropped_total", "Structured fields dropped at ingest, by name (SB-1106).", map[string]string{"connector": k, "field": n}, 1)
			}
		},
		BadReviewBy: func(k string) {
			m.Add("sb_ingest_review_by_ignored_total", "Review-by dates ingest couldn't read and ignored (SB-2705).", map[string]string{"connector": k}, 1)
		},
		RerouteDropped: func(k string) {
			m.Add("sb_ingest_reroute_dropped_total", "Documents Reroute took out because they couldn't be fetched.", map[string]string{"connector": k}, 1)
		}}
}

// SyncerFor is the durable-cursor syncer for one connector kind.
func (s *Service) SyncerFor(kind string) ingest.Syncer {
	return ingest.Syncer{Pipeline: s.Pipeline(kind), Connector: kind, Cursors: s.Cell.Cursors, Recorder: recorder{s}, Now: s.now}
}

type recorder struct{ s *Service }

func (r recorder) SyncSucceeded(tenant, connector string, at time.Time) {
	r.s.mu.Lock()
	if r.s.last == nil {
		r.s.last = map[string]time.Time{}
	}
	r.s.last[connector] = at
	delete(r.s.stale, connector)
	r.s.mu.Unlock()
	r.s.Cell.Metrics.Set("sb_ingest_last_success_age_seconds", "Age of a connector's last successful sync (sb-ingest(7)).",
		map[string]string{"tenant": tenant, "connector": connector}, 0)
	r.record(r.s.Cell.Syncs.Succeeded(context.Background(), connector, at))
}

// record counts a sync record that couldn't be written: the console then shows the
// connector stale, which is the safe way to be wrong (SB-1607).
func (r recorder) record(err error) {
	if err != nil {
		r.s.Cell.Metrics.Add("sb_health_record_failures_total", "Sync records the console reads that couldn't be written (SB-1607).", nil, 1)
	}
}

func (r recorder) SyncFailed(tenant, connector string) {
	r.s.Cell.Metrics.Add("sb_ingest_sync_failures_total", "Failed syncs by connector (sb-ingest(7)).",
		map[string]string{"tenant": tenant, "connector": connector}, 1)
	r.record(r.s.Cell.Syncs.Failed(context.Background(), connector, r.s.now()))
}

// Reroute re-routes every connector's documents to the cell's current spaces, before the
// first sync after a start: spaces may have changed while the service was down.
func (s *Service) Reroute(ctx context.Context) map[string]error {
	out := map[string]error{}
	for _, k := range s.Kinds() {
		if err := s.Pipeline(k).Reroute(ctx); err != nil {
			out[k] = err
		}
	}
	return out
}

// SyncAll runs one sync of every connector, one after another; a failing connector never
// stops the others. It returns each failure by kind (the jobs service's sync job).
func (s *Service) SyncAll(ctx context.Context) map[string]error {
	out := map[string]error{}
	if _, err := s.EnsureEmbedder(ctx); err != nil {
		for _, k := range s.Kinds() {
			out[k] = err
		}
		return out
	}
	for _, k := range s.Kinds() {
		if err := s.SyncerFor(k).Once(ctx); err != nil {
			out[k] = err
		}
	}
	s.Watch(ctx)
	return out
}

// Watch updates each connector's freshness gauge and sends ingest_stale, once, for each
// connector whose last success (or the service's start, if it has none) is older than
// the 15-minute freshness target.
func (s *Service) Watch(ctx context.Context) {
	now := s.now()
	s.mu.Lock()
	if s.started.IsZero() {
		s.started = now
	}
	if s.stale == nil {
		s.stale = map[string]bool{}
	}
	var alert []string
	ages := map[string]float64{}
	for _, k := range s.Kinds() {
		since, ok := s.last[k]
		if !ok {
			since = s.started
		}
		age := now.Sub(since)
		ages[k] = age.Seconds()
		if age > ingest.FreshnessSLA && !s.stale[k] {
			s.stale[k] = true
			alert = append(alert, k)
		}
	}
	s.mu.Unlock()
	t := s.Cell.Config.Tenant
	for k, a := range ages {
		s.Cell.Metrics.Set("sb_ingest_last_success_age_seconds", "Age of a connector's last successful sync (sb-ingest(7)).",
			map[string]string{"tenant": t, "connector": k}, a)
	}
	for _, k := range alert {
		s.alert(ctx, canary.Alert{Severity: "sev2", Kind: "ingest_stale", Tenant: t, Where: "connector:" + k, Ref: "LAW-13"})
	}
}

func (s *Service) alert(ctx context.Context, a canary.Alert) {
	al := s.Alert
	if al == nil {
		al = s.Cell.Alerter
	}
	if al != nil {
		_ = al.Alert(ctx, a) // the alerter logs and counts its own failures
	}
}

// Backoff is the wait after n consecutive failures: the sync interval doubled each time,
// at most MaxBackoff.
func Backoff(n int) time.Duration {
	d := ingest.SyncEvery
	for i := 0; i < n && d < MaxBackoff; i++ {
		d *= 2
	}
	if d > MaxBackoff {
		d = MaxBackoff
	}
	return d
}

// Run re-routes, then syncs each connector in its own loop until ctx ends: every 60
// seconds plus jitter while it succeeds, backing off while it fails.
func (s *Service) Run(ctx context.Context) {
	if _, err := s.EnsureEmbedder(ctx); err != nil {
		s.Cell.Log.Error("ingest: the index couldn't be rebuilt for a new embedder", "err", err.Error())
	}
	for k, err := range s.Reroute(ctx) {
		s.Cell.Metrics.Add("sb_ingest_reroute_failures_total", "Re-routes at start that failed (SB-1503).", map[string]string{"connector": k}, 1)
		_ = err
	}
	var wg sync.WaitGroup
	for _, k := range s.Kinds() {
		wg.Add(1)
		go func(kind string) {
			defer wg.Done()
			s.loop(ctx, kind)
		}(k)
	}
	wg.Wait()
}

func (s *Service) loop(ctx context.Context, kind string) {
	sleep, jitter := s.Sleep, s.Jitter
	if sleep == nil {
		sleep = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	if jitter == nil {
		jitter = func() time.Duration { return time.Duration(rand.Int64N(int64(ingest.SyncJitterMax))) }
	}
	syncer := s.SyncerFor(kind)
	failures := 0
	for ctx.Err() == nil {
		wait := ingest.SyncEvery + jitter()
		if err := syncer.Once(ctx); err != nil {
			failures++
			wait = Backoff(failures)
		} else {
			failures = 0
		}
		s.Watch(ctx)
		if sleep(ctx, wait) != nil {
			return
		}
	}
}

// LegacyEmbedder is the embedder of an index built before the cell recorded one.
const LegacyEmbedder = "local-hashing-64"

// EnsureEmbedder rebuilds the index when the embedder changed since it was built (sb-ingest(7);
// SB-1710): it drops the index and starts every connector again from the beginning, so no
// two embedders' vectors are ever searched together. The embedder is recorded under state.
func (s *Service) EnsureEmbedder(ctx context.Context) (bool, error) {
	c := s.Cell
	want := c.Embedder.Model()
	file := filepath.Join(c.Config.State, "index", "embedder")
	have := LegacyEmbedder
	if b, err := os.ReadFile(file); err == nil {
		have = strings.TrimSpace(string(b))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if have == want {
		return false, nil
	}
	if err := c.Index.Drop(ctx, c.Config.Tenant); err != nil {
		return false, err
	}
	for _, k := range s.Kinds() {
		if err := c.Cursors.SaveCursor(ctx, c.Config.Tenant, k, ""); err != nil {
			return false, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return false, err
	}
	return true, os.WriteFile(file, []byte(want+"\n"), 0o600)
}
