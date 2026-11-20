package laws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	actmem "example.com/second-brain/internal/adapters/actions/memory"
	auditmem "example.com/second-brain/internal/adapters/audit/memory"
	killmem "example.com/second-brain/internal/adapters/kill/memory"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/barrier"
	"example.com/second-brain/internal/core/kill"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/core/toolbus"
)

// LAW-22 beyond reads (SB-1203): through the real policies, retrieval's item check (the
// one drafts use), and the tool bus with barrier.Cell, nothing crosses a barrier: no ask
// uses a document held or owned across one, no draft replies to such an item, no tool call
// or proposal targets one, and nobody approves for a requester across one.

// BarrierActs is a cell's drafts, actions and approvals, beside its asks.
type BarrierActs interface {
	// Draft reports whether user may draft a reply to document doc.
	Draft(user, doc string) (bool, error)
	// Act reports whether user may call a tool (consequential or not) on document doc.
	Act(user, doc string, consequential bool) (bool, error)
	// Approve reports whether approver may approve an action requester proposed.
	Approve(approver, requester string) (bool, error)
}

// barrierCounts guards against a vacuous pass: some drafts, actions and approvals allowed.
var barrierCounts struct{ drafts, acts, approvals int }

func law22Acts(r *rand.Rand, sys SpaceSystem) error {
	acts, ok := sys.(BarrierActs)
	if !ok {
		return nil
	}
	sp := genSpaces(r)
	if len(sp.Barriers) == 0 { // the acts need a wall to test
		sp.Barriers = [][2]string{{"group:deal-team", "group:everyone"}}
		if r.Intn(2) == 0 {
			sp.Barriers = [][2]string{{"group:finance", "group:eng"}}
		}
	}
	if err := sys.Load(sp); err != nil {
		return err
	}
	f := factsOf(sp)
	users := make([]string, 0, len(sp.Users))
	for u := range sp.Users {
		users = append(users, u)
	}
	sortStrings(users)
	for _, u := range users {
		for _, d := range sp.Docs {
			ok, err := acts.Draft(u, d.ID)
			if err != nil {
				return err
			}
			if ok && f.barrierBetween(u, d) {
				return fmt.Errorf("%s (groups %v) drafted a reply to %s across a barrier %v (holders %v, owner %q)", u, sp.Users[u], d.ID, sp.Barriers, d.Spaces, d.Owner)
			}
			if ok {
				barrierCounts.drafts++
			}
			cons := r.Intn(2) == 0
			ok, err = acts.Act(u, d.ID, cons)
			if err != nil {
				return err
			}
			if ok && f.barrierBetween(u, d) {
				return fmt.Errorf("%s (groups %v) acted on %s across a barrier %v (holders %v, owner %q)", u, sp.Users[u], d.ID, sp.Barriers, d.Spaces, d.Owner)
			}
			if ok {
				barrierCounts.acts++
			}
		}
		for _, v := range users {
			if u == v {
				continue
			}
			ok, err := acts.Approve(u, v)
			if err != nil {
				return err
			}
			if ok && f.peopleCross(u, v) {
				return fmt.Errorf("%s (groups %v) approved for %s (groups %v) across a barrier %v", u, sp.Users[u], v, sp.Users[v], sp.Barriers)
			}
			if ok {
				barrierCounts.approvals++
			}
		}
	}
	return nil
}

func sortStrings(xs []string) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}

// barrierSpaces is a cell with barriers: asks through retrieval (askSpaces), and drafts,
// tool calls and approvals through the real code paths.
type barrierSpaces struct {
	*askSpaces
	noTarget  bool // mutation: the bus gets no barrier facts for targets
	noApprove bool // mutation: the bus gets no barrier facts for approvals
	noOwners  bool // mutation: ownership isn't checked
}

func newBarrierSpaces() *barrierSpaces { return &barrierSpaces{askSpaces: newAskSpaces()} }

// Load places every document at /d/ID, and gives each space a scope of exactly the
// documents it holds, so routing an item finds the holders the index has.
func (b *barrierSpaces) Load(sp SpaceSpec) error {
	if b.noOwners {
		docs := append([]SpaceDoc(nil), sp.Docs...)
		for i := range docs {
			docs[i].Owner = ""
		}
		sp.Docs = docs
	}
	if err := b.askSpaces.Load(sp); err != nil {
		return err
	}
	for i := range b.all {
		var pats []string
		for _, d := range sp.Docs {
			if contains(d.Spaces, b.all[i].Name) {
				pats = append(pats, "/d/"+d.ID)
			}
		}
		b.all[i].Scope = map[string][]string{"docs": pats}
	}
	return nil
}

func (b *barrierSpaces) doc(id string) (ports.Document, bool) {
	for _, d := range b.spec.Docs {
		if d.ID == id {
			return ports.Document{Ref: ports.SourceRef{Kind: "docs", ID: id}, Version: 1, Title: id, Text: "about " + id,
				Location: "/d/" + id, Label: d.Label, Owner: d.Owner}, true
		}
	}
	if id == "neutral" {
		return ports.Document{Ref: ports.SourceRef{Kind: "docs", ID: id}, Version: 1, Location: "/neutral", Label: "internal"}, true
	}
	return ports.Document{}, false
}

func (b *barrierSpaces) scope(user string, extra ...string) tenancy.Scope {
	s, _ := tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, user, append(dirGroups(b.spec.Users[user]), extra...), spaces.Company)
	return s
}

func (b *barrierSpaces) spacesNow(context.Context) ([]ports.Space, [][2]string, error) {
	return b.all, b.spec.Barriers, nil
}

func (b *barrierSpaces) Draft(user, id string) (bool, error) {
	auth := retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": viewers{b.askSpaces}},
		Policy: policy.Kernel{Engine: b.ev, Recorder: okRecorder{}}, Spaces: b.spacesNow, Owners: b.ownerGroups}
	d, _ := b.doc(id)
	item := ports.Chunk{ID: "draft-context", Tenant: spaceTenant, Kind: "docs", SourceID: id, Cite: "docs:" + id + "@1"}
	return auth.ItemAllowed(context.Background(), b.scope(user), ports.UserRef{ID: user}, item, d)
}

// lawSource is the target system: Fetch places items, Act always succeeds.
type lawSource struct {
	viewers
	b *barrierSpaces
}

func (s lawSource) Fetch(_ context.Context, ref ports.SourceRef) (ports.Document, error) {
	if d, ok := s.b.doc(ref.ID); ok {
		return d, nil
	}
	return ports.Document{}, errors.New("not found")
}

func (lawSource) Act(_ context.Context, a ports.Action, _ ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{TargetID: "ok"}, nil
}

const barrierApprovers = "grp_approvers"

func (b *barrierSpaces) bus() (*toolbus.Bus, error) {
	now := time.Unix(1_800_000_000, 0)
	clock := func() time.Time { return now }
	checker := &kill.Checker{Tenant: spaceTenant, Store: killmem.New(), Now: clock}
	if err := checker.Refresh(context.Background()); err != nil {
		return nil, err
	}
	log := &audit.Log{Tenant: spaceTenant, Release: func() string { return "rel_law" }, Store: auditmem.New(), Now: clock}
	src := lawSource{viewers{b.askSpaces}, b}
	n := 0
	bus := &toolbus.Bus{Tenant: spaceTenant, Tools: map[string]toolbus.Tool{
		"lookup": {Name: "lookup", Connector: "docs"},
		"update": {Name: "update", Connector: "docs", Consequential: true, Workflow: "doc-update", Approvers: barrierApprovers},
	}, Kill: checker, Policy: policy.Kernel{Engine: b.ev, Recorder: log}, Audit: log, Actions: actmem.New(),
		Connectors: map[string]ports.Connector{"docs": src}, Now: clock, NewID: func(p string) string { n++; return fmt.Sprintf("%s%d", p, n) },
		Receipts: actmem.NewReceipts()}
	cell := barrier.Cell{Spaces: b.spacesNow, Owners: b.ownerGroups, Locate: src.Fetch,
		GroupsOf: func(_ context.Context, u string) ([]string, error) { return dirGroups(b.spec.Users[u]), nil }}
	if b.noTarget || b.noApprove {
		bus.Barriers = partial{cell, b.noTarget, b.noApprove}
	} else {
		bus.Barriers = cell
	}
	return bus, nil
}

// partial is barrier.Cell with one of its facts missing (the mutations).
type partial struct {
	c                   barrier.Cell
	noTarget, noApprove bool
}

func (p partial) Target(ctx context.Context, s tenancy.Scope, t string) (map[string]any, error) {
	if p.noTarget {
		return nil, nil
	}
	return p.c.Target(ctx, s, t)
}

func (p partial) Approval(ctx context.Context, s tenancy.Scope, r string) (map[string]any, error) {
	if p.noApprove {
		return nil, nil
	}
	return p.c.Approval(ctx, s, r)
}

func (b *barrierSpaces) Act(user, id string, consequential bool) (bool, error) {
	bus, err := b.bus()
	if err != nil {
		return false, err
	}
	tool := "lookup"
	if consequential {
		tool = "update"
	}
	payload, _ := json.Marshal(map[string]string{"doc": id})
	_, err = bus.Invoke(context.Background(), b.scope(user), toolbus.Call{Agent: "ask", Tool: tool, Target: "docs:" + id, Payload: payload})
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, toolbus.ErrPerm):
		return false, nil
	default:
		return false, err
	}
}

func (b *barrierSpaces) Approve(approver, requester string) (bool, error) {
	bus, err := b.bus()
	if err != nil {
		return false, err
	}
	ctx := context.Background()
	payload, _ := json.Marshal(map[string]string{"doc": "neutral"})
	a, err := bus.Invoke(ctx, b.scope(requester), toolbus.Call{Agent: "ask", Tool: "update", Target: "docs:neutral", Payload: payload})
	if err != nil {
		return false, fmt.Errorf("proposing on a neutral target: %w", err)
	}
	_, err = bus.Approve(ctx, b.scope(approver, barrierApprovers), a.ID, a.PayloadSHA256, "")
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, toolbus.ErrNoApproval), errors.Is(err, toolbus.ErrPerm):
		return false, nil
	default:
		return false, err
	}
}
