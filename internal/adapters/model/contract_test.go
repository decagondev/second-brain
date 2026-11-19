package model_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/model/anthropic"
	"example.com/second-brain/internal/adapters/model/groq"
	"example.com/second-brain/internal/adapters/model/modeltest"
	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/adapters/model/openrouter"
	"example.com/second-brain/internal/core/ports"
)

const key = "sk-test-model-key"

func adapters(base string) map[string]ports.Model {
	k := func(context.Context) (string, error) { return key, nil }
	return map[string]ports.Model{
		"openaicompat": openaicompat.Client{BaseURL: base, Key: k},
		"groq":         groq.Client{BaseURL: base, Key: k},
		"openrouter":   openrouter.Client{BaseURL: base, Key: k, AppName: "second-brain"},
		"anthropic":    anthropic.Client{BaseURL: base, Key: k},
	}
}

// The shared model contract (ADR-0019): every adapter passes it.
func TestContract(t *testing.T) {
	reply := `{"claims":[{"text":"Refunds up to 500 need no approval.","cites":["C1"]}]}`
	for name := range adapters("") {
		t.Run(name, func(t *testing.T) {
			f := modeltest.New(key, reply)
			defer f.Close()
			m := adapters(f.URL)[name]
			ctx := context.Background()
			req := ports.ModelRequest{Model: "m-1", System: "sys prompt", Messages: []ports.Message{{Role: "user", Content: "What is the refund limit?"}}, JSON: true}

			resp, err := m.Complete(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Text != reply || resp.Usage.Input == 0 || resp.Usage.Output == 0 {
				t.Fatalf("response %+v", resp)
			}
			body := f.Bodies()[0]
			for _, want := range []string{`"m-1"`, "sys prompt", "What is the refund limit?"} {
				if !strings.Contains(body, want) {
					t.Errorf("request body lacks %s: %s", want, body)
				}
			}

			f.RateLimit(1)
			if _, err := m.Complete(ctx, req); !errors.Is(err, ports.ErrRateLimited) {
				t.Errorf("429: got %v, want ErrRateLimited", err)
			}
			f.Fail()
			if _, err := m.Complete(ctx, req); !errors.Is(err, ports.ErrModelDown) {
				t.Errorf("500: got %v, want ErrModelDown", err)
			}

			wrong := map[string]ports.Model{
				"openaicompat": openaicompat.Client{BaseURL: f.URL, Key: func(context.Context) (string, error) { return "nope", nil }},
				"groq":         groq.Client{BaseURL: f.URL, Key: func(context.Context) (string, error) { return "nope", nil }},
				"openrouter":   openrouter.Client{BaseURL: f.URL, Key: func(context.Context) (string, error) { return "nope", nil }},
				"anthropic":    anthropic.Client{BaseURL: f.URL, Key: func(context.Context) (string, error) { return "nope", nil }},
			}[name]
			_, err = wrong.Complete(ctx, req)
			if !errors.Is(err, ports.ErrModelDown) || strings.Contains(err.Error(), "nope") {
				t.Errorf("bad key: got %v (must be ErrModelDown and not echo the key)", err)
			}
		})
	}
}

func TestOpenRouterPinsTheProvider(t *testing.T) {
	f := modeltest.New(key, "ok")
	defer f.Close()
	m := openrouter.Client{BaseURL: f.URL, Key: func(context.Context) (string, error) { return key, nil }, AppName: "second-brain"}
	if _, err := m.Complete(context.Background(), ports.ModelRequest{Model: "vendor/m", Messages: []ports.Message{{Role: "user", Content: "hi"}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.Bodies()[0], `"allow_fallbacks":false`) {
		t.Errorf("OpenRouter fallback routing not turned off: %s", f.Bodies()[0])
	}
	if f.Headers()[0].Get("X-Title") != "second-brain" {
		t.Error("X-Title header missing")
	}
}

func TestAnthropicShape(t *testing.T) {
	f := modeltest.New(key, "ok")
	defer f.Close()
	m := anthropic.Client{BaseURL: f.URL, Key: func(context.Context) (string, error) { return key, nil }}
	if _, err := m.Complete(context.Background(), ports.ModelRequest{Model: "m", System: "S", Messages: []ports.Message{{Role: "user", Content: "hi"}}}); err != nil {
		t.Fatal(err)
	}
	b := f.Bodies()[0]
	if !strings.Contains(b, `"system":"S"`) || !strings.Contains(b, `"max_tokens":1024`) || strings.Contains(b, `"role":"system"`) {
		t.Errorf("Anthropic body shape: %s", b)
	}
	if f.Headers()[0].Get("anthropic-version") != anthropic.Version {
		t.Error("anthropic-version header missing")
	}
}
