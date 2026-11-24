package cell

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-3207 through a running cell: an operator sets the notice, anyone signed in reads it,
// and clearing it leaves {}.
func TestNoticeThroughARunningCell(t *testing.T) {
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + t.TempDir()}}
		return cfg
	})
	r.operator(t)
	until := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	if code, _ := r.do(t, "PUT", "/v1/admin/notice", fmt.Sprintf(`{"text":"see https://x.example","until":%q}`, until)); code != http.StatusBadRequest {
		t.Fatalf("a link: %d", code)
	}
	if code, body := r.do(t, "PUT", "/v1/admin/notice", fmt.Sprintf(`{"text":"Upgrade tonight at 18:00.","until":%q}`, until)); code != 200 {
		t.Fatalf("set: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/notice", ""); code != 200 || !strings.Contains(body, `"text":"Upgrade tonight at 18:00."`) {
		t.Fatalf("read: %d %s", code, body)
	}
	if code, body := r.do(t, "DELETE", "/v1/admin/notice", ""); code != 200 || !strings.Contains(body, `"cleared":true`) {
		t.Fatalf("clear: %d %s", code, body)
	}
	if _, body := r.do(t, "GET", "/v1/notice", ""); strings.TrimSpace(body) != "{}" {
		t.Fatalf("after clearing: %s", body)
	}
}
