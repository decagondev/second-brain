package laws

import (
	"context"
	"fmt"

	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// Asks in a cell with spaces (SB-1006): every document of the company is indexed with its
// label and holders, as ingest routes them (SB-1005), and every ask runs retrieval's
// space, label and barrier checks, then the live ACL batch, through the real kernel and
// policies. LAW-19 and LAW-20 are judged on what reaches the context.

type askSpaces struct {
	spec       SpaceSpec
	all        []ports.Space
	idx        *memindex.Index
	ev         policy.Evaluator
	skipSpaces bool // the mutation: retrieval without the space step
}

func newAskSpaces() *askSpaces {
	ev, err := LoadPolicies()
	if err != nil {
		panic(err)
	}
	return &askSpaces{ev: ev}
}

func (a *askSpaces) Load(sp SpaceSpec) error {
	a.spec, a.all = sp, nil
	for _, s := range sp.Spaces {
		a.all = append(a.all, ports.Space{ID: spaces.ID(s.Name), Tenant: spaceTenant, Name: s.Name, Members: s.Members, Label: s.Label})
	}
	a.idx = memindex.New()
	ctx := context.Background()
	for _, d := range sp.Docs {
		holders := make([]string, len(d.Spaces))
		for i, s := range d.Spaces {
			holders[i] = spaces.ID(s)
		}
		ref := ports.SourceRef{Kind: "docs", ID: d.ID}
		c := ports.Chunk{ID: "chk_" + d.ID, Tenant: spaceTenant, Cite: "docs:" + d.ID + "@1", Kind: "docs", SourceID: d.ID,
			Version: 1, Text: "the brain knows " + d.ID, ACLPrefilter: []string{"anyone"}, Label: d.Label, Spaces: holders, Owner: d.Owner}
		if err := a.idx.Replace(ctx, spaceTenant, ref, 1, []ports.Chunk{c}); err != nil {
			return err
		}
	}
	return nil
}

// ownerGroups is the directory: a document owner's groups.
func (a *askSpaces) ownerGroups(_ context.Context, owner string) ([]string, error) {
	gs, ok := a.spec.Users[owner]
	if !ok {
		return []string{}, nil
	}
	return dirGroups(gs), nil
}

// viewers is the source: its live ACL is the spec's viewers.
type viewers struct{ a *askSpaces }

func (viewers) APIVersion() int { return ports.ConnectorAPIVersion }
func (viewers) Fetch(context.Context, ports.SourceRef) (ports.Document, error) {
	return ports.Document{}, fmt.Errorf("not used")
}
func (v viewers) ACL(_ context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	for i, r := range refs {
		out[i] = ports.VerdictDeny
		for _, d := range v.a.spec.Docs {
			if d.ID == r.ID && contains(d.Viewers, u.ID) {
				out[i] = ports.VerdictAllow
			}
		}
	}
	return out, nil
}
func (viewers) Changes(context.Context, ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	return nil, "", nil
}
func (viewers) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, fmt.Errorf("ENOTSUP")
}

func (a *askSpaces) Ask(user, space string) ([]string, bool, error) {
	ctx := context.Background()
	groups := dirGroups(a.spec.Users[user])
	in := spaces.Company
	if space != spaces.Company {
		in = spaces.ID(space)
		member := false
		for _, s := range a.all {
			if s.ID == in && spaces.Member(s, groups) {
				member = true
			}
		}
		if !member {
			return nil, true, nil // ESPACE (spaces.Resolve in the middleware)
		}
	}
	scope, err := tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, user, groups, in)
	if err != nil {
		return nil, false, err
	}
	auth := retrieve.Authoriser{Connectors: map[string]ports.Connector{"docs": viewers{a}},
		Policy: policy.Kernel{Engine: a.ev, Recorder: okRecorder{}}}
	if !a.skipSpaces {
		auth.Spaces = func(context.Context) ([]ports.Space, [][2]string, error) { return a.all, a.spec.Barriers, nil }
		auth.Owners = a.ownerGroups
	}
	res, err := retrieve.Context(ctx, scope, ports.UserRef{ID: user}, nil, a.idx, auth, ports.Query{Text: "brain", Limit: 100})
	if err != nil {
		return nil, false, err
	}
	var used []string
	for _, c := range res.Context {
		used = append(used, c.SourceID)
	}
	return used, false, nil
}
