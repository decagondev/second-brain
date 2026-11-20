package spaces_test

import (
	"context"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/route"
	"example.com/second-brain/internal/core/spaces"
)

const tn = "acme-logistics"

// SB-1401: a document that can't be fetched while re-routing is taken out of the index,
// and the sync goes on instead of failing for good.
func TestRerouteNeverWedgesTheSync(t *testing.T) {
	ctx := context.Background()
	src := mock.NewSource("docs")
	src.PutAt("d1", "Forecast", "numbers", "/Finance/f", "", "ana@acme.example")
	src.PutAt("d2", "Plan", "plans", "/Finance/p", "", "ana@acme.example")
	idx := memindex.New()
	sp := []ports.Space{{ID: spaces.ID("finance"), Tenant: tn, Name: "finance", Members: []string{"group:finance"}, Label: "restricted",
		Scope: map[string][]string{"docs": {"/Finance/**"}}}}
	p := ingest.Pipeline{Tenant: tn, Release: "rel_t", Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: idx,
		Spaces: func(context.Context) ([]ports.Space, error) { return sp, nil }}
	cur, err := p.Sync(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	src.Delete("d1")
	sp[0].Scope = map[string][]string{"docs": {"/Finance/**", "/Shared/**"}} // a scope change: re-route everything
	dropped := 0
	p.RerouteDropped = func(string) { dropped++ }
	for i := 0; i < 2; i++ {
		if cur, err = p.Sync(ctx, cur); err != nil {
			t.Fatalf("sync %d: %v", i, err)
		}
	}
	if _, found, _ := idx.Indexed(ctx, tn, ports.SourceRef{Kind: "docs", ID: "d1"}); found {
		t.Fatal("a document the source can't give stayed in the index under the old routing")
	}
	if _, found, _ := idx.Indexed(ctx, tn, ports.SourceRef{Kind: "docs", ID: "d2"}); !found {
		t.Fatal("the other document was lost")
	}
}

// SB-1401: a sensitive field allowed by one space goes only to the spaces that allow it.
func TestSensitiveFieldsNarrowTheHolders(t *testing.T) {
	ctx := context.Background()
	src := mock.NewSource("hris")
	src.Declare(ports.Field{Name: "name"}, ports.Field{Name: "payRate", Sensitive: true})
	src.PutAt("e1", "Ana", "", "/employees/ops/e1", "", "ana@acme.example")
	src.PutFields("e1", "Ana", map[string]string{"name": "Ana", "payRate": "52000"}, "ana@acme.example")
	idx := memindex.New()
	sp := []ports.Space{
		{ID: spaces.ID("people"), Tenant: tn, Name: "people", Members: []string{"group:people"}, Label: "restricted",
			Scope: map[string][]string{"hris": {"/employees/**"}}, Fields: map[string][]string{"hris": {"payRate"}}},
		{ID: spaces.ID("finance"), Tenant: tn, Name: "finance", Members: []string{"group:finance"}, Label: "restricted",
			Scope: map[string][]string{"hris": {"/employees/**"}}},
	}
	p := ingest.Pipeline{Tenant: tn, Release: "rel_t", Connector: mock.New(src, false), Embedder: hashing.Embedder{}, Index: idx,
		Spaces: func(context.Context) ([]ports.Space, error) { return sp, nil }}
	if _, err := p.Sync(ctx, ""); err != nil {
		t.Fatal(err)
	}
	all, _ := idx.All(ctx, tn)
	if len(all) == 0 {
		t.Fatal("nothing indexed")
	}
	for _, c := range all {
		if len(c.Spaces) != 1 || c.Spaces[0] != spaces.ID("people") {
			t.Fatalf("a chunk with payRate is held by %v; only people allows it", c.Spaces)
		}
	}
}

// SB-1401: changing a field allow-list changes the routing fingerprint, so it re-ingests.
func TestFieldAllowListChangesRoute(t *testing.T) {
	a := []ports.Space{{ID: "spc_people", Label: "restricted", Scope: map[string][]string{"hris": {"*"}}, Fields: map[string][]string{"hris": {"ssn"}}}}
	b := []ports.Space{{ID: "spc_people", Label: "restricted", Scope: map[string][]string{"hris": {"*"}}}}
	if route.Fingerprint(a) == route.Fingerprint(b) {
		t.Fatal("removing a field from an allow-list left the fingerprint unchanged")
	}
}
