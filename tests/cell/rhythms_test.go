package cell

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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

const rhythmsManifest = `tenant: acme-logistics
channel: testing
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  support:
    template: support
    label: internal
    members: [group:support]
    stewards: [group:support-leads]
    start_here: ["docs:/Support/Start/"]
    scope: { docs: ["/Support/**"] }
  ops:
    template: support
    label: internal
    members: [group:ops]
    stewards: [group:ops-leads]
    scope: { docs: ["/Ops/**"] }
`

// SB-4801 to SB-4804 through a running cell: SCIM adding a person starts their onboarding,
// which waits for its plan's approver; its day-one draft lists only the start-here items
// they can open and lands in their own queue; leaving the space stops it at once (LAW-29).
// A project gets a collection, a decision log and a kickoff brief of a similar closed
// project that cites only what its lead can open; a member from another space joins only
// once their steward approves; closing archives the collection read-only, proposes the
// lessons to the stewards and drafts a cited retrospective, and stops the kickoff rhythm.
func TestRhythmsAndProjectsThroughARunningCell(t *testing.T) {
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
	write("Support/.acl", "read domain:acme.example\nowner sam@acme.example\n")
	write("Support/Start/.acl", "read domain:acme.example\nowner sam@acme.example\n")
	write("Support/Start/welcome.md", "Welcome to support: how the payments queue works.")
	write("Support/Start/secret.md", "Escalation contacts for the payments queue.")
	write("Support/Start/secret.md.acl", "read sam@acme.example\n")
	write("Support/payments-decision.md", "Decision: the payments queue retries three times before paging.")
	write("Support/payments-private.md", "Decision: the payments vendor contract renews in March.")
	write("Support/payments-private.md.acl", "read sam@acme.example\n")
	write("Ops/.acl", "read domain:acme.example\n")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, rhythmsManifest, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	ids := map[string]string{}
	for _, n := range []string{"kim", "pat", "lee", "sam"} {
		r.signIn(t, n)
		ids[n] = r.idp.User
	}
	as := func(name string) {
		t.Helper()
		r.idp.User = ids[name]
		if code, body := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in as %s: %d %s", name, code, body)
		}
	}
	scim := func(method, path, body string) string {
		t.Helper()
		code, out := r.do(t, method, path, body, "Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
		if code != http.StatusCreated && code != http.StatusOK {
			t.Fatalf("SCIM %s %s: %d %s", method, path, code, out)
		}
		return out
	}
	groups := map[string]string{}
	for name, members := range map[string][]string{"support": {"sam", "lee"}, "support-leads": {"sam"}, "ops": {"kim"}, "ops-leads": {"kim"}} {
		var ms []string
		for _, m := range members {
			ms = append(ms, `{"value":"`+ids[m]+`"}`)
		}
		var g struct{ ID string }
		_ = json.Unmarshal([]byte(scim("POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`)), &g)
		groups[name] = g.ID
	}
	ctx := context.Background()
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	tick := func() {
		t.Helper()
		if _, err := r.cell.Parts().Rhythms.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	tick() // the first run only notes who is a member

	var rh struct{ Items []ask.RhythmView }
	rhythms := func(who string) []ask.RhythmView {
		t.Helper()
		as(who)
		code, out := r.do(t, "GET", "/v1/spaces/support/rhythms", "")
		rh.Items = nil
		if err := json.Unmarshal([]byte(out), &rh); err != nil || code != 200 {
			t.Fatalf("rhythms: %d %s", code, out)
		}
		return rh.Items
	}
	if got := rhythms("sam"); len(got) != 0 {
		t.Fatalf("members already there got onboarding: %+v", got)
	}

	// SB-4802: SCIM adds pat; the next run proposes their onboarding to the stewards.
	scim("PATCH", "/scim/v2/Groups/"+groups["support"], `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"add","path":"members","value":[{"value":"`+ids["pat"]+`"}]}]}`)
	tick()
	got := rhythms("sam")
	if len(got) != 1 || got[0].Rhythm != "onboarding" || got[0].State != ask.RhythmProposed || got[0].Subject != ids["pat"] {
		t.Fatalf("onboarding wasn't proposed for pat: %+v", got)
	}
	run := got[0].ID
	as("lee")
	if code, out := r.do(t, "POST", "/v1/rhythms/"+run+"/approve", ""); code == 200 {
		t.Fatalf("a member who isn't a steward approved the plan: %s", out)
	}
	as("sam")
	if code, out := r.do(t, "POST", "/v1/rhythms/"+run+"/approve", ""); code != 200 || !strings.Contains(out, `"state":"running"`) {
		t.Fatalf("approve: %d %s", code, out)
	}

	// The day-one step waits in pat's own queue, drafted as pat: welcome.md, never secret.md.
	as("pat")
	code, out := r.do(t, "GET", "/v1/spaces/"+url.PathEscape(ask.PersonKey(ids["pat"]))+"/questions", "")
	if code != 200 || !strings.Contains(out, "onboarding, day 0") || !strings.Contains(out, "welcome") {
		t.Fatalf("pat's day-one step: %d %s", code, out)
	}
	if strings.Contains(out, "secret") || strings.Contains(out, "Escalation") {
		t.Fatalf("the day-one draft named a document pat can't open (LAW-01): %s", out)
	}
	if !strings.Contains(out, `"rhythm":{"run":"`+run) {
		t.Fatalf("the step doesn't carry its rhythm: %s", out)
	}
	if code, out := r.do(t, "POST", "/v1/rhythms/"+run+"/consent", `{"share":false}`); code != 200 {
		t.Fatalf("consent: %d %s", code, out)
	}

	// LAW-29: pat leaves the space, and the rhythm stops at once, before any job runs.
	scim("PATCH", "/scim/v2/Groups/"+groups["support"], `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"remove","path":"members[value eq \"`+ids["pat"]+`\"]"}]}`)
	got = rhythms("sam")
	if len(got) != 1 || got[0].State != ask.RhythmStopped || got[0].Why != "its subject left the space" {
		t.Fatalf("leaving the space didn't stop the rhythm: %+v", got)
	}
	for _, st := range got[0].Steps {
		if st.State == ask.StepWaiting {
			t.Fatalf("a stopped rhythm still has a waiting step: %+v", got[0])
		}
	}

	cite := func(who, q string) string {
		t.Helper()
		as(who)
		code, out := r.do(t, "GET", "/v1/search?space=support&q="+url.QueryEscape(q), "")
		var res struct{ Results []ask.Result }
		if err := json.Unmarshal([]byte(out), &res); err != nil || code != 200 || len(res.Results) == 0 {
			t.Fatalf("search %q: %d %s", q, code, out)
		}
		doc, _, _ := strings.Cut(res.Results[0].Cite, "@")
		return doc
	}
	public, private := cite("lee", "payments queue retries"), cite("sam", "vendor contract renews")

	// An earlier project, closed, with a decision lee can open and one only sam can.
	as("sam")
	var pv ask.ProjectView
	project := func(method, path, body string) ask.ProjectView {
		t.Helper()
		code, out := r.do(t, method, path, body)
		pv = ask.ProjectView{}
		if err := json.Unmarshal([]byte(out), &pv); err != nil || code != 200 {
			t.Fatalf("%s %s: %d %s", method, path, code, out)
		}
		return pv
	}
	old := project("POST", "/v1/spaces/support/projects", `{"name":"Payments queue cleanup","members":["lee@acme.example"]}`)
	project("POST", "/v1/projects/"+old.ID+"/decisions", `{"text":"Retry three times before paging","cite":"`+public+`"}`)
	project("POST", "/v1/projects/"+old.ID+"/decisions", `{"text":"Renew the vendor in March","cite":"`+private+`"}`)
	as("lee")
	if v := project("GET", "/v1/projects/"+old.ID, ""); len(v.Decisions) != 1 || v.Hidden != 1 || v.Goal != "" {
		t.Fatalf("lee sees decisions they can't open, or the lead's goal draft: %+v", v)
	}
	as("sam")
	project("POST", "/v1/projects/"+old.ID+"/close", `{"lessons":[]}`)

	// SB-4803: lee starts a similar project with kim from ops, who waits for ops' steward.
	as("lee")
	p := project("POST", "/v1/spaces/support/projects", `{"name":"Payments queue revamp","members":["kim@acme.example"]}`)
	if p.Collection.ID == "" || !strings.Contains(p.Goal, "type: goal") || len(p.Pending) != 1 || p.Kickoff == "" {
		t.Fatalf("start a project: %+v", p)
	}
	code, out = r.do(t, "GET", "/v1/spaces/"+url.PathEscape(ask.PersonKey(ids["lee"]))+"/questions", "")
	if code != 200 || !strings.Contains(out, "kickoff, day 0") || !strings.Contains(out, "Retry three times before paging") {
		t.Fatalf("the kickoff brief: %d %s", code, out)
	}
	if strings.Contains(out, "Renew the vendor") {
		t.Fatalf("the kickoff brief cites what the lead can't open (LAW-01): %s", out)
	}
	as("kim")
	if code, out := r.do(t, "GET", "/v1/projects/"+p.ID, ""); code == 200 {
		t.Fatalf("kim reads the project before their steward approved: %s", out)
	}
	if code, out := r.do(t, "POST", "/v1/projects/"+p.ID+"/members", `{"email":"kim@acme.example"}`); code != 200 {
		t.Fatalf("ops' steward approving kim: %d %s", code, out)
	}
	if v := project("GET", "/v1/projects/"+p.ID, ""); v.Lead || v.ID != p.ID {
		t.Fatalf("kim after approval: %+v", v)
	}

	// SB-4804: lee closes it with a lesson; the collection is read-only and never deleted.
	as("lee")
	project("POST", "/v1/projects/"+p.ID+"/decisions", `{"text":"Retry three times before paging","cite":"`+public+`"}`)
	closed := project("POST", "/v1/projects/"+p.ID+"/close", `{"lessons":[{"text":"Agree the retry count first","cite":"`+public+`"}]}`)
	if closed.State != ask.ProjectClosed || closed.Collection.Closed == "" || closed.CloseOut == "" {
		t.Fatalf("close: %+v", closed)
	}
	if code, out := r.do(t, "PUT", "/v1/collections/"+p.Collection.ID, `{"name":"x","items":[]}`); code == 200 {
		t.Fatalf("a closed project's collection changed: %s", out)
	}
	if code, out := r.do(t, "DELETE", "/v1/collections/"+p.Collection.ID, ""); code == 200 {
		t.Fatalf("a project's collection was deleted: %s", out)
	}
	code, out = r.do(t, "GET", "/v1/spaces/"+url.PathEscape(ask.PersonKey(ids["lee"]))+"/questions", "")
	if code != 200 || !strings.Contains(out, "close-out, day 0") || !strings.Contains(out, "Decided: Retry three times before paging [") ||
		!strings.Contains(out, "Lesson proposed to the stewards: Agree the retry count first [") {
		t.Fatalf("the retrospective: %d %s", code, out)
	}
	as("sam")
	code, out = r.do(t, "GET", "/v1/spaces/support/questions", "")
	if code != 200 || !strings.Contains(out, "Lesson from Payments queue revamp") || !strings.Contains(out, `"kind":"lesson"`) {
		t.Fatalf("the lesson waits for the stewards: %d %s", code, out)
	}
	for _, v := range rhythms("sam") {
		if v.ID == p.Kickoff && (v.State != ask.RhythmStopped || v.Why != "its project closed") {
			t.Fatalf("closing didn't stop the kickoff rhythm: %+v", v)
		}
	}
}
