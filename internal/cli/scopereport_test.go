package cli

import (
	"io"
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

// SB-2905: sb index prune --dry-run lists; without it, it needs a reason and posts it.
func TestIndexPrune(t *testing.T) {
	var posted string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/admin/index/leftovers":
			_, _ = w.Write([]byte(`{"gone":["docs:old.md@3"],"unheld":["docs:loose.md@1"]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/admin/index/prune":
			b, _ := io.ReadAll(r.Body)
			posted = string(b)
			_, _ = w.Write([]byte(`{"pruned":1}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	t.Setenv("SB_TOKEN", "op-token")
	code, out, _ := run("index", "prune", "--dry-run", "--cell", srv.URL)
	if code != 0 || !strings.Contains(out, "docs:old.md@3") || !strings.Contains(out, "Held by no space (kept; fix the scopes): 1") || posted != "" {
		t.Fatalf("dry run: %d %q", code, out)
	}
	if code, _, _ := run("index", "prune", "--cell", srv.URL); code != 2 || posted != "" {
		t.Fatalf("a prune without a reason: exit %d, posted %q", code, posted)
	}
	if code, out, _ := run("index", "prune", "--reason", "tombstone missed", "--cell", srv.URL); code != 0 || !strings.Contains(out, "Removed 1") || posted != `{"reason":"tombstone missed"}` {
		t.Fatalf("prune: %d %q %q", code, out, posted)
	}
}

// SB-3202: sb index refresh needs a cite and a reason, and says what happened.
func TestIndexRefresh(t *testing.T) {
	var posted string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/admin/index/refresh" {
			b, _ := io.ReadAll(r.Body)
			posted = string(b)
			_, _ = w.Write([]byte(`{"document":"docs:a.md","version":7,"changed":true,"removed":false}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	t.Setenv("SB_TOKEN", "op-token")
	if code, _, _ := run("index", "refresh", "docs:a.md", "--cell", srv.URL); code != 2 || posted != "" {
		t.Fatalf("a refresh without a reason: exit %d, posted %q", code, posted)
	}
	code, out, _ := run("index", "refresh", "docs:a.md@3", "--reason", "owner says it changed", "--cell", srv.URL)
	if code != 0 || !strings.Contains(out, "docs:a.md: now version 7.") || posted != `{"cite":"docs:a.md@3","reason":"owner says it changed"}` {
		t.Fatalf("refresh: %d %q %q", code, out, posted)
	}
}
