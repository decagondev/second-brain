package ingest

import (
	"context"
	"sort"
	"strings"
	"time"
)

// DrillEvery is how often each cell runs the rebuild drill (sb-ingest(7), Rebuild).
const DrillEvery = 7 * 24 * time.Hour

// DrillReport is one drill's outcome. Equal false is a LAW-13 violation (Sev-1).
type DrillReport struct {
	Tenant        string
	Before, After int // number of (cite, chunk, ACL) entries
	Equal         bool
	Took          time.Duration
	Missing       []string // in the index before, not after (ids only, never content)
	Extra         []string // after, not before
}

// Drill snapshots the index, drops it, rebuilds it from the sources and compares the
// (cite, chunk id, ACL) sets.
func Drill(ctx context.Context, p Pipeline, now func() time.Time) (DrillReport, error) {
	before, err := snapshot(ctx, p)
	if err != nil {
		return DrillReport{}, err
	}
	start := now()
	if err := p.Rebuild(ctx); err != nil {
		return DrillReport{}, err
	}
	took := now().Sub(start)
	after, err := snapshot(ctx, p)
	if err != nil {
		return DrillReport{}, err
	}
	r := DrillReport{Tenant: p.Tenant, Before: len(before), After: len(after), Took: took}
	for k := range before {
		if !after[k] {
			r.Missing = append(r.Missing, k)
		}
	}
	for k := range after {
		if !before[k] {
			r.Extra = append(r.Extra, k)
		}
	}
	sort.Strings(r.Missing)
	sort.Strings(r.Extra)
	r.Equal = len(r.Missing) == 0 && len(r.Extra) == 0
	return r, nil
}

func snapshot(ctx context.Context, p Pipeline) (map[string]bool, error) {
	cs, err := p.Index.All(ctx, p.Tenant)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(cs))
	for _, c := range cs {
		out[c.Cite+"|"+c.ID+"|"+strings.Join(c.ACLPrefilter, ",")] = true
	}
	return out, nil
}
