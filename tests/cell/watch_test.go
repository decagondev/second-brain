package cell

import (
	"context"
	"encoding/json"
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

// S31.1 through a running cell (SB-3202 to SB-3204): one document re-read now, a changed
// one gets a new version and a gone one leaves the index, both audited; the audit API
// narrows by person and day; health's lines are gauges on /metrics.
func TestRunWithoutSurprisesThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"right.md": "The zebra cutoff is 3pm on weekdays.", "wrong.md": "The zebra cutoff moved in 2019.",
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
	r.operator(t)

	// SB-3202
	if code, _ := r.do(t, "POST", "/v1/admin/index/refresh", `{"cite":"docs:Ops/right.md","reason":""}`); code != http.StatusBadRequest {
		t.Fatalf("a refresh without a reason: %d", code)
	}
	if code, _ := r.do(t, "POST", "/v1/admin/index/refresh", `{"cite":"wiki:x","reason":"check"}`); code != http.StatusBadRequest {
		t.Fatalf("a kind with no connector: %d", code)
	}
	later := time.Now().Add(time.Hour)
	p := filepath.Join(share, "Ops", "right.md")
	if err := os.WriteFile(p, []byte("The zebra cutoff is 4pm on weekdays."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, later, later); err != nil {
		t.Fatal(err)
	}
	code, body := r.do(t, "POST", "/v1/admin/index/refresh", `{"cite":"docs:Ops/right.md@1","reason":"the owner says it changed"}`)
	if code != 200 || !strings.Contains(body, `"changed":true`) || !strings.Contains(body, `"removed":false`) {
		t.Fatalf("refresh: %d %s", code, body)
	}
	all, _ := r.cell.Index.All(ctx, tenant)
	for _, c := range all {
		if c.SourceID == "Ops/right.md" && !strings.Contains(c.Text, "4pm") {
			t.Fatalf("the refreshed document still has its old text: %s", c.Cite)
		}
	}
	if err := os.Remove(filepath.Join(share, "Ops", "wrong.md")); err != nil {
		t.Fatal(err)
	}
	if code, body := r.do(t, "POST", "/v1/admin/index/refresh", `{"cite":"docs:Ops/wrong.md","reason":"deleted at the source"}`); code != 200 || !strings.Contains(body, `"removed":true`) {
		t.Fatalf("refresh of a gone document: %d %s", code, body)
	}
	all, _ = r.cell.Index.All(ctx, tenant)
	for _, c := range all {
		if c.SourceID == "Ops/wrong.md" {
			t.Fatal("a document the source no longer has is still indexed")
		}
	}

	// SB-3204
	var page struct {
		Items []struct{ Actor, Subject, State, At string }
	}
	_, body = r.do(t, "GET", "/v1/admin/audit?kind=config", "")
	_ = json.Unmarshal([]byte(body), &page)
	actor := ""
	for _, it := range page.Items {
		if it.Subject == "docs:Ops/wrong.md" && it.State == "removed" {
			actor = it.Actor
		}
	}
	if actor == "" {
		t.Fatalf("no audit row for the refresh: %s", body)
	}
	today := time.Now().UTC().Format("2006-01-02")
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	count := func(q string) int {
		t.Helper()
		code, body := r.do(t, "GET", "/v1/admin/audit?"+q, "")
		if code != 200 {
			t.Fatalf("%s: %d %s", q, code, body)
		}
		var p struct{ Items []json.RawMessage }
		_ = json.Unmarshal([]byte(body), &p)
		return len(p.Items)
	}
	everyone := count("")
	_, body = r.do(t, "GET", "/v1/admin/audit?actor="+actor, "")
	var minePage struct{ Items []struct{ Actor string } }
	_ = json.Unmarshal([]byte(body), &minePage)
	if len(minePage.Items) == 0 || count("actor=usr_nobody") != 0 {
		t.Fatalf("actor: %s", body)
	}
	for _, it := range minePage.Items {
		if it.Actor != actor {
			t.Fatalf("actor= kept someone else's row: %+v", it)
		}
	}
	// Each call writes rows of its own, so today's count is at least the earlier total.
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	if count("from="+today+"&to="+today) < everyone || count("from="+tomorrow) != 0 || count("to="+yesterday) != 0 {
		t.Fatal("from and to don't narrow by day")
	}
	if code, _ := r.do(t, "GET", "/v1/admin/audit?from=yesterday", ""); code != http.StatusBadRequest {
		t.Fatalf("a bad date: %d", code)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/audit?actor=nobody@acme.example", ""); code != http.StatusBadRequest {
		t.Fatalf("an email no one has: %d", code)
	}

	// SB-3203
	_, body = r.do(t, "GET", "/metrics", "")
	for _, want := range []string{`sb_health_warning{check="connector:docs"}`, `sb_health_unknown{check="health"} 0`, `sb_health_warning{check="job:anchor"}`} {
		if !strings.Contains(body, want) {
			t.Fatalf("/metrics lacks %s", want)
		}
	}
}
