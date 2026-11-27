package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// memBatches is an in-memory BatchStore; reopening it is a restart (the records are kept).
type memBatches map[string]ports.PendingBatch

func (m memBatches) GetBatch(_ context.Context, k string) (ports.PendingBatch, bool, error) {
	b, ok := m[k]
	return b, ok, nil
}
func (m memBatches) PutBatch(_ context.Context, b ports.PendingBatch) error { m[b.Key] = b; return nil }
func (m memBatches) DeleteBatch(_ context.Context, k string) error          { delete(m, k); return nil }
func (m memBatches) ListBatches(context.Context) ([]ports.PendingBatch, error) {
	var out []ports.PendingBatch
	for _, b := range m {
		out = append(out, b)
	}
	return out, nil
}

// slowBatcher's batch ends only once ended is set, and can fail or be cancelled.
type slowBatcher struct {
	model
	submitted, cancelled int
	ended, failed        bool
}

func (b *slowBatcher) SubmitBatch(context.Context, ports.ModelRequest) (string, error) {
	b.submitted++
	return "msgbatch_1", nil
}

func (b *slowBatcher) CollectBatch(context.Context, string) (ports.ModelResponse, bool, error) {
	switch {
	case b.failed:
		return ports.ModelResponse{}, true, ErrUnavailable
	case b.ended:
		return ports.ModelResponse{Text: "later", Usage: ports.Usage{Input: 10, Output: 10}}, true, nil
	}
	return ports.ModelResponse{}, false, nil
}

func (b *slowBatcher) CancelBatch(context.Context, string) error { b.cancelled++; return nil }

// SB-5701: a scheduled step's batch is submitted on one tick and collected on a later one,
// across a restart; the store holds no prompt; a batch not back by the working morning is
// cancelled and the call runs normally; a failed batch is submitted once more.
func TestBatchCollectedOnALaterTick(t *testing.T) {
	now := time.Date(2026, 11, 27, 22, 0, 0, 0, time.UTC) // a Friday night
	store := memBatches{}
	b := &slowBatcher{}
	gw := func() Gateway {
		return Gateway{Provider: "anthropic", Model: b, Secrets: scrub{}, Batches: store, Now: func() time.Time { return now }}
	}
	req := ports.ModelRequest{Batch: true, System: "the brief prompt", Messages: []ports.Message{{Role: "user", Content: "PASSAGE-TEXT what changed?"}}}
	step := ports.WithDeferredBatch(context.Background(), "brief-spc_ops")

	if _, err := gw().Complete(step, req); !errors.Is(err, ports.ErrBatchPending) || b.submitted != 1 || b.calls != 0 {
		t.Fatalf("first tick: %v submitted %d calls %d", err, b.submitted, b.calls)
	}
	raw, _ := json.Marshal(store)
	if strings.Contains(string(raw), "PASSAGE-TEXT") || strings.Contains(string(raw), "brief prompt") {
		t.Fatalf("a prompt is in the batch store: %s", raw)
	}
	for _, rec := range store {
		if rec.Step != "brief-spc_ops" || rec.BatchID != "msgbatch_1" || rec.Deadline != time.Date(2026, 11, 30, 6, 0, 0, 0, time.UTC) {
			t.Fatalf("record %+v", rec)
		}
	}
	now = now.Add(time.Hour)
	if _, err := gw().Complete(step, req); !errors.Is(err, ports.ErrBatchPending) || b.submitted != 1 {
		t.Fatalf("still running: %v submitted %d", err, b.submitted)
	}
	b.ended = true // a restart: a new gateway over the same store
	resp, err := gw().Complete(step, req)
	if err != nil || resp.Text != "later" || len(store) != 0 || b.calls != 0 {
		t.Fatalf("collected: %+v %v store %d calls %d", resp, err, len(store), b.calls)
	}

	// Not back by Monday 06:00: cancelled at the provider, and the call runs normally.
	b.ended = false
	if _, err := gw().Complete(step, req); !errors.Is(err, ports.ErrBatchPending) {
		t.Fatal(err)
	}
	now = time.Date(2026, 11, 30, 6, 0, 0, 0, time.UTC)
	if resp, err := gw().Complete(step, req); err != nil || resp.Text != "ok" || b.cancelled != 1 || b.calls != 1 || len(store) != 0 {
		t.Fatalf("deadline: %+v %v cancelled %d calls %d", resp, err, b.cancelled, b.calls)
	}

	// A failed batch is submitted once more, then runs normally.
	b.submitted, b.failed = 0, true
	_, _ = gw().Complete(step, req)
	if _, err := gw().Complete(step, req); !errors.Is(err, ports.ErrBatchPending) || b.submitted != 2 {
		t.Fatalf("retry: %v submitted %d", err, b.submitted)
	}
	if resp, err := gw().Complete(step, req); err != nil || resp.Text != "ok" || len(store) != 0 {
		t.Fatalf("second failure: %+v %v", resp, err)
	}

	// Without a step, a batchable call waits within the run as before (SB-5604); an ask
	// never batches.
	b.failed, b.ended, b.submitted = false, true, 0
	g := gw()
	g.Sleep = func(context.Context, time.Duration) error { return nil }
	if resp, err := g.Complete(context.Background(), req); err != nil || resp.Text != "later" || len(store) != 0 {
		t.Fatalf("in-run: %+v %v", resp, err)
	}
	if _, err := gw().Complete(step, ports.ModelRequest{}); err != nil || b.submitted != 1 {
		t.Fatalf("an ask batched: %v", err)
	}
}

// SB-5701: a batch whose step never came back is swept at its deadline.
func TestSweepBatches(t *testing.T) {
	now := time.Date(2026, 11, 30, 7, 0, 0, 0, time.UTC)
	store := memBatches{"k1": {Key: "k1", BatchID: "x", Provider: "anthropic", Deadline: now.Add(-time.Hour)},
		"k2": {Key: "k2", BatchID: "y", Provider: "anthropic", Deadline: now.Add(time.Hour)},
		"k3": {Key: "k3", BatchID: "z", Provider: "openaicompat", Deadline: now.Add(-time.Hour)}}
	b := &slowBatcher{}
	g := Gateway{Provider: "anthropic", Model: b, Secrets: scrub{}, Batches: store, Now: func() time.Time { return now }}
	if err := g.SweepBatches(context.Background()); err != nil || b.cancelled != 1 || len(store) != 2 || store["k2"].Key == "" || store["k3"].Key == "" {
		t.Fatalf("sweep: %v cancelled %d left %v", err, b.cancelled, store)
	}
}

func TestBatchDeadline(t *testing.T) {
	for in, want := range map[string]string{
		"2026-11-27T22:00:00Z": "2026-11-30T06:00:00Z", // Friday night: Monday
		"2026-11-24T02:00:00Z": "2026-11-24T06:00:00Z", // Tuesday before six: the same morning
		"2026-11-24T06:00:00Z": "2026-11-25T06:00:00Z",
	} {
		at, _ := time.Parse(time.RFC3339, in)
		if got := BatchDeadline(at, nil).Format(time.RFC3339); got != want {
			t.Errorf("%s: %s, want %s", in, got, want)
		}
	}
}
