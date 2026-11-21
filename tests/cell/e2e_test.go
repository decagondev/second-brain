package cell

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/gdrive/drivetest"
	"example.com/second-brain/internal/adapters/connectors/linear/lineartest"
	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/adapters/postgres/cellstore"
	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

const e2eManifest = `tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: gdrive, tickets: linear }
agents: [ask, draft-reply]
workflows: [ticket-update]
approval: { ticket-update: group:it-leads }
retention_days: 365
budget_usd_month: 1500
spaces:
  legal:
    template: legal
    members: [group:legal]
    stewards: [group:legal-leads]
    scope:
      docs: ["/Legal/**"]
`

// freshDSN is SB_TEST_POSTGRES_DSN in a new schema of its own, or skips.
func freshDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("SB_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SB_TEST_POSTGRES_DSN not set (make integration)")
	}
	ctx := context.Background()
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	name := "e2e_" + hex.EncodeToString(b)
	pool, err := cellstore.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA "+name+" CASCADE")
		pool.Close()
	})
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "search_path=" + name + ",public" // pgvector lives in public
}

type e2e struct {
	running
	plain *httptest.Server // the same cell on plain HTTP, for sb's control-plane calls
	drive *drivetest.Fake
	lin   *lineartest.Fake
	cfg   wire.Config
}

func startE2E(t *testing.T) e2e {
	t.Helper()
	dsn := freshDSN(t)
	drive, lin := drivetest.New(), lineartest.New()
	t.Cleanup(drive.Server.Close)
	t.Cleanup(lin.Server.Close)
	var cfg wire.Config
	r := startWith(t, func(idp *oidctest.IdP, base string) wire.Config {
		cfg = celltest.FixtureFor(t, idp, base, e2eManifest, map[string]string{
			"control-plane": cpToken, "connectors/docs": drive.KeyJSON(), "connectors/tickets": lin.Key,
			"audit/anchor-seed": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))})
		cfg.Connectors = map[string]wire.ConnectorConfig{"docs": {Base: drive.Base()}, "tickets": {Base: lin.Endpoint()}}
		cfg.Database, cfg.Durable = dsn, true
		return cfg
	})
	t.Cleanup(r.cell.Close)
	plain := httptest.NewServer(r.cell.Handler())
	t.Cleanup(plain.Close)
	return e2e{running: r, plain: plain, drive: drive, lin: lin, cfg: cfg}
}

// user provisions a SCIM user and signs them in with a client of their own.
func (e e2e) user(t *testing.T, name string) (*http.Client, string) {
	t.Helper()
	code, body := e.do(t, "POST", "/scim/v2/Users", `{"userName":"`+name+`","emails":[{"value":"`+name+`@acme.example"}]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json")
	if code != http.StatusCreated {
		t.Fatalf("SCIM create %s: %d %s", name, code, body)
	}
	var u struct{ ID string }
	_ = json.Unmarshal([]byte(body), &u)
	e.idp.User = u.ID
	base := *e.idp.Server.Client() // Client() is one shared client: copy it, or every user shares a jar
	hc := &base
	hc.Jar, _ = cookiejar.New(nil)
	resp, err := hc.Get(e.srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return hc, u.ID
}

func (e e2e) group(t *testing.T, name string, members ...string) {
	t.Helper()
	var ms []string
	for _, m := range members {
		ms = append(ms, `{"value":"`+m+`"}`)
	}
	if code, body := e.do(t, "POST", "/scim/v2/Groups", `{"displayName":"`+name+`","members":[`+strings.Join(ms, ",")+`]}`,
		"Authorization", "Bearer scim-token", "Content-Type", "application/scim+json"); code != http.StatusCreated {
		t.Fatalf("SCIM group %s: %d %s", name, code, body)
	}
}

func as(t *testing.T, hc *http.Client, base, method, path, body string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	var v map[string]any
	_ = json.Unmarshal(b, &v)
	if v == nil {
		v = map[string]any{"_raw": string(b)}
	}
	return resp.StatusCode, v
}

// TestWholeCellOnPostgres (SB-1508): a durable cell on Postgres asks, drafts, proposes
// and approves an action that reaches the vendor, places a legal hold, passes sb tenant
// verify --cell, and keeps all of it across a restart.
func TestWholeCellOnPostgres(t *testing.T) {
	e := startE2E(t)
	ctx := context.Background()
	ann, annID := e.user(t, "ann")
	bo, boID := e.user(t, "bo")
	carol, carolID := e.user(t, "carol")
	e.group(t, "it-leads", boID)
	e.group(t, "legal-leads", carolID)
	e.group(t, "everyone", annID, boID, carolID)

	const phrase = "the north depot ships returns on tuesdays"
	e.drive.Put("doc1", "Returns", "text/plain", "Remember: "+phrase+" only.", "ann@acme.example", "bo@acme.example")
	if fails := (&ingestsvc.Service{Cell: e.cell, Alert: &alerts{}}).SyncAll(ctx); len(fails) > 0 {
		t.Fatalf("sync: %v", fails)
	}
	if all, err := e.cell.Index.All(ctx, tenant); err != nil || len(all) == 0 {
		t.Fatalf("nothing indexed: %d %v", len(all), err)
	}
	code, v := as(t, ann, e.srv.URL, "POST", "/v1/ask", `{"question":"When does the north depot ship returns?"}`)
	if code != 200 || !strings.Contains(strings.ToLower(toJSON(v)), "tuesdays") {
		t.Fatalf("ask: %d %v", code, v)
	}
	trace, _ := v["trace"].(string)
	e.lin.AddUser("lu_ann", "ann@acme.example", false)
	e.lin.AddTeam("lt_ops", "OPS", false, "lu_ann")
	e.lin.Put("li_1", "lt_ops", "Late return", "A customer's return is late.")
	if code, v := as(t, ann, e.srv.URL, "POST", "/v1/drafts", `{"context":"tickets:li_1","instruction":"Tell the customer when returns ship."}`); code != 200 {
		t.Fatalf("draft: %d %v", code, v)
	}

	code, v = as(t, ann, e.srv.URL, "POST", "/v1/actions", `{"workflow":"ticket-update","target":"tickets:li_1","payload":{"comment":"Returns ship on Tuesdays."}}`)
	if code != 200 || v["state"] != "PendingApproval" {
		t.Fatalf("propose: %d %v", code, v)
	}
	id, hash := v["id"].(string), v["payload_sha256"].(string)
	if code, v := as(t, ann, e.srv.URL, "POST", "/v1/actions/"+id+"/approve", `{"payload_sha256":"`+hash+`"}`); code == 200 {
		t.Fatalf("LAW-15: the requester approved their own action: %v", v)
	}
	if code, v := as(t, bo, e.srv.URL, "POST", "/v1/actions/"+id+"/approve", `{"payload_sha256":"`+hash+`"}`); code != 200 || v["state"] != "Ran" {
		t.Fatalf("approve: %d %v", code, v)
	}
	if len(e.lin.Comments) != 1 {
		t.Fatalf("the vendor got %d comments, want 1", len(e.lin.Comments))
	}

	if code, v := as(t, ann, e.srv.URL, "POST", "/v1/holds", `{"matter":"m-1","users":["`+annID+`"],"reason":"litigation"}`); code == 200 {
		t.Fatalf("a non-steward placed a hold: %v", v)
	}
	if code, v := as(t, carol, e.srv.URL, "POST", "/v1/holds", `{"matter":"m-1","users":["`+annID+`"],"reason":"litigation"}`); code != 200 {
		t.Fatalf("hold: %d %v", code, v)
	}

	mf := filepath.Join(t.TempDir(), "tenant.yaml")
	_ = os.WriteFile(mf, []byte(e2eManifest), 0o600)
	foreign := filepath.Join(t.TempDir(), "foreign.json")
	_ = os.WriteFile(foreign, []byte(`["docs:beta-freight-doc-1"]`), 0o600)
	t.Setenv("SB_TOKEN", cpToken)
	if code, out := sb(t, "tenant", "verify", mf, "--cell", e.plain.URL, "--sample", "4", "--foreign", foreign); code != 0 {
		t.Fatalf("sb tenant verify --cell: %d\n%s", code, out)
	}

	// A restart on the same database keeps everything.
	again, err := wire.Build(ctx, e.cfg, wire.Options{HTTP: e.idp.Server.Client(), ConnectorHTTP: e.idp.Server.Client(), Log: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if _, found, err := again.Traces.GetTrace(ctx, tenant, trace); err != nil || !found {
		t.Fatalf("the trace didn't survive a restart: %v %v", found, err)
	}
	if a, found, _ := again.Actions.Get(ctx, tenant, id); !found || a.State != "Ran" {
		t.Fatalf("the action didn't survive a restart: %+v", a)
	}
	if hs, _ := again.Holds.List(ctx); len(hs) != 1 {
		t.Fatalf("the hold didn't survive a restart: %v", hs)
	}
	if u, found, _ := again.Directory.User(ctx, tenant, boID); !found || u.UserName != "bo" {
		t.Fatal("the directory didn't survive a restart")
	}
	if rows, _ := again.Audit.Store.Range(ctx, 1, 1000); len(rows) == 0 {
		t.Fatal("the audit log didn't survive a restart")
	}
}

func toJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
