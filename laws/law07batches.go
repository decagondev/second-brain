package laws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/ports"
)

// LAW-07 and LAW-05, batches collected later (SB-5701; sb-laws(7)): a batch kept between
// ticks holds no prompt, and a step gets a result only once the provider has returned one
// for its own request. Random steps, prompts and batch outcomes through the real gateway.

// BatchGateway is what the law checks: a model that keeps batches in store for later ticks.
type BatchGateway func(m ports.Model, store ports.BatchStore, now func() time.Time) ports.Model

// RealBatchGateway is the real one.
var RealBatchGateway BatchGateway = func(m ports.Model, store ports.BatchStore, now func() time.Time) ports.Model {
	return gateway.Gateway{Provider: "anthropic", Model: m, Secrets: passScrub{}, Batches: store, Now: now}
}

type passScrub struct{}

func (passScrub) Scrub(s string) string { return s }

// LawBatchStore is an in-memory ports.BatchStore.
type LawBatchStore map[string]ports.PendingBatch

func (m LawBatchStore) GetBatch(_ context.Context, k string) (ports.PendingBatch, bool, error) {
	b, ok := m[k]
	return b, ok, nil
}
func (m LawBatchStore) PutBatch(_ context.Context, b ports.PendingBatch) error {
	m[b.Key] = b
	return nil
}
func (m LawBatchStore) DeleteBatch(_ context.Context, k string) error { delete(m, k); return nil }
func (m LawBatchStore) ListBatches(context.Context) ([]ports.PendingBatch, error) {
	var out []ports.PendingBatch
	for _, b := range m {
		out = append(out, b)
	}
	return out, nil
}

// lawBatcher answers each request with its own text reversed, once its batch has ended.
type lawBatcher struct {
	r       *rand.Rand
	batches map[string]string // id: the answer
	n       int
}

func answerOf(req ports.ModelRequest) string {
	var b strings.Builder
	for _, m := range req.Messages {
		b.WriteString(m.Content)
	}
	return "answer to " + fmt.Sprint(len(b.String())) + ":" + strings.ToUpper(b.String())
}

func (m *lawBatcher) Complete(_ context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	return ports.ModelResponse{Text: answerOf(req)}, nil
}

func (m *lawBatcher) SubmitBatch(_ context.Context, req ports.ModelRequest) (string, error) {
	m.n++
	id := fmt.Sprintf("batch_%d", m.n)
	m.batches[id] = answerOf(req)
	return id, nil
}

func (m *lawBatcher) CollectBatch(_ context.Context, id string) (ports.ModelResponse, bool, error) {
	switch m.r.Intn(4) {
	case 0:
		return ports.ModelResponse{}, true, errors.New("failed")
	case 1:
		return ports.ModelResponse{}, false, nil
	}
	a, ok := m.batches[id]
	if !ok {
		return ports.ModelResponse{}, true, errors.New("no such batch")
	}
	return ports.ModelResponse{Text: a}, true, nil
}

func law07Batches(r *rand.Rand, mk BatchGateway) error {
	now := time.Date(2026, 11, 27, 18, 0, 0, 0, time.UTC)
	store := LawBatchStore{}
	m := mk(&lawBatcher{r: r, batches: map[string]string{}}, store, func() time.Time { return now })
	for i := 0; i < 40; i++ {
		step := fmt.Sprintf("brief-spc_%d", r.Intn(3))
		req := ports.ModelRequest{Batch: true, System: "SYSTEM-PROMPT", Messages: []ports.Message{{Role: "user", Content: fmt.Sprintf("PROMPT-%d passage", r.Intn(4))}}}
		now = now.Add(time.Duration(r.Intn(4)) * time.Hour)
		resp, err := m.Complete(ports.WithDeferredBatch(context.Background(), step), req)
		switch {
		case errors.Is(err, ports.ErrBatchPending):
		case err != nil:
			return fmt.Errorf("call %d: %v", i, err)
		case resp.Text != answerOf(req):
			return fmt.Errorf("call %d: step %s got a result that isn't its own request's", i, step)
		}
		raw, _ := json.Marshal(store)
		if strings.Contains(string(raw), "PROMPT-") || strings.Contains(string(raw), "SYSTEM-PROMPT") {
			return fmt.Errorf("call %d: a kept batch holds the prompt", i)
		}
	}
	return nil
}
