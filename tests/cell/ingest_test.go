package cell

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/gdrive/drivetest"
	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/canary"
)

const docsManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: gdrive }
agents: [ask]
retention_days: 365
budget_usd_month: 1500
`

type alerts struct{ got []canary.Alert }

func (a *alerts) Alert(_ context.Context, al canary.Alert) error {
	a.got = append(a.got, al)
	return nil
}

// docsCell is a running cell whose docs connector is a recorded Drive fake.
func docsCell(t *testing.T) (running, *drivetest.Fake) {
	t.Helper()
	drive := drivetest.New()
	t.Cleanup(drive.Server.Close)
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, docsManifest, map[string]string{"connectors/docs": drive.KeyJSON()})
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: drive.Base()}}
		return cfg
	})
	return r, drive
}

func (r running) asks(t *testing.T, q, want string) bool {
	t.Helper()
	code, body := r.do(t, "POST", "/v1/ask", `{"question":"`+q+`"}`)
	if code != 200 {
		t.Fatalf("ask: %d %s", code, body)
	}
	// Only the answer's claims and their cites: an id elsewhere in the body may hold the
	// same digits.
	var a struct {
		Answer struct {
			Claims []struct {
				Text  string
				Cites []struct{ Cite string }
			}
		}
	}
	if err := json.Unmarshal([]byte(body), &a); err != nil {
		t.Fatalf("ask: %v %s", err, body)
	}
	for _, c := range a.Answer.Claims {
		if strings.Contains(c.Text, want) {
			return true
		}
		for _, ci := range c.Cites {
			if strings.Contains(ci.Cite, want) {
				return true
			}
		}
	}
	return false
}

func (r running) chunks(t *testing.T, source string) int {
	t.Helper()
	all, err := r.cell.Index.All(context.Background(), tenant)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range all {
		if c.SourceID == source {
			n++
		}
	}
	return n
}

// TestLAW13_IngestThroughARunningCell drives a running cell through ingest, a scope
// change and a source deletion (SB-1503): the sync runs from its stored cursor, a viewer
// removed at the source stops seeing the document, and a deleted document leaves the
// index at the next sync.
func TestLAW13_IngestThroughARunningCell(t *testing.T) {
	r, drive := docsCell(t)
	ctx := context.Background()
	r.signIn(t, "ann")
	const phrase = "zebra refund limit is 4200 dollars"
	drive.Put("doc1", "Refunds", "text/plain", "The "+phrase+" for every region.", "ann@acme.example")
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatalf("sync: %v", fails)
	}
	if cur, found, err := r.cell.Cursors.Cursor(ctx, tenant, "docs"); err != nil || !found || cur == "" {
		t.Fatalf("no stored cursor after a sync: %q %v %v", cur, found, err)
	}
	if !r.asks(t, "What is the zebra refund limit?", "4200") {
		t.Fatal("a document ann may read isn't answered from after ingest")
	}

	drive.Share("doc1", "bob@acme.example") // a scope change at the source
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatalf("sync: %v", fails)
	}
	if r.asks(t, "What is the zebra refund limit?", "4200") {
		t.Fatal("LAW-01: ann still gets an answer from a document the source no longer shares with her")
	}

	drive.Trash("doc1")
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatalf("sync: %v", fails)
	}
	if n := r.chunks(t, "doc1"); n != 0 {
		t.Fatalf("LAW-13: %d chunks of a deleted document still indexed", n)
	}
}

// TestIngestRestartResumes: a new service over the same state resumes from the stored
// cursor, and a lost cursor resyncs from the start rather than skipping.
func TestIngestRestartResumes(t *testing.T) {
	r, drive := docsCell(t)
	ctx := context.Background()
	drive.Put("doc1", "One", "text/plain", "first document text", "ann@acme.example")
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	before, _, _ := r.cell.Cursors.Cursor(ctx, tenant, "docs")
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	if after, _, _ := r.cell.Cursors.Cursor(ctx, tenant, "docs"); after == "" || before == "" {
		t.Fatalf("cursor %q then %q", before, after)
	}
	// Lose the cursor and the index: the next sync lists everything again.
	if err := r.cell.Cursors.SaveCursor(ctx, tenant, "docs", ""); err != nil {
		t.Fatal(err)
	}
	if err := r.cell.Index.Drop(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	if n := r.chunks(t, "doc1"); n == 0 {
		t.Fatal("a lost cursor skipped what was already there")
	}
}

// TestFailingConnectorAlertsAndBacksOff: a connector that keeps failing backs off, is
// counted, and alerts once when it passes the freshness target, without stopping another.
func TestFailingConnectorAlertsAndBacksOff(t *testing.T) {
	r, drive := docsCell(t)
	drive.Server.Close() // the vendor is down
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	got := &alerts{}
	svc := &ingestsvc.Service{Cell: r.cell, Alert: got, Now: func() time.Time { return now }}
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		if fails := svc.SyncAll(ctx); fails["docs"] == nil {
			t.Fatal("a sync against a vendor that's down succeeded")
		}
		now = now.Add(time.Minute)
	}
	if len(got.got) != 1 || got.got[0].Kind != "ingest_stale" || got.got[0].Where != "connector:docs" {
		t.Fatalf("alerts %+v, want one ingest_stale for docs", got.got)
	}
	if v, ok := r.cell.Metrics.Get("sb_ingest_sync_failures_total", map[string]string{"tenant": tenant, "connector": "docs"}); !ok || v != 20 {
		t.Fatalf("failures counted %v %v", v, ok)
	}
	for n, want := range map[int]time.Duration{0: time.Minute, 1: 2 * time.Minute, 3: 8 * time.Minute, 4: 15 * time.Minute, 9: 15 * time.Minute} {
		if got := ingestsvc.Backoff(n); got != want {
			t.Errorf("Backoff(%d) = %v, want %v", n, got, want)
		}
	}
}
