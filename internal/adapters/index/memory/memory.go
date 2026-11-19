// Package memory is an in-process index (development and tests). The Postgres/pgvector
// adapter implements the same port for cells.
package memory

import (
	"context"
	"sort"
	"sync"

	"example.com/second-brain/internal/core/ports"
)

type entry struct {
	version int
	chunks  []ports.Chunk
}

// Index maps tenant -> source -> chunks.
type Index struct {
	mu sync.Mutex
	m  map[string]map[string]entry
}

var _ ports.Index = (*Index)(nil)

func New() *Index { return &Index{m: map[string]map[string]entry{}} }

func key(r ports.SourceRef) string { return r.Kind + "\x00" + r.ID }

func (x *Index) Indexed(_ context.Context, tenant string, ref ports.SourceRef) (int, bool, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	e, ok := x.m[tenant][key(ref)]
	return e.version, ok, nil
}

func (x *Index) Replace(_ context.Context, tenant string, ref ports.SourceRef, version int, chunks []ports.Chunk) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.m[tenant] == nil {
		x.m[tenant] = map[string]entry{}
	}
	x.m[tenant][key(ref)] = entry{version: version, chunks: append([]ports.Chunk(nil), chunks...)}
	return nil
}

func (x *Index) Delete(_ context.Context, tenant string, ref ports.SourceRef) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.m[tenant], key(ref))
	return nil
}

func (x *Index) All(_ context.Context, tenant string) ([]ports.Chunk, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	var out []ports.Chunk
	for _, e := range x.m[tenant] {
		out = append(out, e.chunks...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (x *Index) Drop(_ context.Context, tenant string) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.m, tenant)
	return nil
}
