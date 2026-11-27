package openaicompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// OpenAI's file-based Batch API (sb-runtime(7), Batching; SB-5702): the calls go up as a
// JSONL file with purpose batch, a batch runs them within the 24-hour window, and its output
// and error files are read and then deleted with the input, so nothing is left at the
// provider. Only a provider sb-cell(5) model.batch says supports it uses Batching; a plain
// Client never batches.

// Batching is a Client whose provider has the Batch API.
type Batching struct{ Client }

var (
	_ ports.Batcher        = Batching{}
	_ ports.BatchCanceller = Batching{}
)

// BatchWindow is the completion window every batch asks for.
const BatchWindow = "24h"

type fileObj struct {
	ID string `json:"id"`
}

type batchObj struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	InputFileID  string `json:"input_file_id"`
	OutputFileID string `json:"output_file_id"`
	ErrorFileID  string `json:"error_file_id"`
}

func (c Batching) url(path string) string { return strings.TrimRight(c.BaseURL, "/") + path }

func (c Batching) send(ctx context.Context, method, path, ctype string, body io.Reader, out any) error {
	key, err := c.Key(ctx)
	if err != nil {
		return fmt.Errorf("%w: model key: %v", ports.ErrModelDown, err)
	}
	hr, err := http.NewRequestWithContext(ctx, method, c.url(path), body)
	if err != nil {
		return err
	}
	hr.Header.Set("Authorization", "Bearer "+key)
	if ctype != "" {
		hr.Header.Set("Content-Type", ctype)
	}
	for k, v := range c.Headers {
		hr.Header.Set(k, v)
	}
	if out == nil {
		out = &struct{}{}
	}
	return Do(c.HTTP, hr, out)
}

// SubmitBatch implements ports.Batcher: the one call uploaded as a JSONL file and a batch
// created over it. The input file is deleted when the batch is collected or cancelled.
func (c Batching) SubmitBatch(ctx context.Context, req ports.ModelRequest) (string, error) {
	b, err := c.body(req)
	if err != nil {
		return "", err
	}
	line, err := json.Marshal(map[string]any{"custom_id": "r1", "method": "POST", "url": "/v1/chat/completions", "body": json.RawMessage(b)})
	if err != nil {
		return "", err
	}
	var form bytes.Buffer
	mw := multipart.NewWriter(&form)
	_ = mw.WriteField("purpose", "batch")
	fw, err := mw.CreateFormFile("file", "batch.jsonl")
	if err != nil {
		return "", err
	}
	_, _ = fw.Write(append(line, '\n'))
	if err := mw.Close(); err != nil {
		return "", err
	}
	var f fileObj
	if err := c.send(ctx, http.MethodPost, "/files", mw.FormDataContentType(), &form, &f); err != nil {
		return "", err
	}
	if f.ID == "" {
		return "", fmt.Errorf("%w: an upload without an id", ports.ErrModelDown)
	}
	create, _ := json.Marshal(map[string]string{"input_file_id": f.ID, "endpoint": "/v1/chat/completions", "completion_window": BatchWindow})
	var bo batchObj
	if err := c.send(ctx, http.MethodPost, "/batches", "application/json", bytes.NewReader(create), &bo); err != nil || bo.ID == "" {
		_ = c.deleteFile(ctx, f.ID)
		if err == nil {
			err = fmt.Errorf("%w: a batch without an id", ports.ErrModelDown)
		}
		return "", err
	}
	return bo.ID, nil
}

// CollectBatch implements ports.Batcher: done is false while the batch runs. Once it has
// ended its output (or error) file is read, and the input, output and error files deleted.
func (c Batching) CollectBatch(ctx context.Context, id string) (ports.ModelResponse, bool, error) {
	var bo batchObj
	if err := c.send(ctx, http.MethodGet, "/batches/"+id, "", nil, &bo); err != nil {
		return ports.ModelResponse{}, false, err
	}
	switch bo.Status {
	case "completed":
	case "failed", "expired", "cancelled":
		c.cleanUp(ctx, bo)
		return ports.ModelResponse{}, true, fmt.Errorf("%w: the batch %s", ports.ErrModelDown, bo.Status)
	default: // validating, in_progress, finalizing, cancelling
		return ports.ModelResponse{}, false, nil
	}
	defer c.cleanUp(ctx, bo)
	file := bo.OutputFileID
	if file == "" {
		file = bo.ErrorFileID
	}
	if file == "" {
		return ports.ModelResponse{}, true, fmt.Errorf("%w: the batch had no output", ports.ErrModelDown)
	}
	raw, err := c.content(ctx, file)
	if err != nil {
		return ports.ModelResponse{}, true, err
	}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	for sc.Scan() {
		var line struct {
			CustomID string `json:"custom_id"`
			Response *struct {
				StatusCode int      `json:"status_code"`
				Body       response `json:"body"`
			} `json:"response"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil || line.CustomID != "r1" {
			continue
		}
		if line.Response == nil || line.Response.StatusCode/100 != 2 {
			return ports.ModelResponse{}, true, fmt.Errorf("%w: the batched request failed", ports.ErrModelDown)
		}
		resp, err := line.Response.Body.model()
		return resp, true, err
	}
	return ports.ModelResponse{}, true, fmt.Errorf("%w: the batch had no result", ports.ErrModelDown)
}

// CancelBatch implements ports.BatchCanceller, and deletes the batch's files.
func (c Batching) CancelBatch(ctx context.Context, id string) error {
	var bo batchObj
	if err := c.send(ctx, http.MethodPost, "/batches/"+id+"/cancel", "", nil, &bo); err != nil {
		return err
	}
	c.cleanUp(ctx, bo)
	return nil
}

func (c Batching) content(ctx context.Context, file string) ([]byte, error) {
	key, err := c.Key(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: model key: %v", ports.ErrModelDown, err)
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/files/"+file+"/content"), nil)
	if err != nil {
		return nil, err
	}
	hr.Header.Set("Authorization", "Bearer "+key)
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(hr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ports.ErrModelDown, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: batch output %d", ports.ErrModelDown, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func (c Batching) deleteFile(ctx context.Context, id string) error {
	return c.send(ctx, http.MethodDelete, "/files/"+id, "", nil, nil)
}

// cleanUp deletes every file of a batch; a delete that fails is left to the provider's own
// expiry.
func (c Batching) cleanUp(ctx context.Context, bo batchObj) {
	for _, f := range []string{bo.InputFileID, bo.OutputFileID, bo.ErrorFileID} {
		if f != "" {
			_ = c.deleteFile(ctx, f)
		}
	}
}
