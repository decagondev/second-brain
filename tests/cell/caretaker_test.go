package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-3405, SB-3407 through a running cell: a steward names an active member caretaker of
// a document whose owner is gone, and who to ask shows them until they are deprovisioned;
// the review list suggests a term two documents define and the glossary lacks.
func TestCaretakersAndDefinedTermsThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	files := map[string]string{
		"policy.md":     "Spend above 500 needs a purchase order (PO) first.",
		"policy.md.acl": "owner gone@acme.example\nread domain:acme.example\n",
		"rota.md":       "Every PO (purchase order) is checked by finance. The zebra cutoff (ZC) is 3pm.",
		"rota2.md":      "The zebra cutoff (ZC) moves in winter. A goods receipt note (GRN) is filed.",
		".acl":          "read domain:acme.example\n",
	}
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(share, "Ops", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"ops","members":[{"value":"`+sam+`"},{"value":"`+lee+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}

	code, body := r.do(t, "GET", "/v1/spaces/ops/review", "")
	if code != 200 || !strings.Contains(body, `"owner_gone":true`) || !strings.Contains(body, `"usage":{"space":"ops","fewer_than_5":true}`) {
		t.Fatalf("review: %d %s", code, body)
	}
	var rv struct {
		Terms []struct {
			Term, Meaning string
			Docs          int
		}
	}
	_ = json.Unmarshal([]byte(body), &rv)
	if len(rv.Terms) != 1 || rv.Terms[0].Term != "PO" || rv.Terms[0].Meaning != "purchase order" || rv.Terms[0].Docs != 2 {
		t.Fatalf("terms %+v (ZC is in the glossary, GRN in one document)", rv.Terms)
	}

	doc := "docs:Ops/policy.md"
	for _, bad := range []string{`{"doc":"` + doc + `","email":"nobody@acme.example"}`, `{"doc":"docs:Ops/rota.md","email":"lee@acme.example"}`, `{"doc":"docs:Other/x.md","email":"lee@acme.example"}`} {
		if code, body := r.do(t, "PUT", "/v1/spaces/ops/caretakers", bad); code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", bad, code, body)
		}
	}
	if code, body := r.do(t, "PUT", "/v1/spaces/ops/caretakers", `{"doc":"`+doc+`","email":"Lee@acme.example"}`); code != 200 {
		t.Fatalf("name: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/review", ""); code != 200 || !strings.Contains(body, `"owner":{"email":"lee@acme.example","caretaker":true}`) {
		t.Fatalf("review with a caretaker: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/search?space=ops&q=purchase+order+spend", ""); code != 200 || !strings.Contains(body, `"caretaker":true`) {
		t.Fatalf("who to ask: %d %s", code, body)
	}
	r.idp.User = lee
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if code, _ := r.do(t, "DELETE", "/v1/spaces/ops/caretakers?doc="+doc, ""); code != 200 {
		t.Fatalf("a steward removes it: %d", code) // lee is a steward of ops too
	}
	r.idp.User = sam
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if code, body := r.do(t, "PUT", "/v1/spaces/ops/caretakers", `{"doc":"`+doc+`","email":"lee@acme.example"}`); code != 200 {
		t.Fatalf("name again: %d %s", code, body)
	}
	// Deprovisioned, the caretaker is nobody to ask.
	patch := `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"Replace","path":"active","value":false}]}`
	if code, body := r.do(t, "PATCH", "/scim/v2/Users/"+lee, patch, "Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != 200 {
		t.Fatalf("deactivate: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/search?space=ops&q=purchase+order+spend", ""); code != 200 || strings.Contains(body, "lee@") {
		t.Fatalf("a deprovisioned caretaker: %d %s", code, body)
	}
}
