package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/gdrive/drivetest"
	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-1710 through a running cell: an OpenAI-compatible embedder on the cell's own network
// embeds every chunk, and a change of embedder drops the index and resyncs every connector
// from the start, so no two embedders' vectors are ever searched together (LAW-13).
func TestLAW13_EmbedderChangeRebuildsTheIndex(t *testing.T) {
	const dim = 8
	embed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Input []string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		var data []map[string]any
		for i, s := range req.Input {
			v := make([]float32, dim)
			for _, ch := range strings.ToLower(s) {
				v[int(ch)%dim]++
			}
			data = append(data, map[string]any{"index": i, "embedding": v})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer embed.Close()
	drive := drivetest.New()
	t.Cleanup(drive.Server.Close)
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, docsManifest, map[string]string{"connectors/docs": drive.KeyJSON()})
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: drive.Base()}}
		cfg.Embed.Provider, cfg.Embed.Base, cfg.Embed.Model, cfg.Embed.Dim = "openaicompat", embed.URL+"/v1", "nomic-embed", dim
		return cfg
	})
	ctx := context.Background()
	r.signIn(t, "ann")
	drive.Put("doc1", "Refunds", "text/plain", "The zebra refund limit is 4200 dollars.", "ann@acme.example")
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	check := func() {
		t.Helper()
		all, _ := r.cell.Index.All(ctx, tenant)
		if len(all) == 0 {
			t.Fatal("nothing indexed")
		}
		for _, c := range all {
			if len(c.Vector) != dim {
				t.Fatalf("a chunk with a %d-dimension vector", len(c.Vector))
			}
		}
		if !r.asks(t, "What is the zebra refund limit?", "4200") {
			t.Fatal("no answer from the embedded index")
		}
	}
	check()
	file := filepath.Join(r.cell.Config.State, "index", "embedder")
	if b, _ := os.ReadFile(file); strings.TrimSpace(string(b)) != "openaicompat:nomic-embed:8" {
		t.Fatalf("the embedder recorded is %q", b)
	}
	// As if the index had been built by another embedder: the next sync rebuilds it.
	if err := os.WriteFile(file, []byte(ingestsvc.LegacyEmbedder+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if rebuilt, err := svc.EnsureEmbedder(ctx); err != nil || !rebuilt {
		t.Fatalf("rebuilt %v %v", rebuilt, err)
	}
	if all, _ := r.cell.Index.All(ctx, tenant); len(all) != 0 {
		t.Fatalf("%d chunks of the old embedder kept", len(all))
	}
	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	check()
}
