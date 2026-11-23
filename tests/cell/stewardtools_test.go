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
	"example.com/second-brain/internal/core/ask"
)

// S24.1 through a running cell (SB-2501 to SB-2503): a document marked reviewed leaves the
// list until it is reported again; a claim shows to the other stewards and goes when its
// holder leaves the stewards; a space's templates are its stewards' only.
func TestStewardsToolsThroughARunningCell(t *testing.T) {
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
	scim := func(method, path, body string) {
		t.Helper()
		if code, b := r.do(t, method, path, body, "Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code >= 300 {
			t.Fatalf("SCIM %s %s: %d %s", method, path, code, b)
		}
	}
	signInAs := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	r.signIn(t, "kim")
	kim := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"ops","members":[{"value":"`+sam+`"},{"value":"`+kim+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
	var g struct{ ID string }
	if err := json.Unmarshal([]byte(body), &g); err != nil || code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	signInAs(sam)
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}

	// Reviewed, still right: off the list until a new report.
	report := func() string {
		t.Helper()
		code, body := r.do(t, "POST", "/v1/ask?space=ops", `{"question":"When is the zebra cutoff?"}`)
		var a struct{ Trace string }
		if err := json.Unmarshal([]byte(body), &a); err != nil || code != 200 {
			t.Fatalf("ask: %d %s", code, body)
		}
		wrong := cite(t, body, "wrong.md")
		if code, body := r.do(t, "POST", "/v1/feedback", `{"trace":"`+a.Trace+`","rating":"down","wrong_citation":"`+wrong+`"}`); code != 200 {
			t.Fatalf("feedback: %d %s", code, body)
		}
		return wrong
	}
	wrong := report()
	listed := func() bool {
		t.Helper()
		code, body := r.do(t, "GET", "/v1/spaces/ops/review", "")
		if code != 200 {
			t.Fatalf("review: %d %s", code, body)
		}
		return strings.Contains(body, wrong)
	}
	if !listed() {
		t.Fatal("the reported document isn't on the list")
	}
	doc, _, _ := strings.Cut(wrong, "@")
	if code, body := r.do(t, "POST", "/v1/spaces/ops/review/marks", `{"cite":"`+doc+`@99"}`); code != http.StatusBadRequest {
		t.Fatalf("a mark of a version that isn't current: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/review/marks", `{"cite":"docs:nope.md@1"}`); code != http.StatusForbidden {
		t.Fatalf("a mark of a document the space doesn't hold: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/review/marks", `{"cite":"`+wrong+`"}`); code != 200 {
		t.Fatalf("mark: %d %s", code, body)
	}
	if listed() {
		t.Fatal("a document marked reviewed is still on the list")
	}
	report()
	if !listed() {
		t.Fatal("a document reported after its mark stayed off the list")
	}

	// SB-2702, SB-2703: the mark shows on the cite; an outdated report lists it again, and
	// one for a cite the answer doesn't have is refused.
	if code, body := r.do(t, "POST", "/v1/spaces/ops/review/marks", `{"cite":"`+wrong+`"}`); code != 200 {
		t.Fatalf("mark again: %d %s", code, body)
	}
	_, body = r.do(t, "POST", "/v1/ask?space=ops", `{"question":"When is the zebra cutoff?"}`)
	if !strings.Contains(body, `"cite":"`+wrong+`"`) || !strings.Contains(body, `"reviewed":"`+time.Now().UTC().Format("2006-01-02")+`"`) {
		t.Fatalf("the cite doesn't say it was reviewed: %s", body)
	}
	var a2 struct{ Trace string }
	_ = json.Unmarshal([]byte(body), &a2)
	if code, body := r.do(t, "POST", "/v1/feedback", `{"trace":"`+a2.Trace+`","rating":"down","outdated_citation":"docs:nope.md@1"}`); code != http.StatusBadRequest {
		t.Fatalf("an outdated report of a cite the answer lacks: %d %s", code, body)
	}
	if listed() {
		t.Fatal("still listed after the second mark")
	}
	if code, body := r.do(t, "POST", "/v1/feedback", `{"trace":"`+a2.Trace+`","rating":"down","outdated_citation":"`+wrong+`"}`); code != 200 {
		t.Fatalf("outdated: %d %s", code, body)
	}
	if _, body := r.do(t, "GET", "/v1/spaces/ops/review", ""); !strings.Contains(body, `"outdated":1`) {
		t.Fatalf("an outdated report isn't on the list: %s", body)
	}

	// A claim: kim sees sam's; it goes when sam leaves the stewards. A reply still works.
	// (The test model answers every ask, so the question is put in the store as one sent.)
	if err := r.cell.Traces.PutStewardQuestion(context.Background(), ask.StewardQuestion{ID: "sq_1", Tenant: tenant, Asker: kim,
		Question: "Who signs the payroll forms?", Space: "spc_ops", Trace: "tr_x", At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	_, body = r.do(t, "GET", "/v1/spaces/ops/questions", "")
	id := strings.SplitN(strings.SplitN(body, `"id":"`, 2)[1], `"`, 2)[0]
	claim := "/v1/spaces/ops/questions/" + id + "/claim"
	if code, body := r.do(t, "POST", claim, ""); code != 200 || !strings.Contains(body, `"mine":true`) {
		t.Fatalf("claim: %d %s", code, body)
	}
	signInAs(kim)
	if code, body := r.do(t, "POST", claim, ""); code != 200 || !strings.Contains(body, `"by":"sam@acme.example","mine":false`) {
		t.Fatalf("kim took sam's live claim: %d %s", code, body)
	}
	if _, body := r.do(t, "GET", "/v1/spaces/ops/questions", ""); !strings.Contains(body, `"claim":{"by":"sam@acme.example"`) {
		t.Fatalf("kim's queue doesn't say sam is answering: %s", body)
	}
	scim("PATCH", "/scim/v2/Groups/"+g.ID, `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"remove","path":"members[value eq \"`+sam+`\"]"}]}`)
	if _, body := r.do(t, "GET", "/v1/spaces/ops/questions", ""); strings.Contains(body, `"claim"`) {
		t.Fatalf("the claim of someone no longer a steward still shows: %s", body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/questions/"+id+"/reply", `{"text":"Finance signs them.","cites":[]}`); code != 200 {
		t.Fatalf("a reply to a question someone claimed: %d %s", code, body)
	}

	// Templates: the space's stewards only, at most a name and a text.
	if code, body := r.do(t, "POST", "/v1/spaces/ops/templates", `{"name":"","text":"x"}`); code != http.StatusBadRequest {
		t.Fatalf("a template without a name: %d %s", code, body)
	}
	code, body = r.do(t, "POST", "/v1/spaces/ops/templates", `{"name":"Not ours","text":"Ask finance; they own payroll."}`)
	var tm struct{ ID string }
	if err := json.Unmarshal([]byte(body), &tm); err != nil || code != 200 || !strings.HasPrefix(tm.ID, "rt_") {
		t.Fatalf("add: %d %s", code, body)
	}
	if code, body := r.do(t, "PUT", "/v1/spaces/ops/templates/"+tm.ID, `{"name":"Not ours","text":"Ask finance."}`); code != 200 {
		t.Fatalf("edit: %d %s", code, body)
	}
	if code, body := r.do(t, "PUT", "/v1/spaces/ops/templates/rt_other", `{"name":"x","text":"y"}`); code != http.StatusForbidden {
		t.Fatalf("an edit of a template the space doesn't hold: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/templates", ""); code != 200 || !strings.Contains(body, `"text":"Ask finance."`) {
		t.Fatalf("list: %d %s", code, body)
	}
	signInAs(sam)
	if code, _ := r.do(t, "GET", "/v1/spaces/ops/templates", ""); code != http.StatusForbidden {
		t.Fatalf("someone no longer a steward listed the templates: %d", code)
	}
	signInAs(kim)
	if code, _ := r.do(t, "DELETE", "/v1/spaces/ops/templates/"+tm.ID, ""); code != 200 {
		t.Fatalf("delete: %d", code)
	}
	if code, _ := r.do(t, "DELETE", "/v1/spaces/ops/templates/"+tm.ID, ""); code != http.StatusForbidden {
		t.Fatalf("a second delete: %d", code)
	}
}
