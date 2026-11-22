package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

const notesManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
questions_to: group:answerers
notes: true
`

// S16.2 through a running cell (SB-1704, SB-1705): a question nothing answers goes to the
// company's answerers; one of them replies and publishes the reply as a note; after the
// next sync the asker's question is answered from the note, with a cite.
func TestAnswerOnceThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, notesManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}

	r.signIn(t, "ann")
	ann := r.idp.User
	const q = "What is the zebra shipping cutoff?"
	code, body := r.do(t, "POST", "/v1/ask", `{"question":"`+q+`"}`)
	var a struct {
		Trace  string
		Notice *string
	}
	_ = json.Unmarshal([]byte(body), &a)
	if code != 200 || a.Notice == nil || *a.Notice != "no_sources" {
		t.Fatalf("first ask: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/questions", `{"question":"`+q+`","trace":"`+a.Trace+`"}`); code != 200 || !strings.Contains(body, `"to":"company"`) {
		t.Fatalf("send: %d %s", code, body)
	}

	r.signIn(t, "sam")
	sam := r.idp.User
	if code, _ := r.do(t, "GET", "/v1/spaces/company/questions", ""); code != http.StatusForbidden {
		t.Fatalf("a non-answerer saw the queue: %d", code)
	}
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"answerers","members":[{"value":"`+sam+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("sam's sign-in: %d", code)
	}
	code, body = r.do(t, "GET", "/v1/spaces/company/questions", "")
	var queue struct {
		Items []struct{ ID, Question, State string }
	}
	_ = json.Unmarshal([]byte(body), &queue)
	if code != 200 || len(queue.Items) != 1 || queue.Items[0].Question != q || strings.Contains(body, ann) {
		t.Fatalf("queue: %d %s", code, body)
	}
	id := queue.Items[0].ID
	base := "/v1/spaces/company/questions/" + id
	if code, body := r.do(t, "POST", base+"/reply", `{"text":"The zebra shipping cutoff is 3pm on weekdays.","cites":["docs:nope@1"]}`); code != 400 {
		t.Fatalf("a cite the steward can't open was accepted: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", base+"/reply", `{"text":"The zebra shipping cutoff is 3pm on weekdays.","cites":[]}`); code != 200 {
		t.Fatalf("reply: %d %s", code, body)
	}
	code, body = r.do(t, "POST", base+"/note", `{"title":"Zebra shipping cutoff"}`)
	if code != 200 || !strings.Contains(body, `"readers":2`) || !strings.Contains(body, `"narrower":0`) || strings.Contains(body, `"action"`) && !strings.Contains(body, `"action":""`) {
		t.Fatalf("preview: %d %s", code, body)
	}
	code, body = r.do(t, "POST", base+"/note", `{"title":"Zebra shipping cutoff","confirm":0}`)
	if code != 200 || !strings.Contains(body, `"state":"Ran"`) || !strings.Contains(body, "@sha256:") {
		t.Fatalf("publish: %d %s", code, body)
	}
	if b, err := os.ReadFile(share + "/notes/company/" + id + ".md"); err != nil || !strings.Contains(string(b), "3pm on weekdays") {
		t.Fatalf("the note on the share: %q %v", b, err)
	}

	if fails := svc.SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	r.idp.User = ann
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatalf("ann's sign-in: %d", code)
	}
	if !r.asks(t, q, "notes/company/"+id+".md") {
		t.Fatal("the next ask isn't answered from the note with a cite")
	}
	code, body = r.do(t, "GET", "/v1/questions", "")
	if code != 200 || !strings.Contains(body, `"state":"answered"`) || !strings.Contains(body, "3pm") || !strings.Contains(body, `"note":true`) {
		t.Fatalf("ann's questions: %d %s", code, body)
	}
}
