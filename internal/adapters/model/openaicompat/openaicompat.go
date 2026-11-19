// Package openaicompat is the model adapter for any OpenAI-compatible chat completions
// endpoint (ADR-0019). Groq and OpenRouter reuse it with their own base URLs and headers.
package openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// Client calls POST {BaseURL}/chat/completions.
type Client struct {
	BaseURL string                                    // e.g. https://api.openai.com/v1
	Key     func(ctx context.Context) (string, error) // vault://tenants/T/model, at call time
	HTTP    *http.Client
	Headers map[string]string // extra headers (OpenRouter's attribution headers)
	Extra   map[string]any    // extra top-level body fields (OpenRouter's provider routing)
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model          string            `json:"model"`
	Messages       []message         `json:"messages"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type response struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete implements ports.Model.
func (c Client) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	body := request{Model: req.Model, MaxTokens: req.MaxTokens}
	if req.System != "" {
		body.Messages = append(body.Messages, message{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, message{Role: m.Role, Content: m.Content})
	}
	if req.JSON {
		body.ResponseFormat = map[string]string{"type": "json_object"}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return ports.ModelResponse{}, err
	}
	if len(c.Extra) > 0 {
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		for k, v := range c.Extra {
			if _, taken := m[k]; !taken {
				m[k] = v
			}
		}
		if b, err = json.Marshal(m); err != nil {
			return ports.ModelResponse{}, err
		}
	}
	key, err := c.Key(ctx)
	if err != nil {
		return ports.ModelResponse{}, fmt.Errorf("%w: model key: %v", ports.ErrModelDown, err)
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return ports.ModelResponse{}, err
	}
	hr.Header.Set("Authorization", "Bearer "+key)
	hr.Header.Set("Content-Type", "application/json")
	for k, v := range c.Headers {
		hr.Header.Set(k, v)
	}
	var out response
	if err := Do(c.HTTP, hr, &out); err != nil {
		return ports.ModelResponse{}, err
	}
	if len(out.Choices) == 0 {
		return ports.ModelResponse{}, fmt.Errorf("%w: no choices in the response", ports.ErrModelDown)
	}
	return ports.ModelResponse{Text: out.Choices[0].Message.Content,
		Usage: ports.Usage{Input: out.Usage.PromptTokens, Output: out.Usage.CompletionTokens}}, nil
}

// Do sends hr and decodes a 2xx JSON body into out. 429 is ErrRateLimited; any other
// failure is ErrModelDown. Error text never includes the request (it may hold the key).
func Do(client *http.Client, hr *http.Request, out any) error {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(hr)
	if err != nil {
		return fmt.Errorf("%w: %v", ports.ErrModelDown, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("%w: %v", ports.ErrModelDown, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return ports.ErrRateLimited
	case resp.StatusCode/100 != 2:
		return fmt.Errorf("%w: HTTP %d", ports.ErrModelDown, resp.StatusCode)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%w: bad response: %v", ports.ErrModelDown, err)
	}
	return nil
}
