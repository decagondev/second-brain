package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"example.com/second-brain/internal/adapters/model/openaicompat"
	"example.com/second-brain/internal/core/ports"
)

// The Message Batches API (SB-5604): one request a batch, submitted now and collected once
// the batch has ended, at the batch price.

var _ ports.Batcher = Client{}

type batchReq struct {
	Requests []struct {
		CustomID string  `json:"custom_id"`
		Params   request `json:"params"`
	} `json:"requests"`
}

type batchState struct {
	ID               string `json:"id"`
	ProcessingStatus string `json:"processing_status"`
	ResultsURL       string `json:"results_url"`
}

func (c Client) base() string {
	if c.BaseURL == "" {
		return BaseURL
	}
	return strings.TrimRight(c.BaseURL, "/")
}

func (c Client) do(ctx context.Context, method, url string, body any, out any) error {
	var rd *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	key, err := c.Key(ctx)
	if err != nil {
		return fmt.Errorf("%w: model key: %v", ports.ErrModelDown, err)
	}
	hr, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return err
	}
	hr.Header.Set("x-api-key", key)
	hr.Header.Set("anthropic-version", Version)
	hr.Header.Set("Content-Type", "application/json")
	return openaicompat.Do(c.HTTP, hr, out)
}

// SubmitBatch implements ports.Batcher: a batch of the one request.
func (c Client) SubmitBatch(ctx context.Context, req ports.ModelRequest) (string, error) {
	var b batchReq
	b.Requests = append(b.Requests, struct {
		CustomID string  `json:"custom_id"`
		Params   request `json:"params"`
	}{CustomID: "r1", Params: c.body(req)})
	var st batchState
	if err := c.do(ctx, http.MethodPost, c.base()+"/v1/messages/batches", b, &st); err != nil {
		return "", err
	}
	if st.ID == "" {
		return "", fmt.Errorf("%w: a batch without an id", ports.ErrModelDown)
	}
	return st.ID, nil
}

// CollectBatch implements ports.Batcher: done is false while the batch runs.
func (c Client) CollectBatch(ctx context.Context, id string) (ports.ModelResponse, bool, error) {
	var st batchState
	if err := c.do(ctx, http.MethodGet, c.base()+"/v1/messages/batches/"+id, nil, &st); err != nil {
		return ports.ModelResponse{}, false, err
	}
	if st.ProcessingStatus != "ended" {
		return ports.ModelResponse{}, false, nil
	}
	if st.ResultsURL == "" || !strings.HasPrefix(st.ResultsURL, c.base()+"/") {
		return ports.ModelResponse{}, false, fmt.Errorf("%w: the batch's results aren't at the provider", ports.ErrModelDown)
	}
	key, err := c.Key(ctx)
	if err != nil {
		return ports.ModelResponse{}, false, fmt.Errorf("%w: model key: %v", ports.ErrModelDown, err)
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodGet, st.ResultsURL, nil)
	if err != nil {
		return ports.ModelResponse{}, false, err
	}
	hr.Header.Set("x-api-key", key)
	hr.Header.Set("anthropic-version", Version)
	resp, err := c.HTTP.Do(hr)
	if err != nil {
		return ports.ModelResponse{}, false, fmt.Errorf("%w: %v", ports.ErrModelDown, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ports.ModelResponse{}, false, fmt.Errorf("%w: batch results %d", ports.ErrModelDown, resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	for sc.Scan() {
		var line struct {
			CustomID string `json:"custom_id"`
			Result   struct {
				Type    string   `json:"type"`
				Message response `json:"message"`
			} `json:"result"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil || line.CustomID != "r1" {
			continue
		}
		if line.Result.Type != "succeeded" {
			return ports.ModelResponse{}, true, fmt.Errorf("%w: the batched request %s", ports.ErrModelDown, line.Result.Type)
		}
		return line.Result.Message.model(), true, nil
	}
	return ports.ModelResponse{}, true, fmt.Errorf("%w: the batch had no result", ports.ErrModelDown)
}

var _ ports.BatchCanceller = Client{}

// CancelBatch implements ports.BatchCanceller (SB-5701): a batch not back by the morning.
func (c Client) CancelBatch(ctx context.Context, id string) error {
	var st batchState
	return c.do(ctx, http.MethodPost, c.base()+"/v1/messages/batches/"+id+"/cancel", nil, &st)
}
