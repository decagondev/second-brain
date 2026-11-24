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
	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ask"
)

// SB-3601 to SB-3604 through a running cell: a steward pins a cited reply, and a close ask
// in the space shows it first, until an asker can't open one of its cites; verified only
// answers from marked documents alone; a mark takes its own interval; and unhelpful
// ratings since the mark list the document again.
func TestTrustEachAnswerThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(share, "Ops", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("cutoff.md", "The zebra cutoff is 3pm on weekdays.")
	write("rota.md", "The zebra cutoff rota is kept by the night team.")
	write(".acl", "read domain:acme.example\n")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, opsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	signInAs := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"ops","members":[{"value":"`+sam+`"},{"value":"`+lee+`"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group: %d %s", code, body)
	}
	signInAs(sam)
	sync := func() {
		t.Helper()
		if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(context.Background()); len(fails) > 0 {
			t.Fatal(fails)
		}
	}
	sync()
	asked := func(body string) ask.Response {
		t.Helper()
		code, out := r.do(t, "POST", "/v1/ask?space=ops", body)
		var a ask.Response
		if err := json.Unmarshal([]byte(out), &a); err != nil || code != 200 {
			t.Fatalf("ask: %d %s", code, out)
		}
		return a
	}
	cutoff := ""
	for _, c := range asked(`{"question":"When is the zebra cutoff?"}`).Answer.Claims {
		for _, ci := range c.Cites {
			if strings.HasPrefix(ci.Cite, "docs:Ops/cutoff.md@") {
				cutoff = ci.Cite
			}
		}
	}
	if cutoff == "" {
		t.Fatal("no cite of the cutoff document")
	}

	// SB-3601: a cited reply is pinned; one without cites can't be.
	for id, cites := range map[string][]string{"sq_1": {cutoff}, "sq_2": nil} {
		if err := r.cell.Traces.PutStewardQuestion(context.Background(), ask.StewardQuestion{ID: id, Tenant: "acme-logistics", Asker: lee,
			Question: "When is the zebra cutoff?", Space: "spc_ops", Trace: "tr_x", At: time.Now(),
			Reply: &ask.StewardReply{By: sam, Text: "3pm on weekdays.", Cites: cites, At: time.Now()}}); err != nil {
			t.Fatal(err)
		}
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/pins", `{"question_id":"sq_2"}`); code != http.StatusBadRequest {
		t.Fatalf("a reply without cites pinned: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/pins", `{"question_id":"sq_1","question":"When is the zebra cutoff?"}`); code != 200 || !strings.Contains(body, `"id":"pin_`) {
		t.Fatalf("pin: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/pins", ""); code != 200 || strings.Count(body, `"id":"pin_`) != 1 {
		t.Fatalf("pins: %d %s", code, body)
	}
	signInAs(lee)
	a := asked(`{"question":"When is the zebra cutoff?"}`)
	if a.Pinned == nil || a.Pinned.Text != "3pm on weekdays." || a.Pinned.Verified != time.Now().UTC().Format(time.DateOnly) || len(a.Answer.Claims) == 0 {
		t.Fatalf("the pin isn't first: %+v", a)
	}
	if a := asked(`{"question":"Who keeps the payroll forms in the archive?"}`); a.Pinned != nil {
		t.Fatalf("an unlike question got the pin: %+v", a.Pinned)
	}
	if code, out := r.do(t, "POST", "/v1/ask", `{"question":"When is the zebra cutoff?"}`); code != 200 || strings.Contains(out, `"pinned"`) {
		t.Fatalf("a company-wide ask got the space's pin: %d %s", code, out)
	}

	// SB-3602: verified only answers from marked documents alone.
	signInAs(sam)
	if a := asked(`{"question":"zebra cutoff rota","verified":true}`); a.Notice == nil || *a.Notice != ask.NoticeNoSources {
		t.Fatalf("verified only with nothing marked: %+v", a)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/review/marks", `{"cite":"`+cutoff+`","days":45}`); code != http.StatusBadRequest {
		t.Fatalf("an interval of 45 days: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/spaces/ops/review/marks", `{"cite":"`+cutoff+`","days":30}`); code != 200 {
		t.Fatalf("mark: %d %s", code, body)
	}
	a = asked(`{"question":"zebra cutoff rota","verified":true}`)
	for _, c := range a.Answer.Claims {
		for _, ci := range c.Cites {
			if !strings.HasPrefix(ci.Cite, "docs:Ops/cutoff.md@") {
				t.Fatalf("verified only cited an unmarked document: %+v", a.Answer)
			}
		}
	}
	if code, body := r.do(t, "GET", "/v1/search?space=ops&q=zebra+cutoff+rota&verified=true", ""); code != 200 || strings.Contains(body, "rota.md") || !strings.Contains(body, "cutoff.md") {
		t.Fatalf("verified search: %d %s", code, body)
	}

	// SB-3607: how many of the cited documents someone has checked.
	a = asked(`{"question":"zebra cutoff rota"}`)
	rota := ""
	docs := map[string]bool{}
	for _, c := range a.Answer.Claims {
		for _, ci := range c.Cites {
			docs[strings.Split(ci.Cite, "@")[0]] = true
			if strings.HasPrefix(ci.Cite, "docs:Ops/rota.md@") {
				rota = ci.Cite
			}
		}
	}
	if a.Verified == nil || a.Verified.Of != len(docs) || a.Verified.Marked != map[bool]int{true: 1, false: 0}[docs["docs:Ops/cutoff.md"]] {
		t.Fatalf("verified count: %+v of %v", a.Verified, docs)
	}
	if rota == "" {
		t.Fatal("no cite of the rota document")
	}

	// SB-3605: documents that disagreed this month, by pair.
	if err := r.cell.Traces.PutTrace(context.Background(), ask.Trace{ID: "tr_disagree", Tenant: "acme-logistics", User: lee, Space: "spc_ops", At: time.Now(),
		Response: ask.Response{Answer: ask.Answer{Claims: []answer.Claim{{Text: "The cutoff is 3pm or 4pm.", Disagree: true,
			Cites: []answer.Cite{{Cite: cutoff}, {Cite: rota}}}}}}}); err != nil {
		t.Fatal(err)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/review", ""); code != 200 || !strings.Contains(body, `"conflicts":[{`) || !strings.Contains(body, `"answers":`) {
		t.Fatalf("conflicts: %d %s", code, body)
	}

	// SB-3606: a report needs the asker's Not helpful rating, and names them.
	reported := asked(`{"question":"When is the zebra cutoff?"}`)
	report := `{"question":"When is the zebra cutoff?","trace":"` + reported.Trace + `","report":true,"note":"It changed last week."}`
	if code, body := r.do(t, "POST", "/v1/questions?space=ops", report); code != http.StatusBadRequest {
		t.Fatalf("a report without a rating: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/feedback", `{"trace":"`+reported.Trace+`","rating":"down"}`); code != 200 {
		t.Fatalf("rating: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/questions?space=ops", report); code != 200 {
		t.Fatalf("report: %d %s", code, body)
	}

	// SB-3601: once the asker can't open a cite, the pin isn't shown to them.
	write("cutoff.md.acl", "read sam@acme.example\n")
	sync()
	signInAs(lee)
	if a := asked(`{"question":"When is the zebra cutoff?"}`); a.Pinned != nil {
		t.Fatalf("a pin with a cite lee can't open: %+v", a.Pinned)
	}
	// A steward sees only the conflicts and reported claims they can open.
	if code, body := r.do(t, "GET", "/v1/spaces/ops/review", ""); code != 200 || strings.Contains(body, `"conflicts":[`) || !strings.Contains(body, `"conflicts_hidden":1`) {
		t.Fatalf("conflicts lee can't open: %d %s", code, body)
	}
	code, body := r.do(t, "GET", "/v1/spaces/ops/questions", "")
	var queue struct {
		Items []struct {
			Report *ask.ReportView `json:"report"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &queue); err != nil || code != 200 || len(queue.Items) == 0 || queue.Items[0].Report == nil {
		t.Fatalf("the report as lee: %d %s", code, body)
	}
	rep := queue.Items[0].Report
	if rep.By != sam || rep.Note != "It changed last week." || rep.Hidden == 0 || strings.Contains(strings.Join(rep.Claims, " "), "3pm") {
		t.Fatalf("the report as lee: %+v", rep)
	}
	for _, d := range rep.Documents {
		if strings.HasPrefix(d.Cite, "docs:Ops/cutoff.md@") {
			t.Fatalf("a reported document lee can't open: %+v", rep)
		}
	}
	signInAs(sam)
	if code, _ := r.do(t, "DELETE", "/v1/spaces/ops/pins/pin_nope", ""); code != http.StatusBadRequest {
		t.Fatalf("unpin of none: %d", code)
	}
}
