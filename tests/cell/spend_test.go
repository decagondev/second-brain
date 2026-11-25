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
	"example.com/second-brain/internal/core/budget"
)

// SB-5001 to SB-5004 through a running cell. An operator reads twelve weeks of AI use and
// its CSV, with small rows masked and nobody named. A frontier space answers on main. A
// person at 80% of the personal limit is told once; at 100% their long asks go to the
// economy tier, until a steward lifts the limit for the month, which is audited first. An
// agent's targets show in the value report.
func TestSpendWithPurposeThroughARunningCell(t *testing.T) {
	share := t.TempDir()
	if err := os.MkdirAll(filepath.Join(share, "Ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"right.md": "The zebra cutoff is 3pm on weekdays.", ".acl": "read domain:acme.example\n"} {
		if err := os.WriteFile(filepath.Join(share, "Ops", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := strings.Replace(opsManifest, "agents: [ask]\n",
		"agents:\n  ask: {}\n  incident-brief: { job: \"Writes the incident brief\", targets: [{ metric: published_as_is, at_least: 0.5 }] }\n"+
			"limits: { monthly_usd_per_person: 10 }\n", 1) +
		"  top:\n    members: [group:ops]\n    routing: frontier\n    scope: { docs: [\"/Ops/**\"] }\n"
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.FixtureFor(t, idp, base, m, nil)
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: "file://" + share}}
		cfg.Model.Small.ID = "small"
		return cfg
	})
	ctx := context.Background()
	told := &agentsTold{}
	r.cell.Send = told
	r.signIn(t, "lee")
	lee := r.idp.User
	r.signIn(t, "sam")
	sam := r.idp.User
	for name, members := range map[string][]string{"ops": {sam, lee}, "sb-operators": {sam}} {
		var ms []string
		for _, x := range members {
			ms = append(ms, `{"value":"`+x+`"}`)
		}
		if code, body := r.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
			"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
			t.Fatalf("SCIM group: %d %s", code, body)
		}
	}
	as := func(id string) {
		t.Helper()
		r.idp.User = id
		if code, _ := r.do(t, "GET", "/login", ""); code != http.StatusNotFound {
			t.Fatalf("sign-in: %d", code)
		}
	}
	as(sam)
	if fails := (&ingestsvc.Service{Cell: r.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatal(fails)
	}
	tierOf := func(space, q string) string {
		t.Helper()
		code, body := r.do(t, "POST", "/v1/ask?space="+space, `{"question":"`+q+`"}`)
		var a struct{ Trace string }
		if err := json.Unmarshal([]byte(body), &a); err != nil || code != 200 {
			t.Fatalf("ask: %d %s", code, body)
		}
		tr, _, err := r.cell.Traces.GetTrace(ctx, tenant, a.Trace)
		if err != nil {
			t.Fatal(err)
		}
		return tr.Tier
	}
	long := "When is the zebra cutoff " + strings.Repeat("on a weekday ", 10) + "?"

	// SB-5002: frontier answers even a short ask on main; ops is balanced.
	if got := tierOf("top", "When is the zebra cutoff?"); got != "main" {
		t.Fatalf("a frontier space went to %q", got)
	}
	for i := 0; i < 5; i++ {
		if got := tierOf("ops", "When is the zebra cutoff?"); got != "small" {
			t.Fatalf("a short ask in a balanced space went to %q", got)
		}
	}

	// SB-5001: the operator's view, masked under five, and the CSV with the same numbers.
	code, out := r.do(t, "GET", "/v1/admin/ai/usage", "")
	var u ask.AIUsage
	if err := json.Unmarshal([]byte(out), &u); err != nil || code != 200 || u.Weeks != 12 || u.Total.Uses < 6 || u.Limit == nil {
		t.Fatalf("AI use: %d %s", code, out)
	}
	var ops, top *ask.UsageRow
	for i, row := range u.Rows {
		switch {
		case row.Space == "ops" && row.Tier == "small":
			ops = &u.Rows[i]
		case row.Space == "top":
			top = &u.Rows[i]
		}
	}
	if ops == nil || ops.Uses != 5 || ops.People != 1 || top == nil || !top.Masked || top.Uses != 0 {
		t.Fatalf("rows: %s", out)
	}
	if strings.Contains(out, sam) || strings.Contains(out, "sam@") {
		t.Fatalf("AI use names a person: %s", out)
	}
	code, csv := r.do(t, "GET", "/v1/admin/ai/usage?format=csv", "")
	if code != 200 || !strings.HasPrefix(csv, "week,space,feature,agent,tier,uses,people,usd") || !strings.Contains(csv, ",top,ask,ask,main,fewer than 5,fewer than 5,") {
		t.Fatalf("CSV: %d %s", code, csv)
	}
	as(lee)
	if code, _ := r.do(t, "GET", "/v1/admin/ai/usage", ""); code != http.StatusForbidden {
		t.Fatalf("a non-operator read AI use: %d", code)
	}

	// SB-5003: at 80% lee is told once; at 100% lee's long ask goes to small.
	pm, ok := any(r.cell.Usage.Spend()).(budget.PersonMeter)
	if !ok {
		t.Fatal("the ledger keeps no person spend")
	}
	month := time.Now().UTC().Format("2006-01")
	if err := pm.AddPerson(ctx, tenant, lee, month, 8.5); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := r.cell.NotifyJob(told).Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := strings.Count(strings.Join(told.got, "\n"), "limit "+lee+": You have used 80%"); n != 1 {
		t.Fatalf("lee was told %d times: %v", n, told.got)
	}
	if got := tierOf("ops", long); got != "main" {
		t.Fatalf("under the limit a long ask went to %q", got)
	}
	if err := pm.AddPerson(ctx, tenant, lee, month, 2); err != nil {
		t.Fatal(err)
	}
	if got := tierOf("ops", long); got != "small" {
		t.Fatalf("at the limit a long ask went to %q", got)
	}

	// A steward lifts it for the month, audited.
	as(sam)
	if code, out := r.do(t, "POST", "/v1/spaces/ops/limits/lift", `{"person":"nobody@acme.example","reason":"quarter close"}`); code != http.StatusBadRequest {
		t.Fatalf("a lift for a non-member: %d %s", code, out)
	}
	if code, out := r.do(t, "POST", "/v1/spaces/ops/limits/lift", `{"person":"lee@acme.example","reason":"quarter close"}`); code != 200 || !strings.Contains(out, `"lifted":true`) {
		t.Fatalf("lift: %d %s", code, out)
	}
	if rows := r.auditRows(t); !strings.Contains(rows, `"subject":"limit:`+lee+`"`) || !strings.Contains(rows, "quarter close") {
		t.Fatalf("the lift isn't audited: %s", rows)
	}
	as(lee)
	if got := tierOf("ops", long); got != "main" {
		t.Fatalf("after the lift a long ask went to %q", got)
	}

	// SB-5004: the agent's job and target show in the value report.
	as(sam)
	var v ask.ValueReport
	code, out = r.do(t, "GET", "/v1/admin/agents/value", "")
	if err := json.Unmarshal([]byte(out), &v); err != nil || code != 200 || len(v.Goals) != 1 ||
		v.Goals[0].Agent != "incident-brief" || v.Goals[0].Job != "Writes the incident brief" || len(v.Goals[0].Targets[0].Weeks) != ask.GoalWeeks {
		t.Fatalf("goals: %d %s", code, out)
	}
}
