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
	"example.com/second-brain/internal/core/ask"
)

// SB-3705 to SB-3707 through a running cell: an asker asks a document's owner, who alone
// sees and answers it; a steward drafts a note from a term and is warned it repeats a
// document before publishing.
func TestFillItInOnceThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	for name, text := range map[string]string{
		"rates.md":     "Carrier rates for the zebra lane are fixed until March.",
		"rates.md.acl": "read domain:acme.example\nowner bob@acme.example\n",
	} {
		if err := os.WriteFile(filepath.Join(share, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, notesManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	as := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	r.signIn(t, "bob")
	bob := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"answerers","members":[{"value":"`+sam+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	as(sam)
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
		t.Fatal(fails)
	}

	// SB-3706
	code, out := r.do(t, "POST", "/v1/ask", `{"question":"What are the zebra lane carrier rates?"}`)
	var a ask.Response
	if err := json.Unmarshal([]byte(out), &a); err != nil || code != 200 || len(a.People) == 0 || a.People[0].Email != "bob@acme.example" {
		t.Fatalf("ask: %d %s", code, out)
	}
	if code, body := r.do(t, "POST", "/v1/questions", `{"question":"Until when exactly?","trace":"`+a.Trace+`","to":"nobody@acme.example"}`); code != http.StatusBadRequest {
		t.Fatalf("someone the answer doesn't name: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/questions", `{"question":"Until when exactly?","trace":"`+a.Trace+`","to":"bob@acme.example"}`); code != 200 || !strings.Contains(body, `"to":"bob@acme.example"`) {
		t.Fatalf("ask the owner: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/me/questions", ""); code != 200 || strings.Contains(body, "Until when") {
		t.Fatalf("the asker sees the owner's queue: %d %s", code, body)
	}
	as(bob)
	code, body := r.do(t, "GET", "/v1/spaces/me/questions", "")
	var queue struct {
		Items []struct{ ID, Question string }
	}
	if err := json.Unmarshal([]byte(body), &queue); err != nil || code != 200 || len(queue.Items) != 1 || queue.Items[0].Question != "Until when exactly?" || strings.Contains(body, sam) {
		t.Fatalf("the owner's queue: %d %s", code, body)
	}
	id := queue.Items[0].ID
	if code, body := r.do(t, "POST", "/v1/spaces/me/questions/"+id+"/reply", `{"text":"Until 31 March."}`); code != 200 {
		t.Fatalf("owner reply: %d %s", code, body)
	}
	if code, _ := r.do(t, "POST", "/v1/spaces/me/questions/"+id+"/note", `{"title":"Rates"}`); code != http.StatusBadRequest {
		t.Fatalf("an owner's reply published as a note: %d", code)
	}
	as(sam)
	if code, body := r.do(t, "GET", "/v1/questions", ""); code != 200 || !strings.Contains(body, "Until 31 March.") {
		t.Fatalf("the asker's reply: %d %s", code, body)
	}

	// SB-3705, SB-3707
	if code, _ := r.do(t, "POST", "/v1/spaces/company/drafts", `{"term":"zebra","count":0}`); code != http.StatusBadRequest {
		t.Fatalf("a draft with no count: %d", code)
	}
	code, body = r.do(t, "POST", "/v1/spaces/company/drafts", `{"term":"zebra","count":2}`)
	var d struct{ ID string }
	if err := json.Unmarshal([]byte(body), &d); err != nil || code != 200 || d.ID == "" {
		t.Fatalf("draft: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/company/questions", ""); code != 200 || !strings.Contains(body, `"draft":{"title":"zebra","comment":"Asked 2 times this week, found nothing"}`) {
		t.Fatalf("the draft in the queue: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/company/questions/"+d.ID+"/reply", `{"text":"Carrier rates for the zebra lane are fixed until March."}`); code != 200 {
		t.Fatalf("reply: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/company/questions/"+d.ID+"/note", `{"title":"Zebra rates"}`); code != 200 || !strings.Contains(body, `"similar":[{"cite":"docs:rates.md@`) {
		t.Fatalf("already written: %d %s", code, body)
	}

	// A deprovisioned owner can't be asked (LAW-12).
	patch := `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"Replace","path":"active","value":false}]}`
	if code, body := r.do(t, "PATCH", "/scim/v2/Users/"+bob, patch, "Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != 200 {
		t.Fatalf("deactivate: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/questions", `{"question":"And after March?","trace":"`+a.Trace+`","to":"bob@acme.example"}`); code != http.StatusBadRequest {
		t.Fatalf("a deprovisioned owner asked: %d %s", code, body)
	}
}
