// Package spaces is a cell's space registry and the space a request is in (sb-spaces(7),
// The space registry and the request context; SB-1002). Sync makes the registry match
// the manifest and is called only by `sb tenant apply`. Resolve turns a request's
// ?space= into a space id, or refuses it: an unknown space, a non-member and a registry
// that can't answer are all refusals, never a company-wide request instead.
package spaces

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/keywrap"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// Company is the space of a company-wide request.
const Company = "company"

// ID is a space's registry id.
func ID(name string) string { return "spc_" + name }

// ErrSpace refuses a request in a space the registry doesn't hold or the asker isn't a
// member of. One error for both, so a non-member can't learn whether a space exists.
var ErrSpace = errors.New("spaces: no such space for you; ask its stewards for membership, or ask company-wide")

// ErrUnavailable is a registry that can't answer. Unknown never means company-wide.
var ErrUnavailable = errors.New("spaces: the space registry is unavailable")

// Keys is where Sync keeps space data keys (SB-1003): the cell's wrap inventory and the
// custodian whose root key wraps a new space's key.
type Keys struct {
	Wraps     keywrap.Store
	Custodian keywrap.Custodian
}

// Sync makes the registry hold exactly the manifest's spaces: puts every declared space
// and deletes every other. With keys, a new space gets its data key before its row
// exists, and a dropped space's key is shredded after its row is gone (sb-tenancy(7)).
// Any error stops it; the caller retries the whole of it.
func Sync(ctx context.Context, st ports.SpaceStore, keys *Keys, t manifest.Tenant) error {
	have, err := st.Spaces(ctx, t.Tenant)
	if err != nil {
		return fmt.Errorf("spaces: %w", err)
	}
	names := make([]string, 0, len(t.Spaces))
	for n := range t.Spaces {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		s := t.Spaces[n]
		if keys != nil {
			if err := keywrap.Seed(ctx, t.Tenant, keys.Wraps, keys.Custodian, keywrap.PurposeSpace(ID(n))); err != nil {
				return fmt.Errorf("spaces: key for %s: %w", n, err)
			}
		}
		if err := st.PutSpace(ctx, ports.Space{ID: ID(n), Tenant: t.Tenant, Name: n, Template: s.Template, Label: s.Label,
			Members: s.Members, Stewards: s.Stewards, Agents: s.Agents, Scope: s.Scope, RetentionDays: s.RetentionDays}); err != nil {
			return fmt.Errorf("spaces: %s: %w", n, err)
		}
	}
	for _, h := range have {
		if _, keep := t.Spaces[h.Name]; !keep {
			if err := st.DeleteSpace(ctx, t.Tenant, h.ID); err != nil {
				return fmt.Errorf("spaces: %s: %w", h.Name, err)
			}
			if keys != nil {
				if err := keywrap.Shred(ctx, t.Tenant, keys.Wraps, keywrap.PurposeSpace(h.ID)); err != nil {
					return fmt.Errorf("spaces: shred %s: %w", h.Name, err)
				}
			}
		}
	}
	return nil
}

// Resolve returns the space id a request is in: Company for no space or "company", or
// the id of a space the asker (with these directory group names) is a member of.
func Resolve(ctx context.Context, st ports.SpaceStore, tenant string, groups []string, requested string) (string, error) {
	if requested == "" || requested == Company {
		return Company, nil
	}
	if st == nil {
		return "", ErrSpace // a cell without a registry has no spaces
	}
	s, found, err := st.Space(ctx, tenant, ID(requested))
	switch {
	case err != nil:
		return "", ErrUnavailable
	case !found || s.Tenant != tenant || s.Name != requested:
		return "", ErrSpace
	case !Member(s, groups):
		return "", ErrSpace
	}
	return s.ID, nil
}

// Member is whether any of the asker's directory groups is one of the space's members.
func Member(s ports.Space, groups []string) bool {
	for _, m := range s.Members {
		for _, g := range groups {
			if strings.TrimPrefix(m, "group:") == g {
				return true
			}
		}
	}
	return false
}
