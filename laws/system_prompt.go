package laws

import (
	"context"
	"errors"
	"time"

	"example.com/second-brain/internal/adapters/model/modeltest"
	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

// The real LAW-07 prompt world (SB-305): the model gateway in front of an
// OpenAI-compatible adapter talking to the recorded fake provider.
func init() {
	NewPromptWorld = func(string) PromptWorld { return newGatewayWorld(false) }
}

type gatewayWorld struct {
	red  *obs.Redactor
	fake *modeltest.Fake
	gw   ports.Model
}

func newGatewayWorld(unguarded bool) *gatewayWorld {
	w := &gatewayWorld{red: obs.NewRedactor(), fake: modeltest.New("sk-law07-key", `{"claims":[]}`)}
	adapter := openaicompat.Client{BaseURL: w.fake.URL, Key: func(context.Context) (string, error) { return "sk-law07-key", nil }}
	if unguarded {
		w.gw = adapter // a mutation: no gateway in front of the provider
	} else {
		w.gw = gateway.Gateway{Tenant: "acme-logistics", Provider: "openaicompat", ModelID: "m-1", Model: adapter, Secrets: w.red, Sleep: func(context.Context, time.Duration) error { return nil }}
	}
	return w
}

func (w *gatewayWorld) Resolve(v string) error { w.red.Register(v); return nil }

func (w *gatewayWorld) Send(system, user string) (bool, error) {
	_, err := w.gw.Complete(context.Background(), ports.ModelRequest{System: system, Messages: []ports.Message{{Role: "user", Content: user}}})
	if errors.Is(err, gateway.ErrSecretInPrompt) {
		return false, nil
	}
	return err == nil, err
}

func (w *gatewayWorld) Received() string { return w.fake.Received() }
