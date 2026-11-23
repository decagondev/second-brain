package retention

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// spaced is a store whose records carry a space.
type spaced struct{ recs []srec }

type srec struct {
	space string
	at    time.Time
}

func (*spaced) Name() string { return "traces" }
func (s *spaced) Expire(_ context.Context, _ string, before time.Time, held Held) (int, int, error) {
	n, k := 0, 0
	var keep []srec
	for _, r := range s.recs {
		switch {
		case !r.at.Before(before):
			keep = append(keep, r)
		case held.Is(r.space, ""):
			keep, k = append(keep, r), k+1
		default:
			n++
		}
	}
	s.recs = keep
	return n, k, nil
}
func (*spaced) Erase(context.Context, string, string) (int, error) { return 0, nil }
func (s *spaced) Plan(_ context.Context, _ string, before time.Time, held Held) (int, int, error) {
	n, k := 0, 0
	for _, r := range s.recs {
		switch {
		case !r.at.Before(before):
		case held.Is(r.space, ""):
			k++
		default:
			n++
		}
	}
	return n, k, nil
}
func (*spaced) Export(context.Context, string, string) ([]json.RawMessage, error) { return nil, nil }

// SB-1401: a space that keeps less ages out sooner; other spaces keep the tenant's days.
func TestSpaceRetentionIsEnforced(t *testing.T) {
	now := time.Date(2027, 6, 1, 3, 0, 0, 0, time.UTC)
	st := &spaced{recs: []srec{
		{"spc_finance", now.Add(-100 * 24 * time.Hour)}, // older than finance's 90, younger than 365
		{"spc_people", now.Add(-100 * 24 * time.Hour)},
		{"", now.Add(-100 * 24 * time.Hour)}, // unknown space: the tenant's days
	}}
	j := Job{Tenant: "acme-logistics", Days: 365, Stores: []Store{st}, Audit: &auditor{}, Now: func() time.Time { return now },
		SpaceDays: map[string]int{"spc_finance": 90}}
	if _, err := j.Sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(st.recs) != 2 || st.recs[0].space != "spc_people" || st.recs[1].space != "" {
		t.Fatalf("left %+v", st.recs)
	}
}
