// Package groq is the Groq model adapter: the OpenAI-compatible wire format at Groq's
// base URL (ADR-0019). JSON is asked for in the prompt too, since not every Groq model
// has a JSON mode; the composer copes with output that is not JSON (sb-citations(7)).
package groq

import (
	"context"
	"net/http"

	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/core/ports"
)

// BaseURL is Groq's OpenAI-compatible endpoint.
const BaseURL = "https://api.groq.com/openai/v1"

// Client calls Groq.
type Client struct {
	BaseURL string // empty means BaseURL; tests point it at the fake
	Key     func(ctx context.Context) (string, error)
	HTTP    *http.Client
}

// Complete implements ports.Model.
func (c Client) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	return openaicompat.Client{BaseURL: base, Key: c.Key, HTTP: c.HTTP}.Complete(ctx, req)
}
