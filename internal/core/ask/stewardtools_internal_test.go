package ask

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// SB-2501: a mark covers its version only, for review_after_days, and until a later report.
func TestReviewMarkCovers(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	m := ReviewMark{Version: 3, At: now.Add(-24 * time.Hour)}
	c := ports.Chunk{Version: 3}
	for _, tc := range []struct {
		name string
		m    ReviewMark
		ok   bool
		c    ports.Chunk
		flag time.Time
		want bool
	}{
		{"marked", m, true, c, time.Time{}, true},
		{"no mark", m, false, c, time.Time{}, false},
		{"a new version", m, true, ports.Chunk{Version: 4}, time.Time{}, false},
		{"a report before the mark", m, true, c, m.At.Add(-time.Minute), true},
		{"a report after the mark", m, true, c, m.At.Add(time.Minute), false},
		{"too old", ReviewMark{Version: 3, At: now.Add(-31 * 24 * time.Hour)}, true, c, time.Time{}, false},
	} {
		if got := covers(tc.m, tc.ok, tc.c, tc.flag, 30, now); got != tc.want {
			t.Errorf("%s: covers %v, want %v", tc.name, got, tc.want)
		}
	}
}

// SB-2502: a lapsed claim, or any claim with no directory to check it, shows as none.
func TestClaimViewLapses(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	x := Stewards{Claimant: nil}
	if v, err := x.claimView(t.Context(), tenancy.Scope{}, &StewardClaim{By: "sam", Until: now.Add(time.Minute)}, nil, now); v != nil || err != nil {
		t.Fatalf("a claim shown with no directory: %+v %v", v, err)
	}
	x.Claimant = func(_ context.Context, u string, _ []string) (string, bool, error) {
		return u + "@acme.example", true, nil
	}
	if v, _ := x.claimView(t.Context(), tenancy.Scope{}, &StewardClaim{By: "sam", Until: now}, nil, now); v != nil {
		t.Fatalf("a lapsed claim shown: %+v", v)
	}
}
