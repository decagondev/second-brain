// Package registry is the control plane's tenant registry in Postgres (SB-105, ADR-0017):
// applied resources, cell records with custody, host routing and the apply journal.
// It holds no tenant content, only names, ids and configuration.
package registry

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

// DB is a registry backed by a pgx pool.
type DB struct{ Pool *pgxpool.Pool }

var (
	_ ports.Registry       = DB{}
	_ ports.RegistryWriter = DB{}
	_ ports.Cells          = DB{}
	_ ports.CellsWriter    = DB{}
	_ ports.Journal        = DB{}
)

// Migrate creates the registry tables if they don't exist. It is idempotent.
func (d DB) Migrate(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, schema)
	return err
}

// Applied returns the tenant's applied resources. A query error is an error, never "no cell".
func (d DB) Applied(ctx context.Context, tenant string) ([]ports.Resource, bool, error) {
	var raw []byte
	err := d.Pool.QueryRow(ctx, `SELECT resources FROM applied WHERE tenant = $1`, tenant).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("registry: %w", err)
	}
	var rs []ports.Resource
	if err := json.Unmarshal(raw, &rs); err != nil {
		return nil, false, fmt.Errorf("registry: applied record for %s is unreadable", tenant)
	}
	return rs, true, nil
}

// Put replaces the tenant's applied resources.
func (d DB) Put(ctx context.Context, tenant string, rs []ports.Resource) error {
	raw, err := json.Marshal(rs)
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `INSERT INTO applied (tenant, resources) VALUES ($1, $2)
		ON CONFLICT (tenant) DO UPDATE SET resources = EXCLUDED.resources`, tenant, raw)
	return err
}

const cellCols = `c.tenant, c.region, c.endpoint, c.custody, c.placement, c.key_custodian,
	COALESCE((SELECT array_agg(h.host ORDER BY h.ord) FROM cell_hosts h WHERE h.tenant = c.tenant), '{}')`

func (d DB) cell(ctx context.Context, where string, arg string) (ports.CellRecord, bool, error) {
	var c ports.CellRecord
	err := d.Pool.QueryRow(ctx, `SELECT `+cellCols+` FROM cells c WHERE `+where, arg).
		Scan(&c.Tenant, &c.Region, &c.Endpoint, &c.Custody, &c.Placement, &c.KeyCustodian, &c.Hosts)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.CellRecord{}, false, nil
	}
	if err != nil {
		return ports.CellRecord{}, false, fmt.Errorf("registry: %w", err)
	}
	return c, true, nil
}

// CellByHost finds the cell that serves host.
func (d DB) CellByHost(ctx context.Context, host string) (ports.CellRecord, bool, error) {
	return d.cell(ctx, `c.tenant = (SELECT tenant FROM cell_hosts WHERE host = $1)`, host)
}

// CellByTenant finds the tenant's cell.
func (d DB) CellByTenant(ctx context.Context, tenant string) (ports.CellRecord, bool, error) {
	return d.cell(ctx, `c.tenant = $1`, tenant)
}

// PutCell records c and its hosts in one transaction. A host that another cell serves
// violates the primary key, and the whole write is rolled back.
func (d DB) PutCell(ctx context.Context, c ports.CellRecord) error {
	return pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO cells (tenant, region, endpoint, custody, placement, key_custodian)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant) DO UPDATE SET region = EXCLUDED.region, endpoint = EXCLUDED.endpoint,
			custody = EXCLUDED.custody, placement = EXCLUDED.placement, key_custodian = EXCLUDED.key_custodian`,
			c.Tenant, c.Region, c.Endpoint, c.Custody, c.Placement, c.KeyCustodian); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM cell_hosts WHERE tenant = $1`, c.Tenant); err != nil {
			return err
		}
		for i, h := range c.Hosts {
			if _, err := tx.Exec(ctx, `INSERT INTO cell_hosts (host, tenant, ord) VALUES ($1, $2, $3)`, h, c.Tenant, i); err != nil {
				return fmt.Errorf("registry: host %s: %w", h, err)
			}
		}
		return nil
	})
}

// Record appends one apply step to the journal.
func (d DB) Record(ctx context.Context, e ports.JournalEntry) error {
	_, err := d.Pool.Exec(ctx, `INSERT INTO journal (tenant, apply_id, step, op, kind, name, phase)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, e.Tenant, e.ApplyID, e.Step, e.Op, e.Kind, e.Name, e.Phase)
	return err
}
