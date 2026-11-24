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
	// ForecastUSD is the month-end forecast for the month still running (SB-3505); nil
	// with Forecast "not enough data yet" before 3 days of it, nil for a past month.
	ForecastUSD *float64 `json:"forecast_usd"`
	Forecast    string   `json:"forecast,omitempty"`
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
	if st.Partial && !now.Before(from) {
		f, ok, err := Forecast(ctx, m, tenant, prices, now)
		if err != nil {
			return Statement{}, err
		}
		if ok {
			st.ForecastUSD = &f
		} else {
			st.Forecast = NotEnough
		}
	}
	return st, nil
}

// Month-end forecast (sb-admin(3), costs; SB-3505).
const (
	ForecastDays    = 7 // the rate is the last this many days'
	ForecastMinDays = 3 // fewer days of the month gone give no forecast
	NotEnough       = "not enough data yet"
)

// Forecast is the running month's spend so far plus the last ForecastDays' daily rate
// (fewer in a younger month) times the days left; ok is false before ForecastMinDays.
func Forecast(ctx context.Context, m Meter, tenant string, prices map[string]Price, now time.Time) (float64, bool, error) {
	now = now.UTC()
	from, to, _ := ParseMonth(now.Format("2006-01"))
	gone := now.Sub(from)
	if gone < ForecastMinDays*24*time.Hour {
		return 0, false, nil
	}
	win := time.Duration(ForecastDays) * 24 * time.Hour
	if gone < win {
		win = gone
	}
	sofar, err := spend(ctx, m, tenant, prices, from, now)
	if err != nil {
		return 0, false, err
	}
	recent, err := spend(ctx, m, tenant, prices, now.Add(-win), now)
	if err != nil {
		return 0, false, err
	}
	rate := recent / win.Hours()
	return cents(sofar + rate*to.Sub(now).Hours()), true, nil
}

func spend(ctx context.Context, m Meter, tenant string, prices map[string]Price, from, to time.Time) (float64, error) {
	us, err := m.Usage(ctx, tenant, from, to)
	if err != nil {
		return 0, err
	}
	usd := 0.0
	for _, u := range us {
		p, ok := prices[u.Provider+"/"+u.Model]
		if !ok {
			return 0, fmt.Errorf("costs: no price for %s/%s in the release", u.Provider, u.Model)
		}
		usd += float64(u.TokensIn)/1e6*p.InPerM + float64(u.TokensOut)/1e6*p.OutPerM
	}
	return usd, nil
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
