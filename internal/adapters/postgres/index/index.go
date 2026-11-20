// Package index is a cell's chunk index in Postgres with pgvector (SB-301, the user's
// choice). Each tenant has its own schema, idx_TENANT; an Index value is bound to one
// tenant and refuses any other scope. The prefilter is part of every query (WHERE acl &&
// tokens), never a post-filter.
package index

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Index is one tenant's index.
type Index struct {
	Pool   *pgxpool.Pool
	Tenant string
	Dim    int // embedding dimensions
}

var (
	_ ports.Index     = Index{}
	_ ports.Retriever = Index{}

	tenantName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)
)

// ErrOtherTenant is any use with another tenant's name or scope.
var ErrOtherTenant = errors.New("ETENANT: this index serves another tenant")

func (x Index) schema() (string, error) {
	if !tenantName.MatchString(x.Tenant) {
		return "", fmt.Errorf("index: bad tenant %q", x.Tenant)
	}
	return pgx.Identifier{"idx_" + strings.ReplaceAll(x.Tenant, "-", "_")}.Sanitize(), nil
}

func (x Index) own(tenant string) error {
	if tenant != x.Tenant {
		return ErrOtherTenant
	}
	return nil
}

// Migrate creates the vector extension, the tenant's schema and its tables. Idempotent.
func (x Index) Migrate(ctx context.Context) error {
	sc, err := x.schema()
	if err != nil {
		return err
	}
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		`CREATE SCHEMA IF NOT EXISTS ` + sc,
		`CREATE TABLE IF NOT EXISTS ` + sc + `.docs (kind text NOT NULL, source_id text NOT NULL, version bigint NOT NULL, PRIMARY KEY (kind, source_id))`,
		`CREATE TABLE IF NOT EXISTS ` + sc + `.chunks (
			id text PRIMARY KEY, tenant text NOT NULL, cite text NOT NULL, kind text NOT NULL, source_id text NOT NULL,
			version bigint NOT NULL, span_start int NOT NULL, span_end int NOT NULL, body text NOT NULL, sha256 text NOT NULL,
			flags text[] NOT NULL DEFAULT '{}', acl text[] NOT NULL DEFAULT '{}', release text NOT NULL DEFAULT '',
			embedding vector(` + strconv.Itoa(x.Dim) + `),
			tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', body)) STORED)`,
		// Spaces (SB-1005): label, holders and the routing fingerprint; added in place.
		`ALTER TABLE ` + sc + `.chunks ADD COLUMN IF NOT EXISTS label text NOT NULL DEFAULT ''`,
		`ALTER TABLE ` + sc + `.chunks ADD COLUMN IF NOT EXISTS spaces text[] NOT NULL DEFAULT '{}'`,
		`ALTER TABLE ` + sc + `.chunks ADD COLUMN IF NOT EXISTS routing text NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS chunks_acl ON ` + sc + `.chunks USING gin (acl)`,
		`CREATE INDEX IF NOT EXISTS chunks_tsv ON ` + sc + `.chunks USING gin (tsv)`,
		`CREATE INDEX IF NOT EXISTS chunks_src ON ` + sc + `.chunks (kind, source_id)`,
	}
	for _, s := range stmts {
		if _, err := x.Pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("index: migrate: %w", err)
		}
	}
	return nil
}

func vec(v []float32) *string {
	if v == nil {
		return nil
	}
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'g', -1, 32)
	}
	s := "[" + strings.Join(parts, ",") + "]"
	return &s
}

func (x Index) Indexed(ctx context.Context, tenant string, ref ports.SourceRef) (int, bool, error) {
	if err := x.own(tenant); err != nil {
		return 0, false, err
	}
	sc, err := x.schema()
	if err != nil {
		return 0, false, err
	}
	var v int
	err = x.Pool.QueryRow(ctx, `SELECT version FROM `+sc+`.docs WHERE kind = $1 AND source_id = $2`, ref.Kind, ref.ID).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return v, err == nil, err
}

// Replace swaps a document's chunks in one transaction.
func (x Index) Replace(ctx context.Context, tenant string, ref ports.SourceRef, version int, chunks []ports.Chunk) error {
	if err := x.own(tenant); err != nil {
		return err
	}
	sc, err := x.schema()
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, x.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM `+sc+`.chunks WHERE kind = $1 AND source_id = $2`, ref.Kind, ref.ID); err != nil {
			return err
		}
		for _, c := range chunks {
			if c.Tenant != x.Tenant {
				return ErrOtherTenant
			}
			if _, err := tx.Exec(ctx, `INSERT INTO `+sc+`.chunks (id, tenant, cite, kind, source_id, version, span_start, span_end,
				body, sha256, flags, acl, release, embedding, label, spaces, routing) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::vector,$15,$16,$17)`,
				c.ID, c.Tenant, c.Cite, c.Kind, c.SourceID, c.Version, c.Start, c.End, c.Text, c.ContentSHA256,
				nonNil(c.Flags), nonNil(c.ACLPrefilter), c.Release, vec(c.Vector), c.Label, nonNil(c.Spaces), c.Routing); err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `INSERT INTO `+sc+`.docs (kind, source_id, version) VALUES ($1,$2,$3)
			ON CONFLICT (kind, source_id) DO UPDATE SET version = EXCLUDED.version`, ref.Kind, ref.ID, version)
		return err
	})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (x Index) Delete(ctx context.Context, tenant string, ref ports.SourceRef) error {
	if err := x.own(tenant); err != nil {
		return err
	}
	sc, err := x.schema()
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, x.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM `+sc+`.chunks WHERE kind = $1 AND source_id = $2`, ref.Kind, ref.ID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM `+sc+`.docs WHERE kind = $1 AND source_id = $2`, ref.Kind, ref.ID)
		return err
	})
}

const cols = `id, tenant, cite, kind, source_id, version, span_start, span_end, body, sha256, flags, acl, release, label, spaces, routing`

func scan(rows pgx.Rows) (ports.Chunk, error) {
	var c ports.Chunk
	err := rows.Scan(&c.ID, &c.Tenant, &c.Cite, &c.Kind, &c.SourceID, &c.Version, &c.Start, &c.End, &c.Text,
		&c.ContentSHA256, &c.Flags, &c.ACLPrefilter, &c.Release, &c.Label, &c.Spaces, &c.Routing)
	if len(c.Flags) == 0 {
		c.Flags = nil
	}
	if len(c.Spaces) == 0 {
		c.Spaces = nil
	}
	return c, err
}

func (x Index) All(ctx context.Context, tenant string) ([]ports.Chunk, error) {
	if err := x.own(tenant); err != nil {
		return nil, err
	}
	sc, err := x.schema()
	if err != nil {
		return nil, err
	}
	rows, err := x.Pool.Query(ctx, `SELECT `+cols+` FROM `+sc+`.chunks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.Chunk
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Drop empties the tenant's index (the rebuild drill, LAW-13).
func (x Index) Drop(ctx context.Context, tenant string) error {
	if err := x.own(tenant); err != nil {
		return err
	}
	sc, err := x.schema()
	if err != nil {
		return err
	}
	_, err = x.Pool.Exec(ctx, `TRUNCATE `+sc+`.chunks, `+sc+`.docs`)
	return err
}

// Search ranks by cosine distance and by full-text rank, each with the prefilter in the
// WHERE clause, and returns the union with both ranks.
func (x Index) Search(ctx context.Context, s tenancy.Scope, tokens []string, q ports.Query) ([]ports.Candidate, error) {
	if err := s.Check(); err != nil {
		return nil, err
	}
	if err := x.own(s.Tenant()); err != nil {
		return nil, err
	}
	sc, err := x.schema()
	if err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	byID := map[string]*ports.Candidate{}
	var order []string
	collect := func(sql string, args []any, setRank func(*ports.Candidate, int)) error {
		rows, err := x.Pool.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		rank := 0
		for rows.Next() {
			c, err := scan(rows)
			if err != nil {
				return err
			}
			rank++
			cand, ok := byID[c.ID]
			if !ok {
				cand = &ports.Candidate{Chunk: c}
				byID[c.ID] = cand
				order = append(order, c.ID)
			}
			setRank(cand, rank)
		}
		return rows.Err()
	}
	if q.Vector != nil {
		if err := collect(`SELECT `+cols+` FROM `+sc+`.chunks WHERE tenant = $1 AND acl && $2 AND embedding IS NOT NULL
			ORDER BY embedding <=> $3::vector, id LIMIT $4`, []any{x.Tenant, tokens, vec(q.Vector), limit},
			func(c *ports.Candidate, r int) { c.VectorRank = r }); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(q.Text) != "" {
		if err := collect(`SELECT `+cols+` FROM `+sc+`.chunks WHERE tenant = $1 AND acl && $2 AND tsv @@ plainto_tsquery('simple', $3)
			ORDER BY ts_rank(tsv, plainto_tsquery('simple', $3)) DESC, id LIMIT $4`, []any{x.Tenant, tokens, q.Text, limit},
			func(c *ports.Candidate, r int) { c.KeywordRank = r }); err != nil {
			return nil, err
		}
	}
	out := make([]ports.Candidate, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}
