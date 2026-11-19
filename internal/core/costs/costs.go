// Package costs is the monthly cost statement (sb-tenant(5), Model billing and cost
// statements; SB-709): counts and prices only, never content or user ids.
package costs

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"time"
)

// Usage is one provider and model's metered tokens in a period.
type Usage struct {
	Provider, Model     string
	TokensIn, TokensOut int
}

// Meter reads metered usage (the provider's reported tokens, SB-609).
type Meter interface {
	Usage(ctx context.Context, tenant string, from, to time.Time) ([]Usage, error)
}

// Price is the release's price per million tokens for one provider and model.
type Price struct{ InPerM, OutPerM float64 }

// Line is one provider and model in a statement.
type Line struct {
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	InPerM    float64 `json:"price_in_per_m"`
	OutPerM   float64 `json:"price_out_per_m"`
	CostUSD   float64 `json:"cost_usd"`
}

// Statement is one tenant's month.
type Statement struct {
	Tenant    string  `json:"tenant"`
	Month     string  `json:"month"`
	Partial   bool    `json:"partial"`
	Lines     []Line  `json:"lines"`
	TotalUSD  float64 `json:"total_usd"`
	BudgetUSD int     `json:"budget_usd_month"`
}

// ParseMonth reads "2027-03" and returns its UTC bounds.
func ParseMonth(s string) (time.Time, time.Time, error) {
	t, err := time.Parse("2006-01", s)
	if err != nil || len(s) != 7 {
		return time.Time{}, time.Time{}, fmt.Errorf("costs: month must be like 2027-03")
	}
	return t, t.AddDate(0, 1, 0), nil
}

func cents(v float64) float64 { return math.Round(v*100) / 100 }

// Build makes the statement for month. A usage line without a price is an error: a statement
// never shows a cost it can't justify.
func Build(ctx context.Context, m Meter, tenant, month string, prices map[string]Price, budget int, now time.Time) (Statement, error) {
	from, to, err := ParseMonth(month)
	if err != nil {
		return Statement{}, err
	}
	us, err := m.Usage(ctx, tenant, from, to)
	if err != nil {
		return Statement{}, err
	}
	st := Statement{Tenant: tenant, Month: month, Partial: now.Before(to), BudgetUSD: budget, Lines: []Line{}}
	for _, u := range us {
		p, ok := prices[u.Provider+"/"+u.Model]
		if !ok {
			return Statement{}, fmt.Errorf("costs: no price for %s/%s in the release", u.Provider, u.Model)
		}
		l := Line{Provider: u.Provider, Model: u.Model, TokensIn: u.TokensIn, TokensOut: u.TokensOut, InPerM: p.InPerM, OutPerM: p.OutPerM,
			CostUSD: cents(float64(u.TokensIn)/1e6*p.InPerM + float64(u.TokensOut)/1e6*p.OutPerM)}
		st.Lines = append(st.Lines, l)
		st.TotalUSD += l.CostUSD
	}
	st.TotalUSD = cents(st.TotalUSD)
	return st, nil
}

// CSV writes the statement's lines with a header row.
func (s Statement) CSV() ([]byte, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"tenant", "month", "partial", "provider", "model", "tokens_in", "tokens_out", "price_in_per_m", "price_out_per_m", "cost_usd"})
	for _, l := range s.Lines {
		_ = w.Write([]string{s.Tenant, s.Month, strconv.FormatBool(s.Partial), l.Provider, l.Model, strconv.Itoa(l.TokensIn), strconv.Itoa(l.TokensOut),
			strconv.FormatFloat(l.InPerM, 'f', -1, 64), strconv.FormatFloat(l.OutPerM, 'f', -1, 64), strconv.FormatFloat(l.CostUSD, 'f', 2, 64)})
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
