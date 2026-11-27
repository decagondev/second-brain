package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Collect batches later (sb-runtime(7), Batching; SB-5701): a scheduled step's batchable
// call is submitted on one notify tick and its batch kept in the BatchStore, never its
// prompt. The step answers ErrBatchPending and runs again on each later tick, rebuilt from
// its source; the tick that finds the batch ended takes its result. A batch not back by
// 06:00 on the next working morning is cancelled and the step's call runs normally, and a
// failed batch is submitted once more, then the call runs normally.

// MorningHour is the hour of the working morning a batch must be back by.
const MorningHour = 6

// BatchDeadline is 06:00 in zone on the first Monday-to-Friday morning after submitted.
func BatchDeadline(submitted time.Time, zone *time.Location) time.Time {
	if zone == nil {
		zone = time.UTC
	}
	t := submitted.In(zone)
	d := time.Date(t.Year(), t.Month(), t.Day(), MorningHour, 0, 0, 0, zone)
	for !d.After(t) || d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d.UTC()
}

// batchKey is the hash of a step and one request: the same step asking the same thing finds
// its batch again, and the store holds no prompt.
func batchKey(step string, req ports.ModelRequest) string {
	h := sha256.New()
	for _, p := range []string{step, req.Model, req.System, strconv.FormatBool(req.JSON), strconv.Itoa(req.MaxTokens)} {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	for _, m := range req.Messages {
		h.Write([]byte(m.Role))
		h.Write([]byte{0})
		h.Write([]byte(m.Content))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func (g Gateway) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}

// pending is the error a step gets while its batch runs.
func pending(id string) error { return fmt.Errorf("%w: %s", ports.ErrBatchPending, id) }

// deferred submits req for step, or collects the batch an earlier tick submitted: ok with
// the result once it has ended, ErrBatchPending while it runs, and false (run normally) when
// the adapter has no batch API, its batch failed twice or the morning deadline passed.
func (g Gateway) deferred(ctx context.Context, req ports.ModelRequest, step string) (ports.ModelResponse, bool, error) {
	b, ok := g.Model.(ports.Batcher)
	if !ok {
		return ports.ModelResponse{}, false, nil
	}
	key := batchKey(step, req)
	rec, found, err := g.Batches.GetBatch(ctx, key)
	if err != nil {
		return ports.ModelResponse{}, false, fmt.Errorf("%w: batches: %v", ErrUnavailable, err)
	}
	now := g.now()
	if !found {
		return g.submit(ctx, b, req, ports.PendingBatch{Key: key, Provider: g.Provider, Caller: req.User, Step: step,
			Submitted: now.UTC(), Deadline: BatchDeadline(now, g.Zone)})
	}
	resp, done, err := b.CollectBatch(ctx, rec.BatchID)
	switch {
	case done && err == nil:
		if err := g.Batches.DeleteBatch(ctx, key); err != nil {
			return ports.ModelResponse{}, false, fmt.Errorf("%w: batches: %v", ErrUnavailable, err)
		}
		g.count("sb_model_batched_total", "Model calls sent through a batch API (SB-5604).", map[string]string{"tenant": g.Tenant, "provider": g.Provider}, 1)
		return resp, true, nil
	case done && rec.Tries < 2: // failed: once more
		return g.submit(ctx, b, req, rec)
	case done, !now.Before(rec.Deadline): // failed twice, or not back by the morning
		g.cancel(ctx, b, rec)
		return ports.ModelResponse{}, false, g.Batches.DeleteBatch(ctx, key)
	default: // still running, or the provider didn't answer this tick
		return ports.ModelResponse{}, false, pending(rec.BatchID)
	}
}

func (g Gateway) submit(ctx context.Context, b ports.Batcher, req ports.ModelRequest, rec ports.PendingBatch) (ports.ModelResponse, bool, error) {
	id, err := b.SubmitBatch(ctx, req)
	if err != nil { // no batch now: run it normally
		if rec.BatchID != "" {
			_ = g.Batches.DeleteBatch(ctx, rec.Key)
		}
		return ports.ModelResponse{}, false, nil //nolint:nilerr // a provider that won't take a batch runs the call normally
	}
	rec.BatchID, rec.Tries = id, rec.Tries+1
	if err := g.Batches.PutBatch(ctx, rec); err != nil {
		g.cancel(ctx, b, rec)
		return ports.ModelResponse{}, false, nil //nolint:nilerr // a batch that can't be kept runs the call normally
	}
	return ports.ModelResponse{}, false, pending(id)
}

func (g Gateway) cancel(ctx context.Context, b ports.Batcher, rec ports.PendingBatch) {
	if c, ok := b.(ports.BatchCanceller); ok && rec.BatchID != "" {
		_ = c.CancelBatch(ctx, rec.BatchID)
	}
}

// SweepBatches cancels and forgets every batch past its morning deadline that no step came
// back for (its request changed, or its step no longer runs), of this gateway's provider.
// The notify job calls it on each tick.
func (g Gateway) SweepBatches(ctx context.Context) error {
	if g.Batches == nil {
		return nil
	}
	all, err := g.Batches.ListBatches(ctx)
	if err != nil {
		return err
	}
	b, _ := g.Model.(ports.Batcher)
	now := g.now()
	for _, rec := range all {
		if now.Before(rec.Deadline) || rec.Provider != g.Provider {
			continue
		}
		if b != nil {
			g.cancel(ctx, b, rec)
		}
		if err := g.Batches.DeleteBatch(ctx, rec.Key); err != nil {
			return err
		}
	}
	return nil
}

// Sweeper is a model whose gateways keep batches for later ticks.
type Sweeper interface {
	SweepBatches(ctx context.Context) error
}

// SweepBatches sweeps both models' batches (SB-5701).
func (f *Fallback) SweepBatches(ctx context.Context) error {
	for _, m := range []ports.Model{f.Primary, f.Secondary} {
		if s, ok := m.(Sweeper); ok {
			if err := s.SweepBatches(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// ErrRegion is a call refused because its provider is outside model.region, or declares no
// region (SB-5705, LAW-33). It is unavailable, so an ask answers with passages.
var ErrRegion = fmt.Errorf("%w: the provider is outside model.region", ErrUnavailable)

// InRegion says a provider of region may take a call when calls must stay in allowed: always
// with no allowed region, for a model on the cell's own machine, or when the two match. A
// provider that declares no region is never in one (LAW-03).
func InRegion(region, allowed string, local bool) bool {
	return allowed == "" || local || (region != "" && region == allowed)
}
