package gateway

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// storm answers 429 until its limit count is used up, and records who it served in order.
type storm struct {
	mu      sync.Mutex
	limits  int
	served  []string
	inCall  int
	maxCall int
}

func (s *storm) Complete(_ context.Context, r ports.ModelRequest) (ports.ModelResponse, error) {
	s.mu.Lock()
	if s.limits > 0 {
		s.limits--
		s.mu.Unlock()
		return ports.ModelResponse{}, ports.ErrRateLimited
	}
	s.inCall++
	if s.inCall > s.maxCall {
		s.maxCall = s.inCall
	}
	s.mu.Unlock()
	time.Sleep(2 * time.Millisecond)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inCall--
	who := r.Space
	if r.Batch {
		who = "scheduled:" + r.Space
	}
	s.served = append(s.served, who)
	return ports.ModelResponse{Text: "ok"}, nil
}

// SB-5703: in a 429 storm calls take turns: asks before scheduled calls, spaces in turn,
// and a scheduled step that can come back later goes to the next tick.
func TestPeopleFirstWhenBusy(t *testing.T) {
	m := &storm{limits: 1}
	busy := &Busy{MaxWait: 2 * time.Second}
	g := Gateway{Provider: "anthropic", Model: m, Secrets: scrub{}, Busy: busy, Backoff: 20 * time.Millisecond}
	// The first call meets the limit and starts the storm.
	if _, err := g.Complete(context.Background(), ports.ModelRequest{Space: "spc_ops"}); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.limits, m.served = 1, nil
	m.mu.Unlock()
	// Start the storm again, then queue: a scheduled call, five asks from ops and one from finance.
	_, _ = g.Complete(context.Background(), ports.ModelRequest{Space: "spc_ops"})
	m.mu.Lock()
	m.served = nil
	m.mu.Unlock()
	busy.mu.Lock()
	busy.storm, busy.until = true, time.Now().Add(50*time.Millisecond)
	busy.mu.Unlock()
	var wg sync.WaitGroup
	run := func(req ports.ModelRequest, ctx context.Context) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := g.Complete(ctx, req); err != nil {
				t.Errorf("%+v: %v", req, err)
			}
		}()
		time.Sleep(3 * time.Millisecond) // queue in this order
	}
	run(ports.ModelRequest{Space: "spc_ops", Batch: true}, context.Background())
	for i := 0; i < 4; i++ {
		run(ports.ModelRequest{Space: "spc_ops"}, context.Background())
	}
	run(ports.ModelRequest{Space: "spc_finance"}, context.Background())
	if _, err := g.Complete(ports.WithDeferredBatch(context.Background(), "brief-spc_ops"), ports.ModelRequest{Space: "spc_ops", Batch: true}); !errors.Is(err, ports.ErrBatchPending) {
		t.Fatalf("a scheduled step waited: %v", err)
	}
	wg.Wait()
	got := m.served
	if len(got) != 6 || got[len(got)-1] != "scheduled:spc_ops" || got[1] != "spc_finance" || m.maxCall != 1 {
		t.Fatalf("served %v at most %d at once", got, m.maxCall)
	}
	c := busy.Counts()
	if c["spc_ops"].Limited < 1 || c["spc_ops"].Waited < 5 || c["spc_finance"].Waited != 1 {
		t.Fatalf("counts %+v", c)
	}
}

// SB-5703: an ask that would wait past MaxWait is ErrBusy (passages), and outside a storm
// calls run side by side.
func TestAskPastTheWaitIsBusy(t *testing.T) {
	m := &storm{limits: 1000}
	g := Gateway{Provider: "anthropic", Model: m, Secrets: scrub{}, Busy: &Busy{MaxWait: 60 * time.Millisecond}, Backoff: 10 * time.Millisecond}
	if _, err := g.Complete(context.Background(), ports.ModelRequest{}); !errors.Is(err, ErrBusy) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
	calm := &storm{}
	g = Gateway{Provider: "anthropic", Model: calm, Secrets: scrub{}, Busy: &Busy{}}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = g.Complete(context.Background(), ports.ModelRequest{}) }()
	}
	wg.Wait()
	if calm.maxCall < 2 {
		t.Fatalf("calls were serialised outside a storm: %d", calm.maxCall)
	}
}
