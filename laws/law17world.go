package laws

import (
	"context"
	"errors"
	"path/filepath"

	"example.com/second-brain/internal/adapters/placement/local"
	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/handoff"
	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

const hoTenant = "gamma-parts"

var hoPaths = []string{"break-glass-1", "break-glass-2", "deploy-role", "iam-1", "iam-2", "platform-kill", "publish-releases", "support-metrics"}

type hoAccess struct{ paths map[string]bool }

func (a *hoAccess) Paths(context.Context, string) ([]access.Path, error) {
	var out []access.Path
	for _, p := range hoPaths {
		if a.paths[p] {
			g := ""
			for _, known := range handoff.Grants {
				if p == known {
					g = p // grant paths are named after their grant here
				}
			}
			out = append(out, access.Path{ID: p, Grant: g})
		}
	}
	return out, nil
}

func (a *hoAccess) ids() []string {
	ps, _ := a.Paths(context.Background(), "")
	var out []string
	for _, p := range ps {
		out = append(out, p.ID)
	}
	return out
}
func (a *hoAccess) Revoke(_ context.Context, _ string, p string) error {
	delete(a.paths, p)
	return nil
}

type hoTarget struct {
	fail    bool
	serving string
}

var errInjected = errors.New("injected failure")

func (t *hoTarget) step() error {
	if t.fail {
		t.fail = false
		return errInjected
	}
	return nil
}
func (t *hoTarget) Provision(context.Context, string, handoff.Plan) error { return t.step() }
func (t *hoTarget) Teardown(context.Context, string, handoff.Plan) error  { return nil }
func (t *hoTarget) Move(context.Context, string, handoff.Plan) error      { return t.step() }
func (t *hoTarget) Verify(context.Context, string, handoff.Plan) error    { return t.step() }
func (t *hoTarget) Cutover(context.Context, string, handoff.Plan) error {
	if err := t.step(); err != nil {
		return err
	}
	t.serving = "tenantCell"
	return nil
}

type hoCustody struct{ custody string }

func (c *hoCustody) SetCustody(_ context.Context, _, custody, _, _ string) error {
	c.custody = custody
	return nil
}

type hoAudit struct{ rows int }

func (a *hoAudit) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	a.rows++
	return audit.Row{Kind: e.Kind}, nil
}

type handoffWorld struct {
	c       *handoff.Controller
	access  *hoAccess
	target  *hoTarget
	keys    map[string]ports.Keys
	wraps   *keywrap.MemoryStore
	pending bool // Fail() was called: the next step's own work fails
}

func newHandoffWorld(root string) *handoffWorld {
	ctx := context.Background()
	t := manifest.Tenant{Tenant: hoTenant, Region: "eu"}
	acct := func(name string) (local.Account, string) {
		a := local.Account{Root: filepath.Join(root, "accounts"), Name: name}
		rs, err := cell.Stamp(ctx, a, a, t)
		if err != nil {
			panic(err)
		}
		for _, r := range rs {
			if r.Kind == cell.KindRootKey {
				return a, r.ID
			}
		}
		panic("no root key")
	}
	fa, fk := acct("factory-acct")
	ta, tk := acct("tenant-acct")
	w := &handoffWorld{access: &hoAccess{paths: map[string]bool{}}, target: &hoTarget{serving: "factoryCell"},
		keys: map[string]ports.Keys{"factory": fa, "tenant": ta}, wraps: &keywrap.MemoryStore{}}
	for _, p := range hoPaths {
		w.access.paths[p] = true
	}
	factory := keywrap.Custodian{Name: "factory", Keys: fa, KeyID: fk}
	if err := keywrap.Seed(ctx, hoTenant, w.wraps, factory, keywrap.PurposeAuditSigning); err != nil {
		panic(err)
	}
	w.c = &handoff.Controller{Store: &handoff.MemoryStore{}, Access: w.access, Target: w.target, Custody: &hoCustody{},
		Audit: &hoAudit{}, Wraps: w.wraps,
		Factory: func(string) keywrap.Custodian { return factory },
		Tenant: func(string, handoff.Plan) keywrap.Custodian {
			return keywrap.Custodian{Name: "tenant", Keys: ta, KeyID: tk}
		}}
	return w
}

func (w *handoffWorld) Plan(planner string, retain []string) (string, error) {
	var extra []string
	for _, g := range retain {
		if g != "publish-releases" {
			extra = append(extra, g)
		}
	}
	return w.c.Plan(context.Background(), hoTenant, planner, cell.TenantAccountPrefix+"gcp/gamma-prod-7731", "eu", extra)
}

func (w *handoffWorld) Approve(side, who, hash string) error {
	return w.c.Approve(context.Background(), hoTenant, side, who, hash)
}

func (w *handoffWorld) Step() error {
	w.target.fail, w.pending = w.pending, false
	_, err := w.c.Step(context.Background(), hoTenant, "fo_ann")
	w.target.fail = false
	return err
}

func (w *handoffWorld) Fail() { w.pending = true }
func (w *handoffWorld) Abort() error {
	return w.c.Abort(context.Background(), hoTenant, "fo_ann", "law test")
}
func (w *handoffWorld) Detach() error { return w.c.Detach(context.Background(), hoTenant, "to_cy") }

func (w *handoffWorld) State() HandoffState {
	ctx := context.Background()
	s, _ := w.c.State(ctx, hoTenant)
	out := HandoffState{Phase: string(s.Phase), Serving: s.Serving, Wraps: map[string]bool{}, Paths: append([]string(nil), hoPaths...),
		Plan: s.Hash, Approved: map[string]string{}, Approvers: map[string]string{}}
	if s.Plan != nil {
		out.Planner, out.Retained = s.Plan.Planner, append([]string(nil), s.Plan.Retain...)
	}
	for side, a := range s.Approvals {
		out.Approved[side], out.Approvers[side] = a.Hash, a.Who
	}
	inv, _ := keywrap.Inventory(ctx, hoTenant, w.wraps, w.keys)
	for _, e := range inv {
		for _, c := range e.Working {
			out.Wraps[c] = true // one purpose today; a custodian counts if it unwraps it
		}
	}
	out.Access = w.access.ids()
	if w.target.serving != s.Serving {
		out.Serving = "inconsistent: target " + w.target.serving + ", controller " + s.Serving
	}
	return out
}
