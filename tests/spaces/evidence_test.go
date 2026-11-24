package spaces_test

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
)

// SB-3002, SB-3006: every chunk records its section and its lines in the document's own
// text, and a rebuild fills both in again.
func TestSectionsAndLinesAreKept(t *testing.T) {
	ctx := context.Background()
	src := mock.NewSource("docs")
	body := "Intro line.\n\n# Leave\n\n" + strings.Repeat("Leave words go here. ", 57) + "\n\n## Carry-over\n\nUp to five days.\nAsk HR."
	src.Put("hb", "Staff handbook", body, "ana@acme.example")
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: tn, Release: "rel_t", Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: idx,
		Now: func() time.Time { return time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC) }}
	if _, err := p.Sync(ctx, ""); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		all, _ := idx.All(ctx, tn)
		sort.Slice(all, func(i, j int) bool { return all[i].Start < all[j].Start })
		if len(all) < 2 {
			t.Fatalf("%d chunks", len(all))
		}
		first, last := all[0], all[len(all)-1]
		if first.Section != "" || first.Lines[0] != 1 {
			t.Fatalf("first: %q %v", first.Section, first.Lines)
		}
		if last.Section != "Carry-over" || last.Lines[1] != strings.Count(body, "\n")+1 {
			t.Fatalf("last: %q %v (want line %d)", last.Section, last.Lines, strings.Count(body, "\n")+1)
		}
		lines := strings.Split(body, "\n")
		if got := lines[last.Lines[0]-1]; !strings.Contains(last.Text, got) {
			t.Fatalf("line %d is %q, not in the chunk %q", last.Lines[0], got, last.Text)
		}
	}
	check()
	if err := p.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}
	check()
}
