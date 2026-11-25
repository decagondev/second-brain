package cell

import (
	"bytes"
	"context"
	"encoding/base64"
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
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/reviews"
)

const proveManifest = `tenant: acme-logistics
channel: testing
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
notes: true
agents: [ask, case-brief]
spaces:
  support:
    template: support
    label: internal
    members: [group:support]
    stewards: [group:support-leads]
    agents: [ask, case-brief]
    updates: true
    scope: { docs: ["/Support/**", "/notes/support/**"] }
`

// SB-4704 to SB-4707 through a running cell on the testing channel: a changed agent's shadow
// runs beside its drafts, never in a queue, its owner sees both and nobody else does, and the
// runs go when the window ends; an agent draft carries its mark, and the note published
// from it keeps generated_by and approved_by, on the file and in the index; the quarter's
// agent review proposes the unused off and its removals become one manifest proposal; and
// the agent's record is signed, verifies, and holds no document text or asker.
func TestProveItThroughARunningCell(t *testing.T) {
	defs := ask.AgentDefs
	t.Cleanup(func() { ask.AgentDefs = defs })
	ask.AgentDefs = bytes.Replace(defs, []byte(`"prompt": "incident-brief", "suite": "agent-incident-brief"`),
		[]byte(`"prompt": "incident-brief", "suite": "agent-incident-brief", "shadow": {"prompt": "answer", "days": 7}`), 1)
	if bytes.Equal(defs, ask.AgentDefs) {
		t.Fatal("incident-brief's definition wasn't found to give it a shadow")
	}
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
	write("Support/.acl", "read domain:acme.example\n")
	write("Support/queue-runbook.md", "Runbook: when the payments queue stops draining, restart the payments queue consumer.")
	write("Support/inc-1.md", "Labels: sev-1\n\nThe payments queue stopped draining at 09:12.")
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, proveManifest, map[string]string{
			"audit/anchor-seed": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))})
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		return cfg
	})
	got := &agentsTold{}
	r.cell.Send = got
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, members := range map[string][]string{"support": {sam, lee}, "support-leads": {sam}, "brain-agents": {lee}, "sb-operators": {sam}} {
		var ms []string
		for _, m := range members {
			ms = append(ms, `{"value":"`+m+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group %s: %d %s", name, code, body)
		}
	}
	ctx := context.Background()
	svc := &ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}
	sync := func() {
		t.Helper()
		if fails := svc.SyncAll(ctx); len(fails) > 0 {
			t.Fatal(fails)
		}
	}
	sync()
	if _, err := r.cell.NotifyJob(got).Run(ctx); err != nil {
		t.Fatal(err)
	}

	// SB-4706: the incident brief waits for sam with its mark; SB-4704: its shadow doesn't.
	code, out := r.do(t, "GET", "/v1/spaces/support/questions", "")
	var q struct{ Items []ask.StewardItem }
	if err := json.Unmarshal([]byte(out), &q); err != nil || code != 200 {
		t.Fatalf("queue: %d %s", code, out)
	}
	var draft ask.StewardItem
	n := 0
	for _, it := range q.Items {
		if it.Thread != nil && it.Thread.Kind == ask.DraftIncident {
			n++
			draft = it
		}
		if it.Thread != nil && it.Thread.Agent != "" && it.Thread.Mark == nil {
			t.Fatalf("an agent draft without its mark: %s", out)
		}
	}
	if n != 1 || draft.Thread.Kind != ask.DraftIncident || draft.Thread.Mark == nil || *draft.Thread.Mark != (ask.Mark{Agent: ask.AgentIncident, Version: r.cell.Release}) {
		t.Fatalf("one marked incident draft, and no shadow, in the queue: %s", out)
	}

	// SB-4704: lee owns incident-brief and sees its run both ways; sam owns nothing.
	var sh ask.ShadowReport
	code, out = r.do(t, "GET", "/v1/agents/shadow", "")
	if err := json.Unmarshal([]byte(out), &sh); err != nil || code != 200 || len(sh.Items) != 0 {
		t.Fatalf("sam owns no agent and sees no shadow: %d %s", code, out)
	}
	r.idp.User = lee
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatal("lee's sign-in")
	}
	code, out = r.do(t, "GET", "/v1/agents/shadow", "")
	if err := json.Unmarshal([]byte(out), &sh); err != nil || code != 200 || len(sh.Items) != 1 || len(sh.Items[0].Runs) != 1 {
		t.Fatalf("lee's shadow: %d %s", code, out)
	}
	run := sh.Items[0].Runs[0]
	if sh.Items[0].Agent != ask.AgentIncident || sh.Items[0].Prompt != "answer" || run.Text == nil || run.Text.Current != draft.Thread.Text || run.Current.Cites == 0 {
		t.Fatalf("lee's shadow run: %s", out)
	}
	if !sh.Items[0].Until.After(time.Now().Add(6*24*time.Hour)) || sh.Items[0].Until.After(time.Now().Add(7*24*time.Hour+time.Minute)) {
		t.Fatalf("the window is a week: %s", out)
	}
	if gone, err := r.cell.Traces.DeleteShadows(ctx, "acme-logistics", time.Now().Add(8*24*time.Hour)); err != nil || gone != 1 {
		t.Fatalf("the week's end deletes the run: %d %v", gone, err)
	}
	if runs, _ := r.cell.Traces.ShadowRuns(ctx, "acme-logistics"); len(runs) != 0 {
		t.Fatalf("shadow runs left after the window: %+v", runs)
	}
	r.idp.User = sam
	if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
		t.Fatal("sam's sign-in")
	}

	// SB-4706: sam publishes the draft as it is; the note keeps the mark, on the file and on
	// every chunk the index holds of it.
	base := "/v1/spaces/support/questions/" + draft.ID
	if code, out := r.do(t, "POST", base+"/reply", `{"text":`+jsonString(draft.Thread.Text)+`,"cites":[]}`); code != 200 {
		t.Fatalf("reply: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", base+"/note", `{"title":"The payments queue incident","confirm":0}`); code != 200 || !strings.Contains(out, `"state":"Ran"`) {
		t.Fatalf("publish: %d %s", code, out)
	}
	note, err := os.ReadFile(filepath.Join(share, "notes", "support", draft.ID+".md"))
	if err != nil {
		t.Fatal(err)
	}
	samEmail := ""
	for _, l := range strings.Split(string(note), "\n") {
		if v, ok := strings.CutPrefix(l, "approved_by: "); ok {
			samEmail = v
		}
	}
	if !strings.Contains(string(note), "generated_by: incident-brief@"+r.cell.Release+"\n") || samEmail == "" || strings.Contains(string(note), "edited:") {
		t.Fatalf("the note's front matter: %s", note)
	}
	sync()
	lister, ok := r.cell.Index.(interface {
		All(context.Context, string) ([]ports.Chunk, error)
	})
	if !ok {
		t.Fatal("the test cell's index can't list its chunks")
	}
	chunks, err := lister.All(ctx, "acme-logistics")
	if err != nil {
		t.Fatal(err)
	}
	marked := 0
	for _, c := range chunks {
		if strings.Contains(c.SourceID, draft.ID) {
			if c.GeneratedBy != "incident-brief@"+r.cell.Release || c.ApprovedBy != samEmail || c.Edited {
				t.Fatalf("the note's chunk %s: %+v", c.Cite, c)
			}
			marked++
		} else if c.GeneratedBy != "" {
			t.Fatalf("%s, which a person wrote, carries a mark", c.Cite)
		}
	}
	if marked == 0 {
		t.Fatal("the note wasn't indexed")
	}

	// SB-4705: the quarter's agent review lists case-brief, unused and proposed off, and the
	// update trigger, which drafted this week; sam removes case-brief, keeps updates, and one
	// proposal follows.
	opened, err := r.cell.RunAgentReviews(ctx, time.Now())
	if err != nil || len(opened) != 1 || opened[0].Kind != reviews.KindAgents || len(opened[0].Items) != 2 || len(opened[0].ProposeOff) == 0 {
		t.Fatalf("the agent review: %+v %v", opened, err)
	}
	if again, err := r.cell.RunAgentReviews(ctx, time.Now()); err != nil || len(again) != 0 {
		t.Fatalf("a second review in the quarter: %+v %v", again, err)
	}
	rv := opened[0]
	for _, it := range rv.Items {
		if (it.User == "case-brief") != (it.Unused && it.LastUse == reviews.Never) {
			t.Fatalf("item %+v", it)
		}
	}
	if d := strings.Join(rv.ProposeOff, "\n"); !strings.Contains(d, "case-brief") || strings.Contains(d, "updates") {
		t.Fatalf("propose_off: %s", d)
	}
	if rows := r.auditRows(t); !strings.Contains(rows, `"subject":"`+rv.ID+`"`) || !strings.Contains(rows, `"state":"opened"`) {
		t.Fatal("opening the agent review isn't audited")
	}
	if code, out := r.do(t, "POST", "/v1/spaces/support/reviews/"+rv.ID+"/decide", `{"user":"case-brief","decision":"remove","reason":"no one asks it"}`); code != 200 {
		t.Fatalf("remove case-brief: %d %s", code, out)
	}
	code, out = r.do(t, "POST", "/v1/spaces/support/reviews/"+rv.ID+"/decide", `{"user":"updates","decision":"certify"}`)
	var done reviews.Review
	if err := json.Unmarshal([]byte(out), &done); err != nil || code != 200 || done.State != reviews.Complete || len(done.Requests) != 1 || done.Requests[0].Proposal == "" {
		t.Fatalf("the completed agent review: %d %s", code, out)
	}
	code, out = r.do(t, "GET", "/v1/admin/proposals", "")
	if code != 200 || !strings.Contains(out, done.Requests[0].Proposal) || !strings.Contains(out, "case-brief") {
		t.Fatalf("the proposal: %d %s", code, out)
	}
	if m := r.cell.Manifest(); !strings.Contains(strings.Join(m.Spaces["support"].Agents, ","), "case-brief") {
		t.Fatal("the review changed the manifest itself")
	}

	// SB-4707: the record, signed, through the control plane and the admin route.
	code, out = r.do(t, "GET", "/v1/admin/agents/incident-brief/record", "")
	if code != 200 {
		t.Fatalf("the record: %d %s", code, out)
	}
	sr, err := ask.VerifyRecord([]byte(out), "")
	if err != nil {
		t.Fatalf("the record doesn't verify: %v\n%s", err, out)
	}
	if sr.Record.Agent != ask.AgentIncident || len(sr.Record.Versions) != 1 || sr.Record.Versions[0].Release != r.cell.Release || len(sr.Record.Value) != 1 || sr.Record.Value[0].Drafts != 1 || sr.Record.Value[0].Published != 1 {
		t.Fatalf("the record's counts: %s", out)
	}
	for _, leak := range []string{"payments queue", lee, sam, samEmail} {
		if leak != "" && strings.Contains(out, leak) {
			t.Fatalf("the record carries %q: %s", leak, out)
		}
	}
	tampered := strings.Replace(out, `"drafts":1`, `"drafts":0`, 1)
	if _, err := ask.VerifyRecord([]byte(tampered), ""); err == nil {
		t.Fatal("a changed record verified")
	}
	code, md := r.do(t, "GET", "/v1/admin/agents/incident-brief/record?format=md", "")
	if code != 200 || !strings.Contains(md, "# Agent record: incident-brief") || !strings.Contains(md, sr.SHA256) {
		t.Fatalf("the record's Markdown: %d %s", code, md)
	}
	if code, _ := r.do(t, "GET", "/v1/admin/agents/nope/record", ""); code != http.StatusBadRequest {
		t.Fatalf("an unknown agent's record: %d", code)
	}
}

// SB-4707: sb agents record reaches a running cell through the registry, checks the record
// against a pinned key, and writes the JSON and its Markdown.
func TestAgentRecordAgainstARunningCell(t *testing.T) {
	l := startLive(t)
	out := t.TempDir()
	sg, err := l.cell.AnchorSigner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if code, o := sb(t, "--endpoint", "file://"+l.cp, "agents", "record", "ask", "--tenant", tenant, "--out", out, "--key", sg.Key()); code != 0 || !strings.Contains(o, "verified") {
		t.Fatalf("sb agents record: %d %s", code, o)
	}
	raw, err := os.ReadFile(filepath.Join(out, "ask-record.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ask.VerifyRecord(raw, sg.Key()); err != nil {
		t.Fatal(err)
	}
	if md, err := os.ReadFile(filepath.Join(out, "ask-record.md")); err != nil || !strings.Contains(string(md), "# Agent record: ask") {
		t.Fatalf("the Markdown: %v %s", err, md)
	}
	if code, o := sb(t, "agents", "record", "ask", "--cell", l.url, "--out", out, "--key", "ed25519:AAAA"); code == 0 {
		t.Fatalf("a record passed against another key: %s", o)
	}
	if code, o := sb(t, "agents", "record", "nope", "--cell", l.url, "--out", out); code == 0 {
		t.Fatalf("an unknown agent's record: %s", o)
	}
}
