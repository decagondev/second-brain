package report

import (
	"fmt"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/ask"
)

// Adoption (sb-observability(7), Weekly report; SB-3804): how many seats asked this week
// and last, and by space for spaces where at least 5 people asked. Counts only.
type Adoption struct {
	Asked         int            `json:"asked"`
	AskedLastWeek int            `json:"asked_last_week"`
	Seats         int            `json:"seats"`
	Spaces        map[string]int `json:"spaces"`
}

// askers are the distinct people who asked among traces (no searches, no limited asks),
// and by space ("company" for company-wide).
func askers(traces []ask.Trace) (map[string]bool, map[string]map[string]bool) {
	all, by := map[string]bool{}, map[string]map[string]bool{}
	for _, t := range traces {
		if t.Response.Agent == ask.AgentSearch || t.Response.Agent == ask.AgentLimited || t.User == "" {
			continue
		}
		all[t.User] = true
		sp := strings.TrimPrefix(t.Space, "spc_")
		if sp == "" {
			sp = "company"
		}
		if by[sp] == nil {
			by[sp] = map[string]bool{}
		}
		by[sp][t.User] = true
	}
	return all, by
}

// AdoptionOf is this week's and last week's traces as adoption, of seats.
func AdoptionOf(week, before []ask.Trace, seats int) *Adoption {
	now, by := askers(week)
	then, _ := askers(before)
	a := &Adoption{Asked: len(now), AskedLastWeek: len(then), Seats: seats, Spaces: map[string]int{}}
	for sp, people := range by {
		if len(people) >= ask.MinUsage {
			a.Spaces[sp] = len(people)
		}
	}
	return a
}

// line is the report's row text.
func (a Adoption) line() string {
	s := fmt.Sprintf("%d of %d seats asked this week (%d last week)", a.Asked, a.Seats, a.AskedLastWeek)
	if len(a.Spaces) == 0 {
		return s
	}
	sps := make([]string, 0, len(a.Spaces))
	for k := range a.Spaces {
		sps = append(sps, k)
	}
	sort.Slice(sps, func(i, j int) bool {
		return a.Spaces[sps[i]] > a.Spaces[sps[j]] || a.Spaces[sps[i]] == a.Spaces[sps[j]] && sps[i] < sps[j]
	})
	parts := make([]string, len(sps))
	for i, k := range sps {
		parts[i] = fmt.Sprintf("%s %d", k, a.Spaces[k])
	}
	return s + "; by space: " + strings.Join(parts, ", ")
}
