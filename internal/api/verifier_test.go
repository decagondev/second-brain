package api

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/attest"
	"example.com/second-brain/internal/core/verify"
	"example.com/second-brain/internal/obs"
)

type quietTarget struct{}

func (quietTarget) Users(context.Context) ([]string, error)     { return nil, nil }
func (quietTarget) Questions(context.Context) ([]string, error) { return nil, nil }
func (quietTarget) Ask(context.Context, string, string) (verify.Answer, error) {
	return verify.Answer{}, nil
}
func (quietTarget) Allowed(context.Context, string, string) (bool, error) { return false, nil }
func (quietTarget) ForeignIDs(context.Context) ([]string, error)          { return nil, nil }
func (quietTarget) Timeout(string) (func(), error)                        { return func() {}, nil }
func (quietTarget) Connectors() []string                                  { return nil }

// SB-905: the verifier endpoints need the control plane, and nothing they return is content.
func TestVerifierEndpoints(t *testing.T) {
	m := obs.NewMetrics()
	m.Add("sb_asks_total", "Answers.", map[string]string{"tenant": "acme-logistics", "outcome": "What are the Q3 bonus amounts?"}, 1)
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	v := Verifier{Release: "rel_x", Metrics: m.Handler(), Target: quietTarget{},
		ControlPlane: func(r *http.Request) bool { return r.Header.Get("Authorization") == "Bearer cp" },
		Attest: func(context.Context) (attest.Signed, error) {
			return attest.Sign(key, attest.Report{Tenant: "acme-logistics", Release: "rel_x", Digests: map[string]string{"image/api": strings.Repeat("a", 64)}, Seq: 1})
		}}
	mux := http.NewServeMux()
	v.Register(mux)
	call := func(method, path, auth, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/metrics", "", ""); w.Code != 200 || strings.Contains(w.Body.String(), "bonus") || !strings.Contains(w.Body.String(), `outcome="invalid"`) {
		t.Fatalf("metrics: %d %s", w.Code, w.Body)
	}
	if w := call("GET", "/v1/verifier/attestation", "", ""); w.Code != 401 {
		t.Fatalf("no identity: %d", w.Code)
	}
	if w := call("GET", "/v1/verifier/attestation", "Bearer someone", ""); w.Code != 403 {
		t.Fatalf("not the control plane: %d", w.Code)
	}
	if w := call("GET", "/v1/verifier/attestation", "Bearer cp", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"release":"rel_x"`) || !strings.Contains(w.Body.String(), `"sig"`) {
		t.Fatalf("attestation: %d %s", w.Code, w.Body)
	}
	if w := call("POST", "/v1/verifier/verify", "Bearer cp", `{"laws":["LAW-99"]}`); w.Code != 400 {
		t.Fatalf("unknown law: %d", w.Code)
	}
	w := call("POST", "/v1/verifier/verify", "Bearer cp", `{"laws":["LAW-14"],"sample":5,"seed":3}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"Status":"unknown"`) {
		t.Fatalf("verify with nothing to sample must be unknown: %d %s", w.Code, w.Body)
	}
}
