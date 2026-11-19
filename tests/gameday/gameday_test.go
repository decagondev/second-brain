// Package gameday rehearses the disaster-recovery game day end to end on local adapters
// (sb-observability(7), Disaster-recovery game day, exactly; SB-613): back up a cell, lose
// its database and index, restore from the copies, rebuild the index from the sources, and
// score the day.
package gameday

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	"example.com/second-brain/internal/adapters/backup/localfs"
	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
	"example.com/second-brain/internal/core/gameday"
	"example.com/second-brain/internal/core/ingest"
)

const tenant = "acme-logistics"

type cell struct {
	rows    *auditmem.Store
	acts    *actmem.Store
	log     *audit.Log
	idx     *memindex.Index
	p       ingest.Pipeline
	signer  audit.Signer
	targets []backup.Target
	now     time.Time
}

func newCell(t *testing.T, src *mock.Source) *cell {
	signer, _ := audit.NewSigner(tenant, bytes.Repeat([]byte{9}, 32))
	c := &cell{signer: signer, now: time.Date(2027, 3, 2, 2, 0, 0, 0, time.UTC),
		targets: []backup.Target{localfs.Dir{Root: t.TempDir(), Label: "primary", Media: "disk"},
			localfs.Dir{Root: t.TempDir(), Label: "nas", Media: "nas"},
			localfs.Dir{Root: t.TempDir(), Label: "offsite", Media: "object-store", Away: true}}}
	c.lose(src)
	return c
}

// lose replaces the database and the index with empty ones.
func (c *cell) lose(src *mock.Source) {
	c.rows, c.acts, c.idx = auditmem.New(), actmem.New(), memindex.New()
	c.log = &audit.Log{Tenant: tenant, Release: func() string { return "rel_x" }, Store: c.rows, Now: func() time.Time { return c.now }}
	c.p = ingest.Pipeline{Tenant: tenant, Release: "rel_x", Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: c.idx}
}

func (c *cell) work(t *testing.T, n int) {
	for i := 0; i < n; i++ {
		c.now = c.now.Add(time.Minute)
		id := fmt.Sprintf("tc_%d_%d", c.now.Unix(), i)
		if _, err := c.log.Write(context.Background(), audit.Event{Actor: "usr_ann", Kind: "approval", Subject: id, Reason: "ok"}); err != nil {
			t.Fatal(err)
		}
		_ = c.acts.Put(context.Background(), approvals.Action{ID: id, Tenant: tenant, State: approvals.Ran})
	}
}

func (c *cell) backup(t *testing.T) {
	job := backup.Job{Tenant: tenant, Audit: c.rows, Signer: c.signer, Approvals: c.acts, Targets: c.targets, Now: func() time.Time { return c.now }}
	if _, err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func (c *cell) index(t *testing.T) []string {
	cs, err := c.idx.All(context.Background(), tenant)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(cs))
	for i, ch := range cs {
		out[i] = ch.Cite + "|" + strings.Join(ch.ACLPrefilter, ",")
	}
	return out
}

func sources() *mock.Source {
	src := mock.NewSource("docs")
	src.Put("d1", "Freight policy", "Invoices over 10k need two approvals.", "ann@acme.example")
	src.Put("d2", "Regions", "The north region closes on Fridays.", "ann@acme.example", "bob@acme.example")
	return src
}

// rehearse runs the scenario with the last backup `gap` before T0 and returns the score.
func rehearse(t *testing.T, gap time.Duration) (gameday.Score, gameday.Restored) {
	ctx := context.Background()
	src := sources()
	c := newCell(t, src)
	if _, err := c.p.Sync(ctx, ""); err != nil {
		t.Fatal(err)
	}
	before := c.index(t)
	c.work(t, 12)
	c.backup(t)
	lastBacked, _ := c.rows.Range(ctx, 1, 1000)
	c.now = c.now.Add(gap)
	t0 := c.now

	c.lose(src) // the database and the index are gone
	res, err := gameday.Restore(ctx, tenant, c.signer.Key(), c.targets, c.rows, c.acts)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.p.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}
	c.now = t0.Add(40 * time.Minute) // the restored cell passes its smoke test
	restored, _ := c.rows.Range(ctx, 1, 1000)
	chain := fmt.Sprint(restored) == fmt.Sprint(lastBacked)
	var buf bytes.Buffer
	if _, err := audit.Export(ctx, c.rows, c.signer, 1, c.now, &buf); err != nil {
		t.Fatal(err)
	}
	if _, err := audit.Verify(&buf, tenant, c.signer.Key()); err != nil {
		chain = false
	}
	c.work(t, 1) // the next row continues the chain
	buf.Reset()
	_, _ = audit.Export(ctx, c.rows, c.signer, 1, c.now, &buf)
	if _, err := audit.Verify(&buf, tenant, c.signer.Key()); err != nil {
		chain = false
	}
	return gameday.Score{Tenant: tenant, Failure: t0, Answered: t0.Add(40 * time.Minute), LastRow: res.LastAt,
		Chain: chain, Index: gameday.SameIndex(before, c.index(t)), Region: true}, res
}

func TestGameDayPasses(t *testing.T) {
	s, res := rehearse(t, 10*time.Minute)
	if !s.Pass() || res.Rows != 12 || res.Actions != 12 || res.Copy != "primary" {
		t.Fatalf("%+v\n%s", res, s.Markdown())
	}
	if !strings.Contains(s.Markdown(), "### DR game day: passed") {
		t.Fatal(s.Markdown())
	}
}

// With nightly local backups, a failure an hour after the last one misses the RPO: the
// rehearsal measures it honestly.
func TestGameDayMeasuresRPO(t *testing.T) {
	s, _ := rehearse(t, time.Hour)
	if s.Pass() || !strings.Contains(s.Markdown(), "| RPO | 1h0m0s | FAIL |") {
		t.Fatal(s.Markdown())
	}
	s.LastRow = s.Failure.Add(-15 * time.Minute)
	if !s.Pass() {
		t.Fatal("an RPO of exactly 15 minutes must pass")
	}
	s.Answered = s.Failure.Add(4*time.Hour + time.Second)
	if s.Pass() {
		t.Fatal("an RTO over 4 hours passed")
	}
}

func TestRestoreSkipsATamperedCopy(t *testing.T) {
	ctx := context.Background()
	src := sources()
	c := newCell(t, src)
	c.work(t, 5)
	c.backup(t)
	p := filepath.Join(c.targets[0].(localfs.Dir).Root, tenant, c.now.Format("2006-01-02"), "audit.ndjson")
	_ = os.Chmod(p, 0o600)
	b, _ := os.ReadFile(p)
	_ = os.WriteFile(p, bytes.Replace(b, []byte(`"reason":"ok"`), []byte(`"reason":"no"`), 1), 0o600)
	c.lose(src)
	res, err := gameday.Restore(ctx, tenant, c.signer.Key(), c.targets, c.rows, c.acts)
	if err != nil || res.Copy != "nas" || len(res.Skipped) != 1 || res.Skipped[0].Target != "primary" {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := gameday.Restore(ctx, tenant, c.signer.Key(), c.targets, c.rows, c.acts); !errors.Is(err, gameday.ErrNotEmpty) {
		t.Fatalf("a second restore into a full store: %v", err)
	}
}

// Two copies that both verify but differ for the same night: a human chooses.
func TestRestoreRefusesDisagreeingCopies(t *testing.T) {
	ctx := context.Background()
	src := sources()
	c := newCell(t, src)
	c.work(t, 3)
	c.backup(t)
	c.work(t, 2) // a longer, equally valid chain
	var buf bytes.Buffer
	_, _ = audit.Export(ctx, c.rows, c.signer, 1, c.now, &buf)
	name := tenant + "/" + c.now.Format("2006-01-02") + "/audit.ndjson"
	sum := sha256.Sum256(buf.Bytes())
	other := localfs.Dir{Root: t.TempDir(), Label: "rogue", Media: "disk"}
	_ = other.Put(ctx, name, buf.Bytes(), c.now.Add(time.Hour))
	_ = other.Put(ctx, name+".sha256", []byte(hex.EncodeToString(sum[:])+"\n"), c.now.Add(time.Hour))
	c.lose(src)
	if _, err := gameday.Restore(ctx, tenant, c.signer.Key(), append(c.targets, other), c.rows, c.acts); !errors.Is(err, gameday.ErrDisagree) {
		t.Fatalf("disagreeing copies: %v", err)
	}
	if rs, _ := c.rows.Range(ctx, 1, 10); len(rs) != 0 {
		t.Fatal("rows were restored from disagreeing copies")
	}
}
