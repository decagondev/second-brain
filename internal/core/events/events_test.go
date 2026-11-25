package events

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// SB-4404: a body of ids only, signed so a receiver can check it, retried with backoff,
// and delivered once.
func TestEventsAreSignedRetriedAndSentOnce(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	var bodies []string
	fails := 1
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !Verify(key, b, r.Header.Get("X-SB-Timestamp"), r.Header.Get("X-SB-Signature"), now) {
			t.Errorf("a body the receiver can't verify: %s", b)
		}
		if fails > 0 {
			fails--
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		bodies = append(bodies, string(b))
	}))
	defer srv.Close()
	var slept []time.Duration
	sent := Memory{}
	j := Job{URL: srv.URL, Kinds: []string{"decision", "action_due"}, Sent: sent, Now: func() time.Time { return now },
		Sleep: func(d time.Duration) { slept = append(slept, d) },
		Key:   func(context.Context) ([]byte, error) { return key, nil },
		Sources: map[string]Source{
			"decision": func(context.Context, time.Time) ([]Event, error) {
				return []Event{{Space: "ops", ID: "docs:a.md", At: now}}, nil
			},
			"goal_at_risk": func(context.Context, time.Time) ([]Event, error) {
				t.Fatal("a kind not chosen was read")
				return nil, nil
			},
		}}
	res, err := j.Run(context.Background())
	if err != nil || res.Sent != 1 || len(slept) != 1 || slept[0] != Backoff[0] {
		t.Fatalf("run: %+v %v %v", res, err, slept)
	}
	if bodies[0] != `{"kind":"decision","space":"ops","id":"docs:a.md","at":"2026-11-25T09:00:00Z"}` {
		t.Fatalf("body: %s", bodies[0])
	}
	if res, err := j.Run(context.Background()); err != nil || res.Sent != 0 || len(bodies) != 1 {
		t.Fatalf("an event went twice: %+v %v", res, err)
	}

	// A receiver that never answers 2xx: three tries, then the next run tries again.
	fails = 100
	j.Sources["decision"] = func(context.Context, time.Time) ([]Event, error) {
		return []Event{{Space: "ops", ID: "docs:b.md", At: now}}, nil
	}
	slept = nil
	if res, err := j.Run(context.Background()); err == nil || res.Failed != 1 || len(slept) != 2 || sent.Delivered("decision|ops|docs:b.md") {
		t.Fatalf("a failing receiver: %+v %v %v", res, err, slept)
	}

	// No key, or a short one: nothing is sent.
	j.Key = func(context.Context) ([]byte, error) { return nil, errors.New("vault") }
	if _, err := j.Run(context.Background()); err == nil {
		t.Fatal("ran without a key")
	}
	j.Key = func(context.Context) ([]byte, error) { return []byte("short"), nil }
	if _, err := j.Run(context.Background()); err == nil {
		t.Fatal("ran with a short key")
	}
	if Verify(key, []byte("{}"), "1", Sign(key, []byte("{}"), time.Unix(1, 0)), now) {
		t.Fatal("an old timestamp verified")
	}
}
