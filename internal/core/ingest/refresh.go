package ingest

import (
	"context"
	"errors"
	"fmt"

	"example.com/second-brain/internal/core/ports"
)

// ErrNoConnector is a refresh of a kind the cell has no connector for.
var ErrNoConnector = errors.New("ingest: no connector of that kind")

// Refreshed is what a refresh of one document did (SB-3202).
type Refreshed struct {
	Version int  `json:"version"`
	Changed bool `json:"changed"`
	Removed bool `json:"removed"`
}

// Refresh re-reads one document from its source now, through Ingest, as a sync would. A
// document the source no longer has is taken out of the index as a sync takes it out; a
// fetch that fails any other way changes nothing.
func (p Pipeline) Refresh(ctx context.Context, ref ports.SourceRef) (Refreshed, error) {
	before, had, err := p.Index.Indexed(ctx, p.Tenant, ref)
	if err != nil {
		return Refreshed{}, err
	}
	if err := p.Ingest(ctx, ref, true); err != nil {
		if !Gone(err) {
			return Refreshed{}, err
		}
		if err := p.Index.Delete(ctx, p.Tenant, ref); err != nil {
			return Refreshed{}, err
		}
		return Refreshed{Removed: true, Changed: had}, nil
	}
	after, found, err := p.Index.Indexed(ctx, p.Tenant, ref)
	if err != nil {
		return Refreshed{}, err
	}
	if !found {
		return Refreshed{}, fmt.Errorf("ingest: %s:%s wasn't indexed", ref.Kind, ref.ID) // e.g. a structured source without its fields
	}
	return Refreshed{Version: after, Changed: !had || after != before}, nil
}
