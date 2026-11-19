// Package anthropic is the Anthropic Messages API adapter (ADR-0019).
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/core/ports"
)

// BaseURL is Anthropic's API.
const BaseURL = "https://api.anthropic.com"

// Version is the anthropic-version header this adapter speaks.
const Version = "2023-06-01"

// Client calls POST {BaseURL}/v1/messages.
type Client struct {
	BaseURL string
	Key     func(ctx context.Context) (string, error)
	HTTP    *http.Client
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete implements ports.Model. Anthropic requires max_tokens (1024 when unset). JSON
// is asked for in the prompt, which the answer prompt already does.
func (c Client) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	body := request{Model: req.Model, System: req.System, MaxTokens: req.MaxTokens}
	if body.MaxTokens == 0 {
		body.MaxTokens = 1024
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, message{Role: m.Role, Content: m.Content})
	}
	b, err := json.Marshal(body)
	if err != nil {
		return ports.ModelResponse{}, err
	}
	key, err := c.Key(ctx)
	if err != nil {
		return ports.ModelResponse{}, fmt.Errorf("%w: model key: %v", ports.ErrModelDown, err)
	}
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/v1/messages", bytes.NewReader(b))
	if err != nil {
		return ports.ModelResponse{}, err
	}
	hr.Header.Set("x-api-key", key)
	hr.Header.Set("anthropic-version", Version)
	hr.Header.Set("Content-Type", "application/json")
	var out response
	if err := openaicompat.Do(c.HTTP, hr, &out); err != nil {
		return ports.ModelResponse{}, err
	}
	var text strings.Builder
	for _, part := range out.Content {
		if part.Type == "text" {
			text.WriteString(part.Text)
		}
	}
	return ports.ModelResponse{Text: text.String(),
		Usage: ports.Usage{Input: out.Usage.InputTokens, Output: out.Usage.OutputTokens}}, nil
}
