package openaicompat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// openaiBatches is a recorded fake of OpenAI's Files and Batches endpoints (SB-5702).
type openaiBatches struct {
	mu      sync.Mutex
	files   map[string]string
	looks   int
	deleted []string
	created string
}

func (f *openaiBatches) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v1/files":
		if err := r.ParseMultipartForm(1 << 20); err != nil || r.FormValue("purpose") != "batch" {
			http.Error(w, "bad upload", http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "no file", http.StatusBadRequest)
			return
		}
		b, _ := io.ReadAll(file)
		f.files["file-in"] = string(b)
		_, _ = io.WriteString(w, `{"id":"file-in","purpose":"batch"}`)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/batches":
		b, _ := io.ReadAll(r.Body)
		f.created = string(b)
		_, _ = io.WriteString(w, `{"id":"batch_1","status":"validating","input_file_id":"file-in"}`)
	case r.Method == http.MethodGet && r.URL.Path == "/v1/batches/batch_1":
		f.looks++
		if f.looks < 2 {
			_, _ = io.WriteString(w, `{"id":"batch_1","status":"in_progress","input_file_id":"file-in"}`)
			return
		}
		f.files["file-out"] = `{"id":"req_1","custom_id":"r1","response":{"status_code":200,"body":{"choices":[{"message":{"role":"assistant","content":"later"}}],"usage":{"prompt_tokens":7,"completion_tokens":3}}},"error":null}` + "\n"
		_, _ = io.WriteString(w, `{"id":"batch_1","status":"completed","input_file_id":"file-in","output_file_id":"file-out"}`)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/content"):
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/files/"), "/content")
		_, _ = io.WriteString(w, f.files[id])
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/files/"):
		id := strings.TrimPrefix(r.URL.Path, "/v1/files/")
		delete(f.files, id)
		f.deleted = append(f.deleted, id)
		_, _ = io.WriteString(w, `{"id":"`+id+`","deleted":true}`)
	default:
		http.NotFound(w, r)
	}
}

// SB-5702: a call goes up as a JSONL file with purpose batch, a 24-hour batch runs it, its
// output is read, and both files are deleted once collected.
func TestOpenAIBatch(t *testing.T) {
	fake := &openaiBatches{files: map[string]string{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	c := Batching{Client{BaseURL: srv.URL + "/v1", Key: func(context.Context) (string, error) { return "k", nil }, HTTP: srv.Client()}}
	id, err := c.SubmitBatch(context.Background(), ports.ModelRequest{Model: "gpt", System: "s", Messages: []ports.Message{{Role: "user", Content: "q"}}, JSON: true})
	if err != nil || id != "batch_1" {
		t.Fatalf("submit: %q %v", id, err)
	}
	if !strings.Contains(fake.files["file-in"], `"url":"/v1/chat/completions"`) || !strings.Contains(fake.files["file-in"], `"custom_id":"r1"`) ||
		!strings.Contains(fake.created, `"completion_window":"24h"`) || !strings.Contains(fake.created, `"input_file_id":"file-in"`) {
		t.Fatalf("upload %q create %q", fake.files["file-in"], fake.created)
	}
	if _, done, err := c.CollectBatch(context.Background(), id); done || err != nil {
		t.Fatalf("collected while running: %v %v", done, err)
	}
	resp, done, err := c.CollectBatch(context.Background(), id)
	if !done || err != nil || resp.Text != "later" || resp.Usage.Input != 7 || resp.Usage.Output != 3 {
		t.Fatalf("collect: %+v %v %v", resp, done, err)
	}
	sort.Strings(fake.deleted)
	if strings.Join(fake.deleted, ",") != "file-in,file-out" || len(fake.files) != 0 {
		t.Fatalf("deleted %v left %v", fake.deleted, fake.files)
	}
	// A plain client has no batch API.
	if _, ok := any(c.Client).(ports.Batcher); ok {
		t.Fatal("a provider without model.batch batches")
	}
}
