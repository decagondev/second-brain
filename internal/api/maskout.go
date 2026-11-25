package api

import (
	"bytes"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/ingest"
)

// MaskOut is the cell's one filter on what its API sends (sb-ingest(7), Nothing sensitive
// goes out; SB-5102): every JSON string, and every CSV, Markdown or plain-text body, passes
// ingest.MaskOut before it leaves. Only POST /v1/tokens, which hands the caller their new
// token once, and GET /metrics, numbers by label, are passed as they are. Masked is called with the request and how many strings
// changed, for the guardrail events (SB-5103); nil records nothing.
func MaskOut(next http.Handler, masked func(r *http.Request, n int)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == http.MethodPost && r.URL.Path == "/v1/tokens") || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		rec := &maskWriter{w: w, header: http.Header{}}
		next.ServeHTTP(rec, r)
		body, n := maskBody(rec.header.Get("Content-Type"), rec.buf.Bytes())
		for k, vs := range rec.header {
			w.Header()[k] = vs
		}
		if n > 0 {
			w.Header().Del("Content-Length")
			if masked != nil {
				masked(r, n)
			}
		}
		code := rec.code
		if code == 0 {
			code = http.StatusOK
		}
		w.WriteHeader(code)
		_, _ = w.Write(body)
	})
}

// maskBody masks a body of a type that carries text; anything else goes as it is.
func maskBody(ct string, b []byte) ([]byte, int) {
	ct = strings.ToLower(ct)
	switch {
	case strings.HasPrefix(ct, "application/json"):
		return ingest.MaskJSON(b)
	case strings.HasPrefix(ct, "text/csv"), strings.HasPrefix(ct, "text/markdown"), strings.HasPrefix(ct, "text/plain"):
		out, changed := ingest.MaskOut(string(b))
		if changed {
			return []byte(out), 1
		}
	}
	return b, 0
}

// maskWriter holds a response until it can be masked whole.
type maskWriter struct {
	w      http.ResponseWriter
	header http.Header
	buf    bytes.Buffer
	code   int
}

func (m *maskWriter) Header() http.Header { return m.header }
func (m *maskWriter) WriteHeader(code int) {
	if m.code == 0 {
		m.code = code
	}
}
func (m *maskWriter) Write(b []byte) (int, error) {
	if m.code == 0 {
		m.code = http.StatusOK
	}
	return m.buf.Write(b)
}
