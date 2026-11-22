// Package webhook sends Sev-1 alerts to an operator's webhook (the user's choice for W5
// ops; sb-observability(7)). The body is the alert's ids only, never content.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"example.com/second-brain/internal/core/canary"
)

// Sender posts alerts as JSON.
type Sender struct {
	URL  string // https only
	HTTP *http.Client
}

// Alert implements canary.Alerter. It retries once on a network error or 5xx.
func (s Sender) Alert(ctx context.Context, a canary.Alert) error { return s.Post(ctx, a) }

// Report is the weekly report as it is posted (sb-observability(7), Weekly leadership
// report; SB-1608): the report's Markdown, which holds counts and never content.
type Report struct {
	Kind     string `json:"kind"` // weekly_report
	Tenant   string `json:"tenant"`
	Week     string `json:"week"`
	Markdown string `json:"markdown"`
}

// Post sends v as JSON, retrying once on a network error or 5xx.
func (s Sender) Post(ctx context.Context, v any) error {
	u, err := url.Parse(s.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("webhook: alert URL must be https")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c := s.HTTP
	if c == nil {
		c = &http.Client{Timeout: 5 * time.Second}
	}
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.Do(req)
		if err != nil {
			last = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode/100 == 2 {
			return nil
		}
		last = fmt.Errorf("webhook: HTTP %d", resp.StatusCode)
		if resp.StatusCode/100 != 5 {
			return last
		}
	}
	return last
}

// Memory records alerts (development and tests).
type Memory struct{ Got []canary.Alert }

// Alert implements canary.Alerter.
func (m *Memory) Alert(_ context.Context, a canary.Alert) error { m.Got = append(m.Got, a); return nil }
