package retrieve

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

type conn struct {
	answer func(ref ports.SourceRef) ports.Verdict
	delay  time.Duration
	err    error
	calls  atomic.Int32
	mu     sync.Mutex
	sizes  []int
}

func (c *conn) APIVersion() int { return ports.ConnectorAPIVersion }
func (c *conn) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, nil
}
func (c *conn) ACL(ctx context.Context, _ ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	c.calls.Add(1)
	c.mu.Lock()
	c.sizes = append(c.sizes, len(refs))
	c.mu.Unlock()
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	out := make([]ports.Verdict, len(refs))
	for i, r := range refs {
		out[i] = c.answer(r)
	}
	return out, nil
}
func (c *conn) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", nil
}
func (c *conn) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, nil
}

func chunks(kind string, n int) []ports.Chunk {
	out := make([]ports.Chunk, n)
	for i := range out {
		out[i] = ports.Chunk{ID: fmt.Sprintf("chk_%s%d", kind, i), Kind: kind, SourceID: fmt.Sprintf("d%d", i)}
	}
	return out
}

func allow(ports.SourceRef) ports.Verdict { return ports.VerdictAllow }

func TestCheckBatchesPerConnector(t *testing.T) {
	docs := &conn{answer: allow}
	a := Authoriser{Connectors: map[string]ports.Connector{"docs": docs}}
	cs := append(chunks("docs", 450), chunks("docs", 3)...) // duplicates collapse per document
	v := a.Check(context.Background(), ports.UserRef{}, cs)
	if len(v) != 450 || !reflect.DeepEqual(docs.sizes, []int{200, 200, 50}) && !reflect.DeepEqual(docs.sizes, []int{200, 50, 200}) && !reflect.DeepEqual(docs.sizes, []int{50, 200, 200}) {
		t.Fatalf("%d verdicts, batches %v", len(v), docs.sizes)
	}
}

// Every way of not answering leaves a document Unknown.
func TestCheckUnknowns(t *testing.T) {
	cases := map[string]Authoriser{
		"no connector": {Connectors: map[string]ports.Connector{}},
		"error":        {Connectors: map[string]ports.Connector{"docs": &conn{answer: allow, err: errors.New("down")}}},
		"late":         {Connectors: map[string]ports.Connector{"docs": &conn{answer: allow, delay: 2 * ACLDeadline}}},
		"short answer": {Connectors: map[string]ports.Connector{"docs": &short{}}},
		"killed":       {Connectors: map[string]ports.Connector{"docs": &conn{answer: allow}}, Killed: func(k string) bool { return k == "docs" }},
	}
	for name, a := range cases {
		v := a.Check(context.Background(), ports.UserRef{}, chunks("docs", 2))
		for k, x := range v {
			if x != ports.VerdictUnknown {
				t.Errorf("%s: %s is %v", name, k, x)
			}
		}
	}
}

type short struct{ conn }

func (*short) ACL(context.Context, ports.UserRef, []ports.SourceRef) ([]ports.Verdict, error) {
	return []ports.Verdict{ports.VerdictAllow}, nil
}

// No allow is reused across requests (ADR-0002): two requests, two live calls.
func TestNoAllowCaching(t *testing.T) {
	docs := &conn{answer: allow}
	a := Authoriser{Connectors: map[string]ports.Connector{"docs": docs}}
	for i := 0; i < 2; i++ {
		a.Check(context.Background(), ports.UserRef{}, chunks("docs", 1))
	}
	if docs.calls.Load() != 2 {
		t.Fatalf("%d ACL calls for two requests", docs.calls.Load())
	}
}

func TestTokensAndFuse(t *testing.T) {
	s, _ := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_ann", []string{"finance"})
	got := Tokens(s, []string{"Ann@Acme.example"})
	for _, want := range []string{"usr_ann", "ann@acme.example", "domain:acme.example", "finance", "group:finance", "anyone"} {
		found := false
		for _, g := range got {
			found = found || g == want
		}
		if !found {
			t.Errorf("token %q missing from %v", want, got)
		}
	}
	f := Fuse([]ports.Candidate{
		{Chunk: ports.Chunk{ID: "a"}, VectorRank: 5},
		{Chunk: ports.Chunk{ID: "b"}, VectorRank: 2, KeywordRank: 3},
		{Chunk: ports.Chunk{ID: "c"}, KeywordRank: 1},
	})
	if f[0].Chunk.ID != "b" {
		t.Fatalf("both rankings should win: %v", f)
	}
}

// SB-2706: of two candidates with the same score, the newer version comes first.
func TestNewerWinsATie(t *testing.T) {
	old, now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f := Fuse([]ports.Candidate{
		{Chunk: ports.Chunk{ID: "a", Seen: old}, VectorRank: 1, KeywordRank: 2},
		{Chunk: ports.Chunk{ID: "b", Seen: now}, VectorRank: 2, KeywordRank: 1},
		{Chunk: ports.Chunk{ID: "c", Seen: now}, VectorRank: 3},
	})
	if f[0].Chunk.ID != "b" || f[1].Chunk.ID != "a" || f[2].Chunk.ID != "c" {
		t.Fatalf("%v", f)
	}
}
