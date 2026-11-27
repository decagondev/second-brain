package ingest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

type throttle struct{ wait time.Duration }

func (throttle) Error() string               { return "429" }
func (throttle) HTTPStatus() int             { return 429 }
func (t throttle) RetryAfter() time.Duration { return t.wait }

// bigFeed is a first sync of n documents that throttles every tenth fetch once, and can be
// made to stop (a kill) at a fetch.
type bigFeed struct {
	feed
	throttled map[string]bool
	stopAt    int
	seen      map[string]int
}

func (f *bigFeed) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	f.fetches++
	f.seen[ref.ID]++
	if f.fetches == f.stopAt {
		return ports.Document{}, errors.New("killed")
	}
	if n := len(f.throttled); f.fetches%10 == 0 && !f.throttled[ref.ID] && n < 1000 {
		f.throttled[ref.ID] = true
		return ports.Document{}, throttle{wait: 3 * time.Second}
	}
	return ports.Document{Ref: ref, Version: 1, Title: ref.ID, Text: "text " + ref.ID}, nil
}

// SB-6004: a first sync throttled every tenth call finishes with nothing lost, waiting each
// Retry-After at its pace; killed midway, the next run resumes from the last checkpoint and
// reads nothing before it again; progress is reported from the start to the end.
func TestFirstSyncIsPacedPatientAndResumes(t *testing.T) {
	const n = 1200
	f := &bigFeed{throttled: map[string]bool{}, stopAt: 700, seen: map[string]int{}}
	for i := 0; i < n; i++ {
		f.changes = append(f.changes, ports.Change{Ref: ports.SourceRef{Kind: "docs", ID: fmt.Sprintf("d%04d", i)}})
	}
	var slept time.Duration
	clock := time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	pace := &Pace{PerMinute: 600, Now: func() time.Time { return clock },
		Sleep: func(_ context.Context, d time.Duration) error { slept += d; clock = clock.Add(d); return nil }}
	idx := &memIndex{n: map[string]int{}}
	cur := memCursors{}
	var cps []Checkpoint
	var waits int
	s := Syncer{Pipeline: Pipeline{Tenant: "acme-logistics", Connector: f, Embedder: flatEmbed{}, Index: idx, Pace: pace, Throttled: func(string) { waits++ }},
		Connector: "docs", Cursors: cur, Recorder: &rec{}, Now: time.Now,
		Progress: func(_ context.Context, _ string, cp Checkpoint) { cps = append(cps, cp) }}
	if err := s.Once(context.Background()); err == nil {
		t.Fatal("want the kill")
	}
	if cur["acme-logistics/docs--first"] != `{"done":500,"total":1200}` {
		t.Fatalf("checkpoint: %q", cur["acme-logistics/docs--first"])
	}
	before := map[string]int{}
	for k, v := range f.seen {
		before[k] = v
	}
	f.stopAt = 0
	if err := s.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(idx.n) != n {
		t.Fatalf("indexed %d of %d", len(idx.n), n)
	}
	for i := 0; i < 500; i++ {
		id := fmt.Sprintf("d%04d", i)
		if f.seen[id] != before[id] {
			t.Fatalf("%s was read again after the checkpoint", id)
		}
	}
	if cur["acme-logistics/docs--first"] != "" || len(cur["acme-logistics/docs"]) != n {
		t.Fatal("the first sync didn't finish cleanly")
	}
	if waits == 0 || slept < time.Duration(waits)*3*time.Second {
		t.Fatalf("429s weren't waited out: %d waits, slept %v", waits, slept)
	}
	if last := cps[len(cps)-1]; last.Done != n || last.Total != n || cps[0].Done != 0 {
		t.Fatalf("progress: first %+v last %+v", cps[0], last)
	}
	// Paced: about a tenth of a second between reads at 600 a minute.
	if slept < time.Duration(n-500)*100*time.Millisecond {
		t.Fatalf("not paced: slept %v", slept)
	}
	// A source that keeps answering 429 fails after MaxRetries waits, not forever.
	p := Pipeline{Tenant: "acme-logistics", Connector: &alwaysThrottled{}, Embedder: flatEmbed{}, Index: idx, Pace: pace}
	if _, err := p.Sync(context.Background(), "x"); err == nil {
		t.Fatal("an endless 429 succeeded")
	}
}

type alwaysThrottled struct{ feed }

func (*alwaysThrottled) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, throttle{}
}
func (*alwaysThrottled) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return ports.ChangeSet{{Ref: ports.SourceRef{Kind: "docs", ID: "z"}}}, "y", nil
}
