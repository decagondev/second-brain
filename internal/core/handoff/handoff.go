// Package handoff is the hand-off controller (sb-factory(7), The hand-off, exactly; SB-705).
// Its phases and guards are those of proofs/tla/Handoff.tla, and a test reads the model to
// check them. Every step writes a kind: custody audit row before it acts (LAW-04), stores
// its phase only after it succeeds, and leaves the phase unchanged when it fails.
package handoff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"example.com/second-brain/internal/core/access"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/canonjson"
	"example.com/second-brain/internal/core/cell"
	"example.com/second-brain/internal/core/keywrap"
)

// Phase is one of the model's phases.
type Phase string

// The phases of Handoff.tla.
const (
	Managed     Phase = "Managed"
	Planned     Phase = "Planned"
	Provisioned Phase = "Provisioned"
	KeyDual     Phase = "KeyDual"
	Moved       Phase = "Moved"
	Verified    Phase = "Verified"
	CutOver     Phase = "CutOver"
	TenantOwned Phase = "TenantOwned"
	Detached    Phase = "Detached"
)

// Transitions are the model's actions as (from, to) pairs, by action name.
var Transitions = map[string][2]Phase{
	"MakePlan":           {Managed, Planned}, // and Planned -> Planned (a new plan)
	"Provision":          {Planned, Provisioned},
	"WrapTenant":         {Provisioned, KeyDual},
	"Move":               {KeyDual, Moved},
	"Verify":             {Moved, Verified},
	"Cutover":            {Verified, CutOver},
	"DestroyFactoryWrap": {CutOver, TenantOwned},
	"Detach":             {TenantOwned, Detached},
}

// InFlight are the phases abort is allowed in.
var InFlight = []Phase{Planned, Provisioned, KeyDual, Moved, Verified}

// Grants a plan may retain; publish-releases is always kept.
var Grants = []string{"publish-releases", "platform-kill", "support-metrics"}

// Plan is what both sides approve, by its hash.
type Plan struct {
	Tenant  string   `json:"tenant"`
	Target  string   `json:"target"`
	Region  string   `json:"region"`
	Revoke  []string `json:"revoke"` // path ids to revoke at cutover
	Keep    []string `json:"keep"`   // path ids that serve a retained grant
	Retain  []string `json:"retain"` // the retained grants
	Planner string   `json:"planner"`
	Serial  int      `json:"serial"` // a new plan for the same contents still has a new hash
}

// Hash is "ph_" and the SHA-256 of the plan's canonical JSON (RFC 8785).
func (p Plan) Hash() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	c, err := canonjson.Encode(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c)
	return "ph_" + hex.EncodeToString(sum[:]), nil
}

// Approval is one side's approval of one hash.
type Approval struct {
	Who  string `json:"who"`
	Hash string `json:"hash"`
}

// State is one cell's hand-off.
type State struct {
	Phase     Phase               `json:"phase"`
	Plan      *Plan               `json:"plan,omitempty"`
	Hash      string              `json:"hash,omitempty"`
	Approvals map[string]Approval `json:"approvals,omitempty"` // "factory", "tenant"
	Serving   string              `json:"serving"`             // "factoryCell" or "tenantCell"
}

// Store keeps hand-off states.
type Store interface {
	Get(ctx context.Context, tenant string) (State, bool, error)
	Put(ctx context.Context, tenant string, s State) error
}

// Access enumerates the factory's access paths to a cell, each with the grant it serves,
// and revokes one by id (SB-706). A path is kept only if its grant is retained (BUG-0012).
type Access interface {
	Paths(ctx context.Context, tenant string) ([]access.Path, error)
	Revoke(ctx context.Context, tenant, id string) error
}

// Target does the infrastructure work of each step in the tenant's account.
type Target interface {
	Provision(ctx context.Context, tenant string, p Plan) error // stamp the cell in the target account
	Teardown(ctx context.Context, tenant string, p Plan) error  // undo Provision (abort)
	Move(ctx context.Context, tenant string, p Plan) error      // restore state in region, rebuild the index
	Verify(ctx context.Context, tenant string, p Plan) error    // laws, audit chain, attestation
	Cutover(ctx context.Context, tenant string, p Plan) error   // traffic to the tenant's cell
}

// Custody records the new custody when a hand-off completes or a cell detaches.
type Custody interface {
	SetCustody(ctx context.Context, tenant, custody, placement, keyCustodian string) error
}

// Auditor writes audit rows (audit.Log).
type Auditor interface {
	Write(ctx context.Context, e audit.Event) (audit.Row, error)
}

// Errors.
var (
	ErrPhase    = errors.New("handoff: EINVAL not allowed in this phase")
	ErrApproval = errors.New("handoff: ENOAPPROVAL")
	ErrPlan     = errors.New("handoff: EINVAL bad plan")
	ErrAccess   = errors.New("handoff: ELAW unlisted factory access after cutover")
)

// Controller runs hand-offs for the cells of one factory.
type Controller struct {
	Store   Store
	Access  Access
	Target  Target
	Custody Custody
	Audit   Auditor
	Wraps   keywrap.Store
	// Factory and Tenant hold each cell's root keys; Tenant is per cell, from the plan.
	Factory func(tenant string) keywrap.Custodian
	Tenant  func(tenant string, p Plan) keywrap.Custodian

	mu sync.Mutex
}

func (c *Controller) state(ctx context.Context, tenant string) (State, error) {
	s, found, err := c.Store.Get(ctx, tenant)
	if err != nil {
		return State{}, err
	}
	if !found {
		return State{Phase: Managed, Serving: "factoryCell"}, nil
	}
	return s, nil
}

func (c *Controller) audit(ctx context.Context, actor, tenant, state, reason string) error {
	_, err := c.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "custody", Subject: "handoff:" + tenant, State: state, Reason: reason})
	return err
}

func in(ps []Phase, p Phase) bool {
	for _, x := range ps {
		if x == p {
			return true
		}
	}
	return false
}

// Plan makes (or replaces) the plan for a cell and returns its hash. A new plan voids every
// approval. Revoke is every current factory path not retained.
func (c *Controller) Plan(ctx context.Context, tenant, planner, target, region string, retain []string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, err := c.state(ctx, tenant)
	if err != nil {
		return "", err
	}
	if s.Phase != Managed && s.Phase != Planned {
		return "", fmt.Errorf("%w: a hand-off is in %s", ErrPhase, s.Phase)
	}
	keep := map[string]bool{"publish-releases": true}
	for _, g := range retain {
		ok := false
		for _, known := range Grants {
			ok = ok || g == known
		}
		if !ok {
			return "", fmt.Errorf("%w: unknown grant %q", ErrPlan, g)
		}
		keep[g] = true
	}
	paths, err := c.Access.Paths(ctx, tenant)
	if err != nil {
		return "", err
	}
	if _, _, ok := cell.ParseTenantAccount(target); !ok {
		return "", fmt.Errorf("%w: target must be %sPROVIDER/ID", ErrPlan, cell.TenantAccountPrefix)
	}
	p := Plan{Tenant: tenant, Target: target, Region: region, Planner: planner}
	for g := range keep {
		p.Retain = append(p.Retain, g)
	}
	for _, x := range paths {
		if x.Grant != "" && keep[x.Grant] {
			p.Keep = append(p.Keep, x.ID)
		} else {
			p.Revoke = append(p.Revoke, x.ID)
		}
	}
	sort.Strings(p.Retain)
	sort.Strings(p.Keep)
	sort.Strings(p.Revoke)
	if s.Plan != nil {
		p.Serial = s.Plan.Serial + 1
	}
	h, err := p.Hash()
	if err != nil {
		return "", err
	}
	if err := c.audit(ctx, planner, tenant, "HandoffPlanned", "plan "+h); err != nil {
		return "", err
	}
	s.Phase, s.Plan, s.Hash, s.Approvals = Planned, &p, h, map[string]Approval{}
	if s.Serving == "" {
		s.Serving = "factoryCell"
	}
	return h, c.Store.Put(ctx, tenant, s)
}

// Approve records one side's approval of the current hash. The factory side can't be the
// planner. The caller has already authorised who (sb-policy(5), Factory roles).
func (c *Controller) Approve(ctx context.Context, tenant, side, who, hash string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, err := c.state(ctx, tenant)
	if err != nil {
		return err
	}
	switch {
	case s.Phase != Planned:
		return fmt.Errorf("%w: approvals are taken only while Planned", ErrPhase)
	case hash != s.Hash:
		return fmt.Errorf("%w: %s is not the current plan", ErrApproval, hash)
	case side != "factory" && side != "tenant":
		return fmt.Errorf("%w: side must be factory or tenant", ErrApproval)
	case side == "factory" && who == s.Plan.Planner:
		return fmt.Errorf("%w: the planner can't approve their own plan", ErrApproval)
	}
	if err := c.audit(ctx, who, tenant, "Approved", side+" approves "+hash); err != nil {
		return err
	}
	s.Approvals[side] = Approval{Who: who, Hash: hash}
	return c.Store.Put(ctx, tenant, s)
}

func approved(s State) bool {
	f, t := s.Approvals["factory"], s.Approvals["tenant"]
	return s.Plan != nil && f.Hash == s.Hash && t.Hash == s.Hash && f.Who != s.Plan.Planner && f.Who != "" && t.Who != ""
}

// Step runs the next step of the hand-off. It returns the phase reached.
func (c *Controller) Step(ctx context.Context, tenant, actor string) (Phase, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, err := c.state(ctx, tenant)
	if err != nil {
		return "", err
	}
	if s.Plan == nil || !in([]Phase{Planned, Provisioned, KeyDual, Moved, Verified, CutOver}, s.Phase) {
		return s.Phase, fmt.Errorf("%w: nothing to run in %s", ErrPhase, s.Phase)
	}
	if !approved(s) {
		return s.Phase, fmt.Errorf("%w: both sides must approve %s", ErrApproval, s.Hash)
	}
	p := *s.Plan
	var next Phase
	var run func() error
	switch s.Phase {
	case Planned:
		next, run = Provisioned, func() error { return c.Target.Provision(ctx, tenant, p) }
	case Provisioned:
		next, run = KeyDual, func() error {
			_, err := keywrap.Rewrap(ctx, tenant, c.Wraps, c.Factory(tenant), c.Tenant(tenant, p))
			return err
		}
	case KeyDual:
		next, run = Moved, func() error { return c.Target.Move(ctx, tenant, p) }
	case Moved:
		next, run = Verified, func() error { return c.Target.Verify(ctx, tenant, p) }
	case Verified:
		next, run = CutOver, func() error { return c.cutover(ctx, tenant, p) }
	case CutOver:
		next, run = TenantOwned, func() error {
			if err := keywrap.Destroy(ctx, tenant, c.Wraps, c.Factory(tenant), c.Tenant(tenant, p)); err != nil {
				return err
			}
			return c.Custody.SetCustody(ctx, tenant, cell.CustodyTenant, p.Target, cell.KeyCustodianTenant)
		}
	case Managed, TenantOwned, Detached:
		return s.Phase, fmt.Errorf("%w: nothing to run in %s", ErrPhase, s.Phase) // refused above too
	}
	if err := c.audit(ctx, actor, tenant, string(next), "step "+string(s.Phase)+" -> "+string(next)+" for "+s.Hash); err != nil {
		return s.Phase, err
	}
	if err := run(); err != nil {
		return s.Phase, fmt.Errorf("handoff: %s -> %s: %w", s.Phase, next, err)
	}
	s.Phase = next
	if next == CutOver {
		s.Serving = "tenantCell"
	}
	return next, c.Store.Put(ctx, tenant, s)
}

// cutover moves traffic, then revokes every path the plan doesn't retain and checks that
// only retained paths remain (LAW-17). A failed revoke is retried by running the step again.
func (c *Controller) cutover(ctx context.Context, tenant string, p Plan) error {
	if err := c.Target.Cutover(ctx, tenant, p); err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, g := range p.Retain {
		keep[g] = true
	}
	paths, err := c.Access.Paths(ctx, tenant)
	if err != nil {
		return err
	}
	for _, x := range paths {
		if x.Grant == "" || !keep[x.Grant] {
			if err := c.Access.Revoke(ctx, tenant, x.ID); err != nil {
				return err
			}
		}
	}
	left, err := c.Access.Paths(ctx, tenant)
	if err != nil {
		return err
	}
	var extra []string
	for _, x := range left {
		if x.Grant == "" || !keep[x.Grant] {
			extra = append(extra, x.ID)
		}
	}
	if len(extra) > 0 {
		return fmt.Errorf("%w: %s", ErrAccess, strings.Join(extra, ", "))
	}
	return nil
}

// Execute runs steps until the hand-off is TenantOwned or a step fails.
func (c *Controller) Execute(ctx context.Context, tenant, actor, hash string) (Phase, error) {
	s, err := c.State(ctx, tenant)
	if err != nil {
		return "", err
	}
	if hash != s.Hash {
		return s.Phase, fmt.Errorf("%w: %s is not the current plan", ErrApproval, hash)
	}
	for {
		ph, err := c.Step(ctx, tenant, actor)
		if err != nil || ph == TenantOwned {
			return ph, err
		}
	}
}

// Abort returns an in-flight hand-off to Managed: the tenant's wraps and provisioned
// resources go, the factory's stay, and the factory cell never stopped serving.
func (c *Controller) Abort(ctx context.Context, tenant, actor, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, err := c.state(ctx, tenant)
	if err != nil {
		return err
	}
	if !in(InFlight, s.Phase) {
		return fmt.Errorf("%w: abort is allowed only before cutover", ErrPhase)
	}
	if err := c.audit(ctx, actor, tenant, string(Managed), "abort from "+string(s.Phase)+": "+reason); err != nil {
		return err
	}
	p := *s.Plan
	if err := keywrap.Abort(ctx, tenant, c.Wraps, c.Tenant(tenant, p), c.Factory(tenant)); err != nil {
		return err
	}
	if s.Phase != Planned {
		if err := c.Target.Teardown(ctx, tenant, p); err != nil {
			return err
		}
	}
	return c.Store.Put(ctx, tenant, State{Phase: Managed, Serving: "factoryCell"})
}

// Detach takes a tenant-owned cell off the train: every factory path goes.
func (c *Controller) Detach(ctx context.Context, tenant, actor string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, err := c.state(ctx, tenant)
	if err != nil {
		return err
	}
	if s.Phase != TenantOwned {
		return fmt.Errorf("%w: only a tenant-owned cell detaches", ErrPhase)
	}
	if err := c.audit(ctx, actor, tenant, string(Detached), "detach"); err != nil {
		return err
	}
	if err := c.Custody.SetCustody(ctx, tenant, cell.CustodyDetached, s.Plan.Target, cell.KeyCustodianTenant); err != nil {
		return err
	}
	paths, err := c.Access.Paths(ctx, tenant)
	if err != nil {
		return err
	}
	for _, x := range paths {
		if err := c.Access.Revoke(ctx, tenant, x.ID); err != nil {
			return err
		}
	}
	s.Phase = Detached
	return c.Store.Put(ctx, tenant, s)
}

// State returns a cell's hand-off state (Managed when there is none).
func (c *Controller) State(ctx context.Context, tenant string) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state(ctx, tenant)
}

// MemoryStore is an in-process Store.
type MemoryStore struct {
	mu sync.Mutex
	m  map[string]State
}

// Get implements Store.
func (m *MemoryStore) Get(_ context.Context, tenant string) (State, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.m[tenant]
	if ok && s.Approvals != nil {
		a := map[string]Approval{}
		for k, v := range s.Approvals {
			a[k] = v
		}
		s.Approvals = a
	}
	return s, ok, nil
}

// Put implements Store.
func (m *MemoryStore) Put(_ context.Context, tenant string, s State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.m == nil {
		m.m = map[string]State{}
	}
	m.m[tenant] = s
	return nil
}
