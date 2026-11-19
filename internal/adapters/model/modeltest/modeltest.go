// Package modeltest is a recorded fake of the model providers' APIs (ADR-0019): it
// serves the OpenAI-compatible chat completions shape and the Anthropic Messages shape,
// and records every request body it receives, so tests can check what left the cell.
package modeltest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// Fake is one fake provider.
type Fake struct {
	*httptest.Server
	Key string // the bearer or x-api-key the fake accepts

	mu        sync.Mutex
	bodies    []string
	headers   []http.Header
	reply     string // the assistant's text
	limitNext int    // answer 429 to this many requests first
	fail      bool
}

// New starts a fake that answers reply to every completion.
func New(key, reply string) *Fake {
	f := &Fake{Key: key, reply: reply}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	return f
}

// RateLimit makes the next n requests answer 429.
func (f *Fake) RateLimit(n int) { f.mu.Lock(); f.limitNext = n; f.mu.Unlock() }

// Fail makes every request answer 500.
func (f *Fake) Fail() { f.mu.Lock(); f.fail = true; f.mu.Unlock() }

// Bodies is every request body received, in order.
func (f *Fake) Bodies() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.bodies...)
}

// Headers is every request's headers, in order.
func (f *Fake) Headers() []http.Header {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]http.Header(nil), f.headers...)
}

// Received is everything the fake was sent, bodies and headers.
func (f *Fake) Received() string {
	var b strings.Builder
	for _, s := range f.Bodies() {
		b.WriteString(s)
		b.WriteString("\n")
	}
	for _, h := range f.Headers() {
		for k, v := range h {
			b.WriteString(k + ": " + strings.Join(v, ",") + "\n")
		}
	}
	return b.String()
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.bodies = append(f.bodies, string(body))
	f.headers = append(f.headers, r.Header.Clone())
	limited, failing := f.limitNext > 0, f.fail
	if limited {
		f.limitNext--
	}
	reply := f.reply
	f.mu.Unlock()

	anthropic := strings.HasSuffix(r.URL.Path, "/v1/messages")
	key := r.Header.Get("x-api-key")
	if !anthropic {
		key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	switch {
	case r.Method != http.MethodPost:
		http.Error(w, `{"error":"method"}`, http.StatusMethodNotAllowed)
		return
	case key != f.Key:
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	case anthropic && r.Header.Get("anthropic-version") == "":
		http.Error(w, `{"error":"anthropic-version required"}`, http.StatusBadRequest)
		return
	case limited:
		http.Error(w, `{"error":"rate_limited"}`, http.StatusTooManyRequests)
		return
	case failing:
		http.Error(w, `{"error":"overloaded"}`, http.StatusInternalServerError)
		return
	}
	var req struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if json.Unmarshal(body, &req) != nil || len(req.Messages) == 0 {
		http.Error(w, `{"error":"messages required"}`, http.StatusBadRequest)
		return
	}
	in := len(body) / 4
	out := len(reply) / 4
	w.Header().Set("Content-Type", "application/json")
	if anthropic {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_fake", "type": "message", "role": "assistant",
			"content": []map[string]any{{"type": "text", "text": reply}},
			"usage":   map[string]int{"input_tokens": in, "output_tokens": out},
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id": "chatcmpl-fake", "object": "chat.completion",
		"choices": []map[string]any{{"index": 0, "message": map[string]string{"role": "assistant", "content": reply}, "finish_reason": "stop"}},
		"usage":   map[string]int{"prompt_tokens": in, "completion_tokens": out, "total_tokens": in + out},
	})
}
