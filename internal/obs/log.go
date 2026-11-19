// Package obs is the platform's structured logging (SB-205). Every logger is built on a
// Redactor: values registered as secrets are replaced by [REDACTED] wherever they appear
// in a record (message, attribute, error, nested group), and attributes named like a
// secret are redacted whatever their value (LAW-07).
package obs

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Redacted replaces every secret in log output.
const Redacted = "[REDACTED]"

var secretName = regexp.MustCompile(`(?i)(secret|password|token|key|authorization|cookie|credential)`)

// Redactor holds the secret values seen so far.
type Redactor struct {
	mu     sync.RWMutex
	values []string // longest first, so overlapping secrets redact fully
}

// NewRedactor returns an empty redactor.
func NewRedactor() *Redactor { return &Redactor{} }

// Register adds a secret value. Very short values are refused as too likely to collide
// with ordinary text; the vault never holds such secrets.
func (r *Redactor) Register(v string) {
	if len(v) < 6 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.values {
		if x == v {
			return
		}
	}
	r.values = append(r.values, v)
	// The same secret, encoded the ways a careless caller might log it (SB-612, PT-0001).
	for _, e := range encodings(v) {
		if e != v && !contains(r.values, e) {
			r.values = append(r.values, e)
		}
	}
	sort.Slice(r.values, func(i, j int) bool { return len(r.values[i]) > len(r.values[j]) })
}

// Scrub returns s with every registered secret replaced.
func (r *Redactor) Scrub(s string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.values {
		s = strings.ReplaceAll(s, v, Redacted)
	}
	return s
}

// NewLogger writes JSON lines to w through the redactor.
func NewLogger(w io.Writer, r *Redactor) *slog.Logger {
	return slog.New(&handler{next: slog.NewJSONHandler(w, nil), r: r})
}

type handler struct {
	next slog.Handler
	r    *Redactor
}

func (h *handler) Enabled(ctx context.Context, l slog.Level) bool { return h.next.Enabled(ctx, l) }

func (h *handler) Handle(ctx context.Context, rec slog.Record) error {
	out := slog.NewRecord(rec.Time, rec.Level, h.r.Scrub(rec.Message), rec.PC)
	rec.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(h.attr(a))
		return true
	})
	return h.next.Handle(ctx, out)
}

func (h *handler) WithAttrs(as []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(as))
	for i, a := range as {
		clean[i] = h.attr(a)
	}
	return &handler{next: h.next.WithAttrs(clean), r: h.r}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{next: h.next.WithGroup(name), r: h.r}
}

// attr redacts one attribute, recursing into groups. Every value is rendered to a string
// first, so a secret inside an error, a struct or a Stringer can't slip through.
func (h *handler) attr(a slog.Attr) slog.Attr {
	if secretName.MatchString(a.Key) {
		return slog.String(a.Key, Redacted)
	}
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindGroup:
		as := v.Group()
		clean := make([]any, len(as))
		for i, x := range as {
			clean[i] = h.attr(x)
		}
		return slog.Group(a.Key, clean...)
	case slog.KindString:
		return slog.String(a.Key, h.r.Scrub(v.String()))
	case slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool, slog.KindDuration, slog.KindTime:
		return slog.Attr{Key: a.Key, Value: v}
	case slog.KindAny, slog.KindLogValuer:
		return slog.String(a.Key, h.r.Scrub(fmt.Sprint(v.Any())))
	}
	return slog.String(a.Key, h.r.Scrub(v.String()))
}

func encodings(v string) []string {
	b := []byte(v)
	return []string{url.QueryEscape(v), url.PathEscape(v), base64.StdEncoding.EncodeToString(b), base64.URLEncoding.EncodeToString(b),
		base64.RawStdEncoding.EncodeToString(b), base64.RawURLEncoding.EncodeToString(b), hex.EncodeToString(b), strings.ToUpper(hex.EncodeToString(b))}
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
