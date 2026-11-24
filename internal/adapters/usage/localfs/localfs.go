// Package localfs is a cell's usage ledger in a file (SB-1504): model tokens by provider
// and model, draft outcomes and budget spend, each with the time it happened, so the
// weekly report, the monthly cost statement and the budget guard read one record. It
// stands in front of the metrics registry: every Add goes on to Next, and the usage
// counters are also appended to ROOT/usage.ndjson. A cell on GCP keeps it in its database.
package localfs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"example.com/second-brain/internal/core/costs"
)

// Counter is the metrics registry (obs.Metrics).
type Counter interface {
	Add(name, help string, labels map[string]string, v float64)
}

// Ledger records usage and forwards every count.
type Ledger struct {
	Root   string
	Tenant string
	Next   Counter
	Now    func() time.Time
	mu     sync.Mutex // guards the file; an entry that can't be written is counted in sb_usage_write_failures_total
}

var _ costs.Meter = (*Ledger)(nil)

type entry struct {
	At       time.Time `json:"at"`
	Kind     string    `json:"kind"` // tokens_in, tokens_out, draft_used, draft_discarded, spend_usd, space_usd, answer_ms
	N        float64   `json:"n"`
	Tenant   string    `json:"tenant"`
	Provider string    `json:"provider,omitempty"`
	Model    string    `json:"model,omitempty"`
	Month    string    `json:"month,omitempty"` // spend_usd: the budget month
	Space    string    `json:"space,omitempty"` // space_usd: the space it ran in, spc_NAME (SB-4105)
}

func (l *Ledger) file() string { return filepath.Join(l.Root, "usage.ndjson") }

// Add implements the gateway's and the drafts' Counter.
func (l *Ledger) Add(name, help string, labels map[string]string, v float64) {
	if l.Next != nil {
		l.Next.Add(name, help, labels, v)
	}
	e := entry{At: l.Now().UTC(), N: v, Tenant: l.Tenant}
	switch {
	case name == "sb_model_tokens_total" && labels["direction"] == "input":
		e.Kind, e.Provider, e.Model = "tokens_in", labels["provider"], labels["model"]
	case name == "sb_model_tokens_total" && labels["direction"] == "output":
		e.Kind, e.Provider, e.Model = "tokens_out", labels["provider"], labels["model"]
	case name == "sb_drafts_total" && labels["outcome"] == "used":
		e.Kind = "draft_used"
	case name == "sb_drafts_total" && labels["outcome"] == "discarded":
		e.Kind = "draft_discarded"
	default:
		return
	}
	l.write(e)
}

// Observe forwards a histogram observation, and records each answer's duration in
// milliseconds for the answer-time watch (SB-3205): a duration, never a question.
func (l *Ledger) Observe(name, help string, labels map[string]string, v float64) {
	if o, ok := l.Next.(interface {
		Observe(name, help string, labels map[string]string, v float64)
	}); ok {
		o.Observe(name, help, labels, v)
	}
	if name == "sb_ask_seconds" {
		l.write(entry{At: l.Now().UTC(), Kind: "answer_ms", N: math.Round(v * 1000), Tenant: l.Tenant})
	}
}

// AnswerMillis are the milliseconds each answer in [from, to) took (SB-3205).
func (l *Ledger) AnswerMillis(_ context.Context, tenant string, from, to time.Time) ([]int, error) {
	var out []int
	err := l.each(tenant, func(e entry) {
		if e.Kind == "answer_ms" && !e.At.Before(from) && e.At.Before(to) {
			out = append(out, int(e.N))
		}
	})
	return out, err
}

func (l *Ledger) write(e entry) {
	if err := l.append(e); err != nil && l.Next != nil {
		l.Next.Add("sb_usage_write_failures_total", "Usage entries that couldn't be written (SB-1504).", map[string]string{"tenant": l.Tenant}, 1)
	}
}

func (l *Ledger) append(e entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(l.Root, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(l.file(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(e)
	_, err = f.Write(append(b, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// each calls f for every entry of tenant. An unreadable line is an error: nothing is ever
// counted from part of the ledger without saying so.
func (l *Ledger) each(tenant string, f func(e entry)) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	fh, err := os.Open(l.file())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = fh.Close() }()
	sc := bufio.NewScanner(fh)
	for line := 1; sc.Scan(); line++ {
		var e entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return fmt.Errorf("usage: line %d: %w", line, err)
		}
		if e.Tenant == tenant {
			f(e)
		}
	}
	return sc.Err()
}

func (l *Ledger) sums(tenant string, from, to time.Time) (map[string]float64, error) {
	out := map[string]float64{}
	err := l.each(tenant, func(e entry) {
		if !e.At.Before(from) && e.At.Before(to) {
			out[e.Kind] += e.N
		}
	})
	return out, err
}

// Tokens implements report.Source's Tokens.
func (l *Ledger) Tokens(_ context.Context, tenant string, from, to time.Time) (int, int, error) {
	s, err := l.sums(tenant, from, to)
	if err != nil {
		return 0, 0, err
	}
	return int(s["tokens_in"]), int(s["tokens_out"]), nil
}

// InputTokens are the input tokens of each model call in [from, to) (SB-2906).
func (l *Ledger) InputTokens(_ context.Context, tenant string, from, to time.Time) ([]int, error) {
	var out []int
	err := l.each(tenant, func(e entry) {
		if e.Kind == "tokens_in" && !e.At.Before(from) && e.At.Before(to) {
			out = append(out, int(e.N))
		}
	})
	return out, err
}

// Drafts implements report.Source's Drafts.
func (l *Ledger) Drafts(_ context.Context, tenant string, from, to time.Time) (int, int, error) {
	s, err := l.sums(tenant, from, to)
	if err != nil {
		return 0, 0, err
	}
	return int(s["draft_used"]), int(s["draft_discarded"]), nil
}

// Usage implements costs.Meter: tokens by provider and model in [from, to).
func (l *Ledger) Usage(_ context.Context, tenant string, from, to time.Time) ([]costs.Usage, error) {
	by := map[[2]string]*costs.Usage{}
	err := l.each(tenant, func(e entry) {
		if e.At.Before(from) || !e.At.Before(to) || (e.Kind != "tokens_in" && e.Kind != "tokens_out") {
			return
		}
		k := [2]string{e.Provider, e.Model}
		if by[k] == nil {
			by[k] = &costs.Usage{Provider: e.Provider, Model: e.Model}
		}
		if e.Kind == "tokens_in" {
			by[k].TokensIn += int(e.N)
		} else {
			by[k].TokensOut += int(e.N)
		}
	})
	if err != nil {
		return nil, err
	}
	out := make([]costs.Usage, 0, len(by))
	for _, u := range by {
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Provider+"/"+out[i].Model < out[j].Provider+"/"+out[j].Model
	})
	return out, nil
}

// Spent implements budget.Meter.
func (l *Ledger) Spent(_ context.Context, tenant, month string) (float64, error) {
	usd := 0.0
	err := l.each(tenant, func(e entry) {
		if e.Kind == "spend_usd" && e.Month == month {
			usd += e.N
		}
	})
	return usd, err
}

// Spend is the budget guard's meter over this ledger (budget.Meter's Add is not the
// metrics Counter's Add, so it is a type of its own).
func (l *Ledger) Spend() *Spend { return &Spend{l: l} }

// Spend implements budget.Meter.
type Spend struct{ l *Ledger }

// Spent implements budget.Meter.
func (s *Spend) Spent(ctx context.Context, tenant, month string) (float64, error) {
	return s.l.Spent(ctx, tenant, month)
}

// Add implements budget.Meter. It returns the write error, so the guard can refuse.
func (s *Spend) Add(_ context.Context, tenant, month string, usd float64) error {
	return s.l.append(entry{At: s.l.Now().UTC(), Kind: "spend_usd", N: usd, Tenant: tenant, Month: month})
}

// AddSpace implements budget.SpaceMeter: one space's share of a call's spend.
func (s *Spend) AddSpace(_ context.Context, tenant, space, month string, usd float64) error {
	return s.l.append(entry{At: s.l.Now().UTC(), Kind: "space_usd", N: usd, Tenant: tenant, Month: month, Space: space})
}

// SpentBySpace implements budget.SpaceMeter.
func (s *Spend) SpentBySpace(_ context.Context, tenant, month string) (map[string]float64, error) {
	out := map[string]float64{}
	err := s.l.each(tenant, func(e entry) {
		if e.Kind == "space_usd" && e.Month == month && e.Space != "" {
			out[e.Space] += e.N
		}
	})
	return out, err
}

// First implements budget.Once with an exclusive create under ROOT, so across every
// service of a cell only the first claim of a key wins (SB-1608).
func (l *Ledger) First(_ context.Context, key string) (bool, error) {
	for _, r := range key {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false, fmt.Errorf("usage: unsafe key %q", key)
		}
	}
	if err := os.MkdirAll(l.Root, 0o700); err != nil {
		return false, err
	}
	f, err := os.OpenFile(filepath.Join(l.Root, "once-"+key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, f.Close()
}
