package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// SB-5102: the API's JSON and text bodies are masked, a new token and the metrics pass as
// they are, other types are left alone, and each masked response is counted.
func TestMaskOutMiddleware(t *testing.T) {
	key := "AKIA" + strings.Repeat("Q7", 8)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/x", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "999")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"text":"card 4111 1111 1111 1111 key ` + key + `"}`))
	})
	mux.HandleFunc("GET /v1/csv", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("a,b\n" + key + ",1\n"))
	})
	for _, p := range []string{"POST /v1/tokens", "GET /metrics", "GET /app.js"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, _ *http.Request) {
			if strings.HasSuffix(p, ".js") {
				w.Header().Set("Content-Type", "text/javascript")
			} else {
				w.Header().Set("Content-Type", "application/json")
			}
			_, _ = w.Write([]byte(`{"token":"` + key + `"}`))
		})
	}
	masked := 0
	h := MaskOut(mux, func(_ *http.Request, n int) { masked += n })
	get := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}
	rec := get("GET", "/v1/x")
	if b := rec.Body.String(); rec.Code != http.StatusCreated || strings.Contains(b, key) || strings.Contains(b, "4111 1111 1111 1111") || rec.Header().Get("Content-Length") != "" {
		t.Fatalf("json: %d %q %v", rec.Code, b, rec.Header())
	}
	if b := get("GET", "/v1/csv").Body.String(); strings.Contains(b, key) || !strings.HasPrefix(b, "a,b\n") {
		t.Fatalf("csv: %q", b)
	}
	if masked != 2 {
		t.Fatalf("masked %d", masked)
	}
	for _, c := range [][2]string{{"POST", "/v1/tokens"}, {"GET", "/metrics"}, {"GET", "/app.js"}} {
		if b := get(c[0], c[1]).Body.String(); !strings.Contains(b, key) {
			t.Fatalf("%s %s was masked: %q", c[0], c[1], b)
		}
	}
}
