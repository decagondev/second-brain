// Package openaicompat is an embedder over an OpenAI-compatible embeddings endpoint
// (sb-ingest(7); SB-1710), such as llama.cpp's server on a self-hosted cell's own machine.
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

// MaxBatch is the most texts in one request.
const MaxBatch = 64

// Embedder calls POST {Base}/embeddings. It needs no key: it is for an endpoint inside the
// cell's own network.
type Embedder struct {
	Base, Name string
	Dim        int
	HTTP       *http.Client
}

var _ ports.Embedder = Embedder{}

// Model is the embedder's id with its size, which the index records (sb-ingest(7)).
func (e Embedder) Model() string { return fmt.Sprintf("openaicompat:%s:%d", e.Name, e.Dim) }

// Embed implements ports.Embedder. A vector of the wrong size, or a short answer, is an
// error: the index never holds a vector it can't compare.
func (e Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += MaxBatch {
		end := min(start+MaxBatch, len(texts))
		b, err := json.Marshal(map[string]any{"model": e.Name, "input": texts[start:end]})
		if err != nil {
			return nil, err
		}
		hr, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(e.Base, "/")+"/embeddings", bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		hr.Header.Set("Content-Type", "application/json")
		var resp struct {
			Data []struct {
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			} `json:"data"`
		}
		if err := do(e.HTTP, hr, &resp); err != nil {
			return nil, err
		}
		if len(resp.Data) != end-start {
			return nil, fmt.Errorf("%w: %d embeddings for %d texts", ports.ErrModelDown, len(resp.Data), end-start)
		}
		batch := make([][]float32, end-start)
		for _, d := range resp.Data {
			if d.Index < 0 || d.Index >= len(batch) || batch[d.Index] != nil || len(d.Embedding) != e.Dim {
				return nil, fmt.Errorf("%w: an embedding of the wrong size or place", ports.ErrModelDown)
			}
			batch[d.Index] = d.Embedding
		}
		out = append(out, batch...)
	}
	return out, nil
}

// do sends hr and decodes a 2xx JSON body; any failure is ErrModelDown, and the error text
// never carries the request.
func do(client *http.Client, hr *http.Request, out any) error {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(hr)
	if err != nil {
		return fmt.Errorf("%w: %v", ports.ErrModelDown, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil || resp.StatusCode/100 != 2 {
		return fmt.Errorf("%w: embeddings HTTP %d", ports.ErrModelDown, resp.StatusCode)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%w: bad embeddings response", ports.ErrModelDown)
	}
	return nil
}
