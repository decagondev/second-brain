package laws

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/route"
	"example.com/second-brain/internal/core/spaces"
)

// LAW-13 and LAW-20 at ingest with spaces (SB-1005). Random documents with locations and
// source labels, random spaces; after every sync, after a scope or label change and the
// next sync, and after a rebuild, every chunk carries exactly the holders and label the
// rules of sb-ingest(7) give for the spaces now (LAW-13: the index is derived, and the
// rebuild equals it), and a document a restricted space holds, or with a label the cell
// doesn't know, is never indexed below restricted unless its source says so (LAW-20).

// Router is the pipeline's routing rule (route.Route in the real system).
type Router func(sp []ports.Space, kind, location, sourceLabel string) ([]string, string)

type routedDoc struct{ id, location, label string }

func genRouted(r *rand.Rand) ([]routedDoc, []ports.Space) {
	folders := []string{"/Engineering", "/Finance", "/People", "/Specs", "/Other"}
	labels := []string{"", "", "", "public", "internal", "confidential", "restricted", "top-secret"}
	var docs []routedDoc
	for i := 0; i < 4+r.Intn(8); i++ {
		docs = append(docs, routedDoc{fmt.Sprintf("d%d", i), folders[r.Intn(len(folders))] + fmt.Sprintf("/f%d", i), labels[r.Intn(len(labels))]})
	}
	return docs, genScopes(r)
}

func genScopes(r *rand.Rand) []ports.Space {
	folders := []string{"/Engineering", "/Finance", "/People", "/Specs"}
	var sp []ports.Space
	for _, n := range []string{"eng", "finance", "people", "product"}[:1+r.Intn(4)] {
		s := ports.Space{ID: spaces.ID(n), Name: n, Label: route.Ladder[r.Intn(4)], Scope: map[string][]string{"docs": {}}}
		for _, f := range folders {
			if r.Intn(3) == 0 {
				s.Scope["docs"] = append(s.Scope["docs"], f+"/**")
			}
		}
		sp = append(sp, s)
	}
	return sp
}

func law13Spaces(r *rand.Rand, rt Router) error {
	ctx := context.Background()
	docs, current := genRouted(r)
	src := mock.NewSource("docs")
	idx := memindex.New()
	p := ingest.Pipeline{Tenant: "acme-logistics", Release: "rel_test", Connector: mock.New(src, false),
		Embedder: hashing.Embedder{}, Index: idx, Route: rt,
		Spaces: func(context.Context) ([]ports.Space, error) { return current, nil }}
	for _, d := range docs {
		src.Put(d.id, "Title "+d.id, "Text of "+d.id, "ana@acme.example")
		src.Place(d.id, d.location, d.label)
	}
	byID := map[string]routedDoc{}
	for _, d := range docs {
		byID[d.id] = d
	}
	check := func(when string) error {
		cs, err := idx.All(ctx, "acme-logistics")
		if err != nil {
			return err
		}
		if len(cs) == 0 {
			return fmt.Errorf("%s: nothing indexed", when)
		}
		for _, c := range cs {
			d := byID[c.SourceID]
			wantH, wantL := route.Route(current, "docs", d.location, d.label)
			if !reflect.DeepEqual(nonEmpty(c.Spaces), nonEmpty(wantH)) || c.Label != wantL {
				return fmt.Errorf("%s: %s at %s (source label %q) has %v %q, want %v %q (spaces %+v)",
					when, d.id, d.location, d.label, c.Spaces, c.Label, wantH, wantL, current)
			}
			if c.Routing != route.Fingerprint(current) {
				return fmt.Errorf("%s: %s was routed under another configuration", when, d.id)
			}
			if err := law20Label(current, d, c); err != nil {
				return fmt.Errorf("%s: %w", when, err)
			}
		}
		return nil
	}
	cur, err := p.Sync(ctx, "")
	if err != nil {
		return err
	}
	if err := check("after the first sync"); err != nil {
		return err
	}
	before, _ := idx.All(ctx, "acme-logistics")
	if err := p.Rebuild(ctx); err != nil {
		return err
	}
	after, _ := idx.All(ctx, "acme-logistics")
	if a, b := routingSet(before), routingSet(after); !reflect.DeepEqual(a, b) {
		return fmt.Errorf("a rebuild routed differently:\n%v\n%v", a, b)
	}
	// A manifest apply changes scopes and labels; the next sync re-routes.
	current = genScopes(r)
	if _, err := p.Sync(ctx, cur); err != nil {
		return err
	}
	return check("after a scope change and the next sync")
}

// law20Label is LAW-20 at ingest: without a known source label, a document a restricted
// space holds is restricted, and a source label the cell doesn't know is restricted.
func law20Label(sp []ports.Space, d routedDoc, c ports.Chunk) error {
	known := false
	for _, l := range route.Ladder {
		known = known || l == d.label
	}
	if d.label != "" && !known && c.Label != "restricted" {
		return fmt.Errorf("%s has the unknown source label %q and was indexed as %q", d.id, d.label, c.Label)
	}
	if d.label != "" {
		return nil
	}
	for _, s := range sp {
		if s.Label == "restricted" && contains(c.Spaces, s.ID) && c.Label != "restricted" {
			return fmt.Errorf("%s is held by restricted %s and was indexed as %q", d.id, s.ID, c.Label)
		}
	}
	return nil
}

func nonEmpty(xs []string) []string {
	if len(xs) == 0 {
		return nil
	}
	return xs
}

func routingSet(cs []ports.Chunk) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID + "|" + c.Label + "|" + strings.Join(c.Spaces, ",") + "|" + c.Routing
	}
	sort.Strings(out)
	return out
}
