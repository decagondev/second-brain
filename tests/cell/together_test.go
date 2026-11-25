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

// SB-5201 to SB-5203 through a running cell. A steward sends a stale document's owner a
// suggested edit, which lands in the owner's own queue; the owner dismisses it and it isn't
// suggested again. The reviewer edits a draft's section in place, rejects a redraft's
// suggestion, and asks an agent from a comment, whose cited reply is kept under the comment.
func TestFixTheSourceThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	write := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Ops/Dept/dock.md", "The quokka dock opens at six every weekday.")
	write("Ops/Dept/dock.md.acl", "read domain:acme.example\nowner bob@acme.example\n")
	write("Ops/Dept/dock-new.md", "From March the quokka dock opens at seven every weekday.")
	write("Ops/Dept/vans.md", "Quokka vans are washed every Friday afternoon.")
	write("Ops/Dept/.acl", "read domain:acme.example\n")
	m := `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  ops:
    members: [group:ops]
    stewards: [group:ops-leads]
    scope: { docs: ["/Ops/Dept/**"] }
`
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ctx := context.Background()
	ids := map[string]string{}
	for _, n := range []string{"bob", "kim"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	for name, who := range map[string][]string{"ops": {"bob", "kim"}, "ops-leads": {"kim"}} {
		var ms []string
		for _, n := range who {
			ms = append(ms, `{"value":"`+ids[n]+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	as := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}

	// SB-5201: kim suggests an edit to bob's stale document.
	as(ids["kim"])
	body := `{"cite":"docs:Ops/Dept/dock.md","why":"stale","evidence":["docs:Ops/Dept/dock-new.md"]}`
	code, out := r.do(t, "POST", "/v1/spaces/ops/review/suggest", body)
	var sent struct{ ID, To, Doc string }
	if err := json.Unmarshal([]byte(out), &sent); err != nil || code != 200 || sent.To != "owner" || strings.Contains(out, "bob@") || strings.Contains(out, "seven") {
		t.Fatalf("suggest: %d %s", code, out)
	}
	if code, _ := r.do(t, "POST", "/v1/spaces/ops/review/suggest", body); code != http.StatusBadRequest {
		t.Fatalf("a second suggestion for the same passage: %d", code)
	}
	if code, _ := r.do(t, "POST", "/v1/spaces/ops/review/suggest", `{"cite":"docs:Ops/Dept/vans.md","why":"stale"}`); code != http.StatusBadRequest {
		t.Fatalf("a document with no owner: %d", code)
	}
	as(ids["bob"])
	code, out = r.do(t, "GET", "/v1/spaces/me/questions", "")
	if code != 200 || !strings.Contains(out, "Suggested edit to") || !strings.Contains(out, `"kind":"suggested-edit"`) || !strings.Contains(out, "opens at seven") {
		t.Fatalf("bob's queue: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/me/suggestions/"+sent.ID+"/dismiss", `{"reason":"The new hours start in March."}`); code != 200 {
		t.Fatalf("dismiss: %d %s", code, out)
	}
	if b, err := os.ReadFile(filepath.Join(share, "Ops/Dept/dock.md")); err != nil || string(b) != "The quokka dock opens at six every weekday." {
		t.Fatalf("the source changed: %q %v", b, err)
	}
	as(ids["kim"])
	if code, out := r.do(t, "POST", "/v1/spaces/ops/review/suggest", body); code != http.StatusBadRequest || !strings.Contains(out, "dismissed") {
		t.Fatalf("a dismissed passage was suggested again: %d %s", code, out)
	}

	// SB-5202, SB-5203: an agent draft kim reviews.
	s, err := r.cell.ScopeAs(ctx, ids["kim"], "spc_ops")
	if err != nil {
		t.Fatal(err)
	}
	sq, err := r.cell.Parts().Stewards.AgentDraft(ctx, s, ask.DraftBrief, "Weekly brief: ask", ask.Request{Question: ask.BriefQuestion})
	if err != nil {
		t.Fatal(err)
	}
	base := "/v1/spaces/ops/questions/" + sq.ID
	code, out = r.do(t, "POST", base+"/comments", `{"section":1,"text":"@ask When are the quokka vans washed?"}`)
	if code != 200 || !strings.Contains(out, `"reply":{"agent":"ask"`) || !strings.Contains(out, "docs:Ops/Dept/vans.md@") {
		t.Fatalf("an @agent comment: %d %s", code, out)
	}
	code, out = r.do(t, "POST", base+"/changes", `{"note":"Shorter."}`)
	var rd ask.RedraftResponse
	if err := json.Unmarshal([]byte(out), &rd); err != nil || code != 200 || len(rd.Suggestions) == 0 || rd.Suggestions[0].State != ask.SuggestionOpen {
		t.Fatalf("changes: %d %s", code, out)
	}
	if strings.Contains(rd.Text, "When are the quokka vans washed?") && !strings.Contains(sq.Thread.Text, "When are the quokka vans washed?") {
		t.Fatalf("the comment's reply became part of the draft: %s", rd.Text)
	}
	before := rd.Suggestions[0].Old
	code, out = r.do(t, "POST", base+"/suggestions/1/reject", "")
	if code != 200 || !strings.Contains(out, `"state":"rejected"`) {
		t.Fatalf("reject: %d %s", code, out)
	}
	cur, _, err := r.cell.Traces.GetStewardQuestion(ctx, "acme-logistics", sq.ID)
	if err != nil || ask.DraftSections(cur.Thread.Text)[0] != before {
		t.Fatalf("the rejected suggestion's text stayed: %v\n%q\n%q", err, cur.Thread.Text, before)
	}
	if code, out := r.do(t, "PUT", base+"/sections/1", `{"text":"The quokka dock opens at seven."}`); code != 200 || !strings.Contains(out, `"editors":["`+ids["kim"]+`"]`) {
		t.Fatalf("edit a section: %d %s", code, out)
	}
	as(ids["bob"])
	if code, _ := r.do(t, "PUT", base+"/sections/1", `{"text":"Bob's words."}`); code != http.StatusForbidden && code != http.StatusBadRequest {
		t.Fatalf("someone else edited the draft: %d", code)
	}
}
