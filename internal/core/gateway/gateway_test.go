package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

type scrub []string

func (s scrub) Scrub(x string) string {
	for _, v := range s {
		x = strings.ReplaceAll(x, v, "[redacted]")
	}
	return x
}

type counter map[string]float64

func (c counter) Add(name, _ string, l map[string]string, v float64) {
	c[name+"/"+l["direction"]+l["reason"]] += v
}

type model struct {
	calls   int
	limited int
	err     error
	got     []ports.ModelRequest
}

func (m *model) Complete(_ context.Context, r ports.ModelRequest) (ports.ModelResponse, error) {
	m.calls++
	m.got = append(m.got, r)
	if m.limited > 0 {
		m.limited--
		return ports.ModelResponse{}, ports.ErrRateLimited
	}
	if m.err != nil {
		return ports.ModelResponse{}, m.err
	}
	return ports.ModelResponse{Text: "ok", Usage: ports.Usage{Input: 10, Output: 3}}, nil
}

func noSleep(context.Context, time.Duration) error { return nil }

func TestSecretRefusedWherever(t *testing.T) {
	for _, req := range []ports.ModelRequest{
		{System: "use sk-live-123456"},
		{Messages: []ports.Message{{Role: "user", Content: "x sk-live-123456 y"}}},
		{System: "ends sk-live-", Messages: []ports.Message{{Role: "user", Content: "123456 split"}}},
	} {
		m, c := &model{}, counter{}
		g := Gateway{Tenant: "acme-logistics", Provider: "groq", ModelID: "m", Model: m, Secrets: scrub{"sk-live-123456"}, Metrics: c}
		if _, err := g.Complete(context.Background(), req); !errors.Is(err, ErrSecretInPrompt) {
			t.Fatalf("got %v, want ErrSecretInPrompt", err)
		}
		if m.calls != 0 {
			t.Fatal("a request with a secret reached the provider")
		}
		if c["sb_model_refusals_total/secret_in_prompt"] != 1 {
			t.Errorf("refusal not counted: %v", c)
		}
	}
}

func TestNoScannerNoCall(t *testing.T) {
	m := &model{}
	if _, err := (Gateway{Model: m}).Complete(context.Background(), ports.ModelRequest{}); err == nil || m.calls != 0 {
		t.Fatal("a gateway without a secret scanner called the model")
	}
}

func TestReleaseModelAndMetering(t *testing.T) {
	m, c := &model{}, counter{}
	g := Gateway{Tenant: "acme-logistics", Provider: "anthropic", ModelID: "release-model", Model: m, Secrets: scrub{}, Metrics: c}
	if _, err := g.Complete(context.Background(), ports.ModelRequest{Model: "from-request"}); err != nil {
		t.Fatal(err)
	}
	if m.got[0].Model != "release-model" {
		t.Errorf("model %q, want the release's", m.got[0].Model)
	}
	if c["sb_model_tokens_total/input"] != 10 || c["sb_model_tokens_total/output"] != 3 {
		t.Errorf("metering: %v", c)
	}
}

func TestRetriesRateLimitsThenGivesUp(t *testing.T) {
	m := &model{limited: 2}
	g := Gateway{Model: m, Secrets: scrub{}, Sleep: noSleep}
	if _, err := g.Complete(context.Background(), ports.ModelRequest{}); err != nil || m.calls != 3 {
		t.Fatalf("err %v after %d calls; want success on the third", err, m.calls)
	}
	m = &model{limited: 10}
	g.Model = m
	if _, err := g.Complete(context.Background(), ports.ModelRequest{}); !errors.Is(err, ErrUnavailable) || m.calls != 4 {
		t.Fatalf("err %v after %d calls; want EUNAVAIL after 4", err, m.calls)
	}
	m = &model{err: ports.ErrModelDown}
	g.Model = m
	if _, err := g.Complete(context.Background(), ports.ModelRequest{}); !errors.Is(err, ErrUnavailable) || m.calls != 1 {
		t.Fatalf("non-rate-limit errors are not retried: %v, %d calls", err, m.calls)
	}
}

func TestDeadlineStopsRetries(t *testing.T) {
	m := &model{limited: 10}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	g := Gateway{Model: m, Secrets: scrub{}, Backoff: time.Second}
	start := time.Now()
	if _, err := g.Complete(ctx, ports.ModelRequest{}); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if time.Since(start) > 500*time.Millisecond || m.calls != 1 {
		t.Fatalf("waited past the deadline (%d calls)", m.calls)
	}
}
