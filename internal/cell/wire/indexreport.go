package wire

import (
	"context"
	"errors"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/route"
)

// indexReports serves the operator's reports on the index (sb-admin(3); SB-2903).
type indexReports struct{ c *Cell }

// newest is each document of a kind once, at its newest version.
func (x indexReports) newest(ctx context.Context, kind string) (map[string]ports.Chunk, error) {
	lk, ok := x.c.Index.(ports.DocLookup)
	if !ok {
		return nil, errors.New("wire: the index can't look up documents")
	}
	chunks, err := lk.Docs(ctx, x.c.Config.Tenant, ports.DocQuery{Under: []ports.SourceRef{{Kind: kind}}})
	if err != nil {
		return nil, err
	}
	out := map[string]ports.Chunk{}
	for _, ch := range chunks {
		if ch.Kind != kind {
			continue
		}
		if k, ok := out[ch.SourceID]; !ok || ch.Version > k.Version {
			out[ch.SourceID] = ch
		}
	}
	return out, nil
}

// ScopeReport counts what each space's scope patterns match among the index's documents.
func (x indexReports) ScopeReport(ctx context.Context) ([]route.PatternCount, error) {
	spaces, err := x.c.Spaces.Spaces(ctx, x.c.Config.Tenant)
	if err != nil {
		return nil, err
	}
	locs := map[string][]string{}
	for _, sp := range spaces {
		for kind := range sp.Scope {
			if _, done := locs[kind]; done {
				continue
			}
			docs, err := x.newest(ctx, kind)
			if err != nil {
				return nil, err
			}
			locs[kind] = []string{}
			for _, d := range docs {
				locs[kind] = append(locs[kind], d.Location)
			}
		}
	}
	return route.Report(spaces, locs), nil
}
