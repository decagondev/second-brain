package costs

import "testing"

// SB-5606: what each answer saved adds up by why, and a month with none says so.
func TestSavedAddsUp(t *testing.T) {
	s := AddSaved("2026-11", []map[string]float64{{"reuse": 0.5, "cache": 0.25}, {"batch": 1, "tier": 0.004}, {"cache": 0.25}})
	if s.ReuseUSD != 0.5 || s.CacheUSD != 0.5 || s.BatchUSD != 1 || s.TierUSD != 0 || s.TotalUSD != 2 || s.Note != "" {
		t.Fatalf("%+v", s)
	}
	if s.Line() != "Saved $2.00 this month: reuse $0.50, batching $1.00, caching $0.50, economy tier $0.00" {
		t.Fatal(s.Line())
	}
	if n := AddSaved("2026-10", nil); n.TotalUSD != 0 || n.Note != NothingSaved || n.Line() != NothingSaved {
		t.Fatalf("%+v", n)
	}
}
