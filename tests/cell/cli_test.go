package cell

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	regfs "example.com/second-brain/internal/adapters/registry/localfs"
	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
	"example.com/second-brain/internal/cli"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/obs"
)

// sb runs the sb command in-process.
func sb(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cli.Run(args, &out, &errOut)
	return code, out.String() + errOut.String()
}

// liveCell is a running local cell on plain HTTP (the control plane's routes need no
// cookie), with a release deployed by its updater, an anchor and attestation key, and the
// control plane's identity; and a local control plane whose registry points at it.
type liveCell struct {
	cell *wire.Cell
	url  string
	cp   string // the control plane's directory
}

func startLive(t *testing.T) liveCell {
	t.Helper()
	idp := oidctest.New(tenant, "sb-acme-logistics", "oidc-secret")
	t.Cleanup(idp.Server.Close)
	seed := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	cfg := celltest.FixtureFor(t, idp, "http://127.0.0.1", docsManifest, map[string]string{
		"control-plane": cpToken, "audit/anchor-seed": seed, "attest-seed": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32))),
		"connectors/docs": `{"type":"service_account"}`})
	store := relfs.Dir{Root: filepath.Join(t.TempDir(), "releases")}
	cfg.Release = wire.ReleaseConfig{Store: "file://" + filepath.ToSlash(store.Root), Registry: "registry.example/sb", Trusted: relKey.Public()}
	cfg.Release.Deploy.Kind = "local"
	rel := publish(t, store, 1, true, true)
	if err := store.SetChannel("stable", rel); err != nil {
		t.Fatal(err)
	}
	job, err := wire.NewUpdater(cfg, &alerts{}, obs.NewMetrics())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	c, err := wire.Build(context.Background(), cfg, wire.Options{HTTP: idp.Server.Client(), Log: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(c.Handler())
	t.Cleanup(srv.Close)
	for i := 0; i < 5; i++ {
		if _, err := c.Audit.Write(context.Background(), audit.Event{Actor: "usr_ann", Kind: "approval", Subject: "tc_" + string(rune('a'+i)), Reason: "ok"}); err != nil {
			t.Fatal(err)
		}
	}

	cp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cp, "tenants"), 0o755); err != nil {
		t.Fatal(err)
	}
	mf := filepath.Join(t.TempDir(), "tenant.yaml")
	if err := os.WriteFile(mf, []byte(docsManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := sb(t, "--endpoint", "file://"+cp, "tenant", "apply", mf); code != 0 {
		t.Fatalf("control-plane apply: %d %s", code, out)
	}
	reg := regfs.Dir{Root: cp}
	rec, _, err := reg.CellByTenant(context.Background(), tenant)
	if err != nil {
		t.Fatal(err)
	}
	rec.Endpoint = srv.URL
	if err := reg.PutCell(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SB_TOKEN", cpToken)
	return liveCell{cell: c, url: srv.URL, cp: cp}
}

// TestAuditAgainstARunningCell (SB-1507, LAW-11): verify, tail and export reach a running
// cell through the registry or --cell, and what they read verifies.
func TestAuditAgainstARunningCell(t *testing.T) {
	l := startLive(t)
	if code, out := sb(t, "--endpoint", "file://"+l.cp, "audit", "verify", "--tenant", tenant); code != 0 || !strings.Contains(out, "chain intact") {
		t.Fatalf("verify through the registry: %d %s", code, out)
	}
	if code, out := sb(t, "audit", "tail", "--cell", l.url, "--kind", "approval", "--no-follow"); code != 0 || strings.Count(out, "approval") != 5 {
		t.Fatalf("tail: %d %s", code, out)
	}
	if code, out := sb(t, "--json", "audit", "tail", "--cell", l.url, "--actor", "usr_nobody", "--no-follow"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("tail with no match: %d %q", code, out)
	}
	f := filepath.Join(t.TempDir(), "audit.ndjson")
	if code, out := sb(t, "audit", "export", "--cell", l.url, "--since", "2000-01-01T00:00:00Z", "--out", f); code != 0 || !strings.Contains(out, "verified") {
		t.Fatalf("export: %d %s", code, out)
	}
	if code, out := sb(t, "--tenant", tenant, "audit", "verify", "--export", f); code != 0 {
		t.Fatalf("the exported file doesn't verify offline: %d %s", code, out)
	}
	t.Setenv("SB_TOKEN", "wrong")
	if code, out := sb(t, "audit", "verify", "--cell", l.url); code != 3 || !strings.Contains(out, "EPERM") {
		t.Fatalf("a wrong identity: %d %s", code, out)
	}
}

// TestHandoffAndDetachRehearsal (SB-1507, LAW-17, LAW-18): the hand-off game day against a
// running local cell, then detach: two sides approve one hash, execute runs every step,
// only the retained grant's path is left, and the detach package verifies on its own.
func TestHandoffAndDetachRehearsal(t *testing.T) {
	l := startLive(t)
	ep := "file://" + l.cp
	paths := `[{"kind":"iam-binding","id":"deploy:roles/run.admin:sb-factory","grant":""},
 {"kind":"iam-binding","id":"kill:roles/sbPlatformKill:sb-factory","grant":"platform-kill"},
 {"kind":"iam-binding","id":"release:roles/pull:sb-factory","grant":"publish-releases"}]`
	if err := os.MkdirAll(filepath.Join(l.cp, "access"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.cp, "access", tenant+".json"), []byte(paths), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SB_ACTOR", "fo_ann")
	if code, out := sb(t, "--endpoint", ep, "factory", "handoff", "plan", tenant, "--to", "tenant-account://local/acme-own", "--region", "us"); code != 2 || !strings.Contains(out, "EINVAL") {
		t.Fatalf("a plan in another region: %d %s", code, out)
	}
	code, out := sb(t, "--json", "--endpoint", ep, "factory", "handoff", "plan", tenant, "--to", "tenant-account://local/acme-own", "--region", "eu", "--retain", "platform-kill")
	var st struct {
		Plan, Phase string
	}
	if code != 0 || json.Unmarshal([]byte(out), &st) != nil || !strings.HasPrefix(st.Plan, "ph_") {
		t.Fatalf("plan: %d %s", code, out)
	}
	ref := st.Plan[:15]
	if code, out := sb(t, "--endpoint", ep, "factory", "handoff", "execute", ref); code != 5 || !strings.Contains(out, "ENOAPPROVAL") {
		t.Fatalf("execute before approvals: %d %s", code, out)
	}
	if code, out := sb(t, "--endpoint", ep, "factory", "handoff", "approve", ref, "--as", "factory"); code == 0 {
		t.Fatalf("the planner approved for the factory: %s", out)
	}
	t.Setenv("SB_ACTOR", "fo_bob")
	if code, out := sb(t, "--endpoint", ep, "factory", "handoff", "approve", ref, "--as", "factory"); code != 0 {
		t.Fatalf("factory approval: %d %s", code, out)
	}
	t.Setenv("SB_ACTOR", "to_dee")
	if code, out := sb(t, "--endpoint", ep, "factory", "handoff", "approve", ref, "--as", "tenant"); code != 0 {
		t.Fatalf("tenant approval: %d %s", code, out)
	}
	t.Setenv("SB_ACTOR", "fo_ann")
	if code, out := sb(t, "--endpoint", ep, "factory", "handoff", "execute", ref); code != 0 || !strings.Contains(out, "TenantOwned") {
		t.Fatalf("execute: %d %s", code, out)
	}
	if code, out := sb(t, "--endpoint", ep, "factory", "access", tenant); code != 0 || strings.Contains(out, "run.admin") {
		t.Fatalf("LAW-17 after hand-off: %d %s", code, out)
	}
	if code, out := sb(t, "--endpoint", ep, "factory", "status", tenant); code != 0 || !strings.Contains(out, "tenant") {
		t.Fatalf("status: %d %s", code, out)
	}

	dir := filepath.Join(t.TempDir(), "export")
	if code, out := sb(t, "--endpoint", ep, "factory", "detach", tenant, "--out", dir); code != 2 {
		t.Fatalf("detach without --accept-unsupported: %d %s", code, out)
	}
	if code, out := sb(t, "--endpoint", ep, "factory", "detach", tenant, "--accept-unsupported", "--out", dir); code != 0 {
		t.Fatalf("detach: %d %s", code, out)
	}
	if code, out := sb(t, "factory", "verify-export", dir); code != 0 {
		t.Fatalf("the package doesn't verify: %d %s", code, out)
	}
	if _, found, _ := (regfs.Dir{Root: l.cp}).CellByTenant(context.Background(), tenant); found {
		t.Fatal("the registry still routes to a detached cell")
	}
	if b, _ := os.ReadFile(filepath.Join(l.cp, "access", tenant+".json")); strings.Contains(string(b), "sb-factory") {
		t.Fatalf("a factory path survived detach: %s", b)
	}
}
