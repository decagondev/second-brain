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

// S43.1 through a running cell (SB-4401, SB-4402): a down-rated answer of ask is a
// candidate case for the people who answer for ask, in their inbox; accepted, it is a case
// of the agent that the evals job runs; dismissed, it leaves the list and never runs.
func TestCasesFromFeedbackThroughARunningCell(t *testing.T) {
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
	ctx := context.Background()
	r.signIn(t, "sam")
	sam := r.idp.User
	for _, g := range []string{"ops", "brain-agents"} {
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+g+`","members":[{"value":"`+sam+`"}]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group %s: %d %s", g, code, body)
		}
	}
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	if code, body := r.do(t, "GET", "/v1/agents/cases", ""); code != 200 || !strings.Contains(body, `"items":[]`) {
		t.Fatalf("no feedback, no candidates: %d %s", code, body)
	}

	askDown := func(q string) (string, string) {
		code, body := r.do(t, "POST", "/v1/ask?space=ops", `{"question":"`+q+`"}`)
		var a struct{ Trace string }
		if err := json.Unmarshal([]byte(body), &a); err != nil || code != 200 || a.Trace == "" {
			t.Fatalf("ask: %d %s", code, body)
		}
		right := strings.SplitN(cite(t, body, "right.md"), "@", 2)[0]
		if code, body := r.do(t, "POST", "/v1/feedback", `{"trace":"`+a.Trace+`","rating":"down"}`); code != 200 {
			t.Fatalf("feedback: %d %s", code, body)
		}
		return a.Trace, right
	}
	first, right := askDown("When is the zebra cutoff?")
	second, _ := askDown("Is the zebra cutoff on weekdays?")

	code, body := r.do(t, "GET", "/v1/agents/cases", "")
	var list struct {
		Items []struct {
			ID, Agent, Space, Question string
			Hidden                     int
			Claims                     []struct{ Text string }
		}
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil || code != 200 || len(list.Items) != 2 {
		t.Fatalf("candidates: %d %s", code, body)
	}
	c := list.Items[1]
	if c.ID != first || c.Agent != "ask" || c.Space != "spc_ops" || c.Question != "When is the zebra cutoff?" || c.Hidden != 0 || len(c.Claims) == 0 {
		t.Fatalf("the older candidate: %+v", c)
	}
	if strings.Contains(body, sam) || strings.Contains(body, `"user"`) {
		t.Fatalf("a candidate names its asker: %s", body)
	}
	if code, body := r.do(t, "GET", "/v1/me/inbox", ""); code != 200 || !strings.Contains(body, `"kind":"case","id":"`+first+`"`) {
		t.Fatalf("inbox: %d %s", code, body)
	}

	// Accept needs an expect the owner can open now, and exactly one of expect and dismiss.
	for _, bad := range []string{`{}`, `{"expect":"` + right + `","dismiss":true}`} {
		if code, body := r.do(t, "POST", "/v1/agents/cases/"+first, bad); code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", bad, code, body)
		}
	}
	if code, body := r.do(t, "POST", "/v1/agents/cases/"+first, `{"expect":"docs:nope.md"}`); code != http.StatusBadRequest {
		t.Fatalf("an expect no one can open: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/agents/cases/tr_nope", `{"dismiss":true}`); code != http.StatusForbidden {
		t.Fatalf("no such candidate: %d %s", code, body)
	}
	code, body = r.do(t, "POST", "/v1/agents/cases/"+first, `{"expect":"`+right+`"}`)
	var ec ask.EvalCase
	if err := json.Unmarshal([]byte(body), &ec); err != nil || code != 200 || ec.Agent != "ask" || ec.From != first || ec.Expect != right {
		t.Fatalf("accept: %d %s", code, body)
	}
	if code, body := r.do(t, "POST", "/v1/agents/cases/"+second, `{"dismiss":true}`); code != 200 {
		t.Fatalf("dismiss: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/agents/cases", ""); code != 200 || strings.Contains(body, first) || strings.Contains(body, second) {
		t.Fatalf("decided candidates are still listed: %d %s", code, body)
	}
	if code, body := r.do(t, "GET", "/v1/spaces/ops/evals", ""); code != 200 || !strings.Contains(body, ec.ID) || strings.Count(body, `"id":"ev_`) != 1 {
		t.Fatalf("the space's cases: %d %s", code, body)
	}

	// The evals job runs the accepted case with its agent, and never the dismissed one.
	r.cell.Release = "rel_new"
	if _, err := r.cell.RunEvals(ctx); err != nil {
		t.Fatal(err)
	}
	cs, _ := r.cell.Traces.EvalCases(ctx, tenant)
	for _, x := range cs {
		if x.Dismissed == (x.LastRelease == "rel_new") || x.Dismissed && x.Question != "" {
			t.Fatalf("after the run: %+v", x)
		}
	}
	rows, err := r.cell.Audit.Store.Range(ctx, 1, 1000)
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]bool{}
	for _, row := range rows {
		if row.Kind == "access" && row.Reason == "eval_case" {
			states[row.State] = true
		}
	}
	if !states["accepted"] || !states["dismissed"] {
		t.Fatalf("audit: %v", states)
	}
}
