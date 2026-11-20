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
	return SyncHeld(ctx, st, keys, t, nil)
}

// ErrHeld is a manifest that would remove a space a legal hold covers (EHOLD; SB-1207).
var ErrHeld = errors.New("spaces: EHOLD a legal hold covers the space")

// SyncHeld is Sync for a cell with legal holds: if any space it would remove is held
// (holds.Set.HeldSpace), it changes nothing and returns ErrHeld, so a held space's row
// stays and its key is never shredded.
func SyncHeld(ctx context.Context, st ports.SpaceStore, keys *Keys, t manifest.Tenant, held func(spaceID string) bool) error {
	have, err := st.Spaces(ctx, t.Tenant)
	if err != nil {
		return fmt.Errorf("spaces: %w", err)
	}
	if held != nil {
		for _, h := range have {
			if _, keep := t.Spaces[h.Name]; !keep && held(h.ID) {
				return fmt.Errorf("%w: %s; release the hold first", ErrHeld, h.Name)
			}
		}
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
			Channels: s.Channels, Fields: s.Fields}); err != nil {
			return fmt.Errorf("spaces: %s: %w", n, err)
		}
	}
	for _, h := range have {
		if _, keep := t.Spaces[h.Name]; !keep {
			// Shred first, then drop the row: if the shred fails, the row is still there and
			// the next apply shreds again; the other way round, a failed shred left the key
			// openable with nothing to retry it (SB-1401). Shred is idempotent.
			if keys != nil {
				if err := keywrap.Shred(ctx, t.Tenant, keys.Wraps, keywrap.PurposeSpace(h.ID)); err != nil {
					return fmt.Errorf("spaces: shred %s: %w", h.Name, err)
				}
			}
			if err := st.DeleteSpace(ctx, t.Tenant, h.ID); err != nil {
				return fmt.Errorf("spaces: %s: %w", h.Name, err)
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
	mine := map[string]bool{} // the asker's directory group names
	for _, g := range groups {
		mine[g] = true
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
	return map[string]any{"space": space, "member_spaces": member, "barred_spaces": barred,
		"barred_groups": BarredGroups(barriers, groups)}
}

// BarredGroups is the groups (group:NAME) across a barrier from someone in groups, in
// either direction (SB-1203). Groups and barriers may name groups as NAME or group:NAME.
func BarredGroups(barriers [][2]string, groups []string) []string {
	name := func(g string) string { return strings.TrimPrefix(g, "group:") }
	mine := map[string]bool{}
	for _, g := range groups {
		mine[name(g)] = true
	}
	var out []string
	for _, b := range barriers {
		a, c := name(b[0]), name(b[1])
		if mine[a] {
			out = append(out, c)
		}
		if mine[c] {
			out = append(out, a)
		}
	}
	return GroupSet(out)
}

// GroupSet is a set of group names as sorted group:NAME values.
func GroupSet(names []string) []string {
	set := map[string]bool{}
	for _, g := range names {
		if g != "" {
			set["group:"+strings.TrimPrefix(g, "group:")] = true
		}
	}
	out := make([]string, 0, len(set))
	for g := range set {
		out = append(out, g)
	}
	sort.Strings(out)
	return out
}

// acrossBarrier: a barrier names groups as group:NAME (the manifest) or NAME; both sides
// are compared by name, as Member compares a space's members with directory groups.
func acrossBarrier(s ports.Space, barriers [][2]string, mine map[string]bool) bool {
	name := func(g string) string { return strings.TrimPrefix(g, "group:") }
	for _, b := range barriers {
		a, c := name(b[0]), name(b[1])
		for _, m := range s.Members {
			if (mine[a] && name(m) == c) || (mine[c] && name(m) == a) {
				return true
			}
		}
	}
	return false
}

// ChunkAttrs are a chunk's policy attributes in a cell with spaces: its label, the ids of
// the spaces that hold its document, and its owners, empty until the caller sets the
// owner's groups (barrier.Owners; SB-1203). An empty label is left out, so the read refuses.
func ChunkAttrs(label string, holders []string) map[string]any {
	a := map[string]any{"spaces": append([]string{}, holders...), "owners": []string{}}
	if label != "" {
		a["label"] = label
	}
	return a
}
