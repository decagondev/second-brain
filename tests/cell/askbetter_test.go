package cell

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

// SB-3307 through a running cell: the stewards set a space's suggested questions, its
// members read them, and someone outside the space gets EPERM.
func TestSuggestedQuestionsThroughARunningCell(t *testing.T) {
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + t.TempDir()}}
		return cfg
	})
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"ops","members":[{"value":"`+sam+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
	var g struct{ ID string }
	if err := json.Unmarshal([]byte(body), &g); err != nil || code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	r.idp.User = sam
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if code, _ := r.do(t, "PUT", "/v1/spaces/ops/suggested", `{"questions":["1","2","3","4","5","6","7","8","9"]}`); code != http.StatusBadRequest {
		t.Fatalf("a ninth question: %d", code)
	}
	if code, body := r.do(t, "PUT", "/v1/spaces/ops/suggested", `{"questions":[" When is the zebra cutoff? ","Who owns the rota?"]}`); code != 200 {
		t.Fatalf("set: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/suggested", ""); code != 200 || !strings.Contains(body, `"questions":["When is the zebra cutoff?","Who owns the rota?"]`) {
		t.Fatalf("a member reads them: %d %s", code, body)
	}
	r.idp.User = lee
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sign-in: %d", code)
	}
	if code, _ := r.do(t, "GET", "/v1/spaces/ops/suggested", ""); code != http.StatusForbidden {
		t.Fatalf("someone outside the space: %d", code)
	}
	if code, _ := r.do(t, "PUT", "/v1/spaces/ops/suggested", `{"questions":[]}`); code != http.StatusForbidden {
		t.Fatalf("a non-steward sets them: %d", code)
	}
}
