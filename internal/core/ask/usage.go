package ask

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Usage by space (sb-observability(7), Usage by space; SB-3506).

// MinUsage is the fewest asks a space's row shows numbers for.
const MinUsage = 5

// SpaceUsage is one space's asks, the share that found nothing and the share rated helpful;
// a space with fewer than MinUsage asks shows only that.
type SpaceUsage struct {
	Space        string   `json:"space"`
	Asks         *int     `json:"asks,omitempty"`
	FewerThan5   bool     `json:"fewer_than_5,omitempty"`
	FoundNothing *float64 `json:"found_nothing_share,omitempty"`
	Helpful      *float64 `json:"helpful_share,omitempty"`
}

// UsageBySpace is a row per space the asks among traces were asked in (by name, or
// company), sorted by name. helpful are the traces whose latest feedback is up.
func UsageBySpace(traces []Trace, helpful map[string]bool) []SpaceUsage {
	type tally struct{ asks, nothing, up int }
	by := map[string]*tally{}
	for _, t := range traces {
		if t.Response.Agent == AgentSearch || t.Response.Agent == AgentLimited {
			continue
		}
		sp := strings.TrimPrefix(t.Space, "spc_")
		if sp == "" {
			sp = "company"
		}
		if by[sp] == nil {
			by[sp] = &tally{}
		}
		by[sp].asks++
		if nothing(t) {
			by[sp].nothing++
		}
		if helpful[t.ID] {
			by[sp].up++
		}
	}
	out := make([]SpaceUsage, 0, len(by))
	for sp, n := range by {
		row := SpaceUsage{Space: sp}
		if n.asks < MinUsage {
			row.FewerThan5 = true
		} else {
			asks := n.asks
			none, up := share(n.nothing, n.asks), share(n.up, n.asks)
			row.Asks, row.FoundNothing, row.Helpful = &asks, &none, &up
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Space < out[j].Space })
	return out
}

func share(n, of int) float64 { return math.Round(float64(n)/float64(of)*1000) / 1000 }

// Helpful are the traces among fbs whose latest feedback is rated up.
func Helpful(in []Feedback) map[string]bool {
	fbs := append([]Feedback(nil), in...)
	sort.SliceStable(fbs, func(i, j int) bool { return fbs[i].At.Before(fbs[j].At) })
	out := map[string]bool{}
	for _, f := range fbs {
		out[f.Trace] = f.Rating == "up"
	}
	return out
}

// ownUsage is the space's own row over the 7 days to now: fewer than 5 with none.
func ownUsage(traces []Trace, fbs []Feedback, name string, now time.Time) *SpaceUsage {
	var week []Trace
	for _, t := range traces {
		if t.Space == SpaceKey(name) && !t.At.Before(now.Add(-7*24*time.Hour)) {
			week = append(week, t)
		}
	}
	for _, row := range UsageBySpace(week, Helpful(fbs)) {
		return &row
	}
	return &SpaceUsage{Space: name, FewerThan5: true}
}
