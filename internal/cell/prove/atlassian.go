package prove

import (
	"context"
	"errors"
	"strings"

	"example.com/second-brain/internal/adapters/atlassian"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/proof"
)

// Confluence and Jira (SB-6201, SB-6202, SB-6204): sb prove reads the named spaces and
// projects, their ACL copies and the live check for a person, and the person's groups.

func init() {
	Runners["SB-6201"] = Runner{"SB-6201", "confluence-v2", confluence}
	Runners["SB-6202"] = Runner{"SB-6202", "jira-v3", atlassianGroups}
	Runners["SB-6204"] = Runner{"SB-6204", "jira-v3", jira}
}

// listAndCheck lists a connector's changes, reads up to 25 of them and asks the live check
// for the person: the criteria SB-6201 and SB-6204 share.
func listAndCheck(ctx context.Context, c ports.Connector, person string, h proof.Hasher) []proof.Criterion {
	var refs []ports.SourceRef
	cs := []proof.Criterion{timed("the named spaces or projects list their items", func(cr *proof.Criterion) {
		set, _, err := c.Changes(ctx, "")
		if err != nil {
			cr.Counts["errors"] = 1
			return
		}
		var ids []string
		for _, ch := range set {
			refs = append(refs, ch.Ref)
			ids = append(ids, ch.Ref.ID)
		}
		cr.Counts["items"] = len(refs)
		cr.Seen = h.IDs(ids)
		cr.Pass = len(refs) > 0
	})}
	cs = append(cs, timed("each item reads with its permissions", func(cr *proof.Criterion) {
		if len(refs) == 0 {
			missing(cr)
			return
		}
		for _, r := range refs[:min(len(refs), 25)] {
			d, err := c.Fetch(ctx, r)
			if err != nil {
				cr.Counts["errors"]++
				continue
			}
			cr.Counts["read"]++
			if len(d.ACLPrefilter) > 0 {
				cr.Counts["with_permissions"]++
			}
			for _, a := range d.ACLPrefilter {
				if a == "anyone" {
					cr.Counts["wide"]++
				}
			}
		}
		cr.Pass = cr.Counts["errors"] == 0 && cr.Counts["with_permissions"] == cr.Counts["read"]
	}))
	cs = append(cs, timed("the live check answers for a person", func(cr *proof.Criterion) {
		if person == "" || len(refs) == 0 {
			missing(cr)
			return
		}
		vs, err := c.ACL(ctx, ports.UserRef{ID: person, Email: person}, refs[:min(len(refs), 25)])
		if err != nil {
			cr.Counts["errors"] = 1
			return
		}
		for _, v := range vs {
			cr.Counts[v.String()]++
		}
		cr.Pass = cr.Counts["unknown"] == 0
	}))
	return cs
}

func confluence(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion {
	if e.Confluence == nil {
		c := proof.Criterion{Name: "the named spaces or projects list their items", Counts: map[string]int{"missing_input": 1}}
		return []proof.Criterion{c}
	}
	return listAndCheck(ctx, e.Confluence, e.Person, h)
}

func jira(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion {
	if e.Jira == nil {
		c := proof.Criterion{Name: "the named spaces or projects list their items", Counts: map[string]int{"missing_input": 1}}
		return []proof.Criterion{c}
	}
	return listAndCheck(ctx, e.Jira, e.Person, h)
}

func atlassianGroups(ctx context.Context, e Env, h proof.Hasher) []proof.Criterion {
	return []proof.Criterion{timed("a person's atlassian groups and their members resolve", func(c *proof.Criterion) {
		if e.Atlassian == nil || e.Person == "" {
			missing(c)
			return
		}
		dir := &atlassian.Directory{Client: e.Atlassian}
		gs, err := dir.GroupsOf(ctx, e.Person)
		if err != nil {
			c.Counts["errors"] = 1
			return
		}
		c.Counts["groups"] = len(gs)
		c.Seen = h.IDs(gs)
		for _, g := range gs[:min(len(gs), 10)] {
			ms, err := dir.Members(ctx, g)
			switch {
			case errors.Is(err, atlassian.ErrTooLarge):
				c.Counts["too_large"]++
			case err != nil:
				c.Counts["errors"]++
			default:
				c.Counts["members"] += len(ms)
				if !containsFold(ms, e.Person) {
					c.Counts["not_listed"]++
				}
			}
		}
		c.Pass = c.Counts["errors"] == 0 && c.Counts["not_listed"] == 0
	})}
}

func containsFold(xs []string, x string) bool {
	for _, y := range xs {
		if strings.EqualFold(y, x) {
			return true
		}
	}
	return false
}
