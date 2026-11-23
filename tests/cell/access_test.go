package cell

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// S21.3 through a running cell (SB-2205): a document the source stopped letting sam open
// is counted, never named; asking for access tells its owner which document and who asked,
// once a week, and sam sees counts only.
func TestAskForAccessThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(share, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("rates.md", "Carrier rates for the zebra lane are fixed until March.")
	write("rates.md.acl", "read domain:acme.example\nowner bob@acme.example\n")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, notesManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	bobs := &told{}
	r.cell.Send = bobs
	r.signIn(t, "bob")
	r.signIn(t, "sam")
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	write("rates.md.acl", "read bob@acme.example\nowner bob@acme.example\n") // the source changes; no sync yet

	code, body := r.do(t, "GET", "/v1/search?q=zebra+lane+rates", "")
	var s struct {
		Trace   string
		Refused int
	}
	if err := json.Unmarshal([]byte(body), &s); err != nil || code != 200 || s.Refused != 1 || strings.Contains(body, "rates.md") {
		t.Fatalf("search: %d %s", code, body)
	}
	code, body = r.do(t, "POST", "/v1/access-requests", `{"trace":"`+s.Trace+`"}`)
	if code != 200 || !strings.Contains(body, `"sent":1,"already":0,"no_owner":0`) {
		t.Fatalf("access request: %d %s", code, body)
	}
	if strings.Contains(body, "rates") || strings.Contains(body, "bob") {
		t.Fatalf("the asker learned which document or who: %s", body)
	}
	got := bobs.take()
	if len(got) != 1 || !strings.HasPrefix(got[0], "bob@acme.example: sam@acme.example asked a question that your document ") ||
		!strings.Contains(got[0], "rates") || strings.Contains(got[0], "zebra") {
		t.Fatalf("told: %v", got)
	}
	if code, body := r.do(t, "POST", "/v1/access-requests", `{"trace":"`+s.Trace+`"}`); code != 200 || !strings.Contains(body, `"sent":0,"already":1`) || len(bobs.take()) != 0 {
		t.Fatalf("a second request the same week: %d %s", code, body)
	}
	r.signIn(t, "eve")
	if code, _ := r.do(t, "POST", "/v1/access-requests", `{"trace":"`+s.Trace+`"}`); code == 200 {
		t.Fatal("someone else's trace was accepted")
	}
}
