package laws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"

	memspaces "example.com/second-brain/internal/adapters/spaces/memory"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-24 The tree never widens (SB-3907; sb-spaces(7), The tree): a space's place in the
// tree never lets anyone see, count or be told about a document they couldn't see from
// the space that holds it. A random company whose spaces form a tree of departments and
// teams; every branch ask may use only what an ask in one of the branch's spaces the
// asker is a member of would use on its own, and a team's settings that flow down never
// change who its members are, its label or what it holds.

// TreeSystem is a cell with spaces in a tree.
type TreeSystem interface {
	SpaceSystem
	// LoadTree loads spec with each team's department, by name.
	LoadTree(spec SpaceSpec, parent map[string]string) error
	// AskBranch asks in space and its teams, as the middleware and retrieval do.
	AskBranch(user, space string) (used []string, refused bool, err error)
}

// NewTreeSystem builds the system under test: branch asks through spaces.Teams, the
// Issuer and retrieve.ContextUpTo (SB-3903).
var NewTreeSystem = func() TreeSystem { return &treeSpaces{askSpaces: newAskSpaces()} }

type treeSpaces struct {
	*askSpaces
	parent map[string]string
	store  *memspaces.Store
	// wide is the mutation: the branch is the union of every team's members, as though a
	// department's tree made it a member of all its teams.
	wide bool
}

func (t *treeSpaces) LoadTree(sp SpaceSpec, parent map[string]string) error {
	if err := t.Load(sp); err != nil {
		return err
	}
	t.parent, t.store = parent, memspaces.New()
	for _, s := range t.all {
		s.Parent = parent[s.Name]
		if err := t.store.PutSpace(context.Background(), s); err != nil {
			return err
		}
	}
	return nil
}

func (t *treeSpaces) AskBranch(user, space string) ([]string, bool, error) {
	ctx := context.Background()
	groups := dirGroups(t.spec.Users[user])
	id, err := spaces.Resolve(ctx, t.store, spaceTenant, groups, space)
	if errors.Is(err, spaces.ErrSpace) {
		return nil, true, nil // ESPACE, as the middleware answers
	}
	if err != nil {
		return nil, false, err
	}
	iss := tenancy.NewIssuer(spaceTenant)
	scope, err := iss.IssueIn(spaceTenant, user, groups, id)
	if err != nil {
		return nil, false, err
	}
	teams, err := spaces.Teams(ctx, t.store, spaceTenant, groups, id)
	if err != nil {
		return nil, false, err
	}
	if t.wide {
		teams = nil
		for n, p := range t.parent {
			if p == space {
				teams = append(teams, spaces.ID(n))
			}
		}
		sort.Strings(teams)
	}
	var branch []tenancy.Scope
	for _, tm := range teams {
		in, err := iss.IssueIn(spaceTenant, user, groups, tm)
		if err != nil {
			return nil, false, err
		}
		if t.wide { // a member everywhere in the branch
			in, _ = iss.IssueIn(spaceTenant, user, append(groups, t.groupsOf(tm)...), tm)
		}
		branch = append(branch, in)
	}
	auth := retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": viewers{t.askSpaces}},
		Policy: policy.Kernel{Engine: t.ev, Recorder: okRecorder{}},
		Spaces: func(context.Context) ([]ports.Space, [][2]string, error) { return t.all, t.spec.Barriers, nil },
		Owners: t.ownerGroups}
	res, err := retrieve.Context(tenancy.WithBranch(ctx, branch), scope, ports.UserRef{ID: user}, nil, t.idx, auth, ports.Query{Text: "brain", Limit: 100})
	if err != nil {
		return nil, false, err
	}
	var used []string
	for _, c := range res.Context {
		used = append(used, c.SourceID)
	}
	return used, false, nil
}

func (t *treeSpaces) groupsOf(id string) []string {
	for _, s := range t.all {
		if s.ID == id {
			return dirGroups(s.Members)
		}
	}
	return nil
}

// genTree makes a random company as genSpaces does, and puts some of its spaces under
// the first as its teams.
func genTree(r *rand.Rand) (SpaceSpec, map[string]string) {
	sp := genSpaces(r)
	parent := map[string]string{}
	for _, s := range sp.Spaces[1:] {
		if r.Intn(2) == 0 {
			parent[s.Name] = sp.Spaces[0].Name
		}
	}
	return sp, parent
}

func law24(r *rand.Rand, sys TreeSystem) error {
	sp, parent := genTree(r)
	if err := sys.LoadTree(sp, parent); err != nil {
		return err
	}
	f := factsOf(sp)
	for _, a := range asks(sp) {
		user, space := a[0], a[1]
		if space == "company" {
			continue
		}
		used, refused, err := sys.AskBranch(user, space)
		if err != nil {
			return err
		}
		if refused {
			continue
		}
		// What each space of the branch the asker is in would use on its own.
		allowed := map[string]bool{}
		for _, s := range append([]string{space}, childrenOf(parent, space)...) {
			if !f.memberOf(user, s) {
				continue
			}
			alone, _, err := sys.Ask(user, s)
			if err != nil {
				return err
			}
			for _, d := range alone {
				allowed[d] = true
			}
		}
		for _, d := range used {
			if !allowed[d] {
				return fmt.Errorf("%s asking across %s's branch (teams %v) used %s, which no space of the branch they are in would use on its own", user, space, childrenOf(parent, space), d)
			}
		}
	}
	return law24Settings(r)
}

func childrenOf(parent map[string]string, space string) []string {
	var out []string
	for n, p := range parent {
		if p == space {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// law24Settings: what flows down to a team (SB-3902) never changes its members, label,
// scope or retention, which Check settled, stricter than or equal to its department's.
func law24Settings(r *rand.Rand) error {
	labels := []string{"internal", "confidential", "restricted"}
	dept, team := labels[r.Intn(2)], labels[r.Intn(3)]
	src := fmt.Sprintf(`tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: folder }
agents: [ask]
spaces:
  finance:
    members: [group:finance]
    stewards: [group:finance-leads]
    label: %s
    glossary: { PO: purchase order }
    start_here: ["docs:/Finance/Start/"]
    scope: { docs: ["/Finance/Dept/**"] }
  ap:
    parent: finance
    members: [group:ap]
    stewards: [group:ap-leads]
    label: %s
    scope: { docs: ["/Finance/AP/**"] }
`, dept, team)
	m, ds := manifest.Check([]byte(src))
	rank := map[string]int{"internal": 1, "confidential": 2, "restricted": 3}
	if rank[team] < rank[dept] {
		if len(ds) == 0 {
			return fmt.Errorf("a %s team under a %s department was accepted", team, dept)
		}
		return nil
	}
	if len(ds) > 0 {
		return fmt.Errorf("a %s team under a %s department was refused: %v", team, dept, ds)
	}
	own, eff := m.Spaces["ap"], m.Effective("ap")
	if fmt.Sprint(own.Members, own.Label, own.Scope, own.RetentionDays) != fmt.Sprint(eff.Members, eff.Label, eff.Scope, eff.RetentionDays) {
		return fmt.Errorf("settings that flow down changed the team: %+v became %+v", own, eff)
	}
	return nil
}
