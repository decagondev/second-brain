package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Policies to read (sb-spaces(7), Policies to read; SB-4103): a steward marks a document as
// a policy for their space and the teams under it. Members who can open it are told once,
// see it until they press I've read it, which is recorded per person and version and
// audited, and a new version asks again. The stewards see who hasn't read it by name;
// members who can't open it are counted.

// SpacePolicy is one document a space asks its branch to read.
type SpacePolicy struct {
	Tenant string    `json:"tenant"`
	Space  string    `json:"space"` // the space's name
	Doc    string    `json:"doc"`   // KIND:ID
	By     string    `json:"by"`
	At     time.Time `json:"at"`
}

// PolicyRead is one person's I've read it, for one version.
type PolicyRead struct {
	Tenant  string    `json:"tenant"`
	Space   string    `json:"space"`
	Doc     string    `json:"doc"`
	User    string    `json:"user"`
	Email   string    `json:"email"`
	Version int       `json:"version"`
	At      time.Time `json:"at"`
}

// PolicyStore keeps policies and reads.
type PolicyStore interface {
	PutPolicy(ctx context.Context, p SpacePolicy) error
	Policies(ctx context.Context, tenant, space string) ([]SpacePolicy, error)
	PutPolicyRead(ctx context.Context, r PolicyRead) error
	PolicyReads(ctx context.Context, tenant, space, doc string) ([]PolicyRead, error)
}

// MaxPolicies is the most policies a space keeps.
const MaxPolicies = 50

// PolicyItem is one policy as a member sees it.
type PolicyItem struct {
	Space   string `json:"space"`
	Doc     string `json:"doc"`
	Cite    string `json:"cite"`
	Title   string `json:"title"`
	URL     string `json:"url,omitempty"`
	Version int    `json:"version"`
	Read    bool   `json:"read"`
}

// PolicyStatus is one policy as its stewards see it.
type PolicyStatus struct {
	PolicyItem
	// NotRead are the branch's members who can open it and haven't read this version, by
	// email; ReadCount those who have; CantOpen the members who can't open it, counted.
	NotRead   []string `json:"not_read"`
	ReadCount int      `json:"read"`
	CantOpen  int      `json:"cant_open"`
}

// current is a document's newest indexed first chunk, if the index has it.
func (x Stewards) current(ctx context.Context, doc string) (ports.Chunk, bool, error) {
	lk, ok := x.Agent.Index.(ports.DocLookup)
	if !ok {
		return ports.Chunk{}, false, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	kind, id, _ := strings.Cut(doc, ":")
	cs, err := lk.Docs(ctx, x.Agent.Tenant, ports.DocQuery{Refs: []ports.SourceRef{{Kind: kind, ID: id}}})
	if err != nil {
		return ports.Chunk{}, false, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	for _, c := range firsts(latest(cs)) {
		return c, true, nil
	}
	return ports.Chunk{}, false, nil
}

// opens is the chunk as s may see it now, through every check a search makes.
func (x Stewards) opens(ctx context.Context, s tenancy.Scope, c ports.Chunk) (bool, error) {
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return false, err
	}
	res, err := retrieve.Allowed(ctx, s, who, x.Agent.Auth, []ports.Chunk{c}, 1)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return len(res.Context) == 1, nil
}

// MarkPolicy is a steward of space name marking doc, which they can open and a space of the
// branch holds, as a policy for the branch; audited first.
func (x Stewards) MarkPolicy(ctx context.Context, s tenancy.Scope, name, doc string) (PolicyItem, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return PolicyItem{}, err
	}
	if x.Policies == nil {
		return PolicyItem{}, fmt.Errorf("%w: this cell keeps no policies", ErrUnavailable)
	}
	if k, id, ok := strings.Cut(doc, ":"); !ok || k == "" || id == "" || strings.Contains(doc, "@") {
		return PolicyItem{}, fmt.Errorf("%w: doc is KIND:ID", ErrInvalid)
	}
	c, found, err := x.current(ctx, doc)
	if err != nil {
		return PolicyItem{}, err
	}
	branch := []string{SpaceKey(name)}
	if x.Children != nil {
		for _, ch := range x.Children(name) {
			branch = append(branch, SpaceKey(ch))
		}
	}
	held := false
	for _, sp := range c.Spaces {
		held = held || contains(branch, sp)
	}
	if !found || !held {
		return PolicyItem{}, fmt.Errorf("%w: no such document in this space or its teams", ErrInvalid)
	}
	if ok, err := x.opens(ctx, s, c); err != nil || !ok {
		if err != nil {
			return PolicyItem{}, err
		}
		return PolicyItem{}, fmt.Errorf("%w: no such document in this space or its teams", ErrInvalid)
	}
	have, err := x.Policies.Policies(ctx, x.Agent.Tenant, name)
	if err != nil {
		return PolicyItem{}, fmt.Errorf("%w: policies: %v", ErrUnavailable, err)
	}
	if len(have) >= MaxPolicies {
		return PolicyItem{}, fmt.Errorf("%w: at most %d policies a space", ErrInvalid, MaxPolicies)
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: "policy:" + doc, State: "policy_marked",
		Reason: "marked as a policy to read", Space: auditSpace(SpaceKey(name))}); err != nil {
		return PolicyItem{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	if err := x.Policies.PutPolicy(ctx, SpacePolicy{Tenant: x.Agent.Tenant, Space: name, Doc: doc, By: s.User(), At: x.Now().UTC()}); err != nil {
		return PolicyItem{}, fmt.Errorf("%w: policies: %v", ErrUnavailable, err)
	}
	r := x.Agent.result(ctx, nil, c)
	return PolicyItem{Space: name, Doc: doc, Cite: c.Cite, Title: r.Title, URL: r.URL, Version: c.Version}, nil
}

// branchPolicies are the policies that apply in space name: its own and its department's.
func (x Stewards) branchPolicies(ctx context.Context, name string) ([]SpacePolicy, error) {
	if x.Policies == nil {
		return nil, nil
	}
	up := []string{name}
	if x.Parent != nil {
		if p := x.Parent(name); p != "" {
			up = append(up, p)
		}
	}
	var out []SpacePolicy
	for _, sp := range up {
		ps, err := x.Policies.Policies(ctx, x.Agent.Tenant, sp)
		if err != nil {
			return nil, fmt.Errorf("%w: policies: %v", ErrUnavailable, err)
		}
		out = append(out, ps...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Space+out[i].Doc < out[j].Space+out[j].Doc })
	return out, nil
}

// readBy reports whether email has read version of the policy.
func readBy(rs []PolicyRead, email string, version int) bool {
	for _, r := range rs {
		if strings.EqualFold(r.Email, email) && r.Version == version {
			return true
		}
	}
	return false
}

// PoliciesFor are the policies that apply in space name that its member s can open now,
// each with whether they've read its current version. The rest aren't listed.
func (x Stewards) PoliciesFor(ctx context.Context, s tenancy.Scope, name string) ([]PolicyItem, error) {
	if err := x.member(ctx, s, name); err != nil {
		return nil, err
	}
	ps, err := x.branchPolicies(ctx, name)
	if err != nil {
		return nil, err
	}
	who, err := x.Agent.who(ctx, s)
	if err != nil {
		return nil, err
	}
	out := []PolicyItem{}
	for _, p := range ps {
		c, found, err := x.current(ctx, p.Doc)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if ok, err := x.opens(ctx, s, c); err != nil || !ok {
			if err != nil {
				return nil, err
			}
			continue
		}
		rs, err := x.Policies.PolicyReads(ctx, x.Agent.Tenant, p.Space, p.Doc)
		if err != nil {
			return nil, fmt.Errorf("%w: policies: %v", ErrUnavailable, err)
		}
		r := x.Agent.result(ctx, nil, c)
		out = append(out, PolicyItem{Space: p.Space, Doc: p.Doc, Cite: c.Cite, Title: r.Title, URL: r.URL, Version: c.Version, Read: readBy(rs, who.Email, c.Version)})
	}
	return out, nil
}

// ReadPolicy records s's I've read it for the version they saw, audited first. A version
// that isn't the current one is refused: they read again.
func (x Stewards) ReadPolicy(ctx context.Context, s tenancy.Scope, name, doc string, version int) error {
	items, err := x.PoliciesFor(ctx, s, name)
	if err != nil {
		return err
	}
	for _, it := range items {
		if it.Doc != doc {
			continue
		}
		if it.Version != version {
			return fmt.Errorf("%w: the policy has a new version; read it again", ErrInvalid)
		}
		who, err := x.Agent.who(ctx, s)
		if err != nil || who.Email == "" {
			return fmt.Errorf("%w: you have no email to record this under", ErrUnavailable)
		}
		if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: "policy:" + doc, State: "policy_read",
			Reason: fmt.Sprintf("read version %d", version), Space: auditSpace(SpaceKey(it.Space))}); err != nil {
			return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
		}
		return x.Policies.PutPolicyRead(ctx, PolicyRead{Tenant: x.Agent.Tenant, Space: it.Space, Doc: doc, User: s.User(),
			Email: strings.ToLower(who.Email), Version: version, At: x.Now().UTC()})
	}
	return fmt.Errorf("%w: no such policy for you here", ErrInvalid)
}

// PolicyStatuses are space name's own policies for its stewards: who of the branch's
// members hasn't read each current version, by email, and how many can't open it.
func (x Stewards) PolicyStatuses(ctx context.Context, s tenancy.Scope, name string) ([]PolicyStatus, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	if x.Policies == nil || x.Members == nil || x.ScopeOf == nil {
		return nil, fmt.Errorf("%w: this cell keeps no policies", ErrUnavailable)
	}
	ps, err := x.Policies.Policies(ctx, x.Agent.Tenant, name)
	if err != nil {
		return nil, fmt.Errorf("%w: policies: %v", ErrUnavailable, err)
	}
	members, err := x.branchMembers(ctx, name)
	if err != nil {
		return nil, err
	}
	out := []PolicyStatus{}
	for _, p := range ps {
		c, found, err := x.current(ctx, p.Doc)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		rs, err := x.Policies.PolicyReads(ctx, x.Agent.Tenant, p.Space, p.Doc)
		if err != nil {
			return nil, fmt.Errorf("%w: policies: %v", ErrUnavailable, err)
		}
		r := x.Agent.result(ctx, nil, c)
		st := PolicyStatus{PolicyItem: PolicyItem{Space: p.Space, Doc: p.Doc, Cite: c.Cite, Title: r.Title, URL: r.URL, Version: c.Version}, NotRead: []string{}}
		for _, m := range members {
			ms, ok, err := x.ScopeOf(ctx, m)
			if err != nil {
				return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
			}
			can := false
			if ok {
				if can, err = x.opens(ctx, ms, c); err != nil {
					return nil, err
				}
			}
			switch {
			case !can:
				st.CantOpen++ // counted, never named: fix its access at the source
			case readBy(rs, m, c.Version):
				st.ReadCount++
			default:
				st.NotRead = append(st.NotRead, m)
			}
		}
		out = append(out, st)
	}
	return out, nil
}

// branchMembers are the active members of space name and its teams, by email, each once.
func (x Stewards) branchMembers(ctx context.Context, name string) ([]string, error) {
	spaces := []string{name}
	if x.Children != nil {
		spaces = append(spaces, x.Children(name)...)
	}
	seen := map[string]bool{}
	var out []string
	for _, sp := range spaces {
		ms, err := x.Members(ctx, SpaceKey(sp))
		if err != nil {
			return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		for _, m := range ms {
			if m = strings.ToLower(m); !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// PolicyNotice is one policy version to tell its readers about once (SB-4103): Key names
// the version, To are the user ids of the branch's members who can open it and haven't
// read it.
type PolicyNotice struct {
	Key, Space, Title string
	To                []string
}

// PolicyNotices are the current versions of the policies of spaces, each with the members
// to tell, checked as each of them now.
func (x Stewards) PolicyNotices(ctx context.Context, spaces []string) ([]PolicyNotice, error) {
	if x.Policies == nil || x.Members == nil || x.ScopeOf == nil {
		return nil, nil
	}
	var out []PolicyNotice
	for _, name := range spaces {
		ps, err := x.Policies.Policies(ctx, x.Agent.Tenant, name)
		if err != nil || len(ps) == 0 {
			if err != nil {
				return nil, err
			}
			continue
		}
		members, err := x.branchMembers(ctx, name)
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			c, found, err := x.current(ctx, p.Doc)
			if err != nil {
				return nil, err
			}
			if !found {
				continue
			}
			rs, err := x.Policies.PolicyReads(ctx, x.Agent.Tenant, p.Space, p.Doc)
			if err != nil {
				return nil, err
			}
			n := PolicyNotice{Key: hashOf(fmt.Sprintf("%s|%s|%d", p.Space, p.Doc, c.Version))[:16], Space: name, Title: c.Title}
			for _, m := range members {
				ms, ok, err := x.ScopeOf(ctx, m)
				if err != nil {
					return nil, err
				}
				if !ok || readBy(rs, m, c.Version) {
					continue
				}
				if can, err := x.opens(ctx, ms, c); err != nil {
					return nil, err
				} else if can {
					n.To = append(n.To, ms.User())
				}
			}
			out = append(out, n)
		}
	}
	return out, nil
}
