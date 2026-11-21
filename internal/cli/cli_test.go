package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestVersion(t *testing.T) {
	if c, out, _ := run("version"); c != 0 || !strings.HasPrefix(out, "sb 0.0.0-dev") {
		t.Fatalf("version: %d %q", c, out)
	}
	c, out, _ := run("--json", "version")
	var v map[string]string
	if c != 0 || json.Unmarshal([]byte(out), &v) != nil || v["version"] == "" {
		t.Fatalf("--json version: %d %q", c, out)
	}
}

// Every command in the sb(1) SYNOPSIS exists and exits ENOTIMPL (2) naming its ticket.
func TestPlannedCommandsAreHonest(t *testing.T) {
	for name, ticket := range planned {
		args := append(strings.Fields(name), "x")
		c, _, errOut := run(args...)
		if c != 2 || !strings.Contains(errOut, ticket) || !strings.Contains(errOut, "[ENOTIMPL]") {
			t.Errorf("%s: exit %d, stderr %q; want 2 and %s ENOTIMPL", name, c, errOut, ticket)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"nosuch"}, {"tenant"}, {"tenant", "nosuch"}, {"--bogus", "version"}, {"--tenant"}} {
		if c, _, errOut := run(args...); c != 2 || !strings.Contains(errOut, "[EUSAGE]") {
			t.Errorf("%v: exit %d %q; want 2 EUSAGE", args, c, errOut)
		}
	}
	if c, _, _ := run(); c != 2 {
		t.Error("no command must exit 2")
	}
	if c, out, _ := run("--help"); c != 0 || !strings.Contains(out, "usage: sb") {
		t.Error("--help must exit 0 with usage")
	}
}

func TestJSONErrors(t *testing.T) {
	c, out, _ := run("--json", "factory", "handoff", "plan", "a")
	var v struct {
		Error struct{ Code, Message, Remedy string }
	}
	if c != 2 || json.Unmarshal([]byte(out), &v) != nil || v.Error.Code != "ENOTIMPL" || v.Error.Remedy == "" {
		t.Fatalf("json error: %d %q", c, out)
	}
}

// The manual's SYNOPSIS and the dispatcher must agree (P2: the manual is the spec).
func TestSynopsisMatchesManual(t *testing.T) {
	for _, cmd := range []string{"tenant", "kill", "unkill", "audit", "release", "eval", "factory"} {
		if !hasSub(cmd) {
			t.Errorf("sb(1) lists %q but the CLI has no subcommands for it", cmd)
		}
	}
}

// SB-708: verify-export fails closed on a directory that isn't a package (the passing case
// is the LAW-11 export law test, over real packages).
func TestVerifyExportFailsClosed(t *testing.T) {
	if code, _, _ := run("factory", "verify-export"); code != 2 {
		t.Fatalf("no dir: exit %d", code)
	}
	code, out, errOut := run("factory", "verify-export", t.TempDir())
	if code != 1 || !strings.Contains(out+errOut, "ELAW") {
		t.Fatalf("empty dir: exit %d %q %q", code, out, errOut)
	}
}

// SB-905: sb tenant verify --cell posts to the cell's verifier and exits on its report.
func TestTenantVerifyCell(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		if r.URL.Path != "/v1/verifier/verify" || !strings.Contains(string(b), `"laws":["LAW-01"]`) {
			w.WriteHeader(400)
			return
		}
		_, _ = w.Write([]byte(`{"Seed":1,"Results":[{"Law":"LAW-01","Status":"pass","Detail":"12 citations, all openable now"}]}`))
	}))
	defer srv.Close()
	t.Setenv("SB_TOKEN", "cp-token")
	code, out, _ := run("tenant", "verify", "../../examples/tenants/pilot.yaml", "--cell", srv.URL, "--law", "LAW-01")
	if code != 0 || !strings.Contains(out, "LAW-01  pass") || got != "Bearer cp-token" {
		t.Fatalf("exit %d %q, auth %q", code, out, got)
	}
	if code, _, _ := run("tenant", "verify", "../../examples/tenants/pilot.yaml", "--cell", "http://evil.example", "--law", "LAW-01"); code != 2 {
		t.Fatalf("a plain-http cell URL: exit %d", code)
	}
}

// SB-1506: sb tenant apply --cell applies to the control plane, then sends the manifest
// to the running cell with the control plane's identity.
func TestTenantApplyCell(t *testing.T) {
	var got, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		if r.URL.Path != "/v1/cell/manifest" || r.Header.Get("Content-Type") != "application/yaml" {
			w.WriteHeader(400)
			return
		}
		_, _ = w.Write([]byte(`{"sha256":"abc","changed":true,"changes":["+ agent ask"]}`))
	}))
	defer srv.Close()
	t.Setenv("SB_TOKEN", "cp-token")
	reg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(reg, "tenants"), 0o755); err != nil {
		t.Fatal(err)
	}
	ep := "file://" + reg
	code, out, errOut := run("--endpoint", ep, "tenant", "apply", "../../examples/tenants/pilot.yaml", "--cell", srv.URL)
	if code != 0 || !strings.Contains(out, "the cell runs sha256:abc") || got != "Bearer cp-token" || !strings.Contains(body, "tenant: acme-logistics") {
		t.Fatalf("exit %d %q %q, auth %q", code, out, errOut, got)
	}
	if code, _, _ := run("--endpoint", ep, "tenant", "apply", "../../examples/tenants/pilot.yaml", "--plan", "--cell", srv.URL); code != 2 {
		t.Fatalf("--plan with --cell: exit %d", code)
	}
}
