package openaicompat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbed(t *testing.T) {
	dim := 3
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if r.URL.Path != "/v1/embeddings" || json.NewDecoder(r.Body).Decode(&req) != nil || req.Model != "nomic" || r.Header.Get("Authorization") != "" {
			http.Error(w, "bad", 400)
			return
		}
		var data []map[string]any
		for i := len(req.Input) - 1; i >= 0; i-- { // out of order, by index
			v := make([]float32, dim)
			v[0] = float32(i)
			data = append(data, map[string]any{"index": i, "embedding": v})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()
	e := Embedder{Base: srv.URL + "/v1", Name: "nomic", Dim: 3, HTTP: srv.Client()}
	texts := make([]string, 70)
	for i := range texts {
		texts[i] = fmt.Sprint(i)
	}
	vs, err := e.Embed(context.Background(), texts)
	if err != nil || len(vs) != 70 || vs[69][0] != 5 || vs[1][0] != 1 { // the second batch restarts at 0
		t.Fatalf("%v %v", len(vs), err)
	}
	if e.Model() != "openaicompat:nomic:3" {
		t.Fatal(e.Model())
	}
	dim = 4
	if _, err := e.Embed(context.Background(), []string{"x"}); err == nil {
		t.Fatal("a vector of the wrong size was accepted")
	}
}
