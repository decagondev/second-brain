package cell

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/core/ingest"
)

// S16.4 through a running cell (SB-1708, SB-1709): a new document is in what changed; the
// weekly drill's rebuild doesn't put it back after it ages out; a non-steward gets no
// review list.
func TestWhatChangedThroughARunningCell(t *testing.T) {
	r, drive := docsCell(t)
	ctx := context.Background()
	r.signIn(t, "ann")
	drive.Put("doc1", "Depot rota", "text/plain", "The depot rota is set every Friday.", "ann@acme.example")
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	code, body := r.do(t, "GET", "/v1/changes?since=7d", "")
	if code != 200 || !strings.Contains(body, "docs:doc1@") {
		t.Fatalf("changes: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/v1/changes?since=400d", ""); code != http.StatusBadRequest {
		t.Fatalf("a window past 90 days: %d", code)
	}
	if rep, err := ingest.Drill(ctx, svc.Pipeline("docs"), time.Now); err != nil || !rep.Equal {
		t.Fatalf("drill: %+v %v", rep, err)
	}
	all, _ := r.cell.Index.All(ctx, tenant)
	for _, c := range all {
		if c.Rebuilt || c.Seen.IsZero() {
			t.Fatalf("the drill's rebuild lost a first-seen time: %+v", c.Cite)
		}
	}
	if code, _ := r.do(t, "GET", "/v1/spaces/support/review", ""); code != http.StatusForbidden {
		t.Fatalf("a review list for no steward of no space: %d", code)
	}
}
