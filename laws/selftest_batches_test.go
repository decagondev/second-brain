package laws

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// naiveBatches keys a step's batch by the step alone, and may keep the prompt as its caller.
type naiveBatches struct {
	m     ports.Model
	store ports.BatchStore
	leak  bool
}

func (n naiveBatches) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	b := n.m.(ports.Batcher)
	step := ports.DeferredStep(ctx)
	rec, found, _ := n.store.GetBatch(ctx, step)
	if !found {
		id, _ := b.SubmitBatch(ctx, req)
		rec = ports.PendingBatch{Key: step, BatchID: id, Step: step}
		if n.leak {
			rec.Caller = req.Messages[0].Content
		}
		_ = n.store.PutBatch(ctx, rec)
		return ports.ModelResponse{}, fmt.Errorf("%w", ports.ErrBatchPending)
	}
	resp, done, err := b.CollectBatch(ctx, rec.BatchID)
	if done {
		_ = n.store.DeleteBatch(ctx, step)
		if err == nil {
			return resp, nil
		}
	}
	return ports.ModelResponse{}, fmt.Errorf("%w", ports.ErrBatchPending)
}

// SB-5701: a gateway that keeps the prompt with a batch is caught (LAW-07), and so is one
// that hands a step the result of an earlier, different request (LAW-05).
func TestSelfBatchesAreCaught(t *testing.T) {
	for _, leak := range []bool{true, false} {
		failsSomewhere(t, fmt.Sprintf("batches keyed by step, leaking %v", leak), func(r *rand.Rand) error {
			return law07Batches(r, func(m ports.Model, s ports.BatchStore, _ func() time.Time) ports.Model {
				return naiveBatches{m: m, store: s, leak: leak}
			})
		})
	}
}
