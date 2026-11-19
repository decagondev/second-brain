package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

type memCursors map[string]ports.Cursor

func (m memCursors) Cursor(_ context.Context, t, c string) (ports.Cursor, bool, error) {
	v, ok := m[t+"/"+c]
	return v, ok, nil
}
func (m memCursors) SaveCursor(_ context.Context, t, c string, v ports.Cursor) error {
	m[t+"/"+c] = v
	return nil
}

// feed is a connector whose change feed is a list; Changes(cursor) returns the rest.
type feed struct {
	changes []ports.Change
	failAt  int // Fetch call number that fails (0 = never)
	fetches int
}

func (f *feed) APIVersion() int { return ports.ConnectorAPIVersion }
func (f *feed) Fetch(_ context.Context, ref ports.SourceRef) (ports.Document, error) {
	f.fetches++
	if f.fetches == f.failAt {
		return ports.Document{}, errors.New("source down")
	}
	return ports.Document{Ref: ref, Version: 1, Title: ref.ID, Text: "text " + ref.ID}, nil
}
func (f *feed) ACL(context.Context, ports.UserRef, []ports.SourceRef) ([]ports.Verdict, error) {
	return nil, nil
}
func (f *feed) Changes(_ context.Context, c ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	from := len(c)
	return ports.ChangeSet(f.changes[from:]), ports.Cursor(make([]byte, len(f.changes))), nil
}
func (f *feed) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, errors.New("ENOTSUP")
}

type memIndex struct{ n map[string]int }

func (m *memIndex) Indexed(_ context.Context, _ string, r ports.SourceRef) (int, bool, error) {
	v, ok := m.n[r.ID]
	return v, ok, nil
}
func (m *memIndex) Replace(_ context.Context, _ string, r ports.SourceRef, v int, _ []ports.Chunk) error {
	m.n[r.ID] = v
	return nil
}
func (m *memIndex) Delete(context.Context, string, ports.SourceRef) error { return nil }
func (m *memIndex) All(context.Context, string) ([]ports.Chunk, error)    { return nil, nil }
func (m *memIndex) Drop(context.Context, string) error                    { return nil }

type flatEmbed struct{}

func (flatEmbed) Model() string { return "flat" }
func (flatEmbed) Embed(_ context.Context, ts []string) ([][]float32, error) {
	return make([][]float32, len(ts)), nil
}

type rec struct{ ok, failed int }

func (r *rec) SyncSucceeded(string, string, time.Time) { r.ok++ }
func (r *rec) SyncFailed(string, string)               { r.failed++ }

// A failed sync keeps the old cursor, and the next run finishes the work.
func TestCursorAdvancesOnlyAfterSuccess(t *testing.T) {
	f := &feed{changes: []ports.Change{{Ref: ports.SourceRef{Kind: "docs", ID: "a"}}, {Ref: ports.SourceRef{Kind: "docs", ID: "b"}}}, failAt: 2}
	cur := memCursors{}
	idx := &memIndex{n: map[string]int{}}
	r := &rec{}
	s := Syncer{Pipeline: Pipeline{Tenant: "acme-logistics", Connector: f, Embedder: flatEmbed{}, Index: idx},
		Connector: "docs", Cursors: cur, Recorder: r, Now: time.Now}
	if err := s.Once(context.Background()); err == nil {
		t.Fatal("want the injected failure")
	}
	if _, saved := cur["acme-logistics/docs"]; saved || r.failed != 1 {
		t.Fatalf("cursor saved after a failed sync: %v, failures %d", cur, r.failed)
	}
	if err := s.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(idx.n) != 2 || r.ok != 1 || len(cur["acme-logistics/docs"]) != 2 {
		t.Fatalf("after resume: index %v, ok %d, cursor %q", idx.n, r.ok, cur["acme-logistics/docs"])
	}
}
