package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// SB-1710: the gate runs on a model at an OpenAI-compatible endpoint, by its exact id.
func TestOpenAICompatModel(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Model string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		asked = append(asked, req.Model)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant",
			"content": `{"claims":[{"text":"Bank reconciliation for November is done.","cites":["C1"]}]}`}}}})
	}))
	defer srv.Close()
	var w world
	w.Tenant = "acme-logistics"
	if _, err := build(context.Background(), w, "openaicompat=nohash", ""); err == nil {
		t.Fatal("a model spec without #MODEL was accepted")
	}
	m := modelAt{id: "qwen-local"}
	m.c.BaseURL = srv.URL
	m.c.HTTP = srv.Client()
	m.c.Key = func(context.Context) (string, error) { return "", nil }
	if _, err := m.Complete(context.Background(), ports.ModelRequest{Messages: []ports.Message{{Role: "user", Content: "q"}}}); err != nil || len(asked) != 1 || asked[0] != "qwen-local" {
		t.Fatalf("%v %v", asked, err)
	}
}
