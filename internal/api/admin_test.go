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
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/tenancy"
)

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
