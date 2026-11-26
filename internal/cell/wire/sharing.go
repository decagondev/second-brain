package wire

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
	"example.com/second-brain/internal/core/ports"
)

// wide is the cell's sharing checks (sb-permissions(7), Shared too widely; SB-5401 to SB-5403).
func (c *Cell) wide(m manifest.Tenant) *ask.Wide {
	tenant := c.Config.Tenant
	name := func(key string) string { return strings.TrimPrefix(key, "spc_") }
	return &ask.Wide{Limit: c.Config.WideGroupSize, Confirms: c.Traces, Changes: c.Traces,
		GroupSize: func(ctx context.Context, g string) (int, bool) {
			for _, n := range groupNames(g) {
				if ms, err := c.Directory.Members(ctx, tenant, n); err == nil {
					return len(ms), true
				}
			}
			return 0, false
		},
		Mode: func(key string) string {
			if _, ok := m.Spaces[name(key)]; !ok {
				return ask.WideHold
			}
			return m.WideOf(name(key))
		},
		Stewards: func(key string) []string {
			st, _, _ := stewardFacts(m)(key)
			return st
		},
		Restricted: func(key string) bool { return m.Effective(name(key)).Label == "restricted" }}
}

// groupNames are the directory names a grant's group may have: as written, and an
// address's local part.
func groupNames(g string) []string {
	out := []string{g}
	if local, _, ok := strings.Cut(g, "@"); ok && local != "" {
		out = append(out, local)
	}
	return out
}

// everyone are the tenant's active people with their first address, for who can see a
// document (SB-5404).
func (c *Cell) everyone(ctx context.Context) ([]ports.UserRef, error) {
	ids, err := c.Directory.ActiveUserIDs(ctx, c.Config.Tenant)
	if err != nil {
		return nil, err
	}
	out := make([]ports.UserRef, 0, len(ids))
	for _, id := range ids {
		u, found, err := c.Directory.User(ctx, c.Config.Tenant, id)
		if err != nil {
			return nil, err
		}
		if !found || !u.Active {
			continue
		}
		ref := ports.UserRef{ID: u.ID}
		if len(u.Emails) > 0 {
			ref.Email = u.Emails[0]
		}
		out = append(out, ref)
	}
	return out, nil
}

// sharingNotices are one notice per sharing widening seen at sync in the last day (SB-5403):
// to the owner when they can still open the document, else to the space's stewards.
// Narrowing is recorded but not announced.
func (c *Cell) sharingNotices(ctx context.Context, now time.Time) ([]notify.Notice, error) {
	evs, err := c.Traces.SharingEvents(ctx, c.Config.Tenant, now.Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	m := c.Manifest()
	var out []notify.Notice
	for _, e := range evs {
		if !e.Widened {
			continue
		}
		var added []string
		for _, g := range e.Added {
			if strings.HasPrefix(g, "group:") || strings.HasPrefix(g, "domain:") || g == "anyone" {
				added = append(added, ask.GrantName(g))
			}
		}
		text := fmt.Sprintf("The sharing of one of your documents (%s) widened at its source: now also %s. Second Brain changes nothing at the source; narrow it there if that wasn't meant",
			e.Doc, strings.Join(added, ", "))
		to, err := c.widenedTo(ctx, m, e)
		if err != nil {
			return nil, err
		}
		for _, id := range to {
			out = append(out, notify.Notice{Key: "widened-" + e.ID + "-" + id, User: id, Kind: notify.KindWidened, Text: text, Tab: "inbox"})
		}
	}
	return out, nil
}

// widenedTo is who a widening is told to: the owner, if they are one active person who can
// still open the document at its source, else the stewards of the spaces that hold it.
func (c *Cell) widenedTo(ctx context.Context, m manifest.Tenant, e ask.SharingEvent) ([]string, error) {
	if e.Owner != "" {
		s, found, err := c.ownerScope(ctx, e.Owner)
		if err != nil {
			return nil, err
		}
		kind, id, _ := strings.Cut(e.Doc, ":")
		if found {
			if conn, ok := c.Parts().Connectors[kind]; ok {
				vs, err := conn.ACL(ctx, ports.UserRef{ID: s.User(), Email: e.Owner}, []ports.SourceRef{{Kind: kind, ID: id}})
				if err == nil && len(vs) == 1 && vs[0] == ports.VerdictAllow {
					return []string{s.User()}, nil
				}
			}
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, sp := range e.Spaces {
		ps, err := c.stewardPeople(m)(ctx, sp)
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			us, err := c.Directory.UsersByEmail(ctx, c.Config.Tenant, p.Email)
			if err != nil {
				return nil, err
			}
			for _, u := range us {
				if u.Active && !seen[u.ID] {
					seen[u.ID] = true
					out = append(out, u.ID)
				}
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// notices are the notify job's one-off notices: sharing widenings (SB-5403).
func (c *Cell) notices(ctx context.Context, now time.Time) ([]notify.Notice, error) {
	return c.sharingNotices(ctx, now)
}
