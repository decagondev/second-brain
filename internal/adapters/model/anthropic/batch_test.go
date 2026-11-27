package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// SB-5604, SB-5605: a cacheable system prompt goes as a block marked ephemeral and cached
// reads come back in Usage; a batch is submitted, looked at until it ends, and its one
// result read from the provider.
func TestCacheAndBatch(t *testing.T) {
	var bodies []string
	looks := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, r.Method+" "+r.URL.Path+" "+string(b))
		switch {
		case r.URL.Path == "/v1/messages":
			_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":90}}`)
		case r.URL.Path == "/v1/messages/batches" && r.Method == http.MethodPost:
			_, _ = io.WriteString(w, `{"id":"msgbatch_1","processing_status":"in_progress"}`)
		case r.URL.Path == "/v1/messages/batches/msgbatch_1":
			looks++
			if looks < 2 {
				_, _ = io.WriteString(w, `{"id":"msgbatch_1","processing_status":"in_progress"}`)
				return
			}
			_, _ = io.WriteString(w, `{"id":"msgbatch_1","processing_status":"ended","results_url":"`+srv.URL+`/v1/messages/batches/msgbatch_1/results"}`)
		case strings.HasSuffix(r.URL.Path, "/results"):
			_, _ = io.WriteString(w, `{"custom_id":"r1","result":{"type":"succeeded","message":{"content":[{"type":"text","text":"later"}],"usage":{"input_tokens":3,"output_tokens":2}}}}`+"\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := Client{BaseURL: srv.URL, Key: func(context.Context) (string, error) { return "k", nil }, HTTP: srv.Client()}
	resp, err := c.Complete(context.Background(), ports.ModelRequest{Model: "m", System: "You answer.", CacheSystem: true, Messages: []ports.Message{{Role: "user", Content: "q"}}})
	if err != nil || resp.Usage.Input != 100 || resp.Usage.Cached != 90 || resp.Text != "hi" {
		t.Fatalf("cached: %+v %v", resp, err)
	}
	var sent struct {
		System []map[string]any `json:"system"`
	}
	if err := json.Unmarshal([]byte(strings.SplitN(bodies[0], " ", 3)[2]), &sent); err != nil || len(sent.System) != 1 || sent.System[0]["cache_control"] == nil {
		t.Fatalf("system block: %s", bodies[0])
	}
	id, err := c.SubmitBatch(context.Background(), ports.ModelRequest{Model: "m", Messages: []ports.Message{{Role: "user", Content: "q"}}})
	if err != nil || id != "msgbatch_1" {
		t.Fatalf("submit: %q %v", id, err)
	}
	if _, done, err := c.CollectBatch(context.Background(), id); done || err != nil {
		t.Fatalf("collected while running: %v %v", done, err)
	}
	if r, done, err := c.CollectBatch(context.Background(), id); !done || err != nil || r.Text != "later" {
		t.Fatalf("collect: %+v %v %v", r, done, err)
	}
}
