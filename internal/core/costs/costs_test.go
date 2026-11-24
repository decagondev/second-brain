package costs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type meter []Usage

func (m meter) Usage(context.Context, string, time.Time, time.Time) ([]Usage, error) { return m, nil }

type broken struct{}

func (broken) Usage(context.Context, string, time.Time, time.Time) ([]Usage, error) {
	return nil, errors.New("meter down")
}

var prices = map[string]Price{"anthropic/claude-x": {InPerM: 3, OutPerM: 15}, "groq/llama-y": {InPerM: 0.5, OutPerM: 0.8}}

func TestStatement(t *testing.T) {
	ctx := context.Background()
	m := meter{{"anthropic", "claude-x", 2_000_000, 100_000}, {"groq", "llama-y", 1_000_000, 1_000_000}}
	st, err := Build(ctx, m, "gamma-parts", "2027-03", prices, 1000, time.Date(2027, 4, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if st.Partial || len(st.Lines) != 2 || st.Lines[0].CostUSD != 7.5 || st.Lines[1].CostUSD != 1.3 || st.TotalUSD != 8.8 {
		t.Fatalf("%+v", st)
	}
	csv, _ := st.CSV()
	if !strings.Contains(string(csv), "gamma-parts,2027-03,false,anthropic,claude-x,2000000,100000,3,15,7.50") {
		t.Fatalf("csv:\n%s", csv)
	}
	if st, _ := Build(ctx, m, "gamma-parts", "2027-03", prices, 1000, time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC)); !st.Partial {
		t.Fatal("a running month isn't marked partial")
	}
	if _, err := Build(ctx, meter{{"openrouter", "unknown", 1, 1}}, "gamma-parts", "2027-03", prices, 1000, time.Now()); err == nil {
		t.Fatal("a cost without a price was shown")
	}
	if _, err := Build(ctx, broken{}, "gamma-parts", "2027-03", prices, 1000, time.Now()); err == nil {
		t.Fatal("an unreadable meter gave a statement")
	}
	for _, bad := range []string{"2027-3", "2027-13", "March", ""} {
		if _, _, err := ParseMonth(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// windowed meters usage per hour: perHour input tokens of claude-x each hour of [from, to).
type windowed struct{ perHour int }

func (w windowed) Usage(_ context.Context, _ string, from, to time.Time) ([]Usage, error) {
	return []Usage{{"anthropic", "claude-x", w.perHour * int(to.Sub(from).Hours()), 0}}, nil
}

// SB-3505: a steady rate forecasts the whole month; before 3 days there is no number, and a
// past month has none.
func TestForecast(t *testing.T) {
	ctx := context.Background()
	m := windowed{perHour: 1_000_000 / 24} // about $3 a day
	now := time.Date(2027, 3, 11, 0, 0, 0, 0, time.UTC)
	f, ok, err := Forecast(ctx, m, "gamma-parts", prices, now)
	if err != nil || !ok || f < 92 || f > 93.5 { // 31 days at ~$3
		t.Fatalf("forecast %v %v %v", f, ok, err)
	}
	st, err := Build(ctx, m, "gamma-parts", "2027-03", prices, 100, now)
	if err != nil || st.ForecastUSD == nil || *st.ForecastUSD != f || st.Forecast != "" {
		t.Fatalf("statement %+v %v", st, err)
	}
	early := time.Date(2027, 3, 2, 12, 0, 0, 0, time.UTC)
	if _, ok, _ := Forecast(ctx, m, "gamma-parts", prices, early); ok {
		t.Fatal("a forecast after a day and a half")
	}
	if st, _ := Build(ctx, m, "gamma-parts", "2027-03", prices, 100, early); st.ForecastUSD != nil || st.Forecast != NotEnough {
		t.Fatalf("early statement %+v", st)
	}
	if st, _ := Build(ctx, m, "gamma-parts", "2027-02", prices, 100, now); st.ForecastUSD != nil || st.Forecast != "" {
		t.Fatalf("a past month forecast %+v", st)
	}
	if _, _, err := Forecast(ctx, broken{}, "gamma-parts", prices, now); err == nil {
		t.Fatal("an unreadable meter forecast")
	}
}
