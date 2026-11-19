package index

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// open gives a fresh tenant name per test so schemas never collide; skips without a DSN.
func open(t *testing.T, tenant string) Index {
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
	x := Index{Pool: pool, Tenant: tenant, Dim: dim}
	if err := x.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	sc, _ := x.schema()
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DROP SCHEMA "+sc+" CASCADE") })
	return x
}

func name(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

const dim = 8

// embed is a tiny stand-in embedder (adapters may not import sibling adapters).
func embed(text string) []float32 {
	v := make([]float32, dim)
	for _, r := range text {
		v[int(r)%dim]++
	}
	return v
}

func chunk(tenant, id, text string, acl ...string) ports.Chunk {
	return ports.Chunk{ID: id, Tenant: tenant, Cite: "docs:" + id + "@1", Kind: "docs", SourceID: id, Version: 1,
		Text: text, ContentSHA256: "h", ACLPrefilter: acl, Vector: embed(text)}
}

func TestIndexAndSearch(t *testing.T) {
	ctx := context.Background()
	acme := name("acme")
	x := open(t, acme)
	for _, c := range []ports.Chunk{
		chunk(acme, "a", "refund approval limit is five hundred", "ann@acme.example"),
		chunk(acme, "b", "holiday rota for the warehouse", "ann@acme.example"),
		chunk(acme, "c", "refund policy for finance only", "group:finance"),
	} {
		if err := x.Replace(ctx, acme, ports.SourceRef{Kind: "docs", ID: c.SourceID}, 1, []ports.Chunk{c}); err != nil {
			t.Fatal(err)
		}
	}
	s, _ := tenancy.NewIssuer(acme).Issue(acme, "usr_ann", nil)
	got, err := x.Search(ctx, s, []string{"ann@acme.example"}, ports.Query{Text: "refund", Vector: embed("refund limit")})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]ports.Candidate{}
	for _, c := range got {
		ids[c.Chunk.ID] = c
	}
	if _, leaked := ids["c"]; leaked {
		t.Fatal("prefilter not applied in the query")
	}
	if ids["a"].KeywordRank != 1 || ids["a"].VectorRank == 0 {
		t.Fatalf("ranks: %+v", ids)
	}
	if v, found, _ := x.Indexed(ctx, acme, ports.SourceRef{Kind: "docs", ID: "a"}); !found || v != 1 {
		t.Fatal("indexed version")
	}
	if err := x.Drop(ctx, acme); err != nil {
		t.Fatal(err)
	}
	if all, _ := x.All(ctx, acme); len(all) != 0 {
		t.Fatal("drop left chunks")
	}
}

// A bound index refuses other tenants, by name and by scope.
func TestOtherTenantRefused(t *testing.T) {
	ctx := context.Background()
	acme := name("acme")
	x := open(t, acme)
	beta, _ := tenancy.NewIssuer("beta-freight").Issue("beta-freight", "usr_b", nil)
	if _, err := x.Search(ctx, beta, []string{"anyone"}, ports.Query{Text: "x"}); !errors.Is(err, ErrOtherTenant) {
		t.Fatalf("search: %v", err)
	}
	if err := x.Replace(ctx, "beta-freight", ports.SourceRef{Kind: "docs", ID: "a"}, 1, nil); !errors.Is(err, ErrOtherTenant) {
		t.Fatalf("replace: %v", err)
	}
	c := chunk("beta-freight", "a", "x", "anyone")
	if err := x.Replace(ctx, acme, ports.SourceRef{Kind: "docs", ID: "a"}, 1, []ports.Chunk{c}); !errors.Is(err, ErrOtherTenant) {
		t.Fatalf("foreign chunk: %v", err)
	}
}
