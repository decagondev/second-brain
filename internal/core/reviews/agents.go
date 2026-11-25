package reviews

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/manifest"
)

// Agent access reviews (sb-agents(7), Agent access reviews; SB-4705): each quarter, beside
// the reviews of people, every space with stewards that runs or lists an agent gets a review
// of its department agents and triggers, each with its last use. The unused ones are proposed
// off as a manifest diff; the stewards decide every item, and the removals become one
// manifest proposal an operator approves. Nothing is changed by the review itself.

// KindAgents marks an agent review; a review of people has no kind.
const KindAgents = "agents"

// UseWindow is how far back an item's use counts: an item with none is unused.
const UseWindow = 90 * 24 * time.Hour

// Trigger items, besides the department agents a space lists by name.
const (
	ItemUpdates    = "updates"
	ItemLeaderWeek = "leader_week"
	briefPrefix    = "brief:"
)

// AgentFacts are what an agent review reads from the cell.
type AgentFacts struct {
	// Department is whether an agent is a department agent a space lists (not ask).
	Department func(agent string) bool
	// Tools are the tools an agent's definition lists.
	Tools func(agent string) []string
	// LastUse is when item last ran in space: an agent's latest draft or answer there, or a
	// trigger's latest draft.
	LastUse func(space, item string) (time.Time, bool)
}

// agentItems are a space's department agents and triggers, in a fixed order.
func agentItems(t manifest.Tenant, name string, f AgentFacts) []Item {
	sp := t.Spaces[name]
	var out []Item
	agents := append([]string{}, sp.Agents...)
	sort.Strings(agents)
	for _, a := range agents {
		if f.Department != nil && f.Department(a) {
			out = append(out, Item{User: a, Via: []string{"agents"}, Tools: toolsOf(f, a)})
		}
	}
	if sp.Brief != nil {
		out = append(out, Item{User: briefPrefix + sp.Brief.Agent, Via: []string{"brief"}, Tools: toolsOf(f, sp.Brief.Agent)})
	}
	if sp.Updates {
		out = append(out, Item{User: ItemUpdates, Via: []string{"updates"}, Tools: toolsOf(f, "update")})
	}
	if t.LeaderWeek(name) {
		out = append(out, Item{User: ItemLeaderWeek, Via: []string{"leader_week"}, Tools: toolsOf(f, "leader-week")})
	}
	return out
}

func toolsOf(f AgentFacts, agent string) []string {
	if f.Tools == nil {
		return []string{}
	}
	ts := append([]string{}, f.Tools(agent)...)
	sort.Strings(ts)
	return ts
}

// OpenAgents opens an agent review for every space in t with stewards and at least one
// item that has none for this quarter.
func OpenAgents(t manifest.Tenant, now time.Time, f AgentFacts, have func(space, quarter string) bool, newID func(prefix string) string) []Review {
	names := make([]string, 0, len(t.Spaces))
	for n, sp := range t.Spaces {
		if len(sp.Stewards) > 0 {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	q := Quarter(now)
	var out []Review
	for _, n := range names {
		if have != nil && have(n, q) {
			continue
		}
		items := agentItems(t, n, f)
		if len(items) == 0 {
			continue
		}
		var unused []string
		for i := range items {
			items[i].LastUse = Never
			if f.LastUse != nil {
				if at, ok := f.LastUse(n, items[i].User); ok {
					items[i].LastUse = at.UTC().Format(time.RFC3339)
					items[i].Unused = now.Sub(at) > UseWindow
				} else {
					items[i].Unused = true
				}
			} else {
				items[i].Unused = true
			}
			if items[i].Unused {
				unused = append(unused, items[i].User)
			}
		}
		r := Review{ID: newID("rev_"), Kind: KindAgents, Space: n, Quarter: q, Stewards: append([]string{}, t.Spaces[n].Stewards...),
			Opened: now.UTC().Format(time.RFC3339), Due: now.Add(Window).UTC().Format(time.RFC3339), State: Open,
			Items: items, Requests: []Request{}, ProposeOff: []string{}}
		if len(unused) > 0 {
			if next, changed := TakeOff(t, n, unused); changed {
				r.ProposeOff = manifest.Diff(t, next)
			}
		}
		out = append(out, r)
	}
	return out
}

// TakeOff is t with items taken off space name: agents out of its agents list, its brief,
// updates and leader's week turned off.
func TakeOff(t manifest.Tenant, name string, items []string) (manifest.Tenant, bool) {
	next := t
	next.Spaces = make(map[string]manifest.Space, len(t.Spaces))
	for k, v := range t.Spaces {
		next.Spaces[k] = v
	}
	sp, ok := next.Spaces[name]
	if !ok {
		return t, false
	}
	changed := false
	for _, it := range items {
		switch {
		case it == ItemUpdates:
			changed = changed || sp.Updates
			sp.Updates = false
		case it == ItemLeaderWeek:
			off := false
			sp.LeaderWeek, changed = &off, true
		case strings.HasPrefix(it, briefPrefix):
			changed = changed || sp.Brief != nil
			sp.Brief = nil
		default:
			var keep []string
			for _, a := range sp.Agents {
				if a == it {
					changed = true
					continue
				}
				keep = append(keep, a)
			}
			if keep == nil {
				keep = []string{}
			}
			sp.Agents = keep
		}
	}
	next.Spaces[name] = sp
	return next, changed
}

// agentRequests turns a complete agent review's removals into one manifest change.
func agentRequests(t manifest.Tenant, r Review) (manifest.Tenant, bool, []Request) {
	var off []string
	reqs := []Request{}
	for _, it := range r.Items {
		if it.Decision == Remove {
			off = append(off, it.User)
			reqs = append(reqs, Request{Kind: "manifest", User: it.User, Group: strings.Join(it.Via, ",")})
		}
	}
	if len(off) == 0 {
		return t, false, reqs
	}
	next, changed := TakeOff(t, r.Space, off)
	return next, changed, reqs
}

// RunAgents opens the quarter's agent reviews that don't exist yet and stores them.
func RunAgents(ctx context.Context, t manifest.Tenant, now time.Time, f AgentFacts, st Store, newID func(string) string) ([]Review, error) {
	all, err := st.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("reviews: the store can't be read: %w", err)
	}
	have := func(space, quarter string) bool {
		for _, r := range all {
			if r.Kind == KindAgents && r.Space == space && r.Quarter == quarter {
				return true
			}
		}
		return false
	}
	opened := OpenAgents(t, now, f, have, newID)
	for _, r := range opened {
		if err := st.Put(ctx, r); err != nil {
			return nil, fmt.Errorf("reviews: review %s can't be stored: %w", r.ID, err)
		}
	}
	return opened, nil
}
