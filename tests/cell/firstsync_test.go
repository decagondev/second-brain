package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// SB-6004 through a running cell: while a connector is in its first sync, health shows how
// far it has come and answers say the source is still indexing; once a sync succeeds both go.
func TestStillIndexingThroughARunningCell(t *testing.T) {
	r := start(t)
	r.signIn(t, "ann")
	kinds := r.cell.Parts().Connectors
	var kind string
	for k := range kinds {
		kind = k
		break
	}
	if kind == "" {
		t.Skip("the fixture has no connector")
	}
	ctx := context.Background()
	if err := r.cell.Syncs.Progress(ctx, kind, 500, 2000, time.Now().Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	ask := func() []string {
		code, body := r.do(t, "POST", "/v1/ask", `{"question":"What is our refund limit?"}`)
		if code != http.StatusOK {
			t.Fatalf("ask: %d %s", code, body)
		}
		var resp struct {
			Indexing []string `json:"notice_indexing"`
		}
		_ = json.Unmarshal([]byte(body), &resp)
		return resp.Indexing
	}
	if got := ask(); len(got) == 0 || got[0] != kind {
		t.Fatalf("notice_indexing: %v, want %s", got, kind)
	}
	st, _, _ := r.cell.Syncs.Sync(ctx, kind)
	if st.First == nil || st.First.Done != 500 {
		t.Fatalf("sync record: %+v", st)
	}
	if p := st.First.Progress(time.Now()); p.LeftMinutes == nil || *p.LeftMinutes < 25 || *p.LeftMinutes > 35 {
		t.Fatalf("time left: %+v", p)
	}
	if err := r.cell.Syncs.Succeeded(ctx, kind, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := ask(); len(got) != 0 {
		t.Fatalf("notice after the first sync: %v", got)
	}
}
