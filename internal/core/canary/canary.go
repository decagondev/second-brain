// Package canary is the secret canaries and the scanner (sb-observability(7), Canaries and
// the secret scanner; SB-411). A finding names where and which canary, never the value,
// and is a Sev-1 alert.
package canary

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Canary is one planted value.
type Canary struct {
	ID    string // cny_… for canaries, sec_… for resolved secrets
	Value string
}

// New makes a canary for tenant from 16 random bytes.
func New(tenant string, rnd io.Reader) (Canary, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rnd, b); err != nil {
		return Canary{}, err
	}
	v := "sbcanary-" + tenant + "-" + hex.EncodeToString(b)
	return Canary{ID: "cny_" + short(v), Value: v}, nil
}

func short(v string) string {
	h := sha256.Sum256([]byte(v))
	return hex.EncodeToString(h[:4])
}

// Finding is one leak: where it was seen, and which canary or secret, by id only.
type Finding struct {
	Where string
	Line  int
	ID    string
}

// Scanner knows every canary and every resolved secret of a cell.
type Scanner struct {
	mu    sync.RWMutex
	known []Canary
}

// Add learns a canary.
func (s *Scanner) Add(c Canary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.known {
		if k.Value == c.Value {
			return
		}
	}
	s.known = append(s.known, c)
}

// Register learns a resolved secret, so the scanner is a secrets.Registrar too. Very
// short values are ignored, as the log redactor does.
func (s *Scanner) Register(v string) {
	if len(v) < 6 {
		return
	}
	s.Add(Canary{ID: "sec_" + short(v), Value: v})
}

// Scan reports every known value in text, line by line.
func (s *Scanner) Scan(where, text string) []Finding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Finding
	for i, line := range strings.Split(text, "\n") {
		for _, k := range s.known {
			if strings.Contains(line, k.Value) {
				out = append(out, Finding{Where: where, Line: i + 1, ID: k.ID})
			}
		}
	}
	return out
}

// Alert is what the alert webhook receives: ids only.
type Alert struct {
	Severity string `json:"severity"` // "sev1"
	Kind     string `json:"kind"`     // "secret_leak"
	Tenant   string `json:"tenant"`
	Where    string `json:"where"`
	Ref      string `json:"ref"` // the canary or secret id
}

// Alerter sends alerts (the webhook adapter in production).
type Alerter interface {
	Alert(ctx context.Context, a Alert) error
}

// Watch scans a stream of text and raises a Sev-1 alert per finding.
type Watch struct {
	Tenant  string
	Scanner *Scanner
	Alerts  Alerter
}

// Check scans text seen at where; it returns the findings after alerting on each.
func (w *Watch) Check(ctx context.Context, where, text string) []Finding {
	fs := w.Scanner.Scan(where, text)
	for _, f := range fs {
		if w.Alerts != nil {
			_ = w.Alerts.Alert(ctx, Alert{Severity: "sev1", Kind: "secret_leak", Tenant: w.Tenant, Where: fmt.Sprintf("%s:%d", f.Where, f.Line), Ref: f.ID})
		}
	}
	return fs
}

// Writer passes writes through to W and checks each one (for log streams).
type Writer struct {
	W     io.Writer
	Watch *Watch
	Where string
}

func (w Writer) Write(p []byte) (int, error) {
	w.Watch.Check(context.Background(), w.Where, string(p))
	return w.W.Write(p)
}
