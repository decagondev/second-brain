package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
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

// pipeline is the ingest pipeline's part that reads leftovers, for one connector kind.
func (x indexReports) pipeline(kind string) ingest.Pipeline {
	c := x.c
	return ingest.Pipeline{Tenant: c.Config.Tenant, Connector: c.Parts().Connectors[kind], Index: c.Index, Kind: kind,
		Spaces: func(ctx context.Context) ([]ports.Space, error) { return c.Spaces.Spaces(ctx, c.Config.Tenant) }}
}

// Leftovers lists, by cite, the index's documents their connector no longer lists and the
// chunks no space holds (SB-2905). A connector that can't be listed fails it.
func (x indexReports) Leftovers(ctx context.Context) (gone, unheld []ports.Chunk, err error) {
	kinds := make([]string, 0, len(x.c.Parts().Connectors))
	for k := range x.c.Parts().Connectors {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		g, u, err := x.pipeline(k).Leftovers(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("wire: %s can't be listed: %w", k, err)
		}
		gone, unheld = append(gone, g...), append(unheld, u...)
	}
	return gone, unheld, nil
}

// Prune removes the documents their connector no longer lists, with an audit row before
// and after; nothing a source still lists is touched (LAW-13: the index is derived).
func (x indexReports) Prune(ctx context.Context, actor, reason string) (int, error) {
	c := x.c
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "config", Subject: "index:prune", State: "requested", Reason: reason}); err != nil {
		return 0, err
	}
	gone, _, err := x.Leftovers(ctx)
	if err != nil {
		return 0, err
	}
	for _, ch := range gone {
		if err := c.Index.Delete(ctx, c.Config.Tenant, ports.SourceRef{Kind: ch.Kind, ID: ch.SourceID}); err != nil {
			return 0, err
		}
	}
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "config", Subject: "index:prune", State: "pruned",
		Reason: fmt.Sprintf("gone=%d", len(gone))}); err != nil {
		return len(gone), err
	}
	return len(gone), nil
}

// IngestPipeline is the normal ingest pipeline for one connector kind, with the cell's
// counters, as a sync uses it and a refresh re-reads one document through it.
func (c *Cell) IngestPipeline(kind string, now func() time.Time) ingest.Pipeline {
	m := c.Metrics
	return ingest.Pipeline{Tenant: c.Config.Tenant, Release: c.Release, Connector: c.Parts().Connectors[kind], Embedder: c.Parts().Agent.Embedder,
		Index: c.Index, Kind: kind, Now: now,
		Spaces: func(ctx context.Context) ([]ports.Space, error) { return c.Spaces.Spaces(ctx, c.Config.Tenant) },
		SecretDropped: func(k, pattern string) {
			m.Add("sb_ingest_secrets_dropped_total", "Chunks dropped for holding a secret (SB-1103).", map[string]string{"connector": k, "pattern": pattern}, 1)
		},
		FieldsDropped: func(k string, names []string) {
			for _, n := range names {
				m.Add("sb_ingest_fields_dropped_total", "Structured fields dropped at ingest, by name (SB-1106).", map[string]string{"connector": k, "field": n}, 1)
			}
		},
		BadReviewBy: func(k string) {
			m.Add("sb_ingest_review_by_ignored_total", "Review-by dates ingest couldn't read and ignored (SB-2705).", map[string]string{"connector": k}, 1)
		},
		RerouteDropped: func(k string) {
			m.Add("sb_ingest_reroute_dropped_total", "Documents Reroute took out because they couldn't be fetched.", map[string]string{"connector": k}, 1)
		},
		Sharing: func(ctx context.Context, ch ingest.SharingChange) { // SB-5403
			if err := c.Parts().Agent.RecordSharing(ctx, ch); err != nil {
				m.Add("sb_sharing_unrecorded_total", "Sharing changes seen at sync that couldn't be kept (SB-5403).", map[string]string{"connector": ch.Ref.Kind}, 1)
			}
		}}
}

// Refresh re-reads one document from its source now (SB-3202), with an audit row before
// and after; the first row is written before anything is read.
func (x indexReports) Refresh(ctx context.Context, actor, reason string, ref ports.SourceRef) (ingest.Refreshed, error) {
	c := x.c
	if _, ok := c.Parts().Connectors[ref.Kind]; !ok {
		return ingest.Refreshed{}, ingest.ErrNoConnector
	}
	subject := ref.Kind + ":" + ref.ID
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "config", Subject: subject, State: "requested", Reason: "index refresh: " + reason}); err != nil {
		return ingest.Refreshed{}, err
	}
	res, err := c.IngestPipeline(ref.Kind, time.Now).Refresh(ctx, ref)
	if err != nil {
		return res, err
	}
	state := "refreshed"
	if res.Removed {
		state = "removed"
	}
	_, err = c.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "config", Subject: subject, State: state, Reason: fmt.Sprintf("version=%d changed=%t", res.Version, res.Changed)})
	return res, err
}

// leftoverCount is the nightly count health reads.
type leftoverCount struct {
	At     time.Time `json:"at"`
	Gone   int       `json:"gone"`
	Unheld int       `json:"unheld"`
}

func leftoverPath(state string) string { return filepath.Join(state, "index", "leftovers.json") }

// CountLeftovers counts the index's leftovers for health (the nightly retention job).
func (c *Cell) CountLeftovers(ctx context.Context, now time.Time) error {
	gone, unheld, err := indexReports{c}.Leftovers(ctx)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(leftoverCount{At: now.UTC(), Gone: len(gone), Unheld: len(unheld)})
	p := leftoverPath(c.Config.State)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p+".tmp", b, 0o600); err != nil {
		return err
	}
	return os.Rename(p+".tmp", p)
}

// lastLeftovers reads the last count; found is false when there is none or it can't be read.
func lastLeftovers(state string) (leftoverCount, bool) {
	b, err := os.ReadFile(leftoverPath(state))
	if err != nil {
		return leftoverCount{}, false
	}
	var l leftoverCount
	if json.Unmarshal(b, &l) != nil || l.At.IsZero() {
		return leftoverCount{}, false
	}
	return l, true
}
