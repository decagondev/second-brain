package spaces_test

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
)

// SB-2705: a document's own review-by date is kept on its chunks, read again on a
// rebuild, and one that isn't a date is ignored and counted.
func TestReviewByDatesAreKept(t *testing.T) {
	ctx := context.Background()
	src := mock.NewSource("docs")
	src.Put("due", "Rota", "---\ntitle: Rota\nreview-by: 2026-01-31\n---\nThe rota changes monthly.", "ana@acme.example")
	src.Put("exp", "Prices", "---\nexpires: \"2027-06-30\"\n---\nPrices.", "ana@acme.example")
	src.Put("bad", "Notes", "---\nreview-by: next spring\n---\nNotes.", "ana@acme.example")
	src.Put("none", "Plain", "No front matter here.\nreview-by: 2026-01-01", "ana@acme.example")
	idx := memindex.New()
	bad := 0
	p := ingest.Pipeline{Tenant: tn, Release: "rel_t", Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: idx,
		Now: func() time.Time { return time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC) }, BadReviewBy: func(string) { bad++ }}
	if _, err := p.Sync(ctx, ""); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		all, _ := idx.All(ctx, tn)
		got := map[string]string{}
		for _, c := range all {
			got[c.SourceID] = ""
			if !c.ReviewBy.IsZero() {
				got[c.SourceID] = c.ReviewBy.Format("2006-01-02")
			}
		}
		if got["due"] != "2026-01-31" || got["exp"] != "2027-06-30" || got["bad"] != "" || got["none"] != "" {
			t.Fatalf("%v", got)
		}
	}
	check()
	if bad != 1 {
		t.Fatalf("ignored %d, want 1", bad)
	}
	if err := p.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}
	check()
}
