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
	"example.com/second-brain/internal/core/ask"
)

// SB-3701 to SB-3703 through a running cell: a meeting transcript is found and cited by its
// time; a space's decisions are listed with no query; a saved chat thread can be discarded.
func TestCaptureConversationsThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"standup.vtt": "WEBVTT\n\n00:12:40.000 --> 00:12:44.000\n<v Ann>The quokka shipment leaves on Friday.</v>\n\nbroken\n\n",
		"carrier.md":  "# Carrier\n\nSome background.\n\n## Decision\n\nWe move the quokka route to beta-freight.\n",
		"notes.md":    "The quokka notes have no decision.",
		".acl":        "read domain:acme.example\n",
	} {
		if err := os.WriteFile(filepath.Join(share, "Ops", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	r.signIn(t, "sam")
	sam := r.idp.User
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"ops","members":[{"value":"`+sam+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	r.idp.User = sam
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}

	// SB-3701
	code, body := r.do(t, "GET", "/v1/search?space=ops&q=quokka+shipment+Friday&type=transcript", "")
	if code != 200 || !strings.Contains(body, "standup.vtt") || !strings.Contains(body, `### 00:12:40\nAnn: The quokka`) || !strings.Contains(body, "1 cue couldn't be read.") || strings.Contains(body, "carrier.md") {
		t.Fatalf("transcript search: %d %s", code, body)
	}

	// SB-3702
	code, body = r.do(t, "GET", "/v1/search?space=ops&type=decision", "")
	if code != 200 || !strings.Contains(body, "carrier.md") || strings.Contains(body, "notes.md") || strings.Contains(body, "standup.vtt") {
		t.Fatalf("decisions: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/search?type=decision", ""); code != http.StatusBadRequest {
		t.Fatalf("company-wide decisions with no query: %d %s", code, body)
	}

	// SB-3703
	if err := r.cell.Traces.PutStewardQuestion(context.Background(), ask.StewardQuestion{ID: "qs_t", Tenant: "acme-logistics", Asker: sam,
		Question: "ann@acme-logistics.example: When does it leave?", Space: "spc_ops", At: time.Now(),
		Thread: &ask.SavedThread{Link: "https://slack.com/archives/C1/p1", Text: "ann@acme-logistics.example: When does it leave?\nben@acme-logistics.example: Friday."}}); err != nil {
		t.Fatal(err)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/questions", ""); code != 200 || !strings.Contains(body, `"thread":{"by":"`+sam+`","link":"https://slack.com/archives/C1/p1"`) {
		t.Fatalf("saved thread: %d %s", code, body)
	}
	if code, body := r.do(t, "DELETE", "/v1/spaces/ops/questions/qs_t", ""); code != 200 {
		t.Fatalf("discard: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/questions", ""); code != 200 || strings.Contains(body, "qs_t") {
		t.Fatalf("discarded thread still queued: %d %s", code, body)
	}
	if code, _ := r.do(t, "DELETE", "/v1/spaces/ops/questions/qs_t", ""); code != http.StatusBadRequest {
		t.Fatalf("discard twice: %d", code)
	}
}
