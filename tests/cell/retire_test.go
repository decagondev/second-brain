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

// S27.1 through a running cell (SB-2801): a steward retires a document from the space's
// answers, the replacement's cite says what it replaces, and Bring back undoes it.
func TestRetireThroughARunningCell(t *testing.T) {
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
	r.signIn(t, "sam")
	sam := r.idp.User
	code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"ops","members":[{"value":"`+sam+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
	if code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}
	ask := func() string {
		t.Helper()
		code, body := r.do(t, "POST", "/v1/ask?space=ops", `{"question":"When is the zebra cutoff?"}`)
		if code != 200 {
			t.Fatalf("ask: %d %s", code, body)
		}
		return body
	}
	body = ask()
	wrong, right := cite(t, body, "wrong.md"), cite(t, body, "right.md")
	doc, _, _ := strings.Cut(wrong, "@")
	newDoc, _, _ := strings.Cut(right, "@")

	if code, body := r.do(t, "POST", "/v1/spaces/ops/retired", `{"cite":"`+wrong+`","reason":""}`); code != http.StatusBadRequest {
		t.Fatalf("a retirement without a reason: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/retired", `{"cite":"docs:nope.md@1","reason":"x"}`); code != http.StatusForbidden {
		t.Fatalf("a retirement of a document the space doesn't hold: %d %s", code, body)
	}
	code, body = r.do(t, "POST", "/v1/spaces/ops/retired", `{"cite":"`+wrong+`","reason":"Moved to the new rota.","replaced_by":"`+newDoc+`"}`)
	var ret struct{ Document, Reason, ReplacedBy string }
	if err := json.Unmarshal([]byte(body), &ret); err != nil || code != 200 || ret.Document != doc {
		t.Fatalf("retire: %d %s", code, body)
	}
	body = ask()
	if strings.Contains(body, "wrong.md") {
		t.Fatalf("a retired document is still cited in its space: %s", body)
	}
	if !strings.Contains(body, `"replaces":"`) {
		t.Fatalf("the replacement's cite doesn't say what it replaces: %s", body)
	}
	if _, body := r.do(t, "GET", "/v1/spaces/ops/retired", ""); !strings.Contains(body, `"document":"`+doc+`"`) || !strings.Contains(body, "Moved to the new rota.") {
		t.Fatalf("the retirement isn't listed: %s", body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/retired/restore", `{"document":"`+doc+`"}`); code != 200 || !strings.Contains(body, `"restored":true`) {
		t.Fatalf("restore: %d %s", code, body)
	}
	if code, _ := r.do(t, "POST", "/v1/spaces/ops/retired/restore", `{"document":"`+doc+`"}`); code != http.StatusForbidden {
		t.Fatalf("a second restore: %d", code)
	}
	if body := ask(); !strings.Contains(body, "wrong.md") {
		t.Fatalf("a document brought back isn't cited again: %s", body)
	}

	// SB-2903: the scope report, for operators only, counts what each pattern matches.
	if code, _ := r.do(t, "GET", "/v1/admin/index/scopes", ""); code != http.StatusForbidden {
		t.Fatalf("a non-operator read the scope report: %d", code)
	}
	r.operator(t)
	if code, body := r.do(t, "GET", "/v1/admin/index/scopes", ""); code != 200 ||
		!strings.Contains(body, `{"space":"ops","kind":"docs","pattern":"/Ops/**","matches":2,"of":2,"problem":"too_wide"}`) {
		t.Fatalf("scope report: %d %s", code, body)
	}
}
