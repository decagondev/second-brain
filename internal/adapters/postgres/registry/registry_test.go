package registry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/second-brain/internal/core/registrytest"
)

// open connects to SB_TEST_POSTGRES_DSN in a fresh schema, or skips. CI's postgres job
// sets the variable; `make check` stays offline.
func open(t *testing.T) DB {
	t.Helper()
	dsn := os.Getenv("SB_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SB_TEST_POSTGRES_DSN not set")
	}
	ctx := context.Background()
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	schemaName := "t_" + hex.EncodeToString(b)
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	d := DB{Pool: pool}
	if err := d.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := d.Migrate(ctx); err != nil {
		t.Fatalf("Migrate must be idempotent: %v", err)
	}
	return d
}

func TestContract(t *testing.T) {
	registrytest.Contract(t, open(t))
}

// A registry that can't be reached is an error, never "no cell".
func TestUnreachableIsAnError(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	d := DB{Pool: pool}
	if _, found, err := d.Applied(context.Background(), "acme-logistics"); err == nil || found {
		t.Fatalf("found=%v err=%v; want an error", found, err)
	}
	if _, found, err := d.CellByHost(context.Background(), "acme-logistics.sb.example"); err == nil || found {
		t.Fatalf("found=%v err=%v; want an error", found, err)
	}
}
