package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// SB-1905: sb tenant retention --plan prints the cell's plan.
func TestTenantRetentionPlan(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/admin/retention/plan" {
			w.WriteHeader(404)
			return
		}
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"days":30,"cutoff":"2026-01-22T03:00:00Z","would_delete":{"traces":120,"feedback":11},"held":7}`))
	}))
	defer srv.Close()
	t.Setenv("SB_TOKEN", "op-token")
	code, out, _ := run("tenant", "retention", "--plan", "--days", "30", "--cell", srv.URL)
	if code != 0 || query != "days=30" || !strings.Contains(out, "in 30 days") || !strings.Contains(out, "total              131") || !strings.Contains(out, "legal hold: 7") {
		t.Fatalf("exit %d %q (query %q)", code, out, query)
	}
	if code, out, _ := run("--json", "tenant", "retention", "--plan", "--cell", srv.URL); code != 0 || !strings.Contains(out, `"held":7`) {
		t.Fatalf("json: %d %q", code, out)
	}
	for _, bad := range [][]string{{"--cell", srv.URL}, {"--plan", "--days", "-1", "--cell", srv.URL}, {"--plan", "--bogus"}} {
		if code, _, _ := run(append([]string{"tenant", "retention"}, bad...)...); code != 2 {
			t.Errorf("%v: exit %d", bad, code)
		}
	}
}
