// Package anthropic is the Anthropic Messages API adapter (ADR-0019).
package anthropic

import (
	"context"
	"encoding/base64"
	"net/http"

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
	Content any    `json:"content"` // a string, or blocks with an image (SB-6101)
}

// imageBlock is a picture in a message (SB-6101).
type imageBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Source *struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source,omitempty"`
}

// block is a system prompt block; cache_control marks the stable prefix cacheable (SB-5605).
type block struct {
	Type         string `json:"type"`
	Text         string `json:"text"`
	CacheControl *struct {
		Type string `json:"type"`
	} `json:"cache_control,omitempty"`
}

type request struct {
	Model     string    `json:"model"`
	System    any       `json:"system,omitempty"` // a string, or blocks when cached
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens     int `json:"input_tokens"`
		OutputTokens    int `json:"output_tokens"`
		CacheReadTokens int `json:"cache_read_input_tokens"`
		CacheWriteToks  int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

// body is the request for req: the system prompt marked cacheable when req asks (SB-5605).
// Only the system prompt is ever marked: what the asker sent is in the messages.
func (c Client) body(req ports.ModelRequest) request {
	body := request{Model: req.Model, MaxTokens: req.MaxTokens}
	if req.System != "" {
		body.System = req.System
		if req.CacheSystem {
			b := block{Type: "text", Text: req.System}
			b.CacheControl = &struct {
				Type string `json:"type"`
			}{Type: "ephemeral"}
			body.System = []block{b}
		}
	}
	if body.MaxTokens == 0 {
		body.MaxTokens = 1024
	}
	for _, m := range req.Messages {
		if m.Image == nil {
			body.Messages = append(body.Messages, message{Role: m.Role, Content: m.Content})
			continue
		}
		img := imageBlock{Type: "image", Source: &struct {
			Type      string `json:"type"`
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
		}{Type: "base64", MediaType: m.Image.MIME, Data: base64.StdEncoding.EncodeToString(m.Image.Data)}}
		body.Messages = append(body.Messages, message{Role: m.Role, Content: []imageBlock{img, {Type: "text", Text: m.Content}}})
	}
	return body
}

// model is the response as the port has it: Anthropic counts cache reads and writes apart
// from input_tokens, so Input is all three and Cached the reads.
func (r response) model() ports.ModelResponse {
	text := ""
	for _, part := range r.Content {
		if part.Type == "text" {
			text += part.Text
		}
	}
	u := r.Usage
	return ports.ModelResponse{Text: text,
		Usage: ports.Usage{Input: u.InputTokens + u.CacheReadTokens + u.CacheWriteToks, Output: u.OutputTokens, Cached: u.CacheReadTokens}}
}

// Complete implements ports.Model. Anthropic requires max_tokens (1024 when unset). JSON
// is asked for in the prompt, which the answer prompt already does.
func (c Client) Complete(ctx context.Context, req ports.ModelRequest) (ports.ModelResponse, error) {
	var out response
	if err := c.do(ctx, http.MethodPost, c.base()+"/v1/messages", c.body(req), &out); err != nil {
		return ports.ModelResponse{}, err
	}
	return out.model(), nil
}
