// Package cellstore keeps the cell's own records in its Postgres database (SB-1508): the
// directory and sessions, traces and conversations, actions and receipts, steward
// proposals, access reviews, legal holds and the chat events inbox. Each record is a JSON
// document keyed by tenant, kind and id, with its time, user and space beside it for the
// queries retention and the reports need; a decision that must not race (an action's
// state, a proposal or review decided) runs in one transaction with the row locked. It is
// the durable twin of the in-memory stores, and the same contract tests run over both.
package cellstore

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

// DB is one cell's records.
type DB struct {
	Pool   *pgxpool.Pool
	Tenant string
	// Now stamps records that carry no time of their own (conversations); nil is time.Now.
	Now func() time.Time
}

// ErrDuplicate is a record whose id is already used.
var ErrDuplicate = errors.New("cellstore: duplicate id")

func (d DB) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

// Migrate creates the tables. It is idempotent.
func (d DB) Migrate(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, schema)
	return err
}

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// meta is what is kept beside a document for queries.
type meta struct {
	at          time.Time
	user, space string
}

func put(ctx context.Context, q querier, tenant, kind, id string, m meta, v any, insertOnly bool) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	sql := `INSERT INTO cell_docs (tenant, kind, id, at, usr, space, body) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if !insertOnly {
		sql += ` ON CONFLICT (tenant, kind, id) DO UPDATE SET at = EXCLUDED.at, usr = EXCLUDED.usr, space = EXCLUDED.space, body = EXCLUDED.body`
	}
	_, err = q.Exec(ctx, sql, tenant, kind, id, m.at, m.user, m.space, b)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return fmt.Errorf("%w: %s %s", ErrDuplicate, kind, id)
	}
	return err
}

func get[T any](ctx context.Context, q querier, tenant, kind, id string, lock bool) (T, bool, error) {
	var v T
	sql := `SELECT body FROM cell_docs WHERE tenant = $1 AND kind = $2 AND id = $3`
	if lock {
		sql += ` FOR UPDATE`
	}
	var b []byte
	err := q.QueryRow(ctx, sql, tenant, kind, id).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, false, nil
	}
	if err != nil {
		return v, false, err
	}
	return v, true, json.Unmarshal(b, &v)
}

// list returns the documents of a kind matching where (with args from $3), ordered.
func list[T any](ctx context.Context, q querier, tenant, kind, where, order string, args ...any) ([]T, error) {
	sql := `SELECT body FROM cell_docs WHERE tenant = $1 AND kind = $2`
	if where != "" {
		sql += ` AND ` + where
	}
	sql += ` ORDER BY ` + order
	rows, err := q.Query(ctx, sql, append([]any{tenant, kind}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// update applies f to one document with its row locked, and stores the result only if f
// returns nil. A missing document is notFound.
func update[T any](ctx context.Context, pool *pgxpool.Pool, tenant, kind, id string, notFound error, m func(T) meta, f func(*T) error) (T, error) {
	var zero T
	tx, err := pool.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	v, found, err := get[T](ctx, tx, tenant, kind, id, true)
	if err != nil {
		return zero, err
	}
	if !found {
		return zero, notFound
	}
	if err := f(&v); err != nil {
		return v, err
	}
	if err := put(ctx, tx, tenant, kind, id, m(v), v, false); err != nil {
		return zero, err
	}
	return v, tx.Commit(ctx)
}

func del(ctx context.Context, q querier, tenant, kind, id string) error {
	_, err := q.Exec(ctx, `DELETE FROM cell_docs WHERE tenant = $1 AND kind = $2 AND id = $3`, tenant, kind, id)
	return err
}

// Open connects to the cell's database and checks it answers.
func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// Lock holds a session-level advisory lock named name until unlock, so the services of one
// cell, started together, don't create the same tables at once.
func Lock(ctx context.Context, pool *pgxpool.Pool, name string) (unlock func(), err error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(hashtext($1))`, name); err != nil {
		conn.Release()
		return nil, err
	}
	return func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtext($1))`, name)
		conn.Release()
	}, nil
}
