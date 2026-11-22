// Package memory is an in-process index (development and tests). The Postgres/pgvector
// adapter implements the same port for cells.
package memory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

type entry struct {
	version int
	chunks  []ports.Chunk
}

// Index maps tenant -> source -> chunks.
type Index struct {
	mu    sync.Mutex
	m     map[string]map[string]entry
	bound string // if set, only this tenant's scopes may search
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
	chunks = append([]ports.Chunk(nil), chunks...)
	if old, ok := x.m[tenant][key(ref)]; ok && old.version == version {
		keepSeen(old.chunks, chunks) // the same version again: it was first seen earlier
	}
	x.m[tenant][key(ref)] = entry{version: version, chunks: chunks}
	return nil
}

// keepSeen gives new chunks of a version the earliest time the index saw that version.
func keepSeen(old, chunks []ports.Chunk) {
	var first time.Time
	for _, c := range old {
		if !c.Seen.IsZero() && (first.IsZero() || c.Seen.Before(first)) {
			first = c.Seen
		}
	}
	if first.IsZero() {
		return
	}
	for i := range chunks {
		chunks[i].Seen = first
	}
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

// ErrOtherTenant is a search with another tenant's scope on a bound index.
var ErrOtherTenant = errors.New("ETENANT: this index serves another tenant")

// Bind restricts the index to one tenant's scopes, as a cell's index is.
func (x *Index) Bind(tenant string) *Index { x.bound = tenant; return x }

var _ ports.Retriever = (*Index)(nil)

// Search ranks the scope tenant's chunks that pass the prefilter, by cosine similarity
// and by keyword overlap, each up to q.Limit.
func (x *Index) Search(_ context.Context, s tenancy.Scope, tokens []string, q ports.Query) ([]ports.Candidate, error) {
	if err := s.Check(); err != nil {
		return nil, err
	}
	if x.bound != "" && s.Tenant() != x.bound {
		return nil, ErrOtherTenant
	}
	tok := map[string]bool{}
	for _, t := range tokens {
		tok[t] = true
	}
	x.mu.Lock()
	var pool []ports.Chunk
	for _, e := range x.m[s.Tenant()] {
		for _, c := range e.chunks {
			for _, a := range c.ACLPrefilter {
				if tok[a] {
					pool = append(pool, c)
					break
				}
			}
		}
	}
	x.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	words := strings.Fields(strings.ToLower(q.Text))
	type scored struct {
		i int
		s float64
	}
	rank := func(score func(c ports.Chunk) float64) map[int]int {
		var ss []scored
		for i, c := range pool {
			if v := score(c); v > 0 {
				ss = append(ss, scored{i, v})
			}
		}
		sort.SliceStable(ss, func(a, b int) bool {
			if ss[a].s != ss[b].s {
				return ss[a].s > ss[b].s
			}
			return pool[ss[a].i].ID < pool[ss[b].i].ID
		})
		out := map[int]int{}
		for r, x := range ss {
			if r >= limit {
				break
			}
			out[x.i] = r + 1
		}
		return out
	}
	vec := rank(func(c ports.Chunk) float64 {
		var d float64
		for i := range c.Vector {
			if i < len(q.Vector) {
				d += float64(c.Vector[i]) * float64(q.Vector[i])
			}
		}
		if len(q.Vector) == 0 {
			return 1e-9 // no query vector: every permitted chunk ties
		}
		return d
	})
	kw := rank(func(c ports.Chunk) float64 {
		t := strings.ToLower(c.Text)
		n := 0.0
		for _, w := range words {
			if strings.Contains(t, w) {
				n++
			}
		}
		return n
	})
	var out []ports.Candidate
	for i, c := range pool {
		vr, kr := vec[i], kw[i]
		if vr > 0 || kr > 0 {
			out = append(out, ports.Candidate{Chunk: c, VectorRank: vr, KeywordRank: kr})
		}
	}
	return out, nil
}
