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
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// S55.1 through a running cell (SB-5601 to SB-5603, LAW-06). The main provider answers every
// call with a server error. Until the fallback passes the release's eval gate, an ask gets
// the passages Sam can open and no answer, the health report says no model could answer and
// the notice banner says why. Once the gate names the release, the fallback answers and says
// so. The same question again reuses that answer without a model call; Ask fresh doesn't.
func TestKeepAnsweringThroughARunningCell(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(down.Close)
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	for p, text := range map[string]string{"Ops/dock.md": "The quokka dock opens at seven.", "Ops/.acl": "read domain:acme.example\n"} {
		if err := os.WriteFile(filepath.Join(share, p), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
`
	var state, release string
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		cfg.Model.Provider, cfg.Model.Base, cfg.Model.ID, cfg.Model.Local = "openaicompat", down.URL, "main-model", true
		cfg.Model.Fallback.Provider = "extractive"
		state = cfg.State
		return cfg
	})
	release = r.cell.Release
	ctx := context.Background()
	r.signIn(t, "sam")
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	type answer struct {
		Notice   *string
		Fallback bool
		Reused   string
		Passages []struct{ Cite string }
		Answer   struct{ Claims []struct{ Text string } }
	}
	askIt := func(body string) answer {
		t.Helper()
		code, out := r.do(t, "POST", "/v1/ask", body)
		var a answer
		if err := json.Unmarshal([]byte(out), &a); err != nil || code != 200 {
			t.Fatalf("ask: %d %s", code, out)
		}
		return a
	}

	// SB-5602: no gated model, so passages only.
	a := askIt(`{"question":"When does the quokka dock open?"}`)
	if a.Notice == nil || *a.Notice != "model_unavailable" || len(a.Answer.Claims) != 0 || len(a.Passages) != 1 || !strings.Contains(a.Passages[0].Cite, "dock.md") || a.Fallback {
		t.Fatalf("passages when down: %+v", a)
	}
	if h, err := r.cell.Health(ctx, time.Now()); err != nil || h.Model == nil || !h.Model.Down || h.Model.Gated {
		t.Fatalf("health: %+v %v", h.Model, err)
	}
	if code, out := r.do(t, "GET", "/v1/notice", ""); code != 200 || !strings.Contains(out, "model is unavailable") {
		t.Fatalf("banner: %d %s", code, out)
	}

	// SB-5601: once the gate names this release, the fallback answers and says so.
	if err := os.MkdirAll(filepath.Dir(wire.FallbackGatePath(state)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wire.FallbackGatePath(state), []byte(release+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	a = askIt(`{"question":"When does the quokka dock open?"}`)
	if !a.Fallback || a.Notice != nil || len(a.Answer.Claims) == 0 {
		t.Fatalf("fallback: %+v", a)
	}
	if h, _ := r.cell.Health(ctx, time.Now()); h.Model == nil || h.Model.Down || !h.Model.Gated {
		t.Fatalf("health after: %+v", h.Model)
	}

	// SB-5603: the same question again is reused; Ask fresh asks again.
	if a := askIt(`{"question":"when does the quokka dock open"}`); a.Reused == "" || len(a.Answer.Claims) == 0 {
		t.Fatalf("not reused: %+v", a)
	}
	if a := askIt(`{"question":"When does the quokka dock open?","fresh":true}`); a.Reused != "" {
		t.Fatalf("ask fresh reused: %+v", a)
	}
}
