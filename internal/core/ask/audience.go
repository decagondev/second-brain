package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/retrieve"
	"example.com/second-brain/internal/core/tenancy"
)

// Who can see this (sb-ask(3), Who can see this; SB-5404): how many people can open a
// document now, asked of its source as each of them, and the groups behind that the person
// may see: their own, and their space's. Other groups are counted, never named.

// MaxAudienceDocs is the most documents one audience takes (a shared answer's cites).
const MaxAudienceDocs = 20

// Audience is the 200 body of GET /v1/documents/{id}/audience.
type Audience struct {
	Documents int `json:"documents"`
	// People can open every document now; Unknown are people the source couldn't answer
	// for, counted apart and never as able.
	People  int `json:"people"`
	Unknown int `json:"unknown,omitempty"`
	// Groups are the groups behind the sharing the person may see; OtherGroups the rest.
	Groups      []string `json:"groups"`
	OtherGroups int      `json:"other_groups"`
	// Anyone is a document open to anyone with the link, and Domains the domains it is open
	// to as a whole.
	Anyone  bool     `json:"anyone,omitempty"`
	Domains []string `json:"domains,omitempty"`
}

// Audience is who can open every one of docs (KIND:ID) now. A document the person can't
// open answers like one that doesn't exist.
func (a Agent) Audience(ctx context.Context, s tenancy.Scope, docs []string) (Audience, error) {
	if err := s.Check(); err != nil || s.Tenant() != a.Tenant {
		return Audience{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	var keys []string
	seenKey := map[string]bool{}
	bad := false
	for _, d := range docs {
		d, _, _ = strings.Cut(strings.TrimSpace(d), "@")
		kind, id, ok := strings.Cut(d, ":")
		bad = bad || !ok || !Kinds[kind] || id == ""
		if !seenKey[d] {
			seenKey[d] = true
			keys = append(keys, d)
		}
	}
	if bad || len(keys) == 0 || len(keys) > MaxAudienceDocs {
		return Audience{}, fmt.Errorf("%w: 1 to %d documents, as KIND:ID", ErrInvalid, MaxAudienceDocs)
	}
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return Audience{}, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var refs []ports.SourceRef
	for _, k := range keys {
		kind, id, _ := strings.Cut(k, ":")
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	cs, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return Audience{}, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	firstOf := firsts(latest(cs))
	var cand []ports.Chunk
	for _, k := range keys {
		c, ok := firstOf[k]
		if !ok {
			return Audience{}, ErrNotFound
		}
		cand = append(cand, c)
	}
	who, err := a.who(ctx, s)
	if err != nil {
		return Audience{}, err
	}
	res, err := retrieve.Allowed(ctx, s, who, a.Auth, cand, len(cand))
	if err != nil {
		return Audience{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if res.Context, err = a.dropHeld(ctx, s, res.Context); err != nil { // SB-5402
		return Audience{}, err
	}
	if len(firsts(res.Context)) != len(cand) {
		return Audience{}, ErrNotFound // one they can't open, or none, looks the same
	}
	out := Audience{Documents: len(cand), Groups: []string{}}
	if a.Everyone != nil {
		people, err := a.Everyone(ctx)
		if err != nil {
			return Audience{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		for _, p := range people {
			v := a.Auth.Check(ctx, p, cand)
			all, unknown := true, false
			for _, c := range cand {
				switch v[c.Kind+":"+c.SourceID] {
				case ports.VerdictAllow:
				case ports.VerdictDeny:
					all = false
				case ports.VerdictUnknown:
					all, unknown = false, true
				default:
					all, unknown = false, true
				}
			}
			switch {
			case all:
				out.People++
			case unknown:
				out.Unknown++
			}
		}
	}
	seen := a.visibleGroups(ctx, s)
	groups := map[string]bool{}
	domains := map[string]bool{}
	for _, c := range cand {
		for _, g := range c.ACLPrefilter {
			switch {
			case g == "anyone":
				out.Anyone = true
			case strings.HasPrefix(g, "domain:"):
				domains[strings.TrimPrefix(g, "domain:")] = true
			case strings.HasPrefix(g, "group:"):
				groups[strings.TrimPrefix(g, "group:")] = true
			}
		}
	}
	for g := range groups {
		if seenGroup(seen, g) {
			out.Groups = append(out.Groups, g)
		} else {
			out.OtherGroups++
		}
	}
	sort.Strings(out.Groups)
	for d := range domains {
		out.Domains = append(out.Domains, d)
	}
	sort.Strings(out.Domains)
	return out, nil
}

// visibleGroups are the group names s's person may see named: their own, and those of the
// space they ask in (its members and stewards), lower case.
func (a Agent) visibleGroups(ctx context.Context, s tenancy.Scope) map[string]bool {
	out := map[string]bool{}
	for _, g := range s.Groups() {
		out[strings.ToLower(g)] = true
	}
	if a.SpaceOf != nil && s.Space() != "company" && s.Space() != "" {
		if sp, ok, err := a.SpaceOf(ctx, s.Space()); err == nil && ok {
			for _, m := range append(append([]string{}, sp.Members...), sp.Stewards...) {
				if g, ok := strings.CutPrefix(m, "group:"); ok {
					out[strings.ToLower(g)] = true
				}
			}
		}
	}
	return out
}

// seenGroup matches a grant's group (a name or an address) to a visible group name.
func seenGroup(seen map[string]bool, g string) bool {
	g = strings.ToLower(g)
	if seen[g] {
		return true
	}
	local, _, ok := strings.Cut(g, "@")
	return ok && seen[local]
}
