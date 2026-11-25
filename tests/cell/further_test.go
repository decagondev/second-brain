package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/hubspot/hubspottest"
	"example.com/second-brain/internal/adapters/connectors/linear/lineartest"
	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/core/ask"
)

const furtherManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder, tickets: linear, crm: hubspot }
agents: [ask, case-brief]
spaces:
  support:
    template: support
    label: internal
    agents: [ask, case-brief]
    members: [group:support]
    stewards: [group:support-leads]
    scope: { docs: ["/Support/**"], tickets: [SUP], crm: ["*"] }
`

// SB-4604 to SB-4606 through a running cell: an incident that had a brief gets one review
// when its ticket closes, with the ticket's action items for the steward to confirm; an
// account brief gathers the record, open tickets, notes and contracts by code, as the
// person, and never an account they can't open; and How this was made shows an answer's
// agent, sources and cost, checked as the reader.
func TestAgentsThatGoFurtherThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	for path, text := range map[string]string{
		"Support/.acl":               "read domain:acme.example\n",
		"Support/gamma-proposal.md":  "Proposal for Gamma Parts: weekend pickups at a fixed rate.",
		"Support/gamma-call.md":      "Call notes: Gamma Parts asked about weekend pickups at the Leeds depot.",
		"Support/gamma-terms.md":     "Gamma Parts gets a discount of thirty percent.",
		"Support/gamma-terms.md.acl": "read boss@acme.example\n",
		"Support/printer-runbook.md": "Runbook: when the label printer stops, power cycle the label printer.",
		"Support/unrelated-notes.md": "The canteen opens at noon.",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(share, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(share, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lin, hub := lineartest.New(), hubspottest.New()
	t.Cleanup(lin.Server.Close)
	t.Cleanup(hub.Server.Close)
	lin.AddUser("U1", "sam@acme.example", false)
	lin.AddTeam("T1", "SUP", false, "U1")
	lin.Put("i1", "T1", "Label printer down", "The label printer stopped at 09:12.")
	lin.Tag("i1", []string{"sev-1"}, "In Progress", "started")
	lin.Put("i2", "T1", "Gamma Parts late deliveries", "Gamma Parts reports late deliveries to the Leeds depot.")
	lin.Tag("i2", nil, "Todo", "unstarted")
	hub.AddUser("h1", "sam@acme.example", "r_sales", false)
	hub.AddUser("h2", "bob@acme.example", "r_sales", false)
	hub.Put("companies/c1", "Gamma Parts", "Buys freight capacity each quarter.", hub.Owner("sam@acme.example"), "")
	hub.SetDomain("companies/c1", "gamma-parts.example")
	hub.Put("companies/c2", "Beta Freight", "Buys warehouse space.", hub.Owner("bob@acme.example"), "")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, furtherManifest, map[string]string{"connectors/tickets": lin.Key,
			"connectors/crm": `{"token": "` + hub.Token + `", "roles": {"r_sales": "owned"}}`})
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}, "tickets": {Base: lin.Endpoint()}, "crm": {Base: hub.Base()}}
		return cfg
	})
	r.signIn(t, "sam")
	sam := r.idp.User
	for _, name := range []string{"support", "support-leads"} {
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[{"value":"`+sam+`"}]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	sync := func() {
		t.Helper()
		if fails := svc.SyncAll(context.Background()); len(fails) > 0 {
			t.Fatal(fails)
		}
	}
	sync()
	run := func() []string {
		t.Helper()
		told := &agentsTold{}
		if _, err := r.cell.NotifyJob(told).Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		return told.got
	}
	queue := func() []ask.StewardItem {
		t.Helper()
		code, out := r.do(t, "GET", "/v1/spaces/support/questions", "")
		var q struct{ Items []ask.StewardItem }
		if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
			t.Fatalf("queue: %d %s", code, out)
		}
		return q.Items
	}
	kinds := func() map[string]int {
		out := map[string]int{}
		for _, it := range queue() {
			if it.Thread != nil {
				out[it.Thread.Kind]++
			}
		}
		return out
	}

	// SB-4604: the brief first; an open ticket has no review.
	if got := strings.Join(run(), "\n"); !strings.Contains(got, "An incident brief is waiting in support") || strings.Contains(got, "review") {
		t.Fatalf("first run told: %s", got)
	}
	if k := kinds(); k[ask.DraftIncident] != 1 || k[ask.DraftReview] != 0 {
		t.Fatalf("drafts: %v", k)
	}
	// It closes, with a follow-up: one review, its action item waiting for the steward.
	lin.Put("i1", "T1", "Label printer down", "The label printer stopped at 09:12 and was power cycled at 09:30.\n\nAction: sam to add a label printer alert by 2026-12-01.")
	lin.Tag("i1", []string{"sev-1"}, "Done", "completed")
	sync()
	if got := strings.Join(run(), "\n"); !strings.Contains(got, "incident "+sam+": An incident review is waiting in support") {
		t.Fatalf("after closing, told: %s", got)
	}
	var review *ask.StewardItem
	for _, it := range queue() {
		if it.Thread != nil && it.Thread.Kind == ask.DraftReview {
			it := it
			review = &it
		}
	}
	if review == nil || review.Thread.Agent != ask.AgentReview || !strings.HasPrefix(review.Question, "Review: ") ||
		len(review.Thread.Actions) != 1 || !strings.Contains(review.Thread.Actions[0].Text, "label printer alert") {
		t.Fatalf("review: %+v", review)
	}
	code, out := r.do(t, "POST", "/v1/spaces/support/actions", `{"draft":"`+review.ID+`","items":[{"text":"add a label printer alert","owner":"sam","due":"2026-12-01"}]}`)
	if code != 200 {
		t.Fatalf("confirm the review's action: %d %s", code, out)
	}
	sync()
	if got := strings.Join(run(), "\n"); strings.Contains(got, "review is waiting") {
		t.Fatalf("a second review: %s", got)
	}

	// SB-4605: an account brief, picked by code and checked as sam.
	code, out = r.do(t, "POST", "/v1/account-brief?space=support", `{"account":"crm:companies/c1"}`)
	var brief struct {
		ID, Kind, Question string
		Sections           []ask.Section
		Account            ask.AccountFacts
	}
	if err := json.Unmarshal([]byte(out), &brief); err != nil || code != 200 {
		t.Fatalf("account brief: %d %s", code, out)
	}
	if brief.Kind != ask.KindAccount || brief.Question != "Gamma Parts" || brief.Account.Domain != "gamma-parts.example" ||
		brief.Account.Tickets != 1 || brief.Account.Notes != 1 || brief.Account.Documents != 1 || len(brief.Sections) != 4 {
		t.Fatalf("account brief: %s", out)
	}
	if strings.Contains(out, "gamma-terms") || strings.Contains(out, "thirty") {
		t.Fatalf("the brief names a document sam can't open: %s", out)
	}
	for _, body := range []string{`{"account":"crm:companies/c2"}`, `{"account":"crm:companies/nope"}`, `{"account":"docs:Support/gamma-call.md"}`} {
		if code, out := r.do(t, "POST", "/v1/account-brief?space=support", body); code != http.StatusBadRequest || strings.Contains(out, "Beta") {
			t.Fatalf("%s: %d %s", body, code, out)
		}
	}
	if code, _ := r.do(t, "POST", "/v1/account-brief", `{"account":"crm:companies/c1"}`); code != http.StatusBadRequest {
		t.Fatalf("company-wide: %d", code)
	}

	// SB-4606: how the brief, and one of its answers, was made.
	code, out = r.do(t, "GET", "/v1/research/"+brief.ID+"/how?space=support", "")
	var how ask.HowMade
	if err := json.Unmarshal([]byte(out), &how); err != nil || code != 200 {
		t.Fatalf("how: %d %s", code, out)
	}
	if how.Agent != "case-brief" || len(how.Steps) != 4 || len(how.Sources) == 0 || how.SourcesHidden != 0 || how.CostUSD < 0 {
		t.Fatalf("how the brief was made: %s", out)
	}
	for _, s := range how.Sources {
		if strings.Contains(s.Cite, "gamma-terms") {
			t.Fatalf("how names a document sam can't open: %s", out)
		}
	}
	code, out = r.do(t, "GET", "/v1/traces/"+brief.Sections[0].Trace+"/how", "")
	if code != 200 || !strings.Contains(out, `"agent":"case-brief"`) || !strings.Contains(out, `"cost_usd"`) || !strings.Contains(out, "crm:companies/c1") {
		t.Fatalf("how one answer was made: %d %s", code, out)
	}
	if code, _ := r.do(t, "GET", "/v1/research/rs_nope/how", ""); code != http.StatusForbidden {
		t.Fatalf("someone else's report: %d", code)
	}
	// sam loses the call notes: How this was made counts them and no longer names them (LAW-01).
	var notes string
	for _, sec := range brief.Sections {
		if strings.Contains(sec.Question, "notes") {
			notes = sec.Trace
		}
	}
	if err := os.WriteFile(filepath.Join(share, "Support/gamma-call.md.acl"), []byte("read boss@acme.example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sync()
	code, out = r.do(t, "GET", "/v1/traces/"+notes+"/how", "")
	if err := json.Unmarshal([]byte(out), &how); err != nil || code != 200 || how.SourcesHidden != 1 || strings.Contains(out, "gamma-call") {
		t.Fatalf("how, after sam lost the notes: %d %s", code, out)
	}
}
