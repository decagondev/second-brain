// Package cell is the api service's smoke test (SB-1502): a whole cell built from one
// configuration by internal/cell/wire, signed in to through a test IdP, and one request
// of each kind sent through the routes the service mounts.
package cell

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

const tenant = "acme-logistics"

type running struct {
	srv  *httptest.Server
	cell *wire.Cell
	idp  *oidctest.IdP
	hc   *http.Client
}

func start(t *testing.T) running {
	t.Helper()
	return startWith(t, func(idp *oidctest.IdP, base string) wire.Config { return celltest.Fixture(t, idp, base) })
}

// startWith starts a cell from the configuration fixture returns.
func startWith(t *testing.T, fixture func(idp *oidctest.IdP, base string) wire.Config) running {
	t.Helper()
	idp := oidctest.New(tenant, "sb-acme-logistics", "oidc-secret")
	t.Cleanup(idp.Server.Close)
	h := http.NotFoundHandler()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }))
	srv.StartTLS()
	t.Cleanup(srv.Close)
	cell, err := wire.Build(context.Background(), fixture(idp, srv.URL), wire.Options{HTTP: idp.Server.Client(), ConnectorHTTP: idp.Server.Client(), Log: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	h = cell.Handler()
	hc := idp.Server.Client() // httptest servers share one certificate, so it trusts the cell too
	hc.Jar, _ = cookiejar.New(nil)
	return running{srv, cell, idp, hc}
}

func (r running) do(t *testing.T, method, path, body string, hdr ...string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, r.srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	resp, err := r.hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// signIn provisions a user over SCIM and signs them in through the IdP.
func (r running) signIn(t *testing.T, name string) {
	t.Helper()
	code, body := r.do(t, "POST", "/scim/v2/Users", `{"userName":"`+name+`","emails":[{"value":"`+name+`@acme.example"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
	if code != http.StatusCreated {
		t.Fatalf("SCIM create: %d %s", code, body)
	}
	var u struct{ ID string }
	if err := json.Unmarshal([]byte(body), &u); err != nil || u.ID == "" {
		t.Fatalf("SCIM create: %s", body)
	}
	r.idp.User = u.ID
	if code, body := r.do(t, "GET", "/login", ""); code != http.StatusNotFound { // ends at GET /, unserved without web
		t.Fatalf("sign-in ended at %d %s, want the redirect to / (404 without a web UI)", code, body)
	}
}

func TestHealthAndReadiness(t *testing.T) {
	r := start(t)
	if code, _ := r.do(t, "GET", "/healthz", ""); code != http.StatusNoContent {
		t.Fatalf("healthz %d", code)
	}
	if code, _ := r.do(t, "GET", "/readyz", ""); code != http.StatusNoContent {
		t.Fatalf("readyz %d", code)
	}
	r.cell.Stop()
	if code, _ := r.do(t, "GET", "/readyz", ""); code != http.StatusServiceUnavailable {
		t.Fatalf("readyz after Stop %d, want 503", code)
	}
	if code, _ := r.do(t, "GET", "/healthz", ""); code != http.StatusNoContent {
		t.Fatalf("healthz after Stop %d: a stopping cell is still alive", code)
	}
}

// TestOneOfEachKind signs in and sends one request to each family of routes the api
// mounts. None may fail with a server error, and nothing behind sign-in answers without it.
func TestOneOfEachKind(t *testing.T) {
	r := start(t)
	if code, _ := r.do(t, "POST", "/v1/ask", `{"question":"What is our refund limit?"}`); code != http.StatusUnauthorized {
		t.Fatalf("ask before sign-in: %d, want 401", code)
	}
	r.signIn(t, "ann")
	cases := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/v1/ask", `{"question":"What is our refund limit?"}`, http.StatusOK},
		{"POST", "/v1/drafts", `{"question":"Reply to the customer about refunds"}`, 0},
		{"GET", "/v1/actions/act_missing", "", http.StatusForbidden}, // never says whether it exists
		{"GET", "/v1/spaces", "", http.StatusOK},
		{"GET", "/v1/admin/kill", "", http.StatusForbidden}, // ann is no operator
		{"GET", "/v1/holds", "", http.StatusForbidden},
		{"GET", "/v1/admin/proposals", "", http.StatusForbidden},
		{"GET", "/v1/admin/reviews", "", http.StatusForbidden},
		{"GET", "/v1/verifier/attestation", "", http.StatusUnauthorized},
		{"POST", "/chat/slack/commands", "text=hi", http.StatusUnauthorized}, // unsigned
		{"POST", "/connectors/slack/events", "{}", http.StatusUnauthorized},
		{"GET", "/metrics", "", http.StatusOK},
	}
	for _, c := range cases {
		code, body := r.do(t, c.method, c.path, c.body)
		if code >= 500 || (c.want != 0 && code != c.want) {
			t.Errorf("%s %s: %d %s, want %d", c.method, c.path, code, body, c.want)
		}
	}
}

func TestRefusesToStart(t *testing.T) {
	idp := oidctest.New(tenant, "sb-acme-logistics", "oidc-secret")
	defer idp.Server.Close()
	for name, change := range map[string]func(c *wire.Config){
		"durable":      func(c *wire.Config) { c.Durable = true },
		"wrong tenant": func(c *wire.Config) { c.Tenant = "beta-freight" },
		"no policies":  func(c *wire.Config) { c.Policies = filepath.Join(c.State, "none") },
		"no state":     func(c *wire.Config) { c.State = filepath.Join(c.State, "missing") },
		"idp down":     func(c *wire.Config) { c.OIDC.Issuer = "https://127.0.0.1:1" },
		"no team id":   func(c *wire.Config) { c.Chat.TeamID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := celltest.Fixture(t, idp, "https://cell.example")
			change(&cfg)
			if _, err := wire.Build(context.Background(), cfg, wire.Options{HTTP: idp.Server.Client()}); !errors.Is(err, wire.ErrConfig) {
				t.Fatalf("Build: %v, want ErrConfig", err)
			}
		})
	}
	t.Run("unknown vendor", func(t *testing.T) {
		cfg := celltest.Fixture(t, idp, "https://cell.example")
		celltest.Write(t, cfg.Manifest, strings.Replace(celltest.Manifest, "docs: gdrive", "docs: sharepoint", 1), 0o600)
		if _, err := wire.Build(context.Background(), cfg, wire.Options{HTTP: idp.Server.Client()}); !errors.Is(err, wire.ErrConfig) {
			t.Fatalf("Build: %v, want ErrConfig", err)
		}
	})
}

// BUG-0026: a cell with its web UI configured panicked at start, because the UI's "GET /"
// clashed with SCIM's "/scim/v2/". Now it starts, serves the UI, and SCIM still answers.
func TestCellWithWebUI(t *testing.T) {
	web := t.TempDir()
	celltest.Write(t, filepath.Join(web, "index.html"), "<!doctype html><title>sb</title>", 0o600)
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg := celltest.Fixture(t, idp, base)
		cfg.Web = web
		return cfg
	})
	if code, body := r.do(t, "GET", "/", ""); code != 200 || !strings.Contains(body, "<title>sb</title>") {
		t.Fatalf("GET /: %d %q", code, body)
	}
	if code, _ := r.do(t, "POST", "/", "x"); code != 405 {
		t.Fatalf("POST /: %d, want 405", code)
	}
	if code, _ := r.do(t, "POST", "/scim/v2/Users", `{"userName":"ann"}`, "Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != 201 {
		t.Fatalf("SCIM beside the UI: %d", code)
	}
	if code, _ := r.do(t, "GET", "/v1/spaces", ""); code != 401 {
		t.Fatalf("/v1 beside the UI: %d, want 401", code)
	}
}
