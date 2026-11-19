package laws

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/connectors/gdrive"
	"example.com/second-brain/internal/adapters/connectors/gdrive/drivetest"
	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

// LAW-13 (SB-206): the index is derived. Ingesting a tenant's sources, dropping the
// index and rebuilding it from the sources yields the same (cite, ACL) set; ingesting
// the same versions again changes nothing.

// SourceDoc is one document the law puts in the source.
type SourceDoc struct {
	ID, Text string
	Viewers  []string
}

// IngestWorld is one tenant's source, pipeline and index.
type IngestWorld interface {
	Load(docs []SourceDoc) error // put documents in the source
	Edit(id, text string) error  // a new version of one document
	IngestAll() error            // run the pipeline over every change so far
	Snapshot() ([]string, error) // the index's (cite, chunk id, ACL) entries, any order
	Drop() error                 // drop the index
	Rebuild() error              // replay Fetch for every source document
}

// NewIngestWorld builds the system under test: the real pipeline (SB-206) over the mock
// source, the local embedder and the in-memory index.
var NewIngestWorld = func() IngestWorld {
	src := mock.NewSource("docs")
	idx := memindex.New()
	return &ingestWorld{src: src, idx: idx, p: ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test",
		Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: idx}}
}

type ingestWorld struct {
	src    *mock.Source
	idx    *memindex.Index
	p      ingest.Pipeline
	cursor ports.Cursor
}

func (w *ingestWorld) Load(docs []SourceDoc) error {
	for _, d := range docs {
		w.src.Put(d.ID, "Title "+d.ID, d.Text, d.Viewers...)
	}
	return nil
}

func (w *ingestWorld) Edit(id, text string) error {
	w.src.Put(id, "Title "+id, text, nil...)
	return nil
}

func (w *ingestWorld) IngestAll() error {
	next, err := w.p.Sync(context.Background(), w.cursor)
	w.cursor = next
	return err
}

func (w *ingestWorld) Snapshot() ([]string, error) {
	cs, err := w.idx.All(context.Background(), "acme-logistics")
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Cite + "|" + c.ID + "|" + strings.Join(c.ACLPrefilter, ",")
	}
	return out, err
}

func (w *ingestWorld) Drop() error { return w.idx.Drop(context.Background(), "acme-logistics") }

func (w *ingestWorld) Rebuild() error { return w.p.Rebuild(context.Background()) }

func law13(r *rand.Rand, w IngestWorld) error {
	words := strings.Fields("bonus policy freight invoice approval quarterly region ticket customer contract shipment budget")
	para := func() string {
		n := 3 + r.Intn(60)
		ws := make([]string, n)
		for i := range ws {
			ws[i] = words[r.Intn(len(words))]
		}
		return strings.Join(ws, " ") + "."
	}
	var docs []SourceDoc
	for i := 0; i < 2+r.Intn(5); i++ {
		var ps []string
		for j := 0; j < 1+r.Intn(8); j++ {
			ps = append(ps, para())
		}
		viewers := []string{"ann@acme.example", "bob@acme.example", "grp_finance"}[:1+r.Intn(3)]
		docs = append(docs, SourceDoc{ID: fmt.Sprintf("d%d", i), Text: strings.Join(ps, "\n\n"), Viewers: viewers})
	}
	if err := w.Load(docs); err != nil {
		return err
	}
	if r.Intn(2) == 0 {
		if err := w.Edit(docs[0].ID, docs[0].Text+"\n\n"+para()); err != nil {
			return err
		}
	}
	if err := w.IngestAll(); err != nil {
		return err
	}
	first, err := w.Snapshot()
	if err != nil {
		return err
	}
	if len(first) == 0 {
		return fmt.Errorf("ingest produced an empty index from %d documents", len(docs))
	}
	if err := w.IngestAll(); err != nil {
		return err
	}
	again, _ := w.Snapshot()
	if !sameSet(first, again) {
		return fmt.Errorf("ingesting the same versions again changed the index:\n%v\n%v", first, again)
	}
	if err := w.Drop(); err != nil {
		return err
	}
	if err := w.Rebuild(); err != nil {
		return err
	}
	rebuilt, err := w.Snapshot()
	if err != nil {
		return err
	}
	if !sameSet(first, rebuilt) {
		return fmt.Errorf("rebuild differs from the ingested index:\n%v\n%v", first, rebuilt)
	}
	return nil
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

// NewDriveIngestWorld runs the same law over the Google Drive connector (SB-203).
var NewDriveIngestWorld = func() IngestWorld {
	f := drivetest.New()
	sa := &gdrive.ServiceAccount{Key: func(context.Context) (string, error) { return f.KeyJSON(), nil },
		HTTP: f.Server.Client(), Redact: obs.NewRedactor(), Now: time.Now}
	idx := memindex.New()
	c := &gdrive.Connector{Base: f.Base(), HTTP: f.Server.Client(), Tokens: sa}
	return &driveIngest{f: f, ingestWorld: ingestWorld{idx: idx, p: ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test",
		Connector: c, Embedder: hashing.Embedder{}, Index: idx}}}
}

type driveIngest struct {
	f *drivetest.Fake
	ingestWorld
}

func (w *driveIngest) Load(docs []SourceDoc) error {
	for _, d := range docs {
		w.f.Put(d.ID, "Title "+d.ID, "application/vnd.google-apps.document", d.Text, d.Viewers...)
	}
	return nil
}

func (w *driveIngest) Edit(id, text string) error {
	w.f.Put(id, "Title "+id, "application/vnd.google-apps.document", text, nil...)
	return nil
}

// law13Drill runs the production drill (ingest.Drill, SB-210) over a random corpus.
func law13Drill(r *rand.Rand) error {
	w := NewIngestWorld().(*ingestWorld)
	var docs []SourceDoc
	for i := 0; i < 1+r.Intn(6); i++ {
		docs = append(docs, SourceDoc{ID: fmt.Sprintf("d%d", i), Text: strings.Repeat(fmt.Sprintf("word%d ", r.Intn(50)), 1+r.Intn(400)),
			Viewers: []string{"ann@acme.example", "grp_finance"}[:1+r.Intn(2)]})
	}
	if err := w.Load(docs); err != nil {
		return err
	}
	if err := w.IngestAll(); err != nil {
		return err
	}
	rep, err := ingest.Drill(context.Background(), w.p, time.Now)
	if err != nil {
		return err
	}
	if !rep.Equal || rep.Before == 0 {
		return fmt.Errorf("drill: %+v", rep)
	}
	return nil
}
