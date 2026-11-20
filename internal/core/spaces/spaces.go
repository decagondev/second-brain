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
			Members: s.Members, Stewards: s.Stewards, Agents: s.Agents, Scope: s.Scope, RetentionDays: s.RetentionDays,
			Channels: s.Channels}); err != nil {
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

// ReadContext is the policy context of a read in a cell with spaces (sb-permissions(7),
// Spaces, labels and barriers, exactly; SB-1004): the space asked in (an id or Company),
// the spaces the asker is a member of, and the spaces held across a barrier from the
// asker. groups are the asker's directory group names; barriers are group:NAME pairs.
func ReadContext(all []ports.Space, barriers [][2]string, groups []string, space string) map[string]any {
	mine := map[string]bool{}
	for _, g := range groups {
		mine["group:"+g] = true
	}
	member, barred := []string{}, []string{}
	for _, s := range all {
		if Member(s, groups) {
			member = append(member, s.ID)
		}
		if acrossBarrier(s, barriers, mine) {
			barred = append(barred, s.ID)
		}
	}
	if space == "" {
		space = Company
	}
	return map[string]any{"space": space, "member_spaces": member, "barred_spaces": barred}
}

func acrossBarrier(s ports.Space, barriers [][2]string, mine map[string]bool) bool {
	for _, b := range barriers {
		for _, m := range s.Members {
			if (mine[b[0]] && m == b[1]) || (mine[b[1]] && m == b[0]) {
				return true
			}
		}
	}
	return false
}

// ChunkAttrs are a chunk's policy attributes in a cell with spaces: its label and the ids
// of the spaces that hold its document. An empty label is left out, so the read refuses.
func ChunkAttrs(label string, holders []string) map[string]any {
	a := map[string]any{"spaces": append([]string{}, holders...)}
	if label != "" {
		a["label"] = label
	}
	return a
}
