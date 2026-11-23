package route

import (
	"sort"

	"example.com/second-brain/internal/core/ports"
)

// The problems a scope pattern can have (sb-space(1), scope-report; SB-2903).
const (
	ProblemNone    = "none"
	ProblemTooWide = "too_wide"
)

// PatternCount is how many of a kind's documents one space's scope pattern matches now.
type PatternCount struct {
	Space   string `json:"space"`
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	Matches int    `json:"matches"`
	Of      int    `json:"of"`
	Problem string `json:"problem"`
}

// Report counts, for each space's scope patterns, the documents whose locations match,
// out of the kind's documents (locations by kind, each document once). A pattern matching
// none is ProblemNone; one matching more than half of its kind's documents is ProblemTooWide.
// It reads locations only and decides nothing.
func Report(spaces []ports.Space, locations map[string][]string) []PatternCount {
	var out []PatternCount
	sps := append([]ports.Space(nil), spaces...)
	sort.Slice(sps, func(i, j int) bool { return sps[i].Name < sps[j].Name })
	for _, sp := range sps {
		kinds := make([]string, 0, len(sp.Scope))
		for k := range sp.Scope {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		for _, k := range kinds {
			for _, pat := range sp.Scope[k] {
				pc := PatternCount{Space: sp.Name, Kind: k, Pattern: pat, Of: len(locations[k])}
				for _, l := range locations[k] {
					if Match(pat, l) {
						pc.Matches++
					}
				}
				switch {
				case pc.Matches == 0:
					pc.Problem = ProblemNone
				case 2*pc.Matches > pc.Of:
					pc.Problem = ProblemTooWide
				}
				out = append(out, pc)
			}
		}
	}
	return out
}
