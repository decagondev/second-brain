package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	killmem "example.com/second-brain/internal/adapters/kill/memory"
	"example.com/second-brain/internal/adapters/policy/cedar"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/costs"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/retention"
	"example.com/second-brain/internal/core/tenancy"
)

var traces *tracemem.Store

type applied struct{ m manifest.Tenant }

func (a applied) Applied(context.Context) (manifest.Tenant, string, error) {
	return a.m, "abc1234", nil
}

func adminServer(t *testing.T) (*http.ServeMux, *auditmem.Store) {
	src, _ := os.ReadFile("../../examples/tenants/pilot.yaml")
	m, ds := manifest.Check(src)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	b, _ := os.ReadFile("../../policy/platform/base.cedar")
	engine, err := cedar.Load(map[string][]byte{"base.cedar": b})
	if err != nil {
		t.Fatal(err)
	}
	rows := auditmem.New()
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: rows, Now: time.Now}
	ks := killmem.New()
	h := Admin{Tenant: "acme-logistics", Policy: policy.Kernel{Engine: engine, Recorder: log}, Manifests: applied{m},
		Kill: kill.Control{Store: ks, Audit: log, Now: time.Now}, Switches: ks, Audit: log, AuditRows: rows, Release: "rel_x",
		Usage: emptyUsage{}, Approvers: []string{"it-leads"}}
	traces = tracemem.New()
	_ = traces.PutTrace(context.Background(), ask.Trace{ID: "tr_1", Tenant: "acme-logistics", User: "usr_ann", At: time.Now()})
	h.Erasure = retention.Job{Tenant: "acme-logistics", Days: 365, Stores: traces.Retention(), Audit: log, Now: time.Now}
	mux := http.NewServeMux()
	h.Register(mux)
	return mux, rows
}

func adminCall(t *testing.T, mux *http.ServeMux, groups []string, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	s, err := tenancy.NewIssuer("acme-logistics").Issue("acme-logistics", "usr_olga", groups)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(tenancy.WithScope(req.Context(), s))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestAdminNeedsTheOperatorRole(t *testing.T) {
	mux, _ := adminServer(t)
	for _, p := range []string{"/v1/admin/manifest", "/v1/admin/kill", "/v1/admin/audit"} {
		if w := adminCall(t, mux, []string{"grp_support"}, "GET", p, ""); w.Code != 403 {
			t.Errorf("%s without the role: %d", p, w.Code)
		}
	}
	if w := adminCall(t, mux, []string{"sb-operators"}, "GET", "/v1/admin/manifest", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "tenant: acme-logistics") {
		t.Fatalf("operator: %d %s", w.Code, w.Body)
	}
}

func TestDiffKillAndAudit(t *testing.T) {
	mux, rows := adminServer(t)
	op := []string{"sb-operators"}
	w := adminCall(t, mux, op, "POST", "/v1/admin/manifest/diff", `[{"op":"add","path":"/agents/-","value":"draft-reply"}]`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"+ agents: [ask, draft-reply]"`) {
		t.Fatalf("diff: %d %s", w.Code, w.Body)
	}
	if w := adminCall(t, mux, op, "POST", "/v1/admin/manifest/diff", `[{"op":"add","path":"/agents/-","value":"autopilot"}]`); w.Code != 422 || !strings.Contains(w.Body.String(), "EMANIFEST") {
		t.Fatalf("invalid patch: %d %s", w.Code, w.Body)
	}
	if w := adminCall(t, mux, op, "POST", "/v1/admin/manifest/diff", `[{"op":"replace","path":"/region","value":"us"}]`); !strings.Contains(w.Body.String(), "EMANIFEST") {
		t.Fatalf("immutable field: %d %s", w.Code, w.Body)
	}
	if w := adminCall(t, mux, op, "POST", "/v1/admin/kill", `{"scope":"agent","name":"ask","reason":"incident 42"}`); w.Code != 200 {
		t.Fatalf("kill: %d %s", w.Code, w.Body)
	}
	if w := adminCall(t, mux, op, "POST", "/v1/admin/kill", `{"scope":"agent","name":"ask"}`); w.Code != 400 {
		t.Fatalf("kill without a reason: %d", w.Code)
	}
	if w := adminCall(t, mux, op, "GET", "/v1/admin/kill", ""); !strings.Contains(w.Body.String(), "agent:ask") {
		t.Fatalf("list: %s", w.Body)
	}
	w = adminCall(t, mux, op, "GET", "/v1/admin/audit?kind=kill", "")
	if !strings.Contains(w.Body.String(), `"kind":"kill"`) || strings.Contains(w.Body.String(), `"kind":"config"`) {
		t.Fatalf("audit filter: %s", w.Body)
	}
	all, _ := rows.Range(context.Background(), 1, 100)
	kinds := map[string]int{}
	for _, r := range all {
		kinds[r.Kind]++
	}
	if kinds["config"] != 1 || kinds["kill"] != 1 || kinds["decision"] == 0 {
		t.Fatalf("audit kinds %v", kinds)
	}
	if w := adminCall(t, mux, op, "GET", "/v1/admin/refusals", ""); w.Code != 501 {
		t.Fatalf("refusals: %d", w.Code)
	}
	if w := adminCall(t, mux, []string{"it-leads"}, "GET", "/v1/admin/usage?week=2027-W10", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"week":"2027-W10"`) {
		t.Fatalf("usage as an approver: %d %s", w.Code, w.Body)
	}
	if w := adminCall(t, mux, []string{"grp_support"}, "GET", "/v1/admin/usage?week=2027-W10", ""); w.Code != 403 {
		t.Fatalf("usage without a role: %d", w.Code)
	}
}

type emptyUsage struct{}

func (emptyUsage) Traces(context.Context, string, time.Time, time.Time) ([]ask.Trace, error) {
	return nil, nil
}
func (emptyUsage) Drafts(context.Context, string, time.Time, time.Time) (int, int, error) {
	return 0, 0, nil
}
func (emptyUsage) Actions(context.Context, string, time.Time, time.Time) ([]approvals.Action, error) {
	return nil, nil
}
func (emptyUsage) Seats(context.Context, string) (int, error) { return 5, nil }
func (emptyUsage) Tokens(context.Context, string, time.Time, time.Time) (int, int, error) {
	return 0, 0, nil
}

func TestEraseSubject(t *testing.T) {
	mux, rows := adminServer(t)
	if w := adminCall(t, mux, []string{"grp_support"}, "POST", "/v1/admin/subjects/erase", `{"user":"usr_ann","reason":"r"}`); w.Code != 403 {
		t.Fatalf("without the role: %d", w.Code)
	}
	op := []string{"sb-operators"}
	for _, bad := range []string{`{"user":"ann@acme.example","reason":"r"}`, `{"user":"usr_ann"}`, `not json`} {
		if w := adminCall(t, mux, op, "POST", "/v1/admin/subjects/erase", bad); w.Code != 400 {
			t.Errorf("%s: %d", bad, w.Code)
		}
	}
	w := adminCall(t, mux, op, "POST", "/v1/admin/subjects/erase", `{"user":"usr_ann","reason":"subject request 42"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"traces":1`) {
		t.Fatalf("erase: %d %s", w.Code, w.Body)
	}
	if _, found, _ := traces.GetTrace(context.Background(), "acme-logistics", "tr_1"); found {
		t.Fatal("the trace is still there")
	}
	all, _ := rows.Range(context.Background(), 1, 100)
	var states []string
	for _, r := range all {
		if r.Kind == "retention" && r.Subject == "erase:usr_ann" && r.Actor == "usr_olga" {
			states = append(states, r.State)
		}
	}
	if strings.Join(states, ",") != "requested,done" {
		t.Fatalf("audit: %v", states)
	}
}

type usageMeter struct{}

func (usageMeter) Usage(context.Context, string, time.Time, time.Time) ([]costs.Usage, error) {
	return []costs.Usage{{Provider: "anthropic", Model: "claude-x", TokensIn: 2_000_000, TokensOut: 100_000}}, nil
}

// SB-709: the cost statement, for operators and approvers, as JSON or CSV.
func TestCostStatement(t *testing.T) {
	src, _ := os.ReadFile("../../examples/tenants/pilot.yaml")
	m, _ := manifest.Check(src)
	b, _ := os.ReadFile("../../policy/platform/base.cedar")
	engine, _ := cedar.Load(map[string][]byte{"base.cedar": b})
	rows := auditmem.New()
	log := &audit.Log{Tenant: "acme-logistics", Release: func() string { return "rel_x" }, Store: rows, Now: time.Now}
	h := Admin{Tenant: "acme-logistics", Policy: policy.Kernel{Engine: engine, Recorder: log}, Manifests: applied{m}, Audit: log, AuditRows: rows,
		Release: "rel_x", Approvers: []string{"it-leads"}, Meter: usageMeter{}, Prices: map[string]costs.Price{"anthropic/claude-x": {InPerM: 3, OutPerM: 15}},
		BudgetUSD: 1000, Now: func() time.Time { return time.Date(2027, 4, 2, 0, 0, 0, 0, time.UTC) }}
	mux := http.NewServeMux()
	h.Register(mux)
	if w := adminCall(t, mux, []string{"it-leads"}, "GET", "/v1/admin/costs?month=2027-03", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"total_usd":7.5`) {
		t.Fatalf("approver: %d %s", w.Code, w.Body)
	}
	if w := adminCall(t, mux, []string{"sb-operators"}, "GET", "/v1/admin/costs?month=2027-03&format=csv", ""); w.Code != 200 || !strings.HasPrefix(w.Body.String(), "tenant,month,partial") {
		t.Fatalf("csv: %d %s", w.Code, w.Body)
	}
	if w := adminCall(t, mux, []string{"grp_support"}, "GET", "/v1/admin/costs?month=2027-03", ""); w.Code != 403 {
		t.Fatalf("no role: %d", w.Code)
	}
	if w := adminCall(t, mux, []string{"it-leads"}, "GET", "/v1/admin/costs?month=March", ""); w.Code != 400 {
		t.Fatalf("bad month: %d", w.Code)
	}
}
