package memory

import (
	"context"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// SB-1701: the same version again keeps the time the index first saw it; a new one doesn't.
func TestFirstSeenIsKept(t *testing.T) {
	ctx := context.Background()
	x := New()
	ref := ports.SourceRef{Kind: "docs", ID: "d1"}
	t1, t2 := time.Unix(1_700_000_000, 0), time.Unix(1_800_000_000, 0)
	put := func(v int, at time.Time) time.Time {
		if err := x.Replace(ctx, "acme-logistics", ref, v, []ports.Chunk{{ID: "c", Tenant: "acme-logistics", Version: v, Seen: at}}); err != nil {
			t.Fatal(err)
		}
		all, _ := x.All(ctx, "acme-logistics")
		return all[0].Seen
	}
	put(1, t1)
	if got := put(1, t2); !got.Equal(t1) {
		t.Fatalf("re-ingesting version 1 moved its first-seen time to %v", got)
	}
	if got := put(2, t2); !got.Equal(t2) {
		t.Fatalf("version 2 kept version 1's time: %v", got)
	}
}
