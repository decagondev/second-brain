package api

import (
	"context"
	"net/http"
	"sort"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/errs"
)

// Branch moderators (sb-spaces(7), Branch moderators; SB-5105): a steward of a space sees,
// for every space below it, the agents with where each is paused, and the names of its
// skills and rhythms and how many routines are shared there. Names and counts only.

// BranchAgent is one agent a space lists, and whether it is paused there.
type BranchAgent struct {
	Name   string `json:"name"`
	Paused bool   `json:"paused"`
}

// BranchSpace is one space below the moderator's.
type BranchSpace struct {
	Space          string        `json:"space"`
	Parent         string        `json:"parent"`
	Agents         []BranchAgent `json:"agents"`
	Skills         []string      `json:"skills"`
	Rhythms        []string      `json:"rhythms"`
	SharedRoutines int           `json:"shared_routines"`
}

// below are every space under name, depth first, each once.
func below(t manifest.Tenant, name string) []string {
	var out []string
	seen := map[string]bool{name: true}
	var walk func(n string)
	walk = func(n string) {
		kids := t.Children(n)
		sort.Strings(kids)
		for _, k := range kids {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
				walk(k)
			}
		}
	}
	walk(name)
	return out
}

// branch is GET /v1/spaces/NAME/branch, for the space's stewards.
func (h Stewards) branch(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s, cur, ok := h.stewardOf(w, r, name)
	if !ok {
		return
	}
	if h.Feature != nil && !h.Feature(r.Context(), s.User(), "branch-moderators") {
		fail(w, errs.New(errs.EPERM, "Branch moderation isn't on for you yet: it is with the pilot group first."))
		return
	}
	out := []BranchSpace{}
	for _, n := range below(cur, name) {
		sp := cur.Spaces[n]
		b := BranchSpace{Space: n, Parent: sp.Parent, Agents: []BranchAgent{}, Skills: []string{}, Rhythms: []string{}}
		for _, a := range sp.Agents {
			b.Agents = append(b.Agents, BranchAgent{Name: a, Paused: h.PausedIn != nil && contains(h.PausedIn(a), n)})
		}
		for k := range sp.Skills {
			b.Skills = append(b.Skills, k)
		}
		for k := range sp.Rhythms {
			b.Rhythms = append(b.Rhythms, k)
		}
		sort.Strings(b.Skills)
		sort.Strings(b.Rhythms)
		if h.SharedCount != nil {
			c, err := h.SharedCount(r.Context(), n)
			if err != nil {
				fail(w, errs.New(errs.EUNAVAIL, "Routines can't be read right now."))
				return
			}
			b.SharedRoutines = c
		}
		out = append(out, b)
	}
	writeJSON(w, map[string]any{"space": name, "below": out})
}

// SharedCounter counts the routines shared with a space (SB-5105).
type SharedCounter func(ctx context.Context, space string) (int, error)
