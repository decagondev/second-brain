// Package gateway is the model gateway (sb-agents(7), MODEL GATEWAY; SB-305, ADR-0019).
// Every model call goes through Complete: it refuses any request that contains a
// resolved secret (LAW-07), retries rate limits within the deadline, and meters tokens.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/ports"
)

// Scrubber is the log redactor: it replaces every registered secret in a string.
type Scrubber interface{ Scrub(s string) string }

// Counter is the metrics sink.
type Counter interface {
	Add(name, help string, labels map[string]string, v float64)
}

// ErrSecretInPrompt is a refused request: it contained a registered secret. The error
// never carries the secret or the prompt.
var ErrSecretInPrompt = errors.New("gateway: EPERM secret_in_prompt")

// ErrUnavailable is a call that failed or ran out of time (EUNAVAIL).
var ErrUnavailable = errors.New("gateway: EUNAVAIL")

// Gateway wraps one provider adapter for one tenant.
type Gateway struct {
	Tenant   string
	Provider string // "openaicompat", "anthropic", "groq" or "openrouter", from the release
	ModelID  string // from the release
	Model    ports.Model
	Secrets  Scrubber
	Metrics  Counter
	Retries  int           // rate-limit retries; 0 means 3
	Backoff  time.Duration // first backoff; 0 means 200ms
	Sleep    func(context.Context, time.Duration) error
	// OnSecret is told when a request held a secret (the Sev-1 secret_leak alert); the
	// request is refused either way.
	OnSecret func(ctx context.Context)
	// Budget, when set, is checked before every call and metered after (SB-609).
	Budget *budget.Guard
	// MainPrice is the main tier's price, for the small tier's gateway: what a call on it
	// saved (SB-5606). nil saves nothing.
	MainPrice *budget.Price
	// BatchPoll is how long to wait between looks at a batch, 0 for 2s, and BatchWait the
	// longest a batched call waits before it runs as a normal one, 0 for 10 minutes (SB-5604).
	BatchPoll, BatchWait time.Duration
}

// BatchRate is what a batched call costs against its list price (SB-5604).
const BatchRate = 0.5

// ErrBudget is a call refused by the budget or rate limit (EBUDGET).
var ErrBudget = budget.ErrBudget

// Complete sends req after the secret check. The request's Model is always the release's.
func (g Gateway) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	if g.Secrets == nil {
		return ports.ModelResponse{}, errors.New("gateway: no secret scanner; refusing to call a model")
	}
	if containsSecret(g.Secrets, req) {
		g.count("sb_model_refusals_total", "Model requests refused by the gateway.", map[string]string{"reason": "secret_in_prompt"}, 1)
		if g.OnSecret != nil {
			g.OnSecret(ctx)
		}
		return ports.ModelResponse{}, ErrSecretInPrompt
	}
	if g.Budget != nil {
		max := req.MaxTokens
		if max == 0 {
			max = 1024
		}
		if err := g.Budget.Allow(ctx, req.User, max); err != nil {
			if errors.Is(err, budget.ErrBudget) {
				return ports.ModelResponse{}, err
			}
			return ports.ModelResponse{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
	}
	req.Model = g.ModelID
	retries, backoff := g.Retries, g.Backoff
	if retries == 0 {
		retries = 3
	}
	if backoff == 0 {
		backoff = 200 * time.Millisecond
	}
	sleep := g.Sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	if req.Batch { // SB-5604: nobody is waiting
		if resp, ok := g.batched(ctx, req); ok {
			g.spent(ctx, req, resp.Usage, true)
			return resp, nil
		}
	}
	for attempt := 0; ; attempt++ {
		resp, err := g.Model.Complete(ctx, req)
		if err == nil {
			g.spent(ctx, req, resp.Usage, false)
			return resp, nil
		}
		if !errors.Is(err, ports.ErrRateLimited) || attempt >= retries {
			return ports.ModelResponse{}, fmt.Errorf("%w: %s: %v", ErrUnavailable, g.Provider, err)
		}
		wait := backoff << attempt
		if dl, ok := ctx.Deadline(); ok && time.Until(dl) < wait {
			return ports.ModelResponse{}, fmt.Errorf("%w: %s: rate limited past the deadline", ErrUnavailable, g.Provider)
		}
		if err := sleep(ctx, wait); err != nil {
			return ports.ModelResponse{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
	}
}

func containsSecret(s Scrubber, req ports.ModelRequest) bool {
	parts := []string{req.System}
	for _, m := range req.Messages {
		parts = append(parts, m.Content)
	}
	for _, p := range parts {
		if s.Scrub(p) != p {
			return true
		}
	}
	// A secret split across parts would still be one string at the provider.
	all := strings.Join(parts, "")
	return s.Scrub(all) != all
}

func (g Gateway) meter(u ports.Usage) {
	l := map[string]string{"tenant": g.Tenant, "provider": g.Provider, "model": g.ModelID}
	in, out := map[string]string{"direction": "input"}, map[string]string{"direction": "output"}
	for k, v := range l {
		in[k], out[k] = v, v
	}
	g.count("sb_model_tokens_total", "Model tokens by tenant, provider, model and direction.", in, float64(u.Input))
	g.count("sb_model_tokens_total", "Model tokens by tenant, provider, model and direction.", out, float64(u.Output))
}

func (g Gateway) count(name, help string, labels map[string]string, v float64) {
	if g.Metrics != nil {
		g.Metrics.Add(name, help, labels, v)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// spent meters one call: the tokens, and its dollars at the cached and batch prices, with
// what those and the economy tier saved against the list price (SB-5604 to SB-5606).
func (g Gateway) spent(ctx context.Context, req ports.ModelRequest, u ports.Usage, batched bool) {
	g.meter(u)
	if g.Budget == nil {
		return
	}
	p := g.Budget.Price
	list := p.Cost(u.Input, 0, u.Output)
	paid := p.Cost(u.Input, u.Cached, u.Output)
	t := budget.TallyOf(ctx)
	t.Save("cache", list-paid)
	if batched {
		t.Save("batch", paid*(1-BatchRate))
		paid *= BatchRate
	}
	if g.MainPrice != nil {
		t.Save("tier", g.MainPrice.Cost(u.Input, 0, u.Output)-list)
	}
	_ = g.Budget.RecordCost(ctx, req.Space, req.User, paid) // SB-5003
	t.AddUSD(paid)                                          // SB-4201
}

// batched sends req through the adapter's batch API and waits for it, looking every
// BatchPoll until ctx ends; a failed batch is tried once more, then false runs it as a
// normal call. An adapter without one is false at once.
func (g Gateway) batched(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, bool) {
	b, ok := g.Model.(ports.Batcher)
	if !ok {
		return ports.ModelResponse{}, false
	}
	poll := g.BatchPoll
	if poll == 0 {
		poll = 2 * time.Second
	}
	sleep := g.Sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	wait := g.BatchWait
	if wait == 0 {
		wait = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	for try := 0; try < 2; try++ {
		id, err := b.SubmitBatch(ctx, req)
		if err != nil {
			continue
		}
		for {
			resp, done, err := b.CollectBatch(ctx, id)
			if done && err == nil {
				g.count("sb_model_batched_total", "Model calls sent through a batch API (SB-5604).", map[string]string{"tenant": g.Tenant, "provider": g.Provider}, 1)
				return resp, true
			}
			if done || err != nil {
				break
			}
			if sleep(ctx, poll) != nil {
				return ports.ModelResponse{}, false
			}
		}
	}
	return ports.ModelResponse{}, false
}
