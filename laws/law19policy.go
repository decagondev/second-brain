package laws

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/core/policy"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// LAW-19 at the policy layer (SB-1004): the shipped Cedar policies with schema v2 decide
// every chunk with its label and holding spaces and the asker's read context, through the
// real kernel, and the source's live ACL is asked last. The ask path in a running cell
// arrives with SB-1006; LAW-20 and LAW-22 stay on NewSpaceSystem until then.

// NewSpacePolicySystem builds the system under test for LAW-19.
var NewSpacePolicySystem = func() SpaceSystem {
	ev, err := LoadPolicies()
	if err != nil {
		panic(err)
	}
	return &spacePolicy{kernel: policy.Kernel{Engine: ev, Recorder: okRecorder{}}, sourceLast: true}
}

type spacePolicy struct {
	kernel     policy.Kernel
	sourceLast bool // false is the mutation: a space's allow skips the source's ACL
	spec       SpaceSpec
	all        []ports.Space
}

const spaceTenant = "acme-logistics"

func (p *spacePolicy) Load(sp SpaceSpec) error {
	p.spec, p.all = sp, nil
	for _, s := range sp.Spaces {
		p.all = append(p.all, ports.Space{ID: spaces.ID(s.Name), Tenant: spaceTenant, Name: s.Name, Members: s.Members, Label: s.Label})
	}
	return nil
}

func dirGroups(gs []string) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = strings.TrimPrefix(g, "group:")
	}
	return out
}

func (p *spacePolicy) Ask(user, space string) ([]string, bool, error) {
	groups := dirGroups(p.spec.Users[user])
	in := spaces.Company
	if space != spaces.Company {
		in = spaces.ID(space)
		member := false
		for _, s := range p.all {
			if s.ID == in && spaces.Member(s, groups) {
				member = true
			}
		}
		if !member {
			return nil, true, nil // ESPACE, before retrieval
		}
	}
	scope, err := tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, user, groups, in)
	if err != nil {
		return nil, false, err
	}
	rctx := spaces.ReadContext(p.all, p.spec.Barriers, groups, scope.Space())
	var used []string
	for _, d := range p.spec.Docs {
		holders := make([]string, len(d.Spaces))
		for i, s := range d.Spaces {
			holders[i] = spaces.ID(s)
		}
		res := policy.Entity{Type: "Chunk", ID: d.ID, Attrs: map[string]any{"cite": "docs:" + d.ID + "@1"}}
		for k, v := range spaces.ChunkAttrs(d.Label, holders) {
			res.Attrs[k] = v
		}
		if p.kernel.Authorise(context.Background(), scope, "read", res, rctx).Decision != policy.Allow {
			continue
		}
		if p.sourceLast && !contains(d.Viewers, user) {
			continue // the source's live ACL, last and the authority
		}
		used = append(used, d.ID)
	}
	return used, false, nil
}

// LAW-03 for space facts (SB-1004): in a cell with spaces, a read missing any fact the
// space rules need (the chunk's label or holders, the asker's spaces or barriers) is
// refused, never decided as if the fact were empty.
func law03SpaceFacts(r *rand.Rand, ev policy.Evaluator) error {
	k := policy.Kernel{Engine: ev, Recorder: okRecorder{}}
	scope, err := tenancy.NewIssuer(spaceTenant).IssueIn(spaceTenant, "usr_1", []string{"finance"}, spaces.ID("finance"))
	if err != nil {
		return err
	}
	// A read every rule allows when every fact is there: the asker's own space, internal.
	attrs := map[string]any{"cite": "docs:d@1", "label": "internal", "spaces": []string{spaces.ID("finance")}}
	rctx := map[string]any{"space": spaces.ID("finance"), "member_spaces": []string{spaces.ID("finance")}, "barred_spaces": []string{}}
	if d := k.Authorise(context.Background(), scope, "read", policy.Entity{Type: "Chunk", ID: "d", Attrs: attrs}, rctx).Decision; d != policy.Allow {
		return fmt.Errorf("a read with every fact present was %s", d)
	}
	drop := []string{"label", "spaces", "member_spaces", "barred_spaces"}[r.Intn(4)]
	a, c := map[string]any{}, map[string]any{}
	for k, v := range attrs {
		a[k] = v
	}
	for k, v := range rctx {
		c[k] = v
	}
	delete(a, drop)
	delete(c, drop)
	if d := k.Authorise(context.Background(), scope, "read", policy.Entity{Type: "Chunk", ID: "d", Attrs: a}, c).Decision; d == policy.Allow {
		return fmt.Errorf("a read without %s was allowed", drop)
	}
	return nil
}
