package notify

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ask"
)

// SB-3704: the stewards' line adds the space's most asked terms, counts only.
func TestDigestTrends(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	j, s, c := world(now)
	c.weekly["weekly"] = Monday(now, nil).AddDate(0, 0, -7)
	j.Questions = questions{}
	j.Clock.(*clock).t = now.Add(-time.Minute)
	j.Digest, j.Zone = true, time.UTC
	j.Askers = func(context.Context, time.Time, time.Time) ([]string, error) { return nil, nil }
	j.Changes = func(context.Context, string, int) ([]string, int, error) { return nil, 0, nil }
	j.Trends = func(context.Context, time.Time, time.Time) (map[string][]ask.TermCount, error) {
		return map[string][]ask.TermCount{"spc_finance": {{Term: "cutoff", Count: 14}, {Term: "PO", Count: 9}}}, nil
	}
	if _, err := j.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.got) != 1 || s.got[0].msg.Text != "finance: 0 answered this week, 0 still open, asked most: cutoff 14, PO 9" {
		t.Fatalf("digests: %+v", s.got)
	}
}
