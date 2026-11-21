package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	accessfs "example.com/second-brain/internal/adapters/access/localfs"
	auditfs "example.com/second-brain/internal/adapters/audit/localfs"
	handofffs "example.com/second-brain/internal/adapters/handoff/localfs"
	"example.com/second-brain/internal/adapters/placement/local"
	regfs "example.com/second-brain/internal/adapters/registry/localfs"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/export"
	"example.com/second-brain/internal/core/handoff"
	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/errs"
)

// localTarget rehearses a hand-off on the local control plane (tenant-account://local/ID):
// it stamps the cell's resources in a second local account, moves a copy of the running
// cell's audit chain there, checks that copy end to end, and records the cutover. Locally
// both accounts are on one machine, so the running cell keeps serving throughout; on GCP
// the same steps are the gcp adapter's (SB-908).
type localTarget struct {
	root, region, cellURL string
}

func (t localTarget) account(p handoff.Plan) (local.Account, error) {
	provider, id, ok := cell.ParseTenantAccount(p.Target)
	if !ok || provider != "local" {
		return local.Account{}, fmt.Errorf("handoff: the local control plane hands off to %slocal/ID only; %s needs its provider's adapter (SB-908)", cell.TenantAccountPrefix, p.Target)
	}
	return local.Account{Root: filepath.Join(t.root, "accounts"), Name: id}, nil
}

func (t localTarget) moved(p handoff.Plan) (string, error) {
	a, err := t.account(p)
	if err != nil {
		return "", err
	}
	return filepath.Join(a.Root, a.Name, p.Tenant, "moved", "audit.ndjson"), nil
}

func (t localTarget) Provision(ctx context.Context, tenant string, p handoff.Plan) error {
	a, err := t.account(p)
	if err != nil {
		return err
	}
	_, err = cell.Stamp(ctx, a, a, manifest.Tenant{Tenant: tenant, Region: t.region})
	return err
}

func (t localTarget) Teardown(_ context.Context, tenant string, p handoff.Plan) error {
	a, err := t.account(p)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(a.Root, a.Name, tenant))
}

func (t localTarget) Move(_ context.Context, _ string, p handoff.Plan) error {
	if t.cellURL == "" {
		return fmt.Errorf("handoff: no running cell to move from; the registry's endpoint isn't a cell URL")
	}
	raw, err := cellDo(http.MethodGet, t.cellURL, "/v1/cell/audit/export", "", nil, 10*time.Minute)
	if err != nil {
		return err
	}
	path, err := t.moved(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// Verify checks the moved chain from its first row to its last, and that it reaches as far
// as the running cell's chain did when the move started.
func (t localTarget) Verify(_ context.Context, tenant string, p handoff.Plan) error {
	path, err := t.moved(p)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	rep, err := audit.Verify(bytes.NewReader(b), tenant, "")
	if err != nil {
		return err
	}
	if rep.Rows == 0 {
		return fmt.Errorf("handoff: the moved audit chain is empty")
	}
	return nil
}

func (t localTarget) Cutover(_ context.Context, _ string, p handoff.Plan) error {
	a, err := t.account(p)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.Root, a.Name, p.Tenant, "cutover"), []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
}

// controller is the local control plane's hand-off controller for one tenant's cell.
func (e *env) controller(tenant string) (*handoff.Controller, ports.CellRecord, error) {
	ep := e.g.Endpoint
	if ep == "" {
		ep = os.Getenv("SB_ENDPOINT")
	}
	root, ok := strings.CutPrefix(ep, "file://")
	if !ok {
		return nil, ports.CellRecord{}, errs.New(errs.EUNAVAIL, "hand-offs run on a local control plane (--endpoint file:///DIR) until the control-plane client (SB-105)")
	}
	deps, err := e.controlPlane()
	if err != nil {
		return nil, ports.CellRecord{}, err
	}
	reg := regfs.Dir{Root: root}
	ctx := context.Background()
	rec, found, err := reg.CellByTenant(ctx, tenant)
	switch {
	case err != nil:
		return nil, ports.CellRecord{}, errs.New(errs.EUNAVAIL, "tenant registry unavailable: %v", err)
	case !found:
		return nil, ports.CellRecord{}, errs.New(errs.EUSAGE, "no cell for tenant %q", tenant)
	}
	cellURL := ""
	if strings.HasPrefix(rec.Endpoint, "https://") || strings.HasPrefix(rec.Endpoint, "http://") {
		cellURL = rec.Endpoint
	}
	t := manifest.Tenant{Tenant: tenant, Region: rec.Region}
	rootKey := func(k ports.Keys) string {
		for _, r := range cell.Template(t) {
			if r.Kind == cell.KindRootKey {
				id, err := k.EnsureRootKey(ctx, r)
				if err == nil {
					return id
				}
			}
		}
		return ""
	}
	store := handofffs.Dir{Root: filepath.Join(root, "handoffs")}
	target := localTarget{root: root, region: rec.Region, cellURL: cellURL}
	ctl := &handoff.Controller{Store: store, Access: accessfs.Dir{Root: filepath.Join(root, "access")}, Target: target, Custody: reg,
		Audit: &audit.Log{Tenant: tenant, Release: func() string { return "control-plane" },
			Store: auditfs.Dir{Root: filepath.Join(root, "audit"), Tenant: tenant}, Now: time.Now},
		Wraps: deps.Wraps,
		Factory: func(string) keywrap.Custodian {
			return keywrap.Custodian{Name: cell.KeyCustodianFactory, Keys: deps.Keys, KeyID: rootKey(deps.Keys)}
		},
		Tenant: func(_ string, p handoff.Plan) keywrap.Custodian {
			a, err := target.account(p)
			if err != nil {
				return keywrap.Custodian{Name: cell.KeyCustodianTenant}
			}
			return keywrap.Custodian{Name: cell.KeyCustodianTenant, Keys: a, KeyID: rootKey(a)}
		}}
	return ctl, rec, nil
}

// findPlan finds the tenant whose current plan has this hash or unique prefix (12+ hex).
func findPlan(store handofffs.Dir, ref string) (string, string, error) {
	hex := strings.TrimPrefix(ref, "ph_")
	if len(hex) < 12 {
		return "", "", errs.New(errs.EUSAGE, "a plan is ph_ and at least 12 of its hex characters")
	}
	files, _ := filepath.Glob(filepath.Join(store.Root, "*.json"))
	var hits [][2]string
	for _, f := range files {
		tenant := strings.TrimSuffix(filepath.Base(f), ".json")
		s, found, err := store.Get(context.Background(), tenant)
		if err != nil {
			return "", "", errs.New(errs.EUNAVAIL, "%v", err)
		}
		if found && strings.HasPrefix(strings.TrimPrefix(s.Hash, "ph_"), hex) {
			hits = append(hits, [2]string{tenant, s.Hash})
		}
	}
	switch len(hits) {
	case 0:
		return "", "", errs.New(errs.EUSAGE, "no current plan %s", ref)
	case 1:
		return hits[0][0], hits[0][1], nil
	}
	return "", "", errs.New(errs.EUSAGE, "%s matches more than one plan", ref)
}

func handoffErr(err error) *errs.Error {
	switch {
	case errors.Is(err, handoff.ErrApproval):
		return errs.New(errs.ENOAPPROVAL, "%v", err)
	case errors.Is(err, handoff.ErrPhase), errors.Is(err, handoff.ErrPlan):
		return errs.New(errs.EINVAL, "%v", err)
	case errors.Is(err, handoff.ErrAccess):
		return errs.New(errs.ELAW, "%v", err)
	}
	return errs.New(errs.EUNAVAIL, "%v", err)
}

// factoryHandoff is sb factory handoff plan|show|approve|execute|abort (sb-factory(1)).
func (e *env) factoryHandoff(args []string) int {
	const name = "sb factory handoff"
	args = e.localArgs(args)
	if len(args) < 2 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb factory handoff plan|show|approve|execute|abort ..."))
	}
	sub, rest := args[0], args[1:]
	flags := map[string][]string{}
	var pos []string
	for i := 0; i < len(rest); i++ {
		if strings.HasPrefix(rest[i], "--") {
			if i+1 >= len(rest) {
				return e.fail(name, errs.New(errs.EUSAGE, "%s needs a value", rest[i]))
			}
			flags[rest[i]] = append(flags[rest[i]], rest[i+1])
			i++
			continue
		}
		pos = append(pos, rest[i])
	}
	one := func(f string) string {
		if v := flags[f]; len(v) == 1 {
			return v[0]
		}
		return ""
	}
	known := map[string][]string{"plan": {"--to", "--region", "--retain"}, "show": {}, "approve": {"--as"}, "execute": {}, "abort": {"--reason"}}
	allowed, ok := known[sub]
	if !ok || len(pos) != 1 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb factory handoff plan TENANT --to %sPROVIDER/ID --region REGION [--retain GRANT]... | show|approve|execute|abort PLAN", cell.TenantAccountPrefix))
	}
	for f := range flags {
		found := false
		for _, a := range allowed {
			found = found || f == a
		}
		if !found {
			return e.fail(name+" "+sub, errs.New(errs.EUSAGE, "unknown option %s", f))
		}
	}
	ctx := context.Background()
	if sub == "plan" {
		tenant := pos[0]
		ctl, rec, err := e.controller(tenant)
		if err != nil {
			return e.fail(name+" plan", err)
		}
		to, region := one("--to"), one("--region")
		if to == "" || region == "" {
			return e.fail(name+" plan", errs.New(errs.EUSAGE, "--to and --region are required"))
		}
		if region != rec.Region {
			return e.fail(name+" plan", errs.New(errs.EINVAL, "--region %s isn't the cell's pinned region %s", region, rec.Region))
		}
		h, err := ctl.Plan(ctx, tenant, actor(), to, region, flags["--retain"])
		if err != nil {
			return e.fail(name+" plan", handoffErr(err))
		}
		return e.showState(ctl, tenant, h)
	}
	store := handofffs.Dir{}
	if ep := strings.TrimPrefix(e.endpoint(), "file://"); ep != e.endpoint() {
		store = handofffs.Dir{Root: filepath.Join(ep, "handoffs")}
	}
	tenant, hash, err := findPlan(store, pos[0])
	if err != nil {
		return e.fail(name+" "+sub, err)
	}
	ctl, _, err := e.controller(tenant)
	if err != nil {
		return e.fail(name+" "+sub, err)
	}
	switch sub {
	case "approve":
		side := one("--as")
		if side != "factory" && side != "tenant" {
			return e.fail(name+" approve", errs.New(errs.EUSAGE, "--as factory|tenant"))
		}
		if err := ctl.Approve(ctx, tenant, side, actor(), hash); err != nil {
			return e.fail(name+" approve", handoffErr(err))
		}
	case "execute":
		if _, err := ctl.Execute(ctx, tenant, actor(), hash); err != nil {
			e.showState(ctl, tenant, hash)
			return e.fail(name+" execute", handoffErr(err))
		}
	case "abort":
		reason := one("--reason")
		if strings.TrimSpace(reason) == "" {
			return e.fail(name+" abort", errs.New(errs.EUSAGE, "--reason TEXT is required"))
		}
		if err := ctl.Abort(ctx, tenant, actor(), reason); err != nil {
			return e.fail(name+" abort", handoffErr(err))
		}
	}
	return e.showState(ctl, tenant, hash)
}

func (e *env) endpoint() string {
	if e.g.Endpoint != "" {
		return e.g.Endpoint
	}
	return os.Getenv("SB_ENDPOINT")
}

func (e *env) showState(ctl *handoff.Controller, tenant, hash string) int {
	s, err := ctl.State(context.Background(), tenant)
	if err != nil {
		return e.fail("sb factory handoff", handoffErr(err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"tenant": tenant, "plan": hash, "phase": s.Phase, "serving": s.Serving, "approvals": s.Approvals, "detail": s.Plan})
	}
	fmt.Fprintf(e.out, "plan     %s\ntenant   %s\nphase    %s\nserving  %s\n", s.Hash, tenant, s.Phase, s.Serving)
	if s.Plan != nil {
		fmt.Fprintf(e.out, "target   %s (region %s)\nretain   %s\nrevoke   %d paths\n", s.Plan.Target, s.Plan.Region, strings.Join(s.Plan.Retain, ", "), len(s.Plan.Revoke))
	}
	sides := make([]string, 0, len(s.Approvals))
	for side := range s.Approvals {
		sides = append(sides, side)
	}
	sort.Strings(sides)
	for _, side := range sides {
		fmt.Fprintf(e.out, "approved %s by %s\n", side, s.Approvals[side].Who)
	}
	return 0
}

// factoryDetach is sb factory detach TENANT --accept-unsupported --out DIR: the export
// package first, checked, then every factory path and the registry entry go.
func (e *env) factoryDetach(args []string) int {
	const name = "sb factory detach"
	args = e.localArgs(args)
	var tenant, out, cellURL string
	accept := false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--accept-unsupported":
			accept = true
		case (a == "--out" || a == "--cell") && i+1 < len(args):
			if a == "--out" {
				out = args[i+1]
			} else {
				cellURL = args[i+1]
			}
			i++
		case tenant == "" && !strings.HasPrefix(a, "-"):
			tenant = a
		default:
			return e.fail(name, errs.New(errs.EUSAGE, "usage: sb factory detach TENANT --accept-unsupported --out DIR"))
		}
	}
	if tenant == "" || out == "" {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb factory detach TENANT --accept-unsupported --out DIR"))
	}
	if !accept {
		return e.fail(name, errs.New(errs.EUSAGE, "--accept-unsupported is required: after detach the factory guarantees nothing about the cell"))
	}
	ctl, rec, err := e.controller(tenant)
	if err != nil {
		return e.fail(name, err)
	}
	ctx := context.Background()
	s, err := ctl.State(ctx, tenant)
	if err != nil {
		return e.fail(name, handoffErr(err))
	}
	if s.Phase != handoff.TenantOwned {
		return e.fail(name, errs.New(errs.EINVAL, "only a cell in tenant custody detaches; %s is %s", tenant, s.Phase))
	}
	if cellURL == "" {
		cellURL = rec.Endpoint
	}
	raw, err := cellDo(http.MethodGet, cellURL, "/v1/cell/package", "", nil, 10*time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	var pkg struct {
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's package is unreadable"))
	}
	files := map[string][]byte{}
	for p, b64 := range pkg.Files {
		b, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "the cell's package is unreadable"))
		}
		files[p] = b
	}
	repo, ok := findRepo()
	if !ok {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the runbooks aren't found; run from inside the repository"))
	}
	books, _ := filepath.Glob(filepath.Join(repo, "docs", "runbooks", "*.md"))
	for _, f := range books {
		b, err := os.ReadFile(f)
		if err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "%v", err))
		}
		files["runbooks/"+filepath.Base(f)] = b
	}
	if err := export.Write(out, export.Package{Tenant: tenant, Files: files}); err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "the package can't be written: %v", err))
	}
	if _, err := export.Verify(ctx, out); err != nil {
		return e.fail(name, errs.New(errs.ELAW, "the package doesn't verify, so nothing was detached: %v", err))
	}
	if err := ctl.Detach(ctx, tenant, actor()); err != nil {
		return e.fail(name, handoffErr(err))
	}
	if err := (regfs.Dir{Root: strings.TrimPrefix(e.endpoint(), "file://")}).Tombstone(ctx, tenant); err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "detached, but the registry entry stays: %v; run detach again", err))
	}
	fmt.Fprintf(e.out, "detach: %s left the factory; the package is in %s and verifies\n", tenant, out)
	return 0
}
