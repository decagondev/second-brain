package cell

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// S28.2 through a running cell (SB-2905, SB-2906): a document deleted at the source whose
// tombstone was missed is a leftover; health counts it once counted; prune removes it with
// a reason and touches nothing the source still lists.
func TestIndexLeftoversThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"right.md": "The zebra cutoff is 3pm on weekdays.", "wrong.md": "The zebra cutoff moved in 2019; see the new rota.",
		".acl": "read domain:acme.example\n"} {
		if err := os.WriteFile(filepath.Join(share, "Ops", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	// Deleted at the source, and no sync since: the index still has it.
	if err := os.Remove(filepath.Join(share, "Ops", "wrong.md")); err != nil {
		t.Fatal(err)
	}
	r.operator(t)
	code, body := r.do(t, "GET", "/v1/admin/index/leftovers", "")
	if code != 200 || !strings.Contains(body, `"gone":["docs:Ops/wrong.md@`) || strings.Contains(body, "right.md") {
		t.Fatalf("leftovers: %d %s", code, body)
	}
	if err := r.cell.CountLeftovers(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, body := r.do(t, "GET", "/v1/admin/health", ""); !strings.Contains(body, `"gone":1`) || !strings.Contains(body, `"prompt":{`) {
		t.Fatalf("health doesn't count the leftover: %s", body)
	}
	if code, _ := r.do(t, "POST", "/v1/admin/index/prune", `{"reason":""}`); code != http.StatusBadRequest {
		t.Fatalf("a prune without a reason: %d", code)
	}
	if code, body := r.do(t, "POST", "/v1/admin/index/prune", `{"reason":"tombstone missed"}`); code != 200 || !strings.Contains(body, `"pruned":1`) {
		t.Fatalf("prune: %d %s", code, body)
	}
	if _, body := r.do(t, "GET", "/v1/admin/index/leftovers", ""); !strings.Contains(body, `"gone":[]`) {
		t.Fatalf("still a leftover after the prune: %s", body)
	}
	all, err := r.cell.Index.All(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}
	right := 0
	for _, c := range all {
		if strings.Contains(c.SourceID, "wrong.md") {
			t.Fatalf("pruned document still indexed: %s", c.Cite)
		}
		if strings.Contains(c.SourceID, "right.md") {
			right++
		}
	}
	if right == 0 {
		t.Fatal("the prune touched a document its source still lists")
	}
}
