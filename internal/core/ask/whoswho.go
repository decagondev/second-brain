package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/spaces"
	"example.com/second-brain/internal/core/tenancy"
)

// MaxOwnersListed is how many of a space's most-cited documents name their owners on its
// people list (SB-3906).
const MaxOwnersListed = 10

// SpacePeople is who's who in a space (SB-3906): its stewards and the owners (or caretakers)
// of its most-cited documents this month that the asker can open, each an active user now,
// and how many members it has, never who.
type SpacePeople struct {
	Space    string          `json:"space"`
	Parent   string          `json:"parent,omitempty"`
	Stewards []answer.Person `json:"stewards"`
	Owners   []PersonRef     `json:"owners"`
	Members  int             `json:"members"`
}

// member reports whether s is a member of space name now.
func (x Stewards) member(ctx context.Context, s tenancy.Scope, name string) error {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant || x.Agent.Auth.Spaces == nil {
		return ErrNotMember
	}
	all, _, err := x.Agent.Auth.Spaces(ctx)
	if err != nil {
		return fmt.Errorf("%w: spaces: %v", ErrUnavailable, err)
	}
	for _, sp := range all {
		if sp.Name == name && sp.Tenant == x.Agent.Tenant && spaces.Member(sp, s.Groups()) {
			return nil
		}
	}
	return ErrNotMember
}

// People is the space's who's who for one of its members (SB-3906). Owners are counted
// from the space's answers this month and named only for documents the asker can open
// now, through every check a search makes; a person who isn't active is left out (LAW-12).
func (x Stewards) People(ctx context.Context, s tenancy.Scope, name string, src ReviewSource) (SpacePeople, error) {
	if err := x.member(ctx, s, name); err != nil {
		return SpacePeople{}, err
	}
	out := SpacePeople{Space: name, Stewards: []answer.Person{}, Owners: []PersonRef{}}
	var err error
	if x.StewardPeople != nil {
		if out.Stewards, err = x.StewardPeople(ctx, SpaceKey(name)); err != nil {
			return out, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
	}
	if x.Members != nil {
		ms, err := x.Members(ctx, SpaceKey(name))
		if err != nil {
			return out, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		out.Members = len(ms)
	}
	if x.Parent != nil {
		if p := x.Parent(name); p != "" && x.member(ctx, s, p) == nil {
			out.Parent = p // named only to a member of the department too
		}
	}
	now := x.Now().UTC()
	traces, err := src.Between(ctx, x.Agent.Tenant, MonthStart(now), now)
	if err != nil {
		return out, fmt.Errorf("%w: traces: %v", ErrUnavailable, err)
	}
	count := map[string]int{}
	for _, t := range traces {
		if t.Space != SpaceKey(name) {
			continue
		}
		seen := map[string]bool{}
		for _, c := range t.Response.Answer.Claims {
			for _, ci := range c.Cites {
				if k := docOf(ci.Cite); k != "" && !seen[k] {
					seen[k] = true
					count[k]++
				}
			}
		}
	}
	top := make([]string, 0, len(count))
	for k := range count {
		top = append(top, k)
	}
	sort.Slice(top, func(i, j int) bool {
		return count[top[i]] > count[top[j]] || count[top[i]] == count[top[j]] && top[i] < top[j]
	})
	if len(top) > MaxOwnersListed {
		top = top[:MaxOwnersListed]
	}
	if len(top) == 0 {
		return out, nil
	}
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return out, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var q ports.DocQuery
	for _, k := range top {
		kind, id, _ := strings.Cut(k, ":")
		q.Refs = append(q.Refs, ports.SourceRef{Kind: kind, ID: id})
	}
	chunks, err := lk.Docs(ctx, x.Agent.Tenant, q)
	if err != nil {
		return out, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return out, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, chunks, len(chunks))
	if err != nil {
		return out, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	own := x.Agent.owners()
	by := map[string]*PersonRef{}
	done := map[string]bool{}
	for _, c := range res.Context {
		k := c.Kind + ":" + c.SourceID
		if done[k] {
			continue
		}
		done[k] = true
		p := own.of(ctx, c)
		if p == nil {
			continue
		}
		r, ok := by[p.Email]
		if !ok {
			r = &PersonRef{Email: p.Email, Title: p.Title, Caretaker: p.Caretaker}
			by[p.Email] = r
		}
		r.Cites += count[k]
	}
	for _, r := range by {
		out.Owners = append(out.Owners, *r)
	}
	sort.Slice(out.Owners, func(i, j int) bool {
		a, b := out.Owners[i], out.Owners[j]
		return a.Cites > b.Cites || a.Cites == b.Cites && a.Email < b.Email
	})
	return out, nil
}

// docOf is a cite's document, KIND:ID without its version.
func docOf(cite string) string {
	k, _, _ := strings.Cut(cite, "@")
	if !strings.Contains(k, ":") {
		return ""
	}
	return k
}
