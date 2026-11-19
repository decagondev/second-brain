// Package openrouter is the OpenRouter model adapter: the OpenAI-compatible wire format
// at OpenRouter's base URL, with its attribution headers (ADR-0019). Model ids are
// "vendor/model", and the release pins one. OpenRouter's fallback routing is turned off
// so a provider is never swapped silently.
package openrouter

import (
	"context"
	"net/http"

	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/core/ports"
)

// BaseURL is OpenRouter's endpoint.
const BaseURL = "https://openrouter.ai/api/v1"

// Client calls OpenRouter.
type Client struct {
	BaseURL string
	Key     func(ctx context.Context) (string, error)
	HTTP    *http.Client
	AppName string // X-Title
	AppURL  string // HTTP-Referer
}

// Complete implements ports.Model.
func (c Client) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	h := map[string]string{}
	if c.AppName != "" {
		h["X-Title"] = c.AppName
	}
	if c.AppURL != "" {
		h["HTTP-Referer"] = c.AppURL
	}
	return openaicompat.Client{BaseURL: base, Key: c.Key, HTTP: c.HTTP, Headers: h,
		Extra: map[string]any{"provider": map[string]any{"allow_fallbacks": false}}}.Complete(ctx, req)
}
