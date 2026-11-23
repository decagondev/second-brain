package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// SB-2903: sb space scope-report names a pattern matching nothing and one too wide.
func TestSpaceScopeReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/admin/index/scopes" {
			w.WriteHeader(404)
			return
		}
		_, _ = w.Write([]byte(`{"patterns":[{"space":"ops","kind":"docs","pattern":"/Opps/**","matches":0,"of":5,"problem":"none"},` +
			`{"space":"all","kind":"docs","pattern":"/**","matches":5,"of":5,"problem":"too_wide"},{"space":"ops","kind":"docs","pattern":"/Ops/**","matches":2,"of":5,"problem":""}]}`))
	}))
	defer srv.Close()
	t.Setenv("SB_TOKEN", "op-token")
	code, out, _ := run("space", "scope-report", "--cell", srv.URL)
	if code != 0 || !strings.Contains(out, "/Opps/**") || !strings.Contains(out, "matches nothing") || !strings.Contains(out, "too wide?") || !strings.Contains(out, "2 of 5\n") {
		t.Fatalf("exit %d %q", code, out)
	}
	if code, out, _ := run("--json", "space", "scope-report", "--cell", srv.URL); code != 0 || !strings.Contains(out, `"problem":"too_wide"`) {
		t.Fatalf("json: %d %q", code, out)
	}
	if code, _, _ := run("space", "scope-report", "--bogus"); code != 2 {
		t.Errorf("a bad flag: exit %d", code)
	}
}
