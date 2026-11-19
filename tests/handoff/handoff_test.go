// Package handoff rehearses the hand-off game day end to end (sb-factory(7), Hand-off game
// day, exactly; SB-710): a cell in the factory's account is handed to a tenant's GCP project
// (the recorded fake) through the real controller, wrap inventory, backups, restore, access
// check and registry, then detached on a copy. Every law the day checks is checked here.
package handoff

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	backupfs "example.com/second-brain/internal/adapters/backup/localfs"
	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/adapters/placement/gcp"
	"example.com/second-brain/internal/adapters/placement/gcp/gcptest"
	"example.com/second-brain/internal/adapters/placement/local"
	regfs "example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/backup"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/gameday"
	"example.com/second-brain/internal/core/handoff"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

const (
	tenant  = "gamma-parts"
	project = "gamma-prod-7731"
)

var (
	factorySA  = "serviceAccount:sb-factory@factory-prod.iam.gserviceaccount.com"
	breakGlass = "group:sb-breakglass@factory.example"
	deployRes  = "projects/" + project + "/serviceAccounts/sb-cell-deployer@" + project + ".iam.gserviceaccount.com"
)

// switchLog is the cell's audit log: the factory cell's until the move, the tenant cell's after.
type switchLog struct {
	mu  sync.Mutex
	cur *audit.Log
}

func (s *switchLog) Write(ctx context.Context, e audit.Event) (audit.Row, error) {
	s.mu.Lock()
	l := s.cur
	s.mu.Unlock()
	return l.Write(ctx, e)
}

// iamAccess is the controller's Access: the factory's IAM paths in the tenant project, revoked
// by removing the binding. (Key wraps go in the controller's last step, not at cutover.)
type iamAccess struct {
	src gcp.IAM
	f   *gcptest.Fake
}

func (a iamAccess) Paths(ctx context.Context, t string) ([]access.Path, error) {
	return a.src.Paths(ctx, t)
}

func (a iamAccess) Revoke(_ context.Context, _ string, id string) error {
	parts := strings.SplitN(id, ":", 3) // resource:role:member (the role holds no ':'; the member does)
	if len(parts) != 3 {
		return fmt.Errorf("bad path %q", id)
	}
	res := map[string]string{"project": "projects/" + project, "deploy-account": deployRes,
		"cell-key": "projects/" + project + "/locations/europe-west1/keyRings/sb-" + tenant + "/cryptoKeys/cell-" + tenant + "-root"}[parts[0]]
	a.f.Unbind(res, parts[1], parts[2])
	return nil
}

// target does the day's infrastructure work.
type target struct {
	t        *testing.T
	proj     gcp.Project
	log      *switchLog
	rows     *auditmem.Store // the factory cell's audit store
	signer   audit.Signer
	acts     *actmem.Store
	src      *mock.Source
	before   []string // the index before the move
	newRows  *auditmem.Store
	newIndex *memindex.Index
	reg      regfs.Dir
	now      func() time.Time
}

func (g *target) Provision(ctx context.Context, _ string, _ handoff.Plan) error {
	_, err := cell.Stamp(ctx, g.proj, g.proj, manifest.Tenant{Tenant: tenant, Region: "eu"})
	return err
}

func (g *target) Teardown(context.Context, string, handoff.Plan) error { return nil }

// Move backs up the factory cell, restores the chain and approvals into the tenant cell's
// empty stores, rebuilds its index from the sources, and points the audit log at it.
func (g *target) Move(ctx context.Context, _ string, _ handoff.Plan) error {
	ts := []backup.Target{backupfs.Dir{Root: g.t.TempDir(), Label: "primary", Media: "disk"},
		backupfs.Dir{Root: g.t.TempDir(), Label: "nas", Media: "nas"},
		backupfs.Dir{Root: g.t.TempDir(), Label: "offsite", Media: "object-store", Away: true}}
	job := backup.Job{Tenant: tenant, Audit: g.rows, Signer: g.signer, Approvals: g.acts, Targets: ts, Now: g.now}
	if _, err := job.Run(ctx); err != nil {
		return err
	}
	g.newRows = auditmem.New()
	if _, err := gameday.Restore(ctx, tenant, g.signer.Key(), ts, g.newRows, actmem.New()); err != nil {
		return err
	}
	g.newIndex = memindex.New()
	p := ingest.Pipeline{Tenant: tenant, Release: "rel_x", Connector: mock.New(g.src, false), Embedder: hashing.Embedder{}, Index: g.newIndex}
	if err := p.Rebuild(ctx); err != nil {
		return err
	}
	g.log.mu.Lock()
	g.log.cur = &audit.Log{Tenant: tenant, Release: func() string { return "rel_x" }, Store: g.newRows, Now: g.now}
	g.log.mu.Unlock()
	return nil
}

func snapshot(ctx context.Context, idx *memindex.Index) []string {
	cs, _ := idx.All(ctx, tenant)
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Cite + "|" + strings.Join(c.ACLPrefilter, ",")
	}
	return out
}

func (g *target) Verify(ctx context.Context, _ string, _ handoff.Plan) error {
	var buf bytes.Buffer
	if _, err := audit.Export(ctx, g.newRows, g.signer, 1, g.now(), &buf); err != nil {
		return err
	}
	if _, err := audit.Verify(&buf, tenant, g.signer.Key()); err != nil {
		return err
	}
	if !gameday.SameIndex(g.before, snapshot(ctx, g.newIndex)) {
		return fmt.Errorf("the rebuilt index differs")
	}
	return nil
}

func (g *target) Cutover(ctx context.Context, _ string, _ handoff.Plan) error {
	c, _, err := g.reg.CellByTenant(ctx, tenant)
	if err != nil {
		return err
	}
	c.Endpoint = "https://cell." + project + ".example"
	return g.reg.PutCell(ctx, c)
}

func TestHandoffGameDay(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2027, 5, 4, 9, 0, 0, 0, time.UTC)
	now := func() time.Time { at = at.Add(time.Second); return at }

	// The factory cell: a local account, its registry record, audit, approvals, index, key wrap.
	root := t.TempDir()
	fa := local.Account{Root: filepath.Join(root, "accounts"), Name: cell.PlacementFactoryAccount}
	rs, err := cell.Stamp(ctx, fa, fa, manifest.Tenant{Tenant: tenant, Region: "eu"})
	if err != nil {
		t.Fatal(err)
	}
	var factoryKey string
	for _, r := range rs {
		if r.Kind == cell.KindRootKey {
			factoryKey = r.ID
		}
	}
	reg := regfs.Dir{Root: filepath.Join(root, "registry")}
	if err := os.MkdirAll(reg.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := reg.PutCell(ctx, ports.CellRecord{Tenant: tenant, Region: "eu", Endpoint: "local://" + tenant, Hosts: []string{tenant + ".sb.example"},
		Custody: cell.CustodyFactory, Placement: cell.PlacementFactoryAccount, KeyCustodian: cell.KeyCustodianFactory}); err != nil {
		t.Fatal(err)
	}
	rows, acts := auditmem.New(), actmem.New()
	log := &switchLog{cur: &audit.Log{Tenant: tenant, Release: func() string { return "rel_x" }, Store: rows, Now: now}}
	for i := 0; i < 9; i++ {
		id := fmt.Sprintf("tc_%d", i)
		if _, err := log.Write(ctx, audit.Event{Actor: "usr_ann", Kind: "approval", Subject: id, Reason: "ok"}); err != nil {
			t.Fatal(err)
		}
		_ = acts.Put(ctx, approvals.Action{ID: id, Tenant: tenant, State: approvals.Ran})
	}
	src := mock.NewSource("docs")
	src.Put("d1", "Parts policy", "Returns over 500 need a supervisor.", "ann@gamma.example")
	src.Put("d2", "Suppliers", "The north depot closes on Fridays.", "ann@gamma.example", "bob@gamma.example")
	idx := memindex.New()
	if _, err := (ingest.Pipeline{Tenant: tenant, Release: "rel_x", Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: idx}).Sync(ctx, ""); err != nil {
		t.Fatal(err)
	}
	wraps := &keywrap.MemoryStore{}
	factory := keywrap.Custodian{Name: cell.KeyCustodianFactory, Keys: fa, KeyID: factoryKey}
	if err := keywrap.Seed(ctx, tenant, wraps, factory, keywrap.PurposeAuditSigning); err != nil {
		t.Fatal(err)
	}
	signer, _ := audit.NewSigner(tenant, bytes.Repeat([]byte{5}, 32))

	// The tenant's project, as the bootstrap script leaves it, plus two factory paths it must lose.
	f := gcptest.New()
	defer f.Server.Close()
	f.Grant("tok-deploy", project, gcp.DeployPermissions...)
	f.Grant("tok-verifier", project, gcp.VerifierPermissions...)
	f.Bind(deployRes, "roles/iam.serviceAccountTokenCreator", factorySA)
	f.Bind("projects/"+project, "roles/owner", breakGlass)
	f.Bind("projects/"+project, "projects/"+project+"/roles/sbPlatformKill", factorySA)
	f.Bind("projects/"+project, "roles/owner", "group:gamma-it@gamma.example") // the tenant's own
	ep := gcp.Endpoints{SQL: f.URL(), KMS: f.URL(), Secrets: f.URL(), PubSub: f.URL()}
	deploy := gcp.Project{ID: project, HTTP: f.Server.Client(), Poll: time.Millisecond, Endpoints: ep,
		Token: func(context.Context) (string, error) { return "tok-deploy", nil }}
	verifier := deploy
	verifier.Token = func(context.Context) (string, error) { return "tok-verifier", nil }
	iam := gcp.IAM{Project: verifier, Factory: []string{factorySA, breakGlass}, Location: "europe-west1"}
	iam.Endpoints.CRM, iam.Endpoints.IAM = f.URL(), f.URL()

	tg := &target{t: t, proj: deploy, log: log, rows: rows, signer: signer, acts: acts, src: src, before: snapshot(ctx, idx), reg: reg, now: now}
	tenantKey := "projects/" + project + "/locations/europe-west1/keyRings/sb-" + tenant + "/cryptoKeys/cell-" + tenant + "-root"
	ctl := &handoff.Controller{Store: &handoff.MemoryStore{}, Access: iamAccess{src: iam, f: f}, Target: tg, Custody: reg, Audit: log, Wraps: wraps,
		Factory: func(string) keywrap.Custodian { return factory },
		Tenant: func(string, handoff.Plan) keywrap.Custodian {
			return keywrap.Custodian{Name: cell.KeyCustodianTenant, Keys: deploy, KeyID: tenantKey}
		}}

	// The day.
	to := cell.TenantAccountPrefix + "gcp/" + project
	h, err := ctl.Plan(ctx, tenant, "fo_ann", to, "eu", []string{"platform-kill"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ctl.Approve(ctx, tenant, "factory", "fo_ann", h); err == nil {
		t.Fatal("the planner approved")
	}
	_ = ctl.Approve(ctx, tenant, "factory", "fo_bob", h)
	_ = ctl.Approve(ctx, tenant, "tenant", "to_dee", h)
	if ph, err := ctl.Execute(ctx, tenant, "fo_ann", h); err != nil || ph != handoff.TenantOwned {
		t.Fatalf("execute: %s %v", ph, err)
	}

	// LAW-17: only the retained grant's path is left, and no key wrap.
	res, err := access.Check(ctx, tenant, []access.Source{iam, access.Wraps{Store: wraps}}, []string{"publish-releases", "platform-kill"})
	if err != nil || len(res.Unlisted) != 0 || len(res.Paths) != 1 || res.Paths[0].Grant != "platform-kill" {
		t.Fatalf("access after hand-off: %+v %v", res, err)
	}
	// LAW-18: the tenant's KMS key unwraps the cell's data key; the factory's wrap is gone.
	inv, _ := keywrap.Inventory(ctx, tenant, wraps, map[string]ports.Keys{cell.KeyCustodianFactory: fa, cell.KeyCustodianTenant: deploy})
	if len(inv) != 1 || fmt.Sprint(inv[0].Working) != "[tenant]" || len(inv[0].Broken) != 0 {
		t.Fatalf("wraps: %+v", inv)
	}
	// LAW-11 across the move: the tenant cell's chain is the factory's, continued, and verifies.
	old, _ := rows.Range(ctx, 1, 1000)
	moved, _ := rows.Range(ctx, 1, 1000)
	cur, _ := tg.newRows.Range(ctx, 1, 1000)
	if len(cur) <= len(old) || fmt.Sprint(cur[:len(moved)]) != fmt.Sprint(moved) {
		t.Fatalf("the chain didn't move intact: %d rows before, %d after", len(old), len(cur))
	}
	var buf bytes.Buffer
	_, _ = audit.Export(ctx, tg.newRows, signer, 1, now(), &buf)
	if _, err := audit.Verify(&buf, tenant, signer.Key()); err != nil {
		t.Fatalf("the moved chain: %v", err)
	}
	custodyRows := 0
	for _, r := range cur {
		if r.Kind == "custody" {
			custodyRows++
		}
	}
	if custodyRows != 9 { // planned, 2 approvals, 6 steps
		t.Fatalf("%d custody rows, want 9", custodyRows)
	}
	c, _, _ := reg.CellByTenant(ctx, tenant)
	if c.Custody != cell.CustodyTenant || c.Placement != to || c.KeyCustodian != cell.KeyCustodianTenant || !strings.Contains(c.Endpoint, project) {
		t.Fatalf("custody record: %+v", c)
	}

	// Detach, on the same cell (in the game day, a copy of it).
	if err := ctl.Detach(ctx, tenant, "to_dee"); err != nil {
		t.Fatal(err)
	}
	res, err = access.Check(ctx, tenant, []access.Source{iam, access.Wraps{Store: wraps}}, nil)
	if err != nil || len(res.Paths) != 0 {
		t.Fatalf("access after detach: %+v %v", res, err)
	}
	if err := reg.Tombstone(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := reg.CellByHost(ctx, tenant+".sb.example"); found {
		t.Fatal("the detached cell still routes")
	}
	hist, _ := reg.History(ctx, tenant)
	if len(hist) != 2 || hist[1].Custody != cell.CustodyDetached {
		t.Fatalf("custody history: %+v", hist)
	}
}
