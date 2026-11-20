package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"example.com/second-brain/internal/core/attest"
	"example.com/second-brain/internal/core/verify"
	"example.com/second-brain/internal/errs"
)

// Verifier serves a deployed cell's /metrics and verifier endpoints (sb-runtime(7),
// Endpoints, exactly; SB-905). Nothing it returns holds content, prompts or secrets.
type Verifier struct {
	Release string
	Metrics http.Handler
	// ControlPlane says whether a request carries the control plane's identity.
	ControlPlane func(r *http.Request) bool
	Attest       func(ctx context.Context) (attest.Signed, error)
	Target       verify.Target
}

// Laws verify may be asked for.
var verifyLaws = map[string]bool{"LAW-01": true, "LAW-02": true, "LAW-03": true, "LAW-14": true, "LAW-17": true}

// Register mounts the endpoints.
func (v Verifier) Register(mux *http.ServeMux) {
	if v.Metrics != nil {
		mux.Handle("GET /metrics", v.Metrics)
	}
	mux.HandleFunc("GET /v1/verifier/attestation", v.guard(v.attestation))
	mux.HandleFunc("POST /v1/verifier/verify", v.guard(v.verify))
}

func (v Verifier) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			v.fail(w, errs.New(errs.EAUTH, "The control plane's identity is required."))
			return
		}
		if v.ControlPlane == nil || !v.ControlPlane(r) {
			v.fail(w, errs.New(errs.EPERM, "Only the control plane may call the verifier."))
			return
		}
		next(w, r)
	}
}

func (v Verifier) attestation(w http.ResponseWriter, r *http.Request) {
	s, err := v.Attest(r.Context())
	if err != nil {
		v.fail(w, errs.New(errs.EUNAVAIL, "The running build can't be read right now."))
		return
	}
	v.ok(w, s)
}

// VerifyRequest is the body of POST /v1/verifier/verify.
type VerifyRequest struct {
	Laws   []string `json:"laws"`
	Sample int      `json:"sample"`
	Seed   int64    `json:"seed"`
}

func (v Verifier) verify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	b, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(b) > MaxBody || json.Unmarshal(b, &req) != nil || req.Sample < 0 || req.Sample > 1000 {
		v.fail(w, errs.New(errs.EINVAL, "The body must be {laws, sample (0 to 1000), seed}."))
		return
	}
	for _, l := range req.Laws {
		if !verifyLaws[l] {
			v.fail(w, errs.New(errs.EINVAL, "Verify can't check %s.", l))
			return
		}
	}
	rep := verify.Run(r.Context(), v.Target, verify.Options{Laws: req.Laws, Sample: req.Sample, Seed: req.Seed})
	v.ok(w, rep)
}

func (v Verifier) ok(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SB-Release", v.Release)
	_ = json.NewEncoder(w).Encode(body)
}

func (v Verifier) fail(w http.ResponseWriter, e *errs.Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errs.HTTP(e.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": string(e.Code), "message": e.Msg, "remedy": errs.Remedy(e.Code)}})
}
