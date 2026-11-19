package audit

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/second-brain/internal/core/audit"
)

func open(t *testing.T) (Store, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("SB_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SB_TEST_POSTGRES_DSN not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	s := Store{Pool: pool, Tenant: "t-" + hex.EncodeToString(b)}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(context.Background()); err != nil { // idempotent
		t.Fatal(err)
	}
	sc, _ := s.schema()
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DROP SCHEMA "+sc+" CASCADE") })
	return s, pool
}

func TestChainInPostgres(t *testing.T) {
	s, pool := open(t)
	ctx := context.Background()
	log := &audit.Log{Tenant: s.Tenant, Release: func() string { return "rel_x" }, Store: s, Now: time.Now}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ { // concurrent writers from two Log values still give a gap-free chain
		wg.Add(1)
		l := log
		if i%2 == 1 {
			l = &audit.Log{Tenant: s.Tenant, Release: func() string { return "rel_x" }, Store: s, Now: time.Now}
		}
		go func() {
			defer wg.Done()
			if _, err := l.Write(ctx, audit.Event{Actor: "usr_1", Kind: "tool_call", Subject: "tc_1", Reason: "ok"}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	signer, _ := audit.NewSigner(s.Tenant, bytes.Repeat([]byte{3}, 32))
	var exp bytes.Buffer
	if n, err := audit.Export(ctx, s, signer, 1, time.Now(), &exp); err != nil || n != 20 {
		t.Fatal(n, err)
	}
	if rep, err := audit.Verify(bytes.NewReader(exp.Bytes()), s.Tenant, signer.Key()); err != nil || rep.LastSeq != 20 {
		t.Fatalf("%+v %v", rep, err)
	}

	sc, _ := s.schema()
	for _, q := range []string{
		`UPDATE ` + sc + `.rows SET reason = 'x' WHERE seq = 1`,
		`DELETE FROM ` + sc + `.rows WHERE seq = 1`,
		`TRUNCATE ` + sc + `.rows`,
	} {
		if _, err := pool.Exec(ctx, q); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s: got %v, want the append-only refusal", q, err)
		}
	}
}

func TestWriterRoleCanOnlyInsertAndSelect(t *testing.T) {
	s, pool := open(t)
	ctx := context.Background()
	role := "sb_audit_test_" + strings.ReplaceAll(s.Tenant[2:], "-", "")
	if _, err := pool.Exec(ctx, `CREATE ROLE `+role); err != nil {
		t.Skipf("cannot create roles here: %v", err)
	}
	t.Cleanup(func() {
		sc, _ := s.schema()
		_, _ = pool.Exec(ctx, `REVOKE ALL ON `+sc+`.rows FROM `+role)
		_, _ = pool.Exec(ctx, `REVOKE ALL ON SCHEMA `+sc+` FROM `+role)
		_, _ = pool.Exec(ctx, `DROP ROLE `+role)
	})
	if err := s.GrantWriter(ctx, role); err != nil {
		t.Fatal(err)
	}
	sc, _ := s.schema()
	var privs string
	err := pool.QueryRow(ctx, `SELECT string_agg(privilege_type, ',' ORDER BY privilege_type) FROM information_schema.role_table_grants
		WHERE grantee = $1 AND table_schema = $2 AND table_name = 'rows'`, role, strings.Trim(sc, `"`)).Scan(&privs)
	if err != nil || privs != "INSERT,SELECT" {
		t.Fatalf("writer privileges %q (%v), want INSERT,SELECT", privs, err)
	}
	if err := s.GrantWriter(ctx, "bad role; drop"); err == nil {
		t.Error("an unsafe role name was accepted")
	}
}
