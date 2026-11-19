// Package audit is a cell's audit log in Postgres (sb-audit(5), Storage; SB-401). Each
// tenant has its own schema, audit_TENANT. The rows table refuses UPDATE, DELETE and
// TRUNCATE with a trigger, whatever the role, and the writer role holds only INSERT and
// SELECT. Appends serialise on a per-tenant advisory lock, so seq stays gap-free.
package audit

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/second-brain/internal/core/audit"
)

// Store is one tenant's audit rows.
type Store struct {
	Pool   *pgxpool.Pool
	Tenant string
}

var (
	_          audit.Store = Store{}
	tenantName             = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)
	roleName               = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
)

func (s Store) schema() (string, error) {
	if !tenantName.MatchString(s.Tenant) {
		return "", fmt.Errorf("audit: bad tenant %q", s.Tenant)
	}
	return pgx.Identifier{"audit_" + strings.ReplaceAll(s.Tenant, "-", "_")}.Sanitize(), nil
}

// Migrate creates the schema, the rows table and the append-only trigger. Idempotent.
func (s Store) Migrate(ctx context.Context) error {
	sc, err := s.schema()
	if err != nil {
		return err
	}
	stmts := []string{
		`CREATE SCHEMA IF NOT EXISTS ` + sc,
		`CREATE TABLE IF NOT EXISTS ` + sc + `.rows (
			seq bigint PRIMARY KEY CHECK (seq > 0), id text NOT NULL UNIQUE, at text NOT NULL, tenant text NOT NULL,
			actor text NOT NULL, kind text NOT NULL, subject text NOT NULL, state text NOT NULL DEFAULT '',
			approval text NOT NULL DEFAULT '', payload_sha256 text NOT NULL DEFAULT '', receipt text NOT NULL DEFAULT '',
			release text NOT NULL, reason text NOT NULL, prev text NOT NULL, hash text NOT NULL UNIQUE)`,
		`CREATE OR REPLACE FUNCTION ` + sc + `.append_only() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN RAISE EXCEPTION 'audit rows are append-only (LAW-11)'; END $$`,
		`DROP TRIGGER IF EXISTS append_only ON ` + sc + `.rows`,
		`CREATE TRIGGER append_only BEFORE UPDATE OR DELETE ON ` + sc + `.rows FOR EACH ROW EXECUTE FUNCTION ` + sc + `.append_only()`,
		`DROP TRIGGER IF EXISTS append_only_truncate ON ` + sc + `.rows`,
		`CREATE TRIGGER append_only_truncate BEFORE TRUNCATE ON ` + sc + `.rows FOR EACH STATEMENT EXECUTE FUNCTION ` + sc + `.append_only()`,
		`REVOKE ALL ON ` + sc + `.rows FROM PUBLIC`,
	}
	for _, q := range stmts {
		if _, err := s.Pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("audit: migrate: %w", err)
		}
	}
	return nil
}

// GrantWriter gives an existing role exactly INSERT and SELECT on the rows table.
func (s Store) GrantWriter(ctx context.Context, role string) error {
	sc, err := s.schema()
	if err != nil {
		return err
	}
	if !roleName.MatchString(role) {
		return fmt.Errorf("audit: bad role %q", role)
	}
	r := pgx.Identifier{role}.Sanitize()
	for _, q := range []string{
		`GRANT USAGE ON SCHEMA ` + sc + ` TO ` + r,
		`REVOKE ALL ON ` + sc + `.rows FROM ` + r,
		`GRANT INSERT, SELECT ON ` + sc + `.rows TO ` + r,
	} {
		if _, err := s.Pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("audit: grant: %w", err)
		}
	}
	return nil
}

// Append implements audit.Store.
func (s Store) Append(ctx context.Context, build func(int64, string) (audit.Row, error)) (audit.Row, error) {
	sc, err := s.schema()
	if err != nil {
		return audit.Row{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return audit.Row{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "audit:"+s.Tenant); err != nil {
		return audit.Row{}, err
	}
	seq, hash := int64(0), audit.Zero
	err = tx.QueryRow(ctx, `SELECT seq, hash FROM `+sc+`.rows ORDER BY seq DESC LIMIT 1`).Scan(&seq, &hash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return audit.Row{}, err
	}
	r, err := build(seq, hash)
	if err != nil {
		return audit.Row{}, err
	}
	if r.Tenant != s.Tenant {
		return audit.Row{}, fmt.Errorf("audit: row for tenant %q in %q's log", r.Tenant, s.Tenant)
	}
	_, err = tx.Exec(ctx, `INSERT INTO `+sc+`.rows (seq, id, at, tenant, actor, kind, subject, state, approval, payload_sha256, receipt, release, reason, prev, hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		r.Seq, r.ID, r.At, r.Tenant, r.Actor, r.Kind, r.Subject, r.State, r.Approval, r.PayloadSHA256, r.Receipt, r.Release, r.Reason, r.Prev, r.Hash)
	if err != nil {
		return audit.Row{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return audit.Row{}, err
	}
	return r, nil
}

// Range implements audit.Store.
func (s Store) Range(ctx context.Context, from int64, limit int) ([]audit.Row, error) {
	sc, err := s.schema()
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT seq, id, at, tenant, actor, kind, subject, state, approval, payload_sha256, receipt, release, reason, prev, hash
		FROM `+sc+`.rows WHERE seq >= $1 ORDER BY seq LIMIT $2`, from, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []audit.Row
	for rows.Next() {
		var r audit.Row
		if err := rows.Scan(&r.Seq, &r.ID, &r.At, &r.Tenant, &r.Actor, &r.Kind, &r.Subject, &r.State, &r.Approval,
			&r.PayloadSHA256, &r.Receipt, &r.Release, &r.Reason, &r.Prev, &r.Hash); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
