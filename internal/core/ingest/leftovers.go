package ingest

import (
	"context"
	"sort"

	"example.com/second-brain/internal/core/ports"
)

// Leftovers are the index's chunks of the pipeline's kind that this connector no longer lists (gone), and,
// in a cell with spaces, the chunks no space holds (unheld), each document once at its
// newest version (sb-admin(3), index leftovers; SB-2905). The listing is the connector's
// full one, as a rebuild reads it; one that fails is an error, never "everything is gone".
func (p Pipeline) Leftovers(ctx context.Context) (gone, unheld []ports.Chunk, err error) {
	kind := p.Kind
	set, _, err := p.Connector.Changes(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	listed := map[string]bool{}
	for _, ch := range set {
		if ch.Ref.Kind == kind || ch.Ref.Kind == "" {
			listed[ch.Ref.ID] = !ch.Deleted
		}
	}
	all, err := p.Index.All(ctx, p.Tenant)
	if err != nil {
		return nil, nil, err
	}
	withSpaces := false
	if p.Spaces != nil {
		sps, err := p.Spaces(ctx)
		if err != nil {
			return nil, nil, err
		}
		withSpaces = len(sps) > 0
	}
	newest := map[string]ports.Chunk{}
	for _, c := range all {
		if c.Kind != kind {
			continue
		}
		if n, ok := newest[c.SourceID]; !ok || c.Version > n.Version || c.Version == n.Version && c.ID < n.ID {
			newest[c.SourceID] = c
		}
	}
	ids := make([]string, 0, len(newest))
	for id := range newest {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := newest[id]
		switch {
		case !listed[id]:
			gone = append(gone, c)
		case withSpaces && len(c.Spaces) == 0:
			unheld = append(unheld, c)
		}
	}
	return gone, unheld, nil
}
