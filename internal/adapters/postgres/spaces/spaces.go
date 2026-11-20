// Package spaces is a cell's space registry in the cell's Postgres database (SB-1002).
// It holds names and configuration only, never content.
package spaces

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/second-brain/internal/core/ports"
)

//go:embed schema.sql
var schema string

// DB is a space registry backed by a pgx pool.
type DB struct{ Pool *pgxpool.Pool }

var _ ports.SpaceStore = DB{}

// Migrate creates the table if it doesn't exist. It is idempotent.
func (d DB) Migrate(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, schema)
	return err
}

// PutSpace creates or replaces a space.
func (d DB) PutSpace(ctx context.Context, s ports.Space) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `INSERT INTO spaces (tenant, id, space) VALUES ($1, $2, $3)
		ON CONFLICT (tenant, id) DO UPDATE SET space = EXCLUDED.space`, s.Tenant, s.ID, raw)
	return err
}

// DeleteSpace removes a space; deleting one that isn't there is not an error.
func (d DB) DeleteSpace(ctx context.Context, tenant, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM spaces WHERE tenant = $1 AND id = $2`, tenant, id)
	return err
}

// Space finds one of the tenant's spaces. A query error is an error, never "not found".
func (d DB) Space(ctx context.Context, tenant, id string) (ports.Space, bool, error) {
	var raw []byte
	err := d.Pool.QueryRow(ctx, `SELECT space FROM spaces WHERE tenant = $1 AND id = $2`, tenant, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.Space{}, false, nil
	}
	if err != nil {
		return ports.Space{}, false, fmt.Errorf("spaces: %w", err)
	}
	var s ports.Space
	if err := json.Unmarshal(raw, &s); err != nil || s.Tenant != tenant || s.ID != id {
		return ports.Space{}, false, fmt.Errorf("spaces: the record for %s is unreadable", id)
	}
	return s, true, nil
}

// Spaces lists the tenant's spaces by id.
func (d DB) Spaces(ctx context.Context, tenant string) ([]ports.Space, error) {
	rows, err := d.Pool.Query(ctx, `SELECT space FROM spaces WHERE tenant = $1 ORDER BY id`, tenant)
	if err != nil {
		return nil, fmt.Errorf("spaces: %w", err)
	}
	defer rows.Close()
	var out []ports.Space
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("spaces: %w", err)
		}
		var s ports.Space
		if err := json.Unmarshal(raw, &s); err != nil || s.Tenant != tenant {
			return nil, errors.New("spaces: a space record is unreadable")
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
