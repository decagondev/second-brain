package spaces_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

type unlisted struct{ ports.Connector }

func (unlisted) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", errors.New("timeout")
}

// SB-2905: a document deleted at the source with its tombstone missed is gone; one no space
// holds is unheld; a source that can't be listed is an error, never "all gone".
func TestLeftovers(t *testing.T) {
	ctx := context.Background()
	src := mock.NewSource("docs")
	src.PutAt("ops1", "Rota", "The rota.", "/Ops/rota", "", "ana@acme.example")
	src.PutAt("loose", "Loose", "Loose notes.", "/Elsewhere/loose", "", "ana@acme.example")
	src.PutAt("old", "Old", "Old rota.", "/Ops/old", "", "ana@acme.example")
	idx := memindex.New()
	spaces := []ports.Space{{ID: "spc_ops", Tenant: tn, Name: "ops", Label: "internal", Scope: map[string][]string{"docs": {"/Ops/**"}}}}
	p := ingest.Pipeline{Tenant: tn, Release: "rel_t", Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: idx, Kind: "docs",
		Spaces: func(context.Context) ([]ports.Space, error) { return spaces, nil },
		Now:    func() time.Time { return time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC) }}
	if _, err := p.Sync(ctx, ""); err != nil {
		t.Fatal(err)
	}
	src.Delete("old") // the cell never syncs this tombstone
	gone, unheld, err := p.Leftovers(ctx)
	if err != nil || len(gone) != 1 || gone[0].SourceID != "old" || len(unheld) != 1 || unheld[0].SourceID != "loose" {
		t.Fatalf("gone %v unheld %v err %v", gone, unheld, err)
	}
	p.Connector = unlisted{p.Connector}
	if _, _, err := p.Leftovers(ctx); err == nil {
		t.Fatal("a source that can't be listed gave leftovers")
	}
}
